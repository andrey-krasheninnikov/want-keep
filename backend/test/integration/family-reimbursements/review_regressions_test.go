//go:build integration

package familyreimbursements_test

import (
	"context"
	"errors"
	"net/http"
	"net/url"
	"testing"
	"time"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matchdomain "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
)

func TestReimbursementCommandCannotReplayAfterRetention(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "10", "asset": "RUB"}, "reason": "Durable debt"}, http.StatusAccepted)
	snapshot := client.command(key)
	u, _ := url.Parse(f.dsn)
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	maintenance, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 2})
	if err != nil {
		t.Fatal(err)
	}
	defer maintenance.Close()
	at := func(days int) calendar.Instant {
		return instant(f.now.Time().Add(time.Duration(days) * 24 * time.Hour).Format(time.RFC3339Nano))
	}
	if count, err := maintenance.CleanupCommandDetails(testContext, at(90), 100); err != nil || count != 1 {
		t.Fatalf("detail cleanup: %d %v", count, err)
	}
	if count, err := maintenance.CleanupCommandTombstones(testContext, at(400), 100); err != nil || count != 1 {
		t.Fatalf("tombstone cleanup: %d %v", count, err)
	}
	request := commands.Request{ID: key, Kind: snapshot.Kind, PayloadHash: snapshot.PayloadHash}
	if _, err := f.executor.Register(testContext, f.p, request); !errors.Is(err, command.ErrCommandExpired) {
		t.Fatalf("old debt command registered again: %v", err)
	}
	if f.count("reimbursements") != 1 {
		t.Fatal("retention changed debt history")
	}
}

func TestUndoCreationIsConflict(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "10", "asset": "RUB"}, "reason": "Original debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+id+"/undo", key, map[string]any{"expectedRevision": 1, "decisionId": current.DecisionId, "reason": "Invalid creation undo"}, http.StatusAccepted)
	if result := client.command(key); result.Status != command.Failed || result.ErrorCode != "decision_conflict" {
		t.Fatalf("creation undo result: %#v", result)
	}
}

func TestUndoExpenseLinkRejectsChangedFormerExpense(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.personalAccount(0, money.RUB, "1000")
	first := f.expense(accountID, "10", money.RUB)
	second := f.expense(accountID, "20", money.RUB)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "30", "asset": "RUB"}, "expenseId": first, "reason": "Linked debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+id+"/corrections", uuid.NewString(), map[string]any{"expectedRevision": 1, "expenseId": second, "reason": "Link another expense"}, http.StatusAccepted)
	changed := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	f.execute(f.p, "transactions.correct", func(ctx context.Context) (command.Result, error) {
		note := "Expense changed while unlinked"
		return f.ledger.Correct(ctx, f.p, journal.Change{OperationID: first, Expected: 1, Correction: ledger.Correction{Note: &note}}, "Update original expense")
	})
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+id+"/undo", key, map[string]any{"expectedRevision": changed.Revision, "decisionId": changed.DecisionId, "reason": "Restore old expense"}, http.StatusAccepted)
	if result := client.command(key); result.Status != command.Failed || result.ErrorCode != "decision_conflict" {
		t.Fatalf("stale expense link restored: %#v", result)
	}
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.ExpenseId == nil || *current.ExpenseId != second || current.Revision != changed.Revision {
		t.Fatalf("failed undo changed debt: %#v", current)
	}
}

func TestAccountOwnershipChangeInvalidatesSettlement(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	transferID := f.transfer(from, to, cash("100", money.RUB), cash("100", money.RUB))
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "100", "asset": "RUB"}, "reason": "Ownership-sensitive debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+id+"/settlements", uuid.NewString(), map[string]any{"expectedRevision": 1, "transferId": transferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "100", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "100", "asset": "RUB"}}, http.StatusAccepted)
	service := accounts.NewServiceWithOwnershipReconciliation(f.store, f.store, nil, f.reimbursements, func() calendar.Instant { return f.now }, uuid.NewString)
	next, _ := household.NewOwnership(f.family.ID, household.Personal, f.members[0].UserID)
	currentAccount, err := f.store.Account(testContext, f.q, from)
	if err != nil {
		t.Fatal(err)
	}
	request := commands.Request{ID: uuid.NewString(), Kind: "accounts.ownership", PayloadHash: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"}
	outcome, err := f.executor.Execute(testContext, f.q, request, func(ctx context.Context) (command.Result, error) {
		return service.ChangeOwnership(ctx, f.q, from, currentAccount.Revision, next, "Owner changed")
	})
	if err != nil || outcome.Status() != command.Succeeded {
		t.Fatalf("ownership change failed: status=%s err=%v result=%#v", outcome.Status(), err, outcome.Snapshot())
	}
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.Outstanding.Amount != "100" || current.Settlements[0].State != "stale" {
		t.Fatalf("ownership change left active settlement: %#v", current)
	}
}

func TestMatchedTransferOtherLegChangeInvalidatesSettlement(t *testing.T) {
	for _, scenario := range []struct {
		name           string
		selectIncoming bool
	}{{"outgoing", false}, {"incoming", true}} {
		t.Run(scenario.name, func(t *testing.T) { matchedTransferOtherLegChange(t, scenario.selectIncoming) })
	}
}

func matchedTransferOtherLegChange(t *testing.T, selectIncoming bool) {
	f := newFixture(t)
	client := f.client(f.p)
	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	date, _ := calendar.ParseDate("2026-09-14")
	month, _ := calendar.ParseMonth("2026-09")
	left, right := uuid.NewString(), uuid.NewString()
	for _, item := range []struct {
		id, accountID, amount string
		kind                  ledger.Type
	}{{left, from, "-100", ledger.Expense}, {right, to, "100", ledger.Income}} {
		fact := ledger.Revision{OperationID: item.id, Revision: 1, ActorID: f.p.UserID(), Reason: "Synthetic transfer leg", Type: item.kind, State: ledger.Posted, OccurredAt: f.now, CashDate: date, ExpenseMonth: month, PayerState: "known", PayerMemberID: f.members[0].ID, Postings: []ledger.Posting{{AccountID: item.accountID, Money: cash(item.amount, money.RUB), Role: ledger.Principal}}}
		f.execute(f.p, "transactions.create", func(ctx context.Context) (command.Result, error) {
			return command.Result{ResourceType: "transaction", ResourceID: item.id, Revision: 1}, f.writer.Append(ctx, f.p, fact, 0)
		})
	}
	linker := matching.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	f.execute(f.p, "transfers.link", func(ctx context.Context) (command.Result, error) {
		return linker.LinkTransfer(ctx, f.p, journal.TransferInput{FromAccountID: from, ToAccountID: to, At: f.now, Sent: cash("100", money.RUB), Received: cash("100", money.RUB), Existing: true}, []matchdomain.Member{{OperationID: left, Revision: 1}, {OperationID: right, Revision: 1}})
	})
	selectedID, otherID := left, right
	if selectIncoming {
		selectedID, otherID = right, left
	}
	selected, found, err := f.store.CurrentLedgerRevision(testContext, f.p, selectedID)
	if err != nil || !found {
		t.Fatal("linked transfer missing", err)
	}
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "100", "asset": "RUB"}, "reason": "Matched transfer debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+id+"/settlements", uuid.NewString(), map[string]any{"expectedRevision": 1, "transferId": selectedID, "transferExpectedRevision": selected.Revision, "transferAmount": map[string]any{"amount": "100", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "100", "asset": "RUB"}}, http.StatusAccepted)
	other, found, err := f.store.CurrentLedgerRevision(testContext, f.p, otherID)
	if err != nil || !found {
		t.Fatal("other transfer leg missing", err)
	}
	other = other.Clone()
	other.Revision++
	other.State = ledger.Reversed
	for index := range other.Participation.Parts {
		other.Participation.Parts[index].State = ledger.Reversed
	}
	other.Reason = "Other leg reversed"
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, other, other.Revision-1) }); err != nil {
		t.Fatal(err)
	}
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.Outstanding.Amount != "100" || current.Settlements[0].State != "stale" {
		t.Fatalf("other leg change left active settlement: %#v", current)
	}
}
