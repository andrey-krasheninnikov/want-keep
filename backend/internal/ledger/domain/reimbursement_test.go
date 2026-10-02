package domain

import (
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestReimbursementSettlementAndUndo(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	for index, amount := range []string{"100", "200"} {
		value, _ := money.NewMoney(amount, money.RUB)
		settlement := ReimbursementSettlement{ID: "settlement-" + amount, DecisionID: "decision-" + amount, TransferID: "transfer-" + amount, TransferKey: "transfer-" + amount, Fingerprint: strings.Repeat("a", 64), TransferRevision: 1, TransferAmount: value, SettledAmount: value, OperationIDs: []string{"transfer-" + amount}, State: SettlementActive, ActorID: "user-a", RecordedAt: instantForReimbursement(t)}
		var err error
		debt, err = debt.AddSettlement(settlement, "user-a", instantForReimbursement(t), settlement.DecisionID)
		if err != nil {
			t.Fatal(err)
		}
		want := []string{"200", "0"}[index]
		if debt.Outstanding.Amount() != want {
			t.Fatalf("outstanding=%s want=%s", debt.Outstanding.Amount(), want)
		}
	}
	if debt.State != ReimbursementSettled {
		t.Fatal(debt.State)
	}
	decision := ReimbursementDecision{ID: "decision-200", Kind: "settlement", ReimbursementID: debt.ID, SettlementID: "settlement-200", ActorID: "user-a", At: instantForReimbursement(t), Reason: "settle", Before: 2, After: 3, Fields: []ReimbursementField{ReimbursementSettlementField}}
	undone, _, err := debt.Undo(decision, Reimbursement{}, "user-b", instantForReimbursement(t), "undo")
	if err != nil || undone.Outstanding.Amount() != "200" || undone.State != ReimbursementOpen {
		t.Fatal(undone.Outstanding.Amount(), undone.State, err)
	}
}

func TestReimbursementSelectiveUndoDetectsABA(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	firstValue, _ := money.NewMoney("400", money.RUB)
	first, fields, err := debt.Correct(ReimbursementChange{Principal: &firstValue}, "user-a", instantForReimbursement(t), "first")
	if err != nil {
		t.Fatal(err)
	}
	original, _ := money.NewMoney("300", money.RUB)
	latest, _, err := first.Correct(ReimbursementChange{Principal: &original}, "user-b", instantForReimbursement(t), "second")
	if err != nil {
		t.Fatal(err)
	}
	decision := ReimbursementDecision{ID: "first", Kind: "correction", ReimbursementID: debt.ID, ActorID: "user-a", At: instantForReimbursement(t), Reason: "change", Before: 1, After: 2, Fields: fields}
	if _, _, err = latest.Undo(decision, debt, "user-a", instantForReimbursement(t), "undo"); err != ErrReimbursementConflict {
		t.Fatalf("err=%v", err)
	}
}

func TestSameAssetSettlementMustUseExactTransferAmount(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	transfer, _ := money.NewMoney("100", money.RUB)
	settled, _ := money.NewMoney("99", money.RUB)
	entry := ReimbursementSettlement{ID: "settlement", DecisionID: "decision", TransferID: "transfer", TransferKey: "transfer", Fingerprint: strings.Repeat("a", 64), TransferRevision: 1, TransferAmount: transfer, SettledAmount: settled, OperationIDs: []string{"transfer"}, State: SettlementActive, ActorID: "user-a", RecordedAt: instantForReimbursement(t)}
	if _, err := debt.AddSettlement(entry, "user-a", instantForReimbursement(t), "decision"); err != ErrInvalidReimbursement {
		t.Fatalf("err=%v", err)
	}
}

func TestActiveSettlementProtectsPartiesAssetAndSettledPrincipal(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	settled, _ := money.NewMoney("100", money.RUB)
	entry := ReimbursementSettlement{ID: "settlement", DecisionID: "settlement-decision", TransferID: "transfer", TransferKey: "transfer", Fingerprint: strings.Repeat("a", 64), TransferRevision: 1, TransferAmount: settled, SettledAmount: settled, OperationIDs: []string{"transfer"}, State: SettlementActive, ActorID: "user-a", RecordedAt: instantForReimbursement(t)}
	current, err := debt.AddSettlement(entry, "user-a", instantForReimbursement(t), entry.DecisionID)
	if err != nil {
		t.Fatal(err)
	}
	other := household.MembershipID("other")
	tooSmall, _ := money.NewMoney("50", money.RUB)
	otherAsset, _ := money.NewMoney("300", money.USD)
	voided := true
	for _, change := range []ReimbursementChange{{CreditorMemberID: &other}, {Principal: &tooSmall}, {Principal: &otherAsset}, {Voided: &voided}} {
		if _, _, err = current.Correct(change, "user-b", instantForReimbursement(t), "change"); err != ErrReimbursementConflict {
			t.Fatalf("unsafe correction accepted: %#v err=%v", change, err)
		}
	}
	if debt.Revision != 1 || len(debt.Settlements) != 0 || debt.Outstanding.Amount() != "300" {
		t.Fatal("source value mutated")
	}
}

func TestReimbursementRejectsUnsafeTextAndRevisionOverflow(t *testing.T) {
	amount, _ := money.NewMoney("1", money.RUB)
	if _, err := NewReimbursement("id", "creditor", "debtor", amount, "", 0, " \x00 ", "user", instantForReimbursement(t), "decision"); err != ErrInvalidReimbursement {
		t.Fatalf("unsafe text error=%v", err)
	}
	decision := ReimbursementDecision{ID: "decision", Kind: "correction", ReimbursementID: "id", ActorID: "user", At: instantForReimbursement(t), Reason: "overflow", Before: ^uint64(0), After: 0, Fields: []ReimbursementField{ReimbursementReasonField}}
	if err := decision.Validate(); err != ErrInvalidReimbursement {
		t.Fatalf("overflow error=%v", err)
	}
}

func TestVoidedReimbursementRetainsAttentionUntilReopened(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	attention, err := debt.RequireAttention("linked_expense_changed", "user-a", instantForReimbursement(t), "attention")
	if err != nil {
		t.Fatal(err)
	}
	voided := true
	closed, _, err := attention.Correct(ReimbursementChange{Voided: &voided}, "user-a", instantForReimbursement(t), "void")
	if err != nil || closed.State != ReimbursementVoided || closed.AttentionReason == "" {
		t.Fatal(closed.State, closed.AttentionReason, err)
	}
	voided = false
	reopened, _, err := closed.Correct(ReimbursementChange{Voided: &voided}, "user-a", instantForReimbursement(t), "reopen")
	if err != nil || reopened.State != ReimbursementAttentionRequired {
		t.Fatal(reopened.State, err)
	}
}

func TestHistoricalSettlementDoesNotBlockAssetCorrection(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	settled, _ := money.NewMoney("100", money.RUB)
	entry := ReimbursementSettlement{ID: "settlement", DecisionID: "settlement-decision", TransferID: "transfer", TransferKey: "transfer", Fingerprint: strings.Repeat("a", 64), TransferRevision: 1, TransferAmount: settled, SettledAmount: settled, OperationIDs: []string{"transfer"}, State: SettlementActive, ActorID: "user-a", RecordedAt: instantForReimbursement(t)}
	current, err := debt.AddSettlement(entry, "user-a", instantForReimbursement(t), entry.DecisionID)
	if err != nil {
		t.Fatal(err)
	}
	current, err = current.MarkSettlement(entry.ID, SettlementStale, "user-a", instantForReimbursement(t), "stale", "")
	if err != nil {
		t.Fatal(err)
	}
	dollars, _ := money.NewMoney("3", money.USD)
	corrected, _, err := current.Correct(ReimbursementChange{Principal: &dollars}, "user-a", instantForReimbursement(t), "currency")
	if err != nil || corrected.Principal.Asset() != money.USD || corrected.Outstanding.Amount() != "3" {
		t.Fatal(corrected.Principal.Asset(), corrected.Outstanding.Amount(), err)
	}
}

func TestReimbursementSettlementLimit(t *testing.T) {
	debt := reimbursementFixture(t, "300")
	debt.Settlements = make([]ReimbursementSettlement, MaxReimbursementSettlements)
	entry := ReimbursementSettlement{DecisionID: "decision"}
	if _, err := debt.AddSettlement(entry, "user-a", instantForReimbursement(t), entry.DecisionID); err != ErrReimbursementConflict {
		t.Fatalf("limit error=%v", err)
	}
	debt.Settlements = append(debt.Settlements, entry)
	if err := debt.Validate(); err != ErrInvalidReimbursement {
		t.Fatalf("oversized aggregate error=%v", err)
	}
}

func reimbursementFixture(t *testing.T, amount string) Reimbursement {
	t.Helper()
	value, _ := money.NewMoney(amount, money.RUB)
	result, err := NewReimbursement("reimbursement", household.MembershipID("creditor"), household.MembershipID("debtor"), value, "", 0, "Synthetic debt", household.UserID("user-a"), instantForReimbursement(t), "create")
	if err != nil {
		t.Fatal(err)
	}
	return result
}

func instantForReimbursement(t *testing.T) calendar.Instant {
	t.Helper()
	value, err := calendar.ParseInstant("2026-09-14T12:00:00.123456789Z")
	if err != nil {
		t.Fatal(err)
	}
	return value
}
