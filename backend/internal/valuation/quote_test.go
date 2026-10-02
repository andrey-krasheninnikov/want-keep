package valuation

import (
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestPlatformQuoteRequiresSpreadInExecutableRate(t *testing.T) {
	at, _ := calendar.ParseInstant("2026-09-01T12:00:00Z")
	amount, _ := money.NewMoney("100", money.USDT)
	rate, _ := money.NewRate(money.USDT, money.RUB, "90")
	quote := PlatformQuote{ID: "synthetic", Provider: "bybit", EvidenceRef: "quote-1", Direction: SellBase, Amount: amount, Rate: rate, FeeCoverage: "excluded", SpreadCoverage: "included", ObservedAt: at, FetchedAt: at}
	if err := quote.Validate(); err != nil {
		t.Fatalf("complete quote rejected: %v", err)
	}
	quote.SpreadCoverage = "excluded"
	if err := quote.Validate(); err == nil {
		t.Fatal("quote with unpriced spread accepted")
	}
}
