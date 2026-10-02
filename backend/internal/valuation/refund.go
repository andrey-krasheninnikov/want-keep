package valuation

import (
	"errors"
	"math/big"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var ErrInvalidRefund = errors.New("invalid refund valuation")

// RefundShare returns the change in a purchase's pinned reporting value.
// Cumulative boundaries avoid losing the final remainder across partial refunds.
func RefundShare(purchase, reporting money.Money, before, after money.Money) (money.Money, error) {
	if purchase.Validate() != nil || reporting.Validate() != nil || before.Validate() != nil || after.Validate() != nil || purchase.Sign() >= 0 || reporting.Sign() >= 0 || purchase.Asset() != before.Asset() || purchase.Asset() != after.Asset() {
		return money.Money{}, ErrInvalidRefund
	}
	principal, _ := new(big.Rat).SetString(purchase.Amount())
	value, _ := new(big.Rat).SetString(reporting.Amount())
	previous, _ := new(big.Rat).SetString(before.Amount())
	current, _ := new(big.Rat).SetString(after.Amount())
	principal.Neg(principal)
	if previous.Sign() < 0 || current.Cmp(previous) < 0 || current.Cmp(principal) > 0 {
		return money.Money{}, ErrInvalidRefund
	}
	share := new(big.Rat).Mul(new(big.Rat).Neg(value), new(big.Rat).Quo(new(big.Rat).Sub(current, previous), principal))
	return money.NewMoney(decimal(share, 80), reporting.Asset())
}
