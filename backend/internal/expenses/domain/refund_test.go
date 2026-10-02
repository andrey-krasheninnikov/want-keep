package domain_test

import (
	"errors"
	"strconv"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func cash(value string, asset money.Asset) money.Money {
	result, err := money.NewMoney(value, asset)
	if err != nil {
		panic(err)
	}
	return result
}

func purchase(amount string, asset money.Asset) ledger.Revision {
	at, _ := calendar.ParseInstant("2026-08-31T12:00:00Z")
	date, _ := calendar.ParseDate("2026-08-31")
	month, _ := calendar.ParseMonth("2026-08")
	p := ledger.Revision{OperationID: "purchase", Revision: 1, ActorID: "user", Reason: "purchase", Type: ledger.Expense, State: ledger.Posted, OccurredAt: at, CashDate: date, ExpenseMonth: month, PayerState: "known", PayerMemberID: "member-a", Postings: []ledger.Posting{{AccountID: "source", Money: cash("-"+amount, asset), Role: ledger.Principal}}, Allocation: ledger.AllocationSnapshot{State: ledger.AllocationResolved, Purpose: ledger.AllocationShared, Mode: ledger.AllocationByAmounts, Origin: ledger.AllocationExplicitPurchase, Inputs: []ledger.AllocationMemberInput{{MemberID: "member-a", Amount: ptr(cash("4", asset))}, {MemberID: "member-b", Amount: ptr(cash("6", asset))}}, Members: []ledger.MemberAmount{{MemberID: "member-a", Money: cash("4", asset)}, {MemberID: "member-b", Money: cash("6", asset)}}}}
	return p
}

func refund(amount string, asset money.Asset) ledger.Revision {
	at, _ := calendar.ParseInstant("2026-09-07T12:00:00Z")
	date, _ := calendar.ParseDate("2026-09-07")
	month, _ := calendar.ParseMonth("2026-09")
	return ledger.Revision{OperationID: "refund", Revision: 1, ActorID: "user", Reason: "refund", Type: ledger.Refund, State: ledger.Posted, OccurredAt: at, CashDate: date, ExpenseMonth: month, RecordedAt: at, PayerState: "not_applicable", Postings: []ledger.Posting{{AccountID: "target", Money: cash(amount, asset), Role: ledger.Principal}}, Allocation: ledger.NotApplicableAllocation()}
}

func ptr(value money.Money) *money.Money { return &value }

func equal(left, right money.Money) bool {
	compared, err := left.Compare(right)
	return err == nil && compared == 0
}

func TestPurchaseLevelRefundPreservesMonthAllocationAndHistoricalValue(t *testing.T) {
	p := purchase("10", money.USD)
	r := refund("4", money.USD)
	zero := cash("0", money.USD)
	basis := expenses.ValuationBasis{Purchase: cash("10", money.USD), Value: cash("900", money.RUB), Ref: "rate-observation"}
	shares, err := expenses.AllocateValuations(basis, cash("10", money.USD), map[string]money.Money{"refund": cash("4", money.USD)})
	if err != nil {
		t.Fatal(err)
	}
	share := shares["refund"]
	result, err := expenses.Calculate(p, r, nil, zero, nil, &share, 1, "returned", household.UserID("user"), r.RecordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if result.ExpenseMonth.String() != "2026-08" || result.CashDate.String() != "2026-09-07" || result.Remaining.Amount() != "6" || result.Valuation == nil || !equal(result.Valuation.Value, cash("360", money.RUB)) {
		t.Fatalf("unexpected refund: %#v", result)
	}
	if len(result.Members) != 2 || !equal(result.Members[0].Amount, cash("1.6", money.USD)) || !equal(result.Members[1].Amount, cash("2.4", money.USD)) {
		t.Fatalf("unexpected members: %#v", result.Members)
	}
}

func TestHistoricalValueAllocationPreservesFrozenTotalAcrossPartialRefunds(t *testing.T) {
	basis := expenses.ValuationBasis{Purchase: cash("6", money.USD), Value: cash("1", money.RUB), Ref: "rate-observation"}
	amounts := map[string]money.Money{}
	for _, id := range []string{"f", "e", "d", "c", "b", "a"} {
		amounts[id] = cash("1", money.USD)
	}
	shares, err := expenses.AllocateValuations(basis, cash("6", money.USD), amounts)
	if err != nil {
		t.Fatal(err)
	}
	total := cash("0", money.RUB)
	for _, id := range []string{"a", "b", "c", "d", "e", "f"} {
		total, err = total.Add(shares[id].Value)
		if err != nil {
			t.Fatal(err)
		}
	}
	if !equal(total, cash("1", money.RUB)) {
		t.Fatalf("allocated value=%s", total.Amount())
	}
}

func TestHistoricalValueAllocationSupportsRefundLimitAndRemainder(t *testing.T) {
	basis := expenses.ValuationBasis{Purchase: cash("1", money.USD), Value: cash("1", money.RUB), Ref: "rate-observation"}
	amounts := make(map[string]money.Money, 1000)
	for index := range 1000 {
		amounts["refund-"+strconv.Itoa(index)] = cash("0.0001", money.USD)
	}
	shares, err := expenses.AllocateValuations(basis, basis.Purchase, amounts)
	if err != nil {
		t.Fatal(err)
	}
	if len(shares) != len(amounts) {
		t.Fatalf("share count=%d", len(shares))
	}
}

func TestItemRefundUsesAuditedItemAllocationAndCap(t *testing.T) {
	p := purchase("10", money.RUB)
	p.Allocation = ledger.NotApplicableAllocation()
	p.ReceiptItems = []ledger.ReceiptItem{{ID: "joint", Name: "Joint", Quantity: "1", Gross: cash("6", money.RUB), Discount: cash("0", money.RUB), CategoryID: "food", Allocation: ledger.AllocationSnapshot{State: ledger.AllocationResolved, Purpose: ledger.AllocationShared, Mode: ledger.AllocationByAmounts, Origin: ledger.AllocationExplicitItem, Inputs: []ledger.AllocationMemberInput{{MemberID: "member-a", Amount: ptr(cash("3", money.RUB))}, {MemberID: "member-b", Amount: ptr(cash("3", money.RUB))}}, Members: []ledger.MemberAmount{{MemberID: "member-a", Money: cash("3", money.RUB)}, {MemberID: "member-b", Money: cash("3", money.RUB)}}}}, {ID: "personal", Name: "Personal", Quantity: "1", Gross: cash("4", money.RUB), Discount: cash("0", money.RUB), CategoryID: "personal", Allocation: ledger.AllocationSnapshot{State: ledger.AllocationResolved, Purpose: ledger.AllocationPersonal, Mode: ledger.AllocationByAmounts, Origin: ledger.AllocationExplicitItem, Inputs: []ledger.AllocationMemberInput{{MemberID: "member-b", Amount: ptr(cash("4", money.RUB))}}, Members: []ledger.MemberAmount{{MemberID: "member-b", Money: cash("4", money.RUB)}}}}}
	p.Allocation = ledger.AllocationSnapshot{State: ledger.AllocationResolved, Purpose: ledger.AllocationShared, Mode: ledger.AllocationComposite, Origin: ledger.AllocationMixed, Basis: &ledger.AllocationInput{Mode: ledger.AllocationUnknown, Origin: ledger.AllocationUnknownOrigin, Reason: "item allocation"}, Members: []ledger.MemberAmount{{MemberID: "member-a", Money: cash("3", money.RUB)}, {MemberID: "member-b", Money: cash("7", money.RUB)}}}
	r := refund("2", money.RUB)
	result, err := expenses.Calculate(p, r, []expenses.ItemPortion{{ItemID: "joint", Amount: cash("2", money.RUB)}}, cash("0", money.RUB), nil, nil, 1, "partial", "user", r.RecordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Members) != 2 || !equal(result.Members[0].Amount, cash("1", money.RUB)) || !equal(result.Members[1].Amount, cash("1", money.RUB)) || result.Categories[0].CategoryID != "food" {
		t.Fatalf("wrong item attribution: %#v", result)
	}
	_, err = expenses.Calculate(p, r, []expenses.ItemPortion{{ItemID: "joint", Amount: cash("2", money.RUB)}}, cash("9", money.RUB), nil, nil, 1, "too much", "user", r.RecordedAt)
	if !errors.Is(err, expenses.ErrRefundExceedsPurchase) {
		t.Fatalf("wanted cap error, got %v", err)
	}
}

func TestItemizedRefundWithoutItemsNeedsClarification(t *testing.T) {
	p := purchase("10", money.RUB)
	p.ReceiptItems = []ledger.ReceiptItem{{ID: "item", Name: "Item", Quantity: "1", Gross: cash("10", money.RUB), Discount: cash("0", money.RUB)}}
	r := refund("4", money.RUB)
	result, err := expenses.Calculate(p, r, nil, cash("0", money.RUB), nil, nil, 1, "unknown item", "user", r.RecordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != expenses.Clarification || len(result.Members) != 0 {
		t.Fatalf("unexpected result: %#v", result)
	}
}

func TestInactivePurchasePreservesLinkWithoutAnalyticalEffect(t *testing.T) {
	p := purchase("10", money.RUB)
	p.AccountingState = ledger.ExcludedFromAccounting
	r := refund("4", money.RUB)
	result, err := expenses.Calculate(p, r, nil, cash("0", money.RUB), nil, nil, 1, "excluded purchase", "user", r.RecordedAt)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != expenses.Inactive || !equal(result.Remaining, cash("10", money.RUB)) || len(result.Members) != 0 || len(result.Categories) != 0 {
		t.Fatalf("unexpected inactive refund: %#v", result)
	}
}

func TestNonContributingPrincipalDoesNotReduceExpense(t *testing.T) {
	p := purchase("10", money.RUB)
	for _, candidate := range []struct {
		name          string
		participation ledger.Participation
	}{
		{"waiting", ledger.Participation{GroupID: "group", Kind: "payment", State: "waiting"}},
		{"non-carrier", ledger.Participation{GroupID: "group", Kind: "payment", State: "linked", Parts: []ledger.Contribution{{CarrierID: "other", State: ledger.Posted}}}},
		{"pending carrier", ledger.Participation{GroupID: "group", Kind: "payment", State: "linked", Parts: []ledger.Contribution{{CarrierID: "refund", State: ledger.Pending}}}},
	} {
		t.Run(candidate.name, func(t *testing.T) {
			r := refund("4", money.RUB)
			r.Participation = candidate.participation
			result, err := expenses.Calculate(p, r, nil, cash("0", money.RUB), nil, nil, 1, "possible duplicate", "user", r.RecordedAt)
			if err != nil || result.State != expenses.Inactive || result.Remaining.Amount() != "10" {
				t.Fatalf("non-contributing refund=%+v err=%v", result, err)
			}
		})
	}
}

func TestInactiveItemRefundStillValidatesItemIdentity(t *testing.T) {
	p := purchase("10", money.RUB)
	p.AccountingState = ledger.ExcludedFromAccounting
	p.ReceiptItems = []ledger.ReceiptItem{{ID: "known", Name: "Known", Quantity: "1", Gross: cash("10", money.RUB), Discount: cash("0", money.RUB)}}
	r := refund("4", money.RUB)
	_, err := expenses.Calculate(p, r, []expenses.ItemPortion{{ItemID: "unknown", Amount: cash("4", money.RUB)}}, cash("0", money.RUB), nil, nil, 1, "invalid item", "user", r.RecordedAt)
	if !errors.Is(err, expenses.ErrInvalidRefund) {
		t.Fatalf("wanted invalid refund, got %v", err)
	}
}

func TestInactiveRefundDoesNotConsumeReplacementCapacity(t *testing.T) {
	p := purchase("10", money.RUB)
	r := refund("10", money.RUB)
	r.AccountingState = ledger.ExcludedFromAccounting
	result, err := expenses.Calculate(p, r, nil, cash("10", money.RUB), nil, nil, 1, "excluded duplicate", "user", r.RecordedAt)
	if err != nil || result.State != expenses.Inactive || !equal(result.Remaining, cash("0", money.RUB)) {
		t.Fatalf("inactive refund=%+v err=%v", result, err)
	}
	p.ReceiptItems = []ledger.ReceiptItem{{ID: "item", Name: "Item", Quantity: "1", Gross: cash("10", money.RUB), Discount: cash("0", money.RUB)}}
	result, err = expenses.Calculate(p, r, []expenses.ItemPortion{{ItemID: "item", Amount: cash("10", money.RUB)}}, cash("10", money.RUB), map[string]money.Money{"item": cash("10", money.RUB)}, nil, 1, "excluded duplicate", "user", r.RecordedAt)
	if err != nil || result.State != expenses.Inactive {
		t.Fatalf("inactive item refund=%+v err=%v", result, err)
	}
}
