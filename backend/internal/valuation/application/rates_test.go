package application

import (
	"context"
	"testing"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

type memoryRates struct{ values []valuation.Observation }

func (m *memoryRates) SaveRateObservation(_ context.Context, value valuation.Observation) (valuation.Observation, error) {
	m.values = append(m.values, value)
	return value, nil
}
func (m *memoryRates) RateObservations(_ context.Context, base, quote money.Asset, date calendar.Date, current bool) ([]valuation.Observation, error) {
	out := []valuation.Observation{}
	for _, value := range m.values {
		if value.Rate.Base() == base && value.Rate.Quote() == quote && (current && value.RequestedDate.String() <= date.String() || !current && value.RequestedDate == date) {
			out = append(out, value)
		}
	}
	return out, nil
}

type fixedSources struct {
	rub  []valuation.Observation
	coin []valuation.Observation
}

func (f fixedSources) USDToRUBCandidates(context.Context, calendar.Date, time.Time) ([]valuation.Observation, error) {
	return f.rub, nil
}
func (f fixedSources) CryptoUSD(_ context.Context, asset money.Asset, date calendar.Date, _ time.Time) (valuation.Observation, error) {
	for _, value := range f.coin {
		if value.Rate.Base() == asset && value.Granularity == "daily" && value.RequestedDate == date {
			return value, nil
		}
	}
	return valuation.Observation{}, valuation.ErrRateUnavailable
}
func (f fixedSources) CryptoCurrent(context.Context, time.Time) ([]valuation.Observation, error) {
	return f.coin, nil
}

func rateObservation(t *testing.T, asset money.Asset, amount, day, transport string) valuation.Observation {
	t.Helper()
	date, _ := calendar.ParseDate(day)
	at, _ := calendar.ParseInstant(day + "T00:00:00Z")
	base, quote := asset, money.USD
	source, provider := "coingecko", map[money.Asset]string{money.USDT: "tether", money.USDC: "usd-coin", money.BTC: "bitcoin", money.ETH: "ethereum"}[asset]
	if asset == money.RUB {
		base, quote, source, provider = money.USD, money.RUB, "cbr", "R01235"
	}
	rate, err := money.NewRate(base, quote, amount)
	if err != nil {
		t.Fatal(err)
	}
	granularity := "daily"
	if transport == "demo_simple_price" {
		granularity = "instant"
	}
	return valuation.Observation{ID: day + transport + string(asset), ProviderAssetID: provider, Source: source, Transport: transport, Revision: 1, RequestedDate: date, EffectiveAt: at, FetchedAt: at, Granularity: granularity, Rate: rate}
}

func TestHistoricalReferenceDoesNotFollowCurrentPrice(t *testing.T) {
	oldRub := rateObservation(t, money.RUB, "90", "2026-09-01", "xml_daily")
	oldUSDT := rateObservation(t, money.USDT, "0.997", "2026-09-01", "demo_history")
	repo := &memoryRates{values: []valuation.Observation{oldRub, oldUSDT}}
	now := time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC)
	currentRub := rateObservation(t, money.RUB, "100", "2026-09-02", "xml_daily")
	currentUSDT := rateObservation(t, money.USDT, "1.01", "2026-09-02", "demo_simple_price")
	service := Service{Repository: repo, Sources: fixedSources{rub: []valuation.Observation{currentRub}, coin: []valuation.Observation{currentUSDT}}, Now: func() time.Time { return now }}
	historicalDate, _ := calendar.ParseDate("2026-09-01")
	old, err := service.Reference(context.Background(), money.RUB, money.USDT, historicalDate, false)
	if err != nil || old.Coverage.State() != reporting.Complete || len(old.Legs) != 2 {
		t.Fatalf("historical cross: %#v %v", old, err)
	}
	currentDate, _ := calendar.ParseDate("2026-09-02")
	newRate, err := service.Reference(context.Background(), money.RUB, money.USDT, currentDate, true)
	if err != nil || newRate.Rate.Value() == old.Rate.Value() {
		t.Fatalf("current price did not change independently: %#v %v", newRate, err)
	}
	again, err := service.Reference(context.Background(), money.RUB, money.USDT, historicalDate, false)
	if err != nil || again.Rate.Value() != old.Rate.Value() {
		t.Fatalf("historical reference changed: %#v %v", again, err)
	}
}

func TestMissingStablecoinAndConflictingCrosscheck(t *testing.T) {
	date, _ := calendar.ParseDate("2026-09-01")
	direct := rateObservation(t, money.RUB, "90", date.String(), "xml_daily")
	fallback := rateObservation(t, money.RUB, "91", date.String(), "frankfurter_v2")
	repo := &memoryRates{values: []valuation.Observation{direct, fallback}}
	service := Service{Repository: repo, Sources: fixedSources{}, Now: func() time.Time { return time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC) }}
	result, err := service.Reference(context.Background(), money.RUB, money.USD, date, false)
	if err != nil || result.Rate.Value() != "0.01111111111111111111111111111111111111111111111111111111111111111111111111111111" || result.Coverage.State() != reporting.Partial {
		t.Fatalf("CBR mismatch hidden: %#v %v", result, err)
	}
	missing, err := service.Reference(context.Background(), money.USDC, money.USD, date, false)
	if err != nil || missing.Reason != "missing_observation" || missing.Coverage.State() != reporting.NoCoverage {
		t.Fatalf("missing USDC became parity: %#v %v", missing, err)
	}
}

func TestHistoricalFallbackIsReplacedByDirectSource(t *testing.T) {
	date, _ := calendar.ParseDate("2026-09-01")
	fallback := rateObservation(t, money.RUB, "91", date.String(), "frankfurter_v2")
	direct := rateObservation(t, money.RUB, "90", date.String(), "xml_daily")
	repo := &memoryRates{values: []valuation.Observation{fallback}}
	service := Service{Repository: repo, Sources: fixedSources{rub: []valuation.Observation{direct}}, Now: func() time.Time { return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC) }}
	result, err := service.Reference(context.Background(), money.USD, money.RUB, date, false)
	if err != nil || result.Rate.Value() != "90" || len(result.Legs) != 1 || result.Legs[0].Transport != "xml_daily" {
		t.Fatalf("fallback did not yield to CBR: %+v %v", result, err)
	}
}

func TestCrosscheckIgnoresDecimalScale(t *testing.T) {
	direct := rateObservation(t, money.RUB, "90.0", "2026-09-01", "xml_daily")
	fallback := rateObservation(t, money.RUB, "90", "2026-09-01", "frankfurter_v2")
	if crosscheckMismatch([]valuation.Observation{direct, fallback}) {
		t.Fatal("equal prices with different scales marked inconsistent")
	}
}

func TestCurrentReferenceAllowsNextLocalCalendarDay(t *testing.T) {
	today, _ := calendar.ParseDate("2026-10-02")
	utcDay := rateObservation(t, money.BTC, "60000", "2026-10-01", "demo_simple_price")
	service := Service{Repository: &memoryRates{}, Sources: fixedSources{coin: []valuation.Observation{utcDay}}, Now: func() time.Time { return time.Date(2026, 10, 1, 22, 0, 0, 0, time.UTC) }}
	current, err := service.Reference(context.Background(), money.BTC, money.USD, today, true)
	if err != nil || current.Reason != "" || current.Rate.Value() != "60000" {
		t.Fatalf("local current reference: %+v %v", current, err)
	}
	historical, err := service.Reference(context.Background(), money.BTC, money.USD, today, false)
	if err != nil || historical.Reason != "missing_observation" {
		t.Fatalf("local historical rate should remain pending: %+v %v", historical, err)
	}
}
