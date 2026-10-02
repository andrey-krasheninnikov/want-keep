package storage

import (
	"strconv"
	"testing"

	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestRefundEffectPositionsRestartForEachDimension(t *testing.T) {
	amount, err := money.NewMoney("1", money.RUB)
	if err != nil {
		t.Fatal(err)
	}
	categories := make([]expenses.CategoryAmount, 1000)
	for index := range categories {
		categories[index] = expenses.CategoryAmount{CategoryID: strconv.Itoa(index), Amount: amount}
	}
	rows := appendRefundEffects(nil, "refund", 1, "native", []expenses.MemberAmount{{MemberID: household.MembershipID("member"), Amount: amount}}, categories, []money.Money{amount})
	if rows[0].Dimension != "member" || rows[0].Position != 0 || rows[1].Dimension != "category" || rows[1].Position != 0 || rows[1000].Position != 999 || rows[1001].Dimension != "unallocated" || rows[1001].Position != 0 {
		t.Fatalf("unexpected dimension positions: first=%+v category-first=%+v category-last=%+v unallocated=%+v", rows[0], rows[1], rows[1000], rows[1001])
	}
}
