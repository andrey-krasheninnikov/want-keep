package valuation

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
	rateapp "github.com/pchkauu/want-keep/backend/internal/valuation/application"
)

type rateMemory struct{ items []valuation.Observation }

func (m *rateMemory) SaveRateObservation(_ context.Context, value valuation.Observation) (valuation.Observation, error) {
	m.items = append(m.items, value)
	return value, nil
}

func TestCurrentTotalDistinguishesKnownPartialFromUnknown(t *testing.T) {
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{boundary: boundary}
	sum, _ := money.NewMoney("90.123456789", money.RUB)
	value, err := server.currentTotalAmount("current_owned_total", currentTotal{sum: sum, known: true, reasons: []string{"source_mismatch"}, stale: true, inputs: []generated.CalculationInput{}})
	coverage, coverageErr := value.Quality.Coverage.AsIncompleteCoverage()
	if err != nil || coverageErr != nil || value.Name != "current_owned_total" || coverage.State != "partial" || value.Quality.Freshness != generated.Freshness(reporting.Stale) {
		t.Fatalf("partial known total: %+v %v", value, err)
	}
	known, err := value.Reporting.AsKnownAmount()
	if err != nil || known.Value.Amount != sum.Amount() {
		t.Fatalf("known subtotal lost: %+v %v", known, err)
	}
	value, err = server.currentTotalAmount("current_owned_total", currentTotal{sum: sum, known: false, reasons: []string{"incomplete_accounts"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = value.Reporting.AsMissingAmount(); err != nil {
		t.Fatalf("unknown total became zero: %v", err)
	}
}

func TestReportEndUsesHouseholdDate(t *testing.T) {
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	now := time.Date(2026, 10, 1, 22, 0, 0, 0, time.UTC)
	today, _ := calendar.ParseDate("2026-10-02")
	future, err := reportEndIsFuture(now, zone, today)
	if err != nil || future {
		t.Fatalf("local current date rejected: %v %v", future, err)
	}
	tomorrow, _ := calendar.ParseDate("2026-10-03")
	future, err = reportEndIsFuture(now, zone, tomorrow)
	if err != nil || !future {
		t.Fatalf("future local date accepted: %v %v", future, err)
	}
}

func TestHistoricalSnapshotKeepsKnownValueWithPartialQuality(t *testing.T) {
	boundary, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	server := &Server{boundary: boundary}
	native, _ := money.NewMoney("10", money.USDC)
	converted, _ := money.NewMoney("9.97", money.USD)
	snapshot := &valuation.Snapshot{Reporting: &converted, CoverageReasons: []string{"source_mismatch"}, Freshness: reporting.Stale}
	value, reasons, err := server.historicalAmount(component{operationID: uuid.NewString(), revision: 1, kind: "expense", native: native, snapshot: snapshot}, money.USD)
	if err != nil || len(reasons) != 1 || reasons[0] != "source_mismatch" || value.Quality.Freshness != "stale" {
		t.Fatalf("partial historical quality: %+v %v %v", value, reasons, err)
	}
	known, err := value.Reporting.AsKnownAmount()
	if err != nil || known.Value.Amount != "9.97" {
		t.Fatalf("partial historical value lost: %+v %v", known, err)
	}
}
func (m *rateMemory) RateObservations(_ context.Context, base, quote money.Asset, date calendar.Date, _ bool) ([]valuation.Observation, error) {
	out := []valuation.Observation{}
	for _, value := range m.items {
		if value.Rate.Base() == base && value.Rate.Quote() == quote && value.RequestedDate == date {
			out = append(out, value)
		}
	}
	return out, nil
}

type noFetch struct{}

func (noFetch) USDToRUBCandidates(context.Context, calendar.Date, time.Time) ([]valuation.Observation, error) {
	return nil, valuation.ErrRateUnavailable
}
func (noFetch) CryptoUSD(context.Context, money.Asset, calendar.Date, time.Time) (valuation.Observation, error) {
	return valuation.Observation{}, valuation.ErrRateUnavailable
}
func (noFetch) CryptoCurrent(context.Context, time.Time) ([]valuation.Observation, error) {
	return nil, valuation.ErrRateUnavailable
}

type fixedSession struct{}

func (fixedSession) Me(context.Context, identity.Token) (identityapp.Access, error) {
	return identityapp.Access{}, nil
}

type quoteRead struct{}

func (quoteRead) WithinFinancialRead(ctx context.Context, _ household.Principal, run func(context.Context) error) error {
	return run(ctx)
}
func (quoteRead) AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error) {
	panic("unused")
}
func (quoteRead) ValuationSnapshot(context.Context, household.Principal, string, uint64, int, money.Asset) (valuation.Snapshot, bool, error) {
	panic("unused")
}
func (quoteRead) LatestPlatformQuote(context.Context, household.Principal, string, money.Asset, money.Asset, valuation.Direction, money.Money) (valuation.PlatformQuote, bool, error) {
	return valuation.PlatformQuote{}, false, nil
}

func TestRatesHTTPRequiresSessionAndKeepsSourceLegs(t *testing.T) {
	day, _ := calendar.ParseDate("2026-09-01")
	at, _ := calendar.ParseInstant("2026-09-01T12:00:00Z")
	btcRate, _ := money.NewRate(money.BTC, money.USD, "60000.123456789")
	rubRate, _ := money.NewRate(money.USD, money.RUB, "90.01")
	repo := &rateMemory{items: []valuation.Observation{
		{ID: uuid.NewString(), ProviderAssetID: "bitcoin", Source: "coingecko", Transport: "demo_history", Revision: 1, RequestedDate: day, EffectiveAt: at, FetchedAt: at, Granularity: "daily", Rate: btcRate},
		{ID: uuid.NewString(), ProviderAssetID: "R01235", Source: "cbr", Transport: "xml_daily", Revision: 1, RequestedDate: day, EffectiveAt: at, FetchedAt: at, Granularity: "daily", Rate: rubRate},
	}}
	now := func() time.Time { return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC) }
	service := rateapp.Service{Repository: repo, Sources: noFetch{}, Now: now}
	server, err := New(service, &accounts.Service{}, &ledger.Queries{}, fixedSession{}, quoteRead{}, security.Config{Environment: "test", Origin: "http://localhost:8080"}, now)
	if err != nil {
		t.Fatal(err)
	}
	request := func(path string, authenticated bool) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(http.MethodGet, "http://localhost:8080"+path, nil)
		if authenticated {
			r.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: "synthetic-session"})
		}
		w := httptest.NewRecorder()
		server.ServeHTTP(w, r)
		return w
	}
	path := "/api/v1/rates?base=BTC&quote=RUB&date=2026-09-01"
	if result := request(path, false); result.Code != http.StatusUnauthorized {
		t.Fatalf("unauthenticated: %d", result.Code)
	}
	result := request(path, true)
	if result.Code != http.StatusOK || result.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("reference: %d %s", result.Code, result.Body.String())
	}
	var body struct {
		Items []struct {
			Method string `json:"method"`
			Legs   []struct {
				Source         string `json:"source"`
				AttributionURL string `json:"attributionUrl"`
			} `json:"legs"`
		} `json:"items"`
	}
	if err = json.Unmarshal(result.Body.Bytes(), &body); err != nil || len(body.Items) != 1 || body.Items[0].Method != "reference" || len(body.Items[0].Legs) != 2 {
		t.Fatalf("source legs: %+v %v", body, err)
	}
	if body.Items[0].Legs[0].Source != "coingecko" || body.Items[0].Legs[0].AttributionURL != "https://www.coingecko.com/en/api" || body.Items[0].Legs[1].Source != "cbr" {
		t.Fatalf("source attribution: %+v", body.Items[0].Legs)
	}
	if result = request("/api/v1/rates?base=BTC&quote=RUB&date=2026-09-03", true); result.Code != http.StatusBadRequest {
		t.Fatalf("future date: %d", result.Code)
	}
	if result = request("/api/v1/rates?base=USDT&quote=USD&provider=bybit&amount=100&direction=sell_base", true); result.Code != http.StatusOK || !strings.Contains(result.Body.String(), "quote_unavailable") {
		t.Fatalf("missing provider quote: %d %s", result.Code, result.Body.String())
	}
}
