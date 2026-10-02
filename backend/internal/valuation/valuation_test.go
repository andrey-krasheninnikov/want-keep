package valuation

import (
	"errors"
	"math/big"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func observation(t *testing.T, asset money.Asset, rate string) Observation {
	t.Helper()
	date, _ := calendar.ParseDate("2026-09-01")
	at, _ := calendar.ParseInstant("2026-09-01T00:00:00Z")
	base, quote := asset, money.USD
	providerID := map[money.Asset]string{money.BTC: "bitcoin", money.ETH: "ethereum", money.USDT: "tether", money.USDC: "usd-coin"}[asset]
	source, transport, granularity := "coingecko", "demo_history", "daily"
	if asset == money.RUB {
		base, quote = money.USD, money.RUB
		providerID, source, transport = "R01235", "cbr", "xml_daily"
	}
	r, err := money.NewRate(base, quote, rate)
	if err != nil {
		t.Fatal(err)
	}
	return Observation{ID: string(asset), ProviderAssetID: providerID, Source: source, Transport: transport, Revision: 1, RequestedDate: date, EffectiveAt: at, FetchedAt: at, Granularity: granularity, Rate: r}
}

func TestCrossRatesKeepNativeAmountsAndSeparateStablecoins(t *testing.T) {
	legs := []Observation{observation(t, money.RUB, "90"), observation(t, money.USDT, "0.997"), observation(t, money.BTC, "50000.00000000000001")}
	native, _ := money.NewMoney("9000.00000000000001", money.RUB)
	converted, rate, used, err := Convert(native, money.USDT, legs)
	if err != nil || converted.Asset() != money.USDT || converted.Amount() == "100" || rate.Value() == "1" || len(used) != 2 || native.Amount() != "9000.00000000000001" {
		t.Fatalf("cross conversion lost provenance or precision: %s %s %#v %v", converted.Amount(), rate.Value(), used, err)
	}
	btc, _ := money.NewMoney("0.00000000000000001", money.BTC)
	result, _, _, err := Convert(btc, money.USD, legs)
	if err != nil || result.Amount() != "0.0000000000005000000000000000001" {
		t.Fatalf("BTC conversion: %s %v", result.Amount(), err)
	}
	for _, asset := range []money.Asset{money.USDC, money.ETH} {
		amount, _ := money.NewMoney("1", asset)
		if _, _, _, err := Convert(amount, money.USD, legs); !errors.Is(err, ErrRateUnavailable) {
			t.Fatalf("missing %s price accepted: %v", asset, err)
		}
	}
}

func TestAllSupportedAssetsKeepNativePrecision(t *testing.T) {
	legs := []Observation{
		observation(t, money.RUB, "90.01"),
		observation(t, money.USDT, "0.998"),
		observation(t, money.USDC, "1.002"),
		observation(t, money.BTC, "60000.123456789"),
		observation(t, money.ETH, "3000.000000001"),
	}
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		native, _ := money.NewMoney("0.000000000000000000000001", asset)
		got, _, _, err := Convert(native, money.USD, legs)
		if err != nil || got.Asset() != money.USD || got.Sign() <= 0 || native.Amount() != "0.000000000000000000000001" {
			t.Fatalf("%s: %s %v", asset, got.Amount(), err)
		}
	}
}

func TestDecimalHalfEven(t *testing.T) {
	for _, tc := range []struct{ source, want string }{{"1/2", "0.5"}, {"1/8", "0.125"}, {"-1/2", "-0.5"}} {
		r, ok := newRat(tc.source)
		if !ok || decimal(r, 3) != tc.want {
			t.Fatalf("decimal(%s) = %s", tc.source, decimal(r, 3))
		}
	}
}

func newRat(value string) (*big.Rat, bool) { return new(big.Rat).SetString(value) }
