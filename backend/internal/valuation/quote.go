package valuation

import (
	"errors"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var ErrInvalidQuote = errors.New("invalid platform quote")

type Direction string

const (
	SellBase Direction = "sell_base"
	BuyBase  Direction = "buy_base"
)

// PlatformQuote is an observed, amount-specific provider offer, never a reference rate.
type PlatformQuote struct {
	ID, Provider, EvidenceRef   string
	Direction                   Direction
	Amount                      money.Money
	Rate                        money.Rate
	Fees                        []money.Money
	FeeCoverage, SpreadCoverage string
	ObservedAt, FetchedAt       calendar.Instant
}

func (q PlatformQuote) Validate() error {
	switch q.Provider {
	case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
	default:
		return ErrInvalidQuote
	}
	if q.ID == "" || q.EvidenceRef == "" || len(q.EvidenceRef) > 2000 || q.Direction != SellBase && q.Direction != BuyBase || q.Amount.Validate() != nil || q.Amount.Sign() <= 0 || q.Rate.Validate() != nil || q.Rate.Base() != q.Amount.Asset() || q.ObservedAt.String() == "" || q.FetchedAt.String() == "" || q.ObservedAt.Time().After(q.FetchedAt.Time()) || len(q.Fees) > 20 {
		return ErrInvalidQuote
	}
	if q.FeeCoverage != "included" && q.FeeCoverage != "excluded" || q.SpreadCoverage != "included" {
		return ErrInvalidQuote
	}
	for _, fee := range q.Fees {
		if fee.Validate() != nil || fee.Sign() < 0 {
			return ErrInvalidQuote
		}
	}
	return nil
}
