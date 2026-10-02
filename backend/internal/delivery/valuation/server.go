package valuation

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	access "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	valuation "github.com/pchkauu/want-keep/backend/internal/valuation"
	rates "github.com/pchkauu/want-keep/backend/internal/valuation/application"
)

type ReadTransactions interface {
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	ValuationSnapshot(context.Context, household.Principal, string, uint64, int, money.Asset) (valuation.Snapshot, bool, error)
	LatestPlatformQuote(context.Context, household.Principal, string, money.Asset, money.Asset, valuation.Direction, money.Money) (valuation.PlatformQuote, bool, error)
}

type Server struct {
	rates    rates.Service
	accounts *accounts.Service
	ledger   *ledger.Queries
	sessions security.Sessions
	reads    ReadTransactions
	guard    *security.Guard
	boundary *contract.Boundary
	now      func() time.Time
	mux      *http.ServeMux
}

func New(rateService rates.Service, accountsService *accounts.Service, ledgerQueries *ledger.Queries, sessions security.Sessions, reads ReadTransactions, config security.Config, now func() time.Time) (*Server, error) {
	if rateService.Repository == nil || rateService.Sources == nil || accountsService == nil || ledgerQueries == nil || sessions == nil || reads == nil || now == nil {
		return nil, contract.ErrInvalidRequest
	}
	guard, err := security.New(config)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	s := &Server{rates: rateService, accounts: accountsService, ledger: ledgerQueries, sessions: sessions, reads: reads, guard: guard, boundary: boundary, now: now, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/v1/rates", s.listRates)
	s.mux.HandleFunc("GET /api/v1/reports/valuation", s.report)
	return s, nil
}

func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *Server) write(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		s.problem(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}

func (s *Server) problem(w http.ResponseWriter, err error) {
	response := (contract.ErrorConverter{}).ToResponse(err, uuid.NewString())
	switch {
	case errors.Is(err, access.ErrUnauthorized):
		response.Status, response.Body.Code, response.Body.Message = 401, "unauthorized", "Sign in to continue."
	case errors.Is(err, access.ErrAttempt):
		response.Status, response.Body.Code, response.Body.Message = 400, "invalid_request", "Check the request."
	case errors.Is(err, valuation.ErrRateUnavailable):
		response.Status, response.Body.Code, response.Body.Message = 503, "valuation_unavailable", "Reference rate is unavailable."
	case errors.Is(err, money.ErrInvalidRate), errors.Is(err, calendar.ErrInvalidTime), errors.Is(err, contract.ErrInvalidRequest):
		response.Status, response.Body.Code, response.Body.Message = 400, "invalid_request", "Check the rate request."
	}
	s.write(w, response.Status, response.Body)
}

type rateRequest struct {
	base, quote money.Asset
	date        calendar.Date
	current     bool
	provider    string
	amount      money.Money
	direction   valuation.Direction
}

func (s *Server) parseRateRequest(r *http.Request, today calendar.Date) (rateRequest, error) {
	values := r.URL.Query()
	for key, entries := range values {
		if len(entries) != 1 {
			return rateRequest{}, contract.ErrInvalidRequest
		}
		switch key {
		case "base", "quote", "date", "provider", "amount", "direction":
		default:
			return rateRequest{}, contract.ErrInvalidRequest
		}
	}
	base, err := money.ParseAsset(values.Get("base"))
	if err != nil {
		return rateRequest{}, contract.ErrInvalidRequest
	}
	quote, err := money.ParseAsset(values.Get("quote"))
	if err != nil || base == quote {
		return rateRequest{}, contract.ErrInvalidRequest
	}
	out := rateRequest{base: base, quote: quote, provider: values.Get("provider"), direction: valuation.Direction(values.Get("direction")), current: values.Get("date") == ""}
	if out.current {
		out.date = today
	} else {
		out.date, err = calendar.ParseDate(values.Get("date"))
	}
	if err != nil || out.date.String() > today.String() {
		return rateRequest{}, contract.ErrInvalidRequest
	}
	if out.provider != "" {
		if !out.current {
			return rateRequest{}, contract.ErrInvalidRequest
		}
		switch out.provider {
		case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
		default:
			return rateRequest{}, contract.ErrInvalidRequest
		}
		out.amount, err = money.NewMoney(values.Get("amount"), base)
		if err != nil || out.amount.Sign() <= 0 || out.direction != valuation.SellBase && out.direction != valuation.BuyBase {
			return rateRequest{}, contract.ErrInvalidRequest
		}
	} else if values.Get("amount") != "" || out.direction != "" {
		return rateRequest{}, contract.ErrInvalidRequest
	}
	return out, nil
}

func (s *Server) listRates(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	var zone calendar.Timezone
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var readErr error
		zone, readErr = s.reads.AccountTimezone(ctx, access.Principal)
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	today, err := householdToday(s.now(), zone)
	if err != nil {
		s.problem(w, err)
		return
	}
	request, err := s.parseRateRequest(r, today)
	if err != nil {
		s.problem(w, err)
		return
	}
	if request.provider != "" {
		s.platformRate(w, r, access.Principal, request)
		return
	}
	result, err := s.rates.Reference(r.Context(), request.base, request.quote, request.date, request.current)
	if err != nil {
		s.problem(w, err)
		return
	}
	if result.Reason != "" {
		s.unavailableRate(w, request, "valuation_unavailable", result.Reason)
		return
	}
	quality, err := s.quality(result.Coverage, result.Freshness, result.Legs)
	if err != nil {
		s.problem(w, err)
		return
	}
	rate, err := (contract.MoneyConverter{}).RateToDTO(result.Rate)
	if err != nil {
		s.problem(w, err)
		return
	}
	legs := make([]generated.ValuationLeg, 0, len(result.Legs))
	observed, fetched := result.Legs[0].EffectiveAt, result.Legs[0].FetchedAt
	granularity := generated.ValuationObservationGranularity("daily")
	ids := []string{string(request.base), string(request.quote), request.date.String()}
	for _, leg := range result.Legs {
		dto, convertErr := toLeg(leg)
		if convertErr != nil {
			s.problem(w, convertErr)
			return
		}
		legs = append(legs, dto)
		ids = append(ids, leg.ID)
		if leg.EffectiveAt.Time().After(observed.Time()) {
			observed = leg.EffectiveAt
		}
		if leg.FetchedAt.Time().After(fetched.Time()) {
			fetched = leg.FetchedAt
		}
		if leg.Granularity == "instant" {
			granularity = "instant"
		}
	}
	identifier := uuid.NewSHA1(uuid.NameSpaceURL, []byte(strings.Join(ids, "/"))).String()
	d := generated.ValuationObservation{Id: identifier, Revision: 1, Rate: rate, RequestedDate: request.date.String(), ObservedAt: observed.String(), FetchedAt: fetched.String(), Source: "reference", Method: "reference", Granularity: granularity, Legs: legs, Quality: quality}
	var item generated.RateObservation
	if err = item.FromValuationObservation(d); err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, generated.ValuationObservationPage{Items: []generated.RateObservation{item}, Quality: quality})
}

func (s *Server) platformRate(w http.ResponseWriter, r *http.Request, principal household.Principal, request rateRequest) {
	var quote valuation.PlatformQuote
	var found bool
	err := s.reads.WithinFinancialRead(r.Context(), principal, func(ctx context.Context) error {
		var readErr error
		quote, found, readErr = s.reads.LatestPlatformQuote(ctx, principal, request.provider, request.base, request.quote, request.direction, request.amount)
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	if !found {
		s.unavailableRate(w, request, "quote_unavailable", "incomplete_quote")
		return
	}
	freshness := reporting.Fresh
	if s.now().Sub(quote.ObservedAt.Time()) > 5*time.Minute {
		freshness = reporting.Stale
	}
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	quality, err := s.quality(coverage, freshness, nil)
	if err != nil {
		s.problem(w, err)
		return
	}
	observed, fetched := generated.Instant(quote.ObservedAt.String()), generated.Instant(quote.FetchedAt.String())
	quality.AsOf, quality.FetchedAt = &observed, &fetched
	rate, err := (contract.MoneyConverter{}).RateToDTO(quote.Rate)
	if err != nil {
		s.problem(w, err)
		return
	}
	amount, err := (contract.MoneyConverter{}).ToDTO(quote.Amount)
	if err != nil {
		s.problem(w, err)
		return
	}
	fees := make([]generated.Money, 0, len(quote.Fees))
	for _, fee := range quote.Fees {
		value, conversionErr := (contract.MoneyConverter{}).ToDTO(fee)
		if conversionErr != nil {
			s.problem(w, conversionErr)
			return
		}
		fees = append(fees, value)
	}
	value := generated.PlatformQuote{Method: "platform_quote", Provider: generated.Provider(quote.Provider), Direction: generated.PlatformQuoteDirection(quote.Direction), Base: generated.Asset(request.base), Quote: generated.Asset(request.quote), ApplicableAmount: generated.PositiveMoney(amount), Rate: rate, ObservedAt: quote.ObservedAt.String(), Fees: fees, FeeCoverage: generated.PlatformQuoteFeeCoverage(quote.FeeCoverage), SpreadCoverage: generated.PlatformQuoteSpreadCoverage(quote.SpreadCoverage), Quality: quality}
	var item generated.RateObservation
	if err = item.FromPlatformQuote(value); err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, generated.ValuationObservationPage{Items: []generated.RateObservation{item}, Quality: quality})
}

func (s *Server) unavailableRate(w http.ResponseWriter, request rateRequest, reason, detail string) {
	coverage, _ := reporting.NewCoverage(reporting.NoCoverage, []string{detail})
	quality, err := s.quality(coverage, reporting.UnknownFreshness, nil)
	if err != nil {
		s.problem(w, err)
		return
	}
	value := generated.UnavailableRate{Method: "unavailable", Base: generated.Asset(request.base), Quote: generated.Asset(request.quote), RequestedDate: request.date.String(), Reason: generated.UnavailableRateReason(reason), Detail: generated.UnavailableRateDetail(detail), Quality: quality}
	var item generated.RateObservation
	if err = item.FromUnavailableRate(value); err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, generated.ValuationObservationPage{Items: []generated.RateObservation{item}, Quality: quality})
}

func toLeg(value valuation.Observation) (generated.ValuationLeg, error) {
	rate, err := (contract.MoneyConverter{}).RateToDTO(value.Rate)
	if err != nil {
		return generated.ValuationLeg{}, err
	}
	link := "https://www.cbr.ru/"
	if value.Source == "coingecko" {
		link = "https://www.coingecko.com/en/api"
	}
	return generated.ValuationLeg{ProviderAssetId: value.ProviderAssetID, Source: value.Source, AttributionUrl: link, Transport: value.Transport, RequestedDate: value.RequestedDate.String(), EffectiveAt: value.EffectiveAt.String(), FetchedAt: value.FetchedAt.String(), Granularity: generated.ValuationLegGranularity(value.Granularity), Revision: generated.Revision(value.Revision), Rate: rate}, nil
}

func (s *Server) quality(coverage reporting.Coverage, freshness reporting.Freshness, legs []valuation.Observation) (generated.DataQuality, error) {
	dto, err := s.boundary.CoverageToDTO(coverage)
	if err != nil {
		return generated.DataQuality{}, err
	}
	result := generated.DataQuality{Coverage: dto, Freshness: generated.Freshness(freshness)}
	if len(legs) != 0 {
		asOf, fetched := legs[0].EffectiveAt, legs[0].FetchedAt
		for _, leg := range legs[1:] {
			if leg.EffectiveAt.Time().Before(asOf.Time()) {
				asOf = leg.EffectiveAt
			}
			if leg.FetchedAt.Time().After(fetched.Time()) {
				fetched = leg.FetchedAt
			}
		}
		at, got := generated.Instant(asOf.String()), generated.Instant(fetched.String())
		result.AsOf, result.FetchedAt = &at, &got
	}
	return result, nil
}
