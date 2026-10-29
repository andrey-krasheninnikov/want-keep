package ledger

import (
	"testing"

	domain "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestClassificationProposalItemDoesNotRequireAllocation(t *testing.T) {
	gross, _ := money.NewMoney("10.25", money.RUB)
	discount, _ := money.NewMoney("0.25", money.RUB)
	item := domain.ReceiptItem{ID: "item", Name: "Milk", Quantity: "1", Gross: gross, Discount: discount}

	dto, err := classificationProposalItemDTO(item)
	if err != nil || dto.Id != item.ID || dto.Net.Amount != "10.00" {
		t.Fatalf("proposal item = %+v, err=%v", dto, err)
	}
	if _, err = ReceiptItemToDTO(item); err == nil {
		t.Fatal("ledger receipt item accepted missing allocation")
	}
}
