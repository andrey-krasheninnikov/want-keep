package valuation

import (
	"errors"
	"testing"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestRefundShareUsesPinnedPurchaseValue(t *testing.T) {
	purchase, _ := money.NewMoney("-10", money.USD)
	reporting, _ := money.NewMoney("-900", money.RUB)
	zero, _ := money.NewMoney("0", money.USD)
	four, _ := money.NewMoney("4", money.USD)
	ten, _ := money.NewMoney("10", money.USD)
	first, err := RefundShare(purchase, reporting, zero, four)
	if err != nil || first.Amount() != "360" {
		t.Fatalf("first refund: %s %v", first.Amount(), err)
	}
	second, err := RefundShare(purchase, reporting, four, ten)
	if err != nil || second.Amount() != "540" {
		t.Fatalf("remaining refund: %s %v", second.Amount(), err)
	}
	overmuch, _ := money.NewMoney("11", money.USD)
	if _, err = RefundShare(purchase, reporting, four, overmuch); !errors.Is(err, ErrInvalidRefund) {
		t.Fatalf("over-refund: %v", err)
	}
}
