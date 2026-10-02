package rates

import (
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (fn roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return fn(r) }

func response(body string, status int) *http.Response {
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body)), ContentLength: int64(len(body)), Header: make(http.Header)}
}

type allowQuota struct{}

func (allowQuota) ReserveRateCall(context.Context, time.Time) error { return nil }

func TestCBRPrimaryAndFilteredFallback(t *testing.T) {
	var fallbackCalled bool
	client := New(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/cbr":
			if r.URL.Query().Get("date_req") != "01/09/2026" {
				t.Error("CBR requested date changed")
			}
			return response(`<?xml version="1.0" encoding="windows-1251"?><ValCurs Date="31.08.2026"><Valute ID="R01235"><Nominal>1</Nominal><Value>80,1234</Value></Valute></ValCurs>`, 200), nil
		case "/fallback":
			fallbackCalled = true
			if r.URL.Query().Get("providers") != "cbr" {
				t.Error("unfiltered provider")
			}
			return response(`{"date":"2026-08-31","base":"USD","quote":"RUB","rate":81.1234}`, 200), nil
		}
		return response("", 404), nil
	})}, "", nil)
	client.cbr, client.frankfurter = "https://example.test/cbr", "https://example.test/fallback"
	date, _ := calendar.ParseDate("2026-09-01")
	got, err := client.USDToRUB(context.Background(), date, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
	if err != nil || got.Transport != "xml_daily" || got.Rate.Value() != "80.123400000000000000000000000000" || fallbackCalled || got.EffectiveAt.String() != "2026-08-31T00:00:00Z" {
		t.Fatalf("primary rate: %+v %v", got, err)
	}
	client.cbr = "https://example.test/missing"
	got, err = client.USDToRUB(context.Background(), date, time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC))
	if err != nil || got.Transport != "frankfurter_v2" || !fallbackCalled {
		t.Fatalf("fallback rate: %+v %v", got, err)
	}
}

func TestCoinGeckoKeyDatesAndSeparateStablecoins(t *testing.T) {
	secretFile := filepath.Join(t.TempDir(), "key")
	if err := os.WriteFile(secretFile, []byte("private-test-key\n"), 0600); err != nil {
		t.Fatal(err)
	}
	client := New(&http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("x-cg-demo-api-key") != "private-test-key" {
			t.Error("missing private header")
		}
		if strings.HasSuffix(r.URL.Path, "/simple/price") {
			return response(`{"bitcoin":{"usd":50000.01,"last_updated_at":1788264000},"ethereum":{"usd":2000,"last_updated_at":1788264000},"tether":{"usd":0.997,"last_updated_at":1788264000},"usd-coin":{"usd":1.001,"last_updated_at":1788264000}}`, 200), nil
		}
		if r.URL.Query().Get("date") != "01-09-2026" || !strings.Contains(r.URL.Path, "/usd-coin/history") {
			t.Error("history request mismatched")
		}
		return response(`{"market_data":{"current_price":{"usd":1.001}}}`, 200), nil
	})}, secretFile, allowQuota{})
	client.cg = "https://example.test"
	now := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	current, err := client.CryptoCurrent(context.Background(), now)
	if err != nil || len(current) != 4 || current[2].Rate.Value() != "0.997" || current[3].Rate.Value() != "1.001" {
		t.Fatalf("current rates: %+v %v", current, err)
	}
	date, _ := calendar.ParseDate("2026-09-01")
	history, err := client.CryptoUSD(context.Background(), money.USDC, date, now)
	if err != nil || history.Rate.Value() != "1.001" || history.Granularity != "daily" {
		t.Fatalf("historical rate: %+v %v", history, err)
	}
	older, _ := calendar.ParseDate("2025-08-01")
	if _, err := client.CryptoUSD(context.Background(), money.BTC, older, now); !errors.Is(err, ErrHistoryRange) {
		t.Fatalf("old history allowed: %v", err)
	}
	localToday, _ := calendar.ParseDate("2026-09-02")
	if _, err := client.CryptoUSD(context.Background(), money.USDC, localToday, now); !errors.Is(err, ErrUnavailable) {
		t.Fatalf("next local day was classified as permanent: %v", err)
	}
	if err := os.Chmod(secretFile, 0644); err != nil {
		t.Fatal(err)
	}
	if _, err := client.CryptoCurrent(context.Background(), now); !errors.Is(err, ErrConfiguration) {
		t.Fatalf("permissive key file allowed: %v", err)
	}
}

func TestProviderDecimalExponentWithoutFloat(t *testing.T) {
	for input, want := range map[string]string{"1.234e-7": "0.0000001234", "9E+3": "9000", "1e0": "1"} {
		got, err := plainDecimal(input)
		if err != nil || got != want {
			t.Fatalf("%s: %s %v", input, got, err)
		}
	}
	for _, input := range []string{"1e999", "1e-999", "1e2e3", ""} {
		if _, err := plainDecimal(input); err == nil {
			t.Fatalf("accepted %s", input)
		}
	}
}
