package application

import (
	"context"
	"errors"
	"math/big"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

type RateRepository interface {
	SaveRateObservation(context.Context, valuation.Observation) (valuation.Observation, error)
	RateObservations(context.Context, money.Asset, money.Asset, calendar.Date, bool) ([]valuation.Observation, error)
}

type Sources interface {
	USDToRUBCandidates(context.Context, calendar.Date, time.Time) ([]valuation.Observation, error)
	CryptoUSD(context.Context, money.Asset, calendar.Date, time.Time) (valuation.Observation, error)
	CryptoCurrent(context.Context, time.Time) ([]valuation.Observation, error)
}

type Service struct {
	Repository RateRepository
	Sources    Sources
	Now        func() time.Time
}

type Reference struct {
	Rate      money.Rate
	Legs      []valuation.Observation
	Coverage  reporting.Coverage
	Freshness reporting.Freshness
	Reason    string
}

func (s Service) Reference(ctx context.Context, base, quote money.Asset, date calendar.Date, current bool) (Reference, error) {
	if _, err := money.ParseAsset(string(base)); err != nil {
		return Reference{}, err
	}
	if _, err := money.ParseAsset(string(quote)); err != nil {
		return Reference{}, err
	}
	if base == quote || date.String() == "" || s.Repository == nil || s.Sources == nil || s.Now == nil {
		return Reference{}, money.ErrInvalidRate
	}
	if dateTime(date).After(s.Now().UTC().Truncate(24 * time.Hour)) {
		return Reference{}, calendar.ErrInvalidTime
	}
	observations := make([]valuation.Observation, 0, 2)
	stale := false
	mismatch := false
	for _, asset := range []money.Asset{base, quote} {
		if asset == money.USD {
			continue
		}
		leg, old, different, err := s.leg(ctx, asset, date, current)
		if err != nil {
			if !errors.Is(err, valuation.ErrRateUnavailable) && !errors.Is(err, valuation.ErrHistoryRange) {
				return Reference{}, err
			}
			reason := "missing_observation"
			if errors.Is(err, valuation.ErrHistoryRange) {
				reason = "history_out_of_range"
			}
			coverage, _ := reporting.NewCoverage(reporting.NoCoverage, []string{reason})
			return Reference{Coverage: coverage, Freshness: reporting.UnknownFreshness, Reason: reason}, nil
		}
		observations = append(observations, leg)
		stale = stale || old
		mismatch = mismatch || different
	}
	one, _ := money.NewMoney("1", base)
	_, rate, used, err := valuation.Convert(one, quote, observations)
	if err != nil {
		return Reference{}, err
	}
	reasons := []string{}
	if stale {
		reasons = append(reasons, "rate_stale")
	}
	if mismatch {
		reasons = append(reasons, "source_mismatch")
	}
	if len(used) == 2 && used[0].EffectiveAt.String()[:10] != used[1].EffectiveAt.String()[:10] {
		reasons = append(reasons, "rate_dates_differ")
	}
	state := reporting.Complete
	if len(reasons) != 0 {
		state = reporting.Partial
	}
	coverage, _ := reporting.NewCoverage(state, reasons)
	freshness := reporting.Fresh
	if stale {
		freshness = reporting.Stale
	}
	return Reference{Rate: rate, Legs: used, Coverage: coverage, Freshness: freshness}, nil
}

func (s Service) leg(ctx context.Context, asset money.Asset, date calendar.Date, current bool) (valuation.Observation, bool, bool, error) {
	base, quote := asset, money.USD
	if asset == money.RUB {
		base, quote = money.USD, money.RUB
	}
	known, err := s.Repository.RateObservations(ctx, base, quote, date, current)
	if err != nil {
		return valuation.Observation{}, false, false, err
	}
	selected := preferred(known)
	now := s.Now().UTC()
	if len(known) > 0 && (!current && (asset == money.RUB && (selected.Transport == "xml_daily" || now.Sub(selected.FetchedAt.Time()) < time.Hour) || asset != money.RUB && now.Sub(dateTime(date)) <= 365*24*time.Hour) || current && now.Sub(selected.FetchedAt.Time()) < time.Hour) {
		return selected, sourceStale(selected, now, current), crosscheckMismatch(known), nil
	}
	if asset != money.RUB && !current && now.Truncate(24*time.Hour).Sub(dateTime(date)) > 365*24*time.Hour {
		if len(known) != 0 {
			return selected, true, crosscheckMismatch(known), nil
		}
		return valuation.Observation{}, false, false, valuation.ErrHistoryRange
	}
	var fetched valuation.Observation
	if asset == money.RUB {
		var candidates []valuation.Observation
		candidates, err = s.Sources.USDToRUBCandidates(ctx, date, now)
		if err == nil {
			for _, candidate := range candidates {
				if _, saveErr := s.Repository.SaveRateObservation(ctx, candidate); saveErr != nil {
					return valuation.Observation{}, false, false, saveErr
				}
			}
			known, err = s.Repository.RateObservations(ctx, base, quote, date, current)
			if err != nil {
				return valuation.Observation{}, false, false, err
			}
			fetched = preferred(known)
		}
	} else if current {
		var batch []valuation.Observation
		batch, err = s.Sources.CryptoCurrent(ctx, now)
		if err == nil {
			for _, value := range batch {
				persisted, saveErr := s.Repository.SaveRateObservation(ctx, value)
				if saveErr != nil {
					return valuation.Observation{}, false, false, saveErr
				}
				if persisted.Rate.Base() == asset {
					fetched = persisted
				}
			}
		}
	} else {
		fetched, err = s.Sources.CryptoUSD(ctx, asset, date, now)
	}
	if err == nil && fetched.ID != "" && asset != money.RUB && !current {
		fetched, err = s.Repository.SaveRateObservation(ctx, fetched)
	}
	if err == nil && fetched.ID != "" {
		return fetched, sourceStale(fetched, now, current), crosscheckMismatch(known), nil
	}
	if len(known) != 0 {
		return selected, true, crosscheckMismatch(known), nil
	}
	if err == nil {
		err = valuation.ErrRateUnavailable
	}
	if errors.Is(err, valuation.ErrHistoryRange) {
		return valuation.Observation{}, false, false, err
	}
	return valuation.Observation{}, false, false, valuation.ErrRateUnavailable
}

func sourceStale(value valuation.Observation, now time.Time, current bool) bool {
	return current && value.Source == "coingecko" && now.Sub(value.EffectiveAt.Time()) > time.Hour
}

func crosscheckMismatch(values []valuation.Observation) bool {
	var direct, fallback *valuation.Observation
	for i := range values {
		switch values[i].Transport {
		case "xml_daily":
			direct = &values[i]
		case "frankfurter_v2":
			fallback = &values[i]
		}
	}
	if direct == nil || fallback == nil || !direct.EffectiveAt.Time().Equal(fallback.EffectiveAt.Time()) {
		return false
	}
	a, _ := new(big.Rat).SetString(direct.Rate.Value())
	b, _ := new(big.Rat).SetString(fallback.Rate.Value())
	return a.Cmp(b) != 0
}

func preferred(values []valuation.Observation) valuation.Observation {
	if len(values) == 0 {
		return valuation.Observation{}
	}
	best := values[0]
	for _, value := range values[1:] {
		if value.EffectiveAt.Time().After(best.EffectiveAt.Time()) || value.EffectiveAt == best.EffectiveAt && value.Transport == "xml_daily" {
			best = value
		}
	}
	return best
}

func dateTime(date calendar.Date) time.Time {
	value, _ := time.Parse(time.DateOnly, date.String())
	return value.UTC()
}
