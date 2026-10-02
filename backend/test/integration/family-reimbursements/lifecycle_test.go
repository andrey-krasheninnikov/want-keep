//go:build integration

package familyreimbursements_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestExplicitDebtSettlementAndExactAssets(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	for _, item := range []struct {
		asset  money.Asset
		amount string
	}{{money.RUB, "300"}, {money.USD, "12.34"}, {money.USDT, "0.000000000001"}, {money.USDC, "1.000000000000000001"}, {money.BTC, "0.00000001"}, {money.ETH, "0.123456789012345678901"}} {
		key := uuid.NewString()
		client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": item.amount, "asset": item.asset}, "reason": "Explicit debt"}, http.StatusAccepted)
		id := client.result(key).ResourceID
		value := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
		if value.Principal.Amount != item.amount || value.Outstanding.Amount != item.amount || value.Principal.Asset != generated.Asset(item.asset) || value.State != "open" || value.DecisionId == "" {
			t.Fatalf("%s precision or state lost: %#v", item.asset, value)
		}
	}

	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	transferID := f.execute(f.p, "transfers.create", func(ctx context.Context) (command.Result, error) {
		return f.ledger.Transfer(ctx, f.p, journal.TransferInput{FromAccountID: from, ToAccountID: to, At: f.now, Sent: cash("300", money.RUB), Received: cash("300", money.RUB), Fees: []journal.FeeInput{{AccountID: from, Amount: cash("10", money.RUB)}}})
	}).ResourceID
	transfer, _, err := f.store.CurrentLedgerRevision(testContext, f.p, transferID)
	if err != nil {
		t.Fatal(err)
	}
	components, err := transfer.Components()
	if err != nil || len(components) != 1 || components[0].Kind != "fee" || components[0].Money.Amount() != "-10" {
		t.Fatal("principal or fee accounting is wrong", components, err)
	}

	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "300", "asset": "RUB"}, "reason": "Explicit debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	for index, amount := range []string{"100", "200"} {
		key = uuid.NewString()
		client.call(http.MethodPost, "/reimbursements/"+id+"/settlements", key, map[string]any{"expectedRevision": index + 1, "transferId": transferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": amount, "asset": "RUB"}, "settledAmount": map[string]any{"amount": amount, "asset": "RUB"}}, http.StatusAccepted)
	}
	value := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if value.Revision != 3 || value.State != "settled" || value.Outstanding.Amount != "0" || len(value.Settlements) != 2 {
		t.Fatalf("settlement result: %#v", value)
	}
	if got := f.balance(from, "owned"); got != "690" {
		t.Fatalf("transfer applied more than once: %s", got)
	}
}

func TestTransferCapacityCrossAssetAndExplicitCreation(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	transferID := f.transfer(from, to, cash("300", money.RUB), cash("300", money.RUB))
	for _, amount := range []string{"100", "200"} {
		key := uuid.NewString()
		client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": amount, "asset": "RUB"}, "reason": "Split settlement"}, http.StatusAccepted)
		id := client.result(key).ResourceID
		key = uuid.NewString()
		client.call(http.MethodPost, "/reimbursements/"+id+"/settlements", key, map[string]any{"expectedRevision": 1, "transferId": transferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": amount, "asset": "RUB"}, "settledAmount": map[string]any{"amount": amount, "asset": "RUB"}}, http.StatusAccepted)
	}
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "1", "asset": "RUB"}, "reason": "Over capacity"}, http.StatusAccepted)
	overID := client.result(key).ResourceID
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+overID+"/settlements", key, map[string]any{"expectedRevision": 1, "transferId": transferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "1", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "1", "asset": "RUB"}}, http.StatusAccepted)
	if snapshot := client.command(key); snapshot.Status != command.Failed || snapshot.ErrorCode != "decision_conflict" {
		t.Fatalf("over-capacity result: %#v", snapshot)
	}

	usdFrom := f.personalAccount(1, money.USD, "100")
	usdtTo := f.personalAccount(0, money.USDT, "0")
	exchangeID := f.transfer(usdFrom, usdtTo, cash("100", money.USD), cash("99", money.USDT))
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "100", "asset": "USD"}, "reason": "Cross asset debt"}, http.StatusAccepted)
	crossID := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+crossID+"/settlements", uuid.NewString(), map[string]any{"expectedRevision": 1, "transferId": exchangeID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "99", "asset": "USDT"}, "settledAmount": map[string]any{"amount": "100", "asset": "USD"}}, http.StatusAccepted)

	reverseID := f.transfer(to, from, cash("1", money.RUB), cash("1", money.RUB))
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "1", "asset": "RUB"}, "reason": "Wrong direction"}, http.StatusAccepted)
	reverseDebtID := client.result(key).ResourceID
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+reverseDebtID+"/settlements", key, map[string]any{"expectedRevision": 1, "transferId": reverseID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "1", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "1", "asset": "RUB"}}, http.StatusAccepted)
	if snapshot := client.command(key); snapshot.Status != command.Failed || snapshot.ErrorCode != "invalid_transaction" {
		t.Fatalf("wrong transfer direction: %#v", snapshot)
	}

	shared := f.personalAccount(-1, money.RUB, "1")
	sharedTransferID := f.transfer(shared, to, cash("1", money.RUB), cash("1", money.RUB))
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "1", "asset": "RUB"}, "reason": "Shared account transfer"}, http.StatusAccepted)
	sharedDebtID := client.result(key).ResourceID
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+sharedDebtID+"/settlements", key, map[string]any{"expectedRevision": 1, "transferId": sharedTransferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "1", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "1", "asset": "RUB"}}, http.StatusAccepted)
	if snapshot := client.command(key); snapshot.Status != command.Failed || snapshot.ErrorCode != "invalid_transaction" {
		t.Fatalf("shared transfer accepted: %#v", snapshot)
	}

	if count := f.count("reimbursements"); count != 6 {
		t.Fatalf("allocation or transfer created implicit debt: %d", count)
	}
}

func TestCorrectionUndoAndReferenceInvalidation(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.personalAccount(0, money.RUB, "1000")
	expenseID := f.expense(accountID, "50", money.RUB)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "300", "asset": "RUB"}, "expenseId": expenseID, "reason": "A"}, http.StatusAccepted)
	id := client.result(key).ResourceID

	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+id+"/corrections", key, map[string]any{"expectedRevision": 1, "debtReason": "B", "reason": "Correct reason"}, http.StatusAccepted)
	second := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+id+"/corrections", key, map[string]any{"expectedRevision": 2, "debtReason": "A", "reason": "Return reason"}, http.StatusAccepted)
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+id+"/undo", key, map[string]any{"expectedRevision": 3, "decisionId": second.DecisionId, "reason": "Selective undo"}, http.StatusAccepted)
	if snapshot := client.command(key); snapshot.Status != command.Failed || snapshot.ErrorCode != "decision_conflict" {
		t.Fatalf("A-B-A conflict result: %#v", snapshot)
	}
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	client.call(http.MethodPost, "/reimbursements/"+id+"/undo", uuid.NewString(), map[string]any{"expectedRevision": 3, "decisionId": current.DecisionId, "reason": "Undo latest"}, http.StatusAccepted)
	current = decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.Reason != "B" || current.Revision != 4 {
		t.Fatalf("latest correction not selectively undone: %#v", current)
	}

	result := f.execute(f.p, "transactions.correct", func(ctx context.Context) (command.Result, error) {
		note := "Only text changed"
		return f.ledger.Correct(ctx, f.p, journal.Change{OperationID: expenseID, Expected: 1, Correction: ledger.Correction{Note: &note}}, "Non-financial edit")
	})
	if result.Revision != 2 {
		t.Fatal("expense correction missing")
	}
	current = decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.State != "attention_required" || current.AttentionReason == nil {
		t.Fatalf("expense revision did not require attention: %#v", current)
	}
	key = uuid.NewString()
	client.call(http.MethodPost, "/reimbursements/"+id+"/undo", key, map[string]any{"expectedRevision": current.Revision, "decisionId": current.DecisionId, "reason": "Invalid undo"}, http.StatusAccepted)
	if snapshot := client.command(key); snapshot.Status != command.Failed || snapshot.ErrorCode != "decision_conflict" {
		t.Fatalf("reference invalidation was undone: %#v", snapshot)
	}
}

func TestSettlementSurvivesTextEditAndStalesOnFinancialEdit(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	transferID := f.transfer(from, to, cash("300", money.RUB), cash("300", money.RUB))
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "200", "asset": "RUB"}, "reason": "Transfer-linked debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+id+"/settlements", uuid.NewString(), map[string]any{"expectedRevision": 1, "transferId": transferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "100", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "100", "asset": "RUB"}}, http.StatusAccepted)

	f.execute(f.p, "transactions.correct", func(ctx context.Context) (command.Result, error) {
		note := "Reference only"
		return f.ledger.Correct(ctx, f.p, journal.Change{OperationID: transferID, Expected: 1, Correction: ledger.Correction{Note: &note}}, "Text edit")
	})
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.Outstanding.Amount != "100" || current.Settlements[0].State != "active" {
		t.Fatalf("text edit invalidated settlement: %#v", current)
	}

	revision, _, err := f.store.CurrentLedgerRevision(testContext, f.p, transferID)
	if err != nil {
		t.Fatal(err)
	}
	principal := []ledger.Posting{}
	for _, posting := range revision.Postings {
		if posting.Role == ledger.Principal {
			principal = append(principal, posting)
		}
	}
	principal[0].Money = cash("-250", money.RUB)
	principal[1].Money = cash("250", money.RUB)
	f.execute(f.p, "transactions.correct", func(ctx context.Context) (command.Result, error) {
		return f.ledger.Correct(ctx, f.p, journal.Change{OperationID: transferID, Expected: 2, Correction: ledger.Correction{Principal: &principal}}, "Financial edit")
	})
	current = decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.Outstanding.Amount != "200" || current.Settlements[0].State != "stale" {
		t.Fatalf("financial edit did not restore debt: %#v", current)
	}
}

func TestReversedTransferRestoresDebt(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	transferID := f.transfer(from, to, cash("100", money.RUB), cash("100", money.RUB))
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "100", "asset": "RUB"}, "reason": "Transfer-linked debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+id+"/settlements", uuid.NewString(), map[string]any{"expectedRevision": 1, "transferId": transferID, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "100", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "100", "asset": "RUB"}}, http.StatusAccepted)

	reversed, _, err := f.store.CurrentLedgerRevision(testContext, f.p, transferID)
	if err != nil {
		t.Fatal(err)
	}
	reversed = reversed.Clone()
	reversed.Revision++
	reversed.State = ledger.Reversed
	reversed.Reason = "Transfer reversed"
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		return f.writer.Append(ctx, f.p, reversed, reversed.Revision-1)
	}); err != nil {
		t.Fatal(err)
	}
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.Outstanding.Amount != "100" || current.State != "open" || current.Settlements[0].State != "stale" {
		t.Fatalf("reversed transfer did not restore debt: %#v", current)
	}
}

func TestVoidedDebtRecordsLinkedExpenseChange(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.personalAccount(0, money.RUB, "1000")
	expenseID := f.expense(accountID, "50", money.RUB)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "300", "asset": "RUB"}, "expenseId": expenseID, "reason": "Linked debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	client.call(http.MethodPost, "/reimbursements/"+id+"/corrections", uuid.NewString(), map[string]any{"expectedRevision": 1, "voided": true, "reason": "Temporarily void"}, http.StatusAccepted)

	f.execute(f.p, "transactions.correct", func(ctx context.Context) (command.Result, error) {
		note := "Updated source expense"
		return f.ledger.Correct(ctx, f.p, journal.Change{OperationID: expenseID, Expected: 1, Correction: ledger.Correction{Note: &note}}, "Update expense")
	})
	current := decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.State != "voided" || current.AttentionReason == nil || current.Revision != 3 {
		t.Fatalf("voided debt lost expense invalidation: %#v", current)
	}
	client.call(http.MethodPost, "/reimbursements/"+id+"/corrections", uuid.NewString(), map[string]any{"expectedRevision": 3, "voided": false, "reason": "Reopen debt"}, http.StatusAccepted)
	current = decode[generated.Reimbursement](t, client.call(http.MethodGet, "/reimbursements/"+id, "", nil, http.StatusOK))
	if current.State != "attention_required" || current.AttentionReason == nil {
		t.Fatalf("reopened debt did not require attention: %#v", current)
	}
}

func TestReplayIsolationPaginationAndCSRF(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	key := uuid.NewString()
	payload := map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "10", "asset": "RUB"}, "reason": "Replay-safe debt"}
	client.call(http.MethodPost, "/reimbursements", key, payload, http.StatusAccepted)
	client.call(http.MethodPost, "/reimbursements", key, payload, http.StatusAccepted)
	changed := map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "11", "asset": "RUB"}, "reason": "Replay-safe debt"}
	client.call(http.MethodPost, "/reimbursements", key, changed, http.StatusConflict)
	if f.count("reimbursements") != 1 {
		t.Fatal("replay created another debt")
	}
	secondKey := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", secondKey, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "20", "asset": "RUB"}, "reason": "Second debt"}, http.StatusAccepted)

	request := client.call(http.MethodGet, "/reimbursements?limit=1", "", nil, http.StatusOK)
	page := decode[generated.ReimbursementPage](t, request)
	if len(page.Items) != 1 || page.NextCursor == nil {
		t.Fatal("page did not return result")
	}
	next := decode[generated.ReimbursementPage](t, client.call(http.MethodGet, "/reimbursements?limit=1&cursor="+*page.NextCursor, "", nil, http.StatusOK))
	if len(next.Items) != 1 || next.Items[0].Id == page.Items[0].Id {
		t.Fatal("keyset cursor repeated a debt")
	}
	client.call(http.MethodGet, "/reimbursements?limit=1&asset=USD&cursor="+*page.NextCursor, "", nil, http.StatusBadRequest)
	other := f.otherFamily()
	if _, err := f.store.LoadCommand(testContext, other.p, key); err == nil {
		t.Fatal("foreign command result visible")
	}

	raw := strings.NewReader(`{"creditorMemberId":"` + string(f.members[0].ID) + `","debtorMemberId":"` + string(f.members[1].ID) + `","amount":{"amount":"1","asset":"RUB"},"reason":"CSRF"}`)
	req, _ := http.NewRequest(http.MethodPost, "http://localhost/api/v1/reimbursements", raw)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("Idempotency-Key", uuid.NewString())
	req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(client.token)})
	response := httptest.NewRecorder()
	client.handler.ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("missing CSRF accepted: %d", response.Code)
	}
}

func TestRollbackAndRestartCommandRecovery(t *testing.T) {
	f := newFixture(t)
	request := commands.Request{ID: uuid.NewString(), Kind: "reimbursements.create", PayloadHash: strings.Repeat("c", 64)}
	_, err := f.executor.Execute(testContext, f.p, request, func(ctx context.Context) (command.Result, error) {
		if _, createErr := f.reimbursements.Create(ctx, f.p, journal.ReimbursementCreateInput{
			CreditorMemberID: f.members[0].ID,
			DebtorMemberID:   f.members[1].ID,
			Amount:           cash("10", money.RUB),
			Reason:           "Rolled back debt",
		}); createErr != nil {
			return command.Result{}, createErr
		}
		return command.Result{}, errors.New("synthetic failure before commit")
	})
	if err == nil || f.count("reimbursements") != 0 {
		t.Fatal("failed command left a partial debt")
	}
	pending, loadErr := f.store.LoadCommand(testContext, f.p, request.ID)
	if loadErr != nil || pending.Status() != command.Pending {
		t.Fatal("infrastructure failure did not retain pending command")
	}

	client := f.client(f.p)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "25", "asset": "RUB"}, "reason": "Committed debt"}, http.StatusAccepted)
	f.restart()
	recovered, loadErr := f.store.LoadCommand(testContext, f.p, key)
	if loadErr != nil || recovered.Status() != command.Succeeded {
		t.Fatal("committed command result was not recovered after restart")
	}
	result, ok := recovered.Result()
	if !ok {
		t.Fatal("recovered command has no result")
	}
	if _, loadErr = f.store.Reimbursement(testContext, f.p, result.ResourceID); loadErr != nil {
		t.Fatal("committed debt was not recovered after restart")
	}
}

func (f *fixture) balance(id, field string) string {
	f.t.Helper()
	value, err := f.store.Balance(testContext, f.p, id, field)
	if err != nil {
		f.t.Fatal(err)
	}
	amount, known := value.Amount.Value()
	if !known {
		f.t.Fatal("unknown balance")
	}
	return amount.Amount()
}

func (f *fixture) count(table string) int {
	f.t.Helper()
	var count int
	if err := f.admin.QueryRow(testContext, "SELECT count(*) FROM want_keep."+table).Scan(&count); err != nil {
		f.t.Fatal(err)
	}
	return count
}

func (f *fixture) otherFamily() *fixture {
	f.t.Helper()
	other := &fixture{t: f.t, store: f.store, admin: f.admin, dsn: f.dsn, now: f.now, writer: f.writer, executor: f.executor, family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Other synthetic family"}}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Other member"}
	other.members = []household.Membership{{ID: household.MembershipID(uuid.NewString()), HouseholdID: other.family.ID, UserID: user.ID, Active: true}}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err := f.store.InitializeHousehold(testContext, other.family, []household.User{user}, other.members, zone, 2); err != nil {
		f.t.Fatal(err)
	}
	other.p, _ = other.members[0].Principal()
	return other
}
