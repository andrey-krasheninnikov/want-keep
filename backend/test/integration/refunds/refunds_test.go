//go:build integration

package refunds_test

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matchingdomain "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

func decodeResponse[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func createExpense(t *testing.T, client *client, accountID string, asset money.Asset, amount, half string) generated.CommandSucceeded {
	t.Helper()
	memberA, memberB := client.f.members[0].ID, client.f.members[1].ID
	input := map[string]any{
		"type": "expense", "accountId": accountID, "occurredAt": "2026-08-31T12:00:00Z",
		"amount": map[string]any{"amount": amount, "asset": asset},
		"payer":  map[string]any{"state": "known", "memberId": memberA},
		"allocation": map[string]any{"mode": "amounts", "purpose": "shared", "members": []any{
			map[string]any{"memberId": memberA, "amount": map[string]any{"amount": half, "asset": asset}},
			map[string]any{"memberId": memberB, "amount": map[string]any{"amount": half, "asset": asset}},
		}},
	}
	return decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions", uuid.NewString(), input, http.StatusAccepted))
}

func createRefund(t *testing.T, client *client, purchaseID, accountID string, asset money.Asset, amount, key string) generated.CommandSucceeded {
	t.Helper()
	input := map[string]any{"purchaseId": purchaseID, "purchaseExpectedRevision": 1, "receivingAccountId": accountID, "occurredAt": "2026-09-07T12:00:00Z", "amount": map[string]any{"amount": amount, "asset": asset}, "returnedItems": []any{}, "fees": []any{}, "reason": "Confirmed return"}
	return decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/refunds", key, input, http.StatusAccepted))
}

func readTransaction(t *testing.T, client *client, id string) generated.Transaction {
	t.Helper()
	return decodeResponse[generated.Transaction](t, client.call(http.MethodGet, "/transactions/"+id, "", nil, http.StatusOK))
}

func TestRefundKeepsCashDateAndReducesOriginalExpenseAllocation(t *testing.T) {
	f := newFixture(t)
	first, second := f.client(f.p), f.client(f.q)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, first, accountID, money.RUB, "1000", "500")
	if purchase.Status != "succeeded" || f.available(accountID, f.p) != "4000" {
		t.Fatalf("purchase result=%+v balance=%s", purchase, f.available(accountID, f.p))
	}
	key := uuid.NewString()
	created := createRefund(t, second, purchase.Result.Id, accountID, money.RUB, "400", key)
	if created.Status != "succeeded" || f.available(accountID, f.q) != "4400" {
		t.Fatalf("refund result=%+v balance=%s", created, f.available(accountID, f.q))
	}
	purchaseView := readTransaction(t, first, purchase.Result.Id)
	if len(purchaseView.Refunds) != 1 {
		t.Fatalf("refund attribution missing: %+v", purchaseView.Refunds)
	}
	refund := purchaseView.Refunds[0]
	if refund.ExpenseMonth != "2026-08" || refund.CashDate != "2026-09-07" || refund.Amount.Amount != "400" || len(refund.Members) != 2 || refund.Members[0].Amount.Amount != "200.000000" || refund.Members[1].Amount.Amount != "200.000000" {
		t.Fatalf("refund attribution=%+v", refund)
	}
	missing, err := refund.Valuation.AsUnavailableRefundValuation()
	if err != nil || missing.State != "unavailable" || missing.Reason != "historical_basis_unavailable" {
		t.Fatalf("valuation=%+v err=%v", missing, err)
	}
	refundView := readTransaction(t, second, created.Result.Id)
	if refundView.OriginalTransactionId == nil || *refundView.OriginalTransactionId != purchase.Result.Id || len(refundView.EconomicComponents) != 0 {
		t.Fatalf("refund transaction=%+v", refundView)
	}
	replayed := createRefund(t, second, purchase.Result.Id, accountID, money.RUB, "400", key)
	if replayed.Result.Id != created.Result.Id || f.available(accountID, f.p) != "4400" {
		t.Fatalf("replay=%+v balance=%s", replayed, f.available(accountID, f.p))
	}
}

func TestCreditCardRefundDoesNotInventOwnedMoney(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	cashID := f.account(money.RUB, "5000")
	cardID := f.accountWithProduct(money.RUB, "0", "credit_card")
	purchase := createExpense(t, client, cashID, money.RUB, "1000", "500")
	created := createRefund(t, client, purchase.Result.Id, cardID, money.RUB, "400", uuid.NewString())
	revision, found, err := f.store.CurrentLedgerRevision(testContext, f.p, created.Result.Id)
	if err != nil || !found || len(revision.Postings) != 1 || revision.Postings[0].Funding != "unknown" {
		t.Fatalf("credit-card refund revision=%+v found=%v err=%v", revision, found, err)
	}
	for _, field := range []string{"owned", "available", "debt"} {
		balance, err := f.store.Balance(testContext, f.p, cardID, field)
		if err != nil {
			t.Fatal(err)
		}
		if _, known := balance.Amount.Value(); known {
			t.Fatalf("credit-card %s falsely known: %+v", field, balance)
		}
	}
	if f.available(cashID, f.p) != "4000" {
		t.Fatalf("purchase cash changed twice: %s", f.available(cashID, f.p))
	}
}

func TestRefundUsesFrozenHistoricalValuation(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.USD, "100")
	purchase := createExpense(t, client, accountID, money.USD, "10", "5")
	date, _ := calendar.ParseDate("2026-08-31")
	rate, _ := money.NewRate(money.USD, money.RUB, "90")
	observation, err := f.store.SaveRateObservation(testContext, valuation.Observation{ID: uuid.NewString(), ProviderAssetID: "R01235", Source: "cbr", Transport: "xml_daily", Revision: 1, RequestedDate: date, EffectiveAt: instant("2026-08-31T00:00:00Z"), FetchedAt: f.now, Granularity: "daily", Rate: rate})
	if err != nil {
		t.Fatal(err)
	}
	converted := cash("-900", money.RUB)
	snapshot := valuation.Snapshot{OperationID: purchase.Result.Id, OperationRevision: 1, ComponentIndex: 0, ComponentKind: "expense", ValuationRevision: 1, Native: cash("-10", money.USD), Reporting: &converted, Target: money.RUB, RequestedDate: date, Freshness: reporting.Fresh, Legs: []valuation.Observation{observation}, RecordedAt: f.now}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.SaveValuationSnapshot(ctx, f.p, snapshot) }); err != nil {
		t.Fatal(err)
	}
	created := createRefund(t, client, purchase.Result.Id, accountID, money.USD, "4", uuid.NewString())
	view := readTransaction(t, client, created.Result.Id)
	known, err := view.Refunds[0].Valuation.AsKnownRefundValuation()
	if err != nil || known.State != "known" || known.Amount.Asset != "RUB" || known.Amount.Amount != "360.000000" || !strings.HasPrefix(known.BasisRef, "valuation:"+purchase.Result.Id+":1:0:RUB:") {
		t.Fatalf("valuation=%+v err=%v", known, err)
	}
	correction := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+purchase.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 1, "reason": "Correct merchant label", "merchant": "Shop",
	}, http.StatusAccepted))
	if correction.Result.Revision != 2 {
		t.Fatalf("merchant correction=%+v", correction)
	}
	view = readTransaction(t, client, created.Result.Id)
	known, err = view.Refunds[0].Valuation.AsKnownRefundValuation()
	if err != nil || known.Amount.Amount != "360.000000" || !strings.HasPrefix(known.BasisRef, "valuation:"+purchase.Result.Id+":1:0:RUB:") {
		t.Fatalf("valuation after purchase correction=%+v err=%v", known, err)
	}
	conflict := decodeResponse[generated.CommandFailed](t, client.call(http.MethodPost, "/transactions/"+purchase.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 2, "reason": "Change purchase date", "occurredAt": "2026-08-30T12:00:00Z",
	}, http.StatusAccepted))
	if conflict.Error.Code != "decision_conflict" || readTransaction(t, client, purchase.Result.Id).Revision != 2 {
		t.Fatalf("incompatible purchase correction=%+v", conflict)
	}
}

func TestExcludedRefundDoesNotBlockReplacement(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	first := createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "1000", uuid.NewString())
	excluded := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+first.Result.Id+"/exclude", uuid.NewString(), map[string]any{
		"expectedRevision": 1, "reason": "Duplicate refund",
	}, http.StatusAccepted))
	if excluded.Status != "succeeded" || f.available(accountID, f.p) != "4000" {
		t.Fatalf("exclude=%+v balance=%s", excluded, f.available(accountID, f.p))
	}
	count, err := f.store.ActiveRefundCountForPurchase(testContext, f.p, purchase.Result.Id)
	if err != nil || count != 0 {
		t.Fatalf("active refunds after exclusion=%d err=%v", count, err)
	}
	replacement := createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "1000", uuid.NewString())
	if replacement.Status != "succeeded" || f.available(accountID, f.p) != "5000" {
		t.Fatalf("replacement=%+v balance=%s", replacement, f.available(accountID, f.p))
	}
	view := readTransaction(t, client, purchase.Result.Id)
	states := map[string]generated.RefundAttributionState{}
	for _, link := range view.Refunds {
		states[link.Id] = link.State
	}
	if len(view.Refunds) != 2 || states[first.Result.Id] != "inactive" || states[replacement.Result.Id] != "applied" {
		t.Fatalf("refund links=%+v", view.Refunds)
	}
}

func TestPartialRefundsPreserveFrozenValuationTotal(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.USD, "100")
	purchase := createExpense(t, client, accountID, money.USD, "6", "3")
	if _, err := f.admin.Exec(testContext, `INSERT INTO want_keep.transaction_historical_values(household_id,operation_id,operation_revision,basis_ref,native_amount,native_asset,reporting_amount,reporting_asset) VALUES($1,$2,1,'synthetic:rounding',6,'USD',1,'RUB')`, f.family.ID, purchase.Result.Id); err != nil {
		t.Fatal(err)
	}
	for range 6 {
		createRefund(t, client, purchase.Result.Id, accountID, money.USD, "1", uuid.NewString())
	}
	view := readTransaction(t, client, purchase.Result.Id)
	if len(view.Refunds) != 6 {
		t.Fatalf("refund count=%d", len(view.Refunds))
	}
	total := cash("0", money.RUB)
	for _, refund := range view.Refunds {
		known, err := refund.Valuation.AsKnownRefundValuation()
		if err != nil || refund.Remaining.Amount != "0" {
			t.Fatalf("refund=%+v valuation=%+v err=%v", refund, known, err)
		}
		value := cash(known.Amount.Amount, money.RUB)
		total, err = total.Add(value)
		if err != nil {
			t.Fatal(err)
		}
	}
	if total.Amount() != "1.000000" {
		t.Fatalf("valuation total=%s", total.Amount())
	}
}

func TestRefundCorrectionAndExclusionRecalculateOnce(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	created := createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "400", uuid.NewString())
	corrected := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+created.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 1,
		"reason":           "Correct confirmed refund amount",
		"principal": []any{map[string]any{
			"accountId": accountID,
			"money":     map[string]any{"amount": "500", "asset": "RUB"},
			"role":      "principal",
			"funding":   "own",
			"treatment": "movement",
		}},
	}, http.StatusAccepted))
	if corrected.Status != "succeeded" || corrected.Result.Revision != 2 || f.available(accountID, f.p) != "4500" {
		t.Fatalf("correction=%+v balance=%s", corrected, f.available(accountID, f.p))
	}
	view := readTransaction(t, client, created.Result.Id)
	if len(view.Refunds) != 1 || view.Refunds[0].RefundRevision != 2 || view.Refunds[0].Amount.Amount != "500" {
		t.Fatalf("corrected attribution=%+v", view.Refunds)
	}
	original := decodeResponse[generated.Transaction](t, client.call(http.MethodGet, "/transactions/"+created.Result.Id+"/revisions/1", "", nil, http.StatusOK))
	if len(original.Refunds) != 1 || original.Refunds[0].RefundRevision != 1 || original.Refunds[0].Amount.Amount != "400" || original.Refunds[0].Remaining.Amount != "600" {
		t.Fatalf("historical attribution=%+v", original.Refunds)
	}
	history := decodeResponse[generated.TransactionHistoryPage](t, client.call(http.MethodGet, "/transactions/"+created.Result.Id+"/history", "", nil, http.StatusOK))
	if len(history.Items) < 2 || len(history.Items[0].Transaction.Refunds) != 1 || history.Items[0].Transaction.Refunds[0].Amount.Amount != "500" || history.Items[0].Before == nil || len(history.Items[0].Before.Refunds) != 1 || history.Items[0].Before.Refunds[0].Amount.Amount != "400" {
		t.Fatalf("refund history=%+v", history.Items)
	}
	excluded := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+created.Result.Id+"/exclude", uuid.NewString(), map[string]any{"expectedRevision": 2, "reason": "Exclude duplicate source record"}, http.StatusAccepted))
	if excluded.Status != "succeeded" || excluded.Result.Revision != 3 || f.available(accountID, f.p) != "4000" {
		t.Fatalf("exclusion=%+v balance=%s", excluded, f.available(accountID, f.p))
	}
	view = readTransaction(t, client, created.Result.Id)
	if view.Refunds[0].State != "inactive" || view.Refunds[0].Remaining.Amount != "1000" {
		t.Fatalf("inactive attribution=%+v", view.Refunds[0])
	}
	if view.DecisionId == nil {
		t.Fatal("exclusion decision missing")
	}
	undone := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+created.Result.Id+"/undo", uuid.NewString(), map[string]any{
		"decisionId": *view.DecisionId,
		"reason":     "Restore valid refund",
		"expectedRevisions": []any{
			map[string]any{"transactionId": created.Result.Id, "expectedRevision": 3},
		},
	}, http.StatusAccepted))
	if undone.Result.Revision != 4 || f.available(accountID, f.p) != "4500" {
		t.Fatalf("undo=%+v balance=%s", undone, f.available(accountID, f.p))
	}
	view = readTransaction(t, client, created.Result.Id)
	if view.Refunds[0].State != "applied" || view.Refunds[0].Amount.Amount != "500" {
		t.Fatalf("restored attribution=%+v", view.Refunds[0])
	}
	f.transition(created.Result.Id, "reversed")
	if f.available(accountID, f.p) != "4000" {
		t.Fatalf("reversed balance=%s", f.available(accountID, f.p))
	}
	view = readTransaction(t, client, created.Result.Id)
	if view.Refunds[0].State != "inactive" || view.Refunds[0].Remaining.Amount != "1000" {
		t.Fatalf("reversed attribution=%+v", view.Refunds[0])
	}
	var reviewRequests int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.refund_review_requests WHERE household_id=$1 AND refund_operation_id=$2`, f.family.ID, created.Result.Id).Scan(&reviewRequests); err != nil || reviewRequests != 5 {
		t.Fatalf("review requests=%d err=%v", reviewRequests, err)
	}
}

func TestPurchaseExclusionAndUndoRecalculateRefundAttribution(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	created := createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "400", uuid.NewString())
	excluded := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+purchase.Result.Id+"/exclude", uuid.NewString(), map[string]any{"expectedRevision": 1, "reason": "Exclude invalid purchase"}, http.StatusAccepted))
	if excluded.Result.Revision != 2 || f.available(accountID, f.p) != "5400" {
		t.Fatalf("purchase exclusion=%+v balance=%s", excluded, f.available(accountID, f.p))
	}
	view := readTransaction(t, client, created.Result.Id)
	if view.Refunds[0].State != "inactive" || view.Refunds[0].Remaining.Amount != "1000" {
		t.Fatalf("inactive attribution=%+v", view.Refunds[0])
	}
	purchaseView := readTransaction(t, client, purchase.Result.Id)
	if purchaseView.DecisionId == nil {
		t.Fatal("purchase exclusion decision missing")
	}
	undo := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+purchase.Result.Id+"/undo", uuid.NewString(), map[string]any{
		"decisionId": *purchaseView.DecisionId,
		"reason":     "Restore purchase",
		"expectedRevisions": []any{
			map[string]any{"transactionId": purchase.Result.Id, "expectedRevision": 2},
		},
	}, http.StatusAccepted))
	if undo.Result.Revision != 3 || f.available(accountID, f.p) != "4400" {
		t.Fatalf("purchase undo=%+v balance=%s", undo, f.available(accountID, f.p))
	}
	view = readTransaction(t, client, created.Result.Id)
	if view.Refunds[0].State != "applied" || view.Refunds[0].Remaining.Amount != "600" {
		t.Fatalf("restored attribution=%+v", view.Refunds[0])
	}
}

func TestExistingRefundLinkDoesNotMoveMoneyTwice(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	refundID := f.postRefund(accountID, "400")
	before := f.available(accountID, f.p)
	linked := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+refundID+"/links", uuid.NewString(), map[string]any{
		"kind":   "refund",
		"reason": "Link imported refund to purchase",
		"expectedRevisions": []any{
			map[string]any{"transactionId": refundID, "expectedRevision": 1},
			map[string]any{"transactionId": purchase.Result.Id, "expectedRevision": 1},
		},
		"refund": map[string]any{"purchaseId": purchase.Result.Id, "expectedRevision": 0, "returnedItems": []any{}},
	}, http.StatusAccepted))
	if linked.Status != "succeeded" || linked.Result.Id != refundID || f.available(accountID, f.p) != before {
		t.Fatalf("link=%+v before=%s after=%s", linked, before, f.available(accountID, f.p))
	}
	if view := readTransaction(t, client, purchase.Result.Id); len(view.Refunds) != 1 || view.Refunds[0].Id != refundID {
		t.Fatalf("linked refund=%+v", view.Refunds)
	}
	history := decodeResponse[generated.RefundHistoryPage](t, client.call(http.MethodGet, "/transactions/"+refundID+"/refund-history", "", nil, http.StatusOK))
	if len(history.Items) != 1 || history.Items[0].Refund.Revision != 1 || history.Items[0].Refund.PurchaseId != purchase.Result.Id || history.Items[0].ActorId != string(f.p.UserID()) {
		t.Fatalf("linked refund history=%+v", history)
	}
	updated := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+refundID+"/links", uuid.NewString(), map[string]any{
		"kind":   "refund",
		"reason": "Confirm imported refund link",
		"expectedRevisions": []any{
			map[string]any{"transactionId": refundID, "expectedRevision": 1},
			map[string]any{"transactionId": purchase.Result.Id, "expectedRevision": 1},
		},
		"refund": map[string]any{"purchaseId": purchase.Result.Id, "expectedRevision": 1, "returnedItems": []any{}},
	}, http.StatusAccepted))
	if updated.Status != "succeeded" || f.available(accountID, f.p) != before {
		t.Fatalf("updated link=%+v", updated)
	}
	firstPage := decodeResponse[generated.RefundHistoryPage](t, client.call(http.MethodGet, "/transactions/"+refundID+"/refund-history?limit=1", "", nil, http.StatusOK))
	if len(firstPage.Items) != 1 || firstPage.Items[0].Refund.Revision != 2 || firstPage.NextCursor == nil {
		t.Fatalf("first refund history page=%+v", firstPage)
	}
	secondPage := decodeResponse[generated.RefundHistoryPage](t, client.call(http.MethodGet, "/transactions/"+refundID+"/refund-history?limit=1&cursor="+*firstPage.NextCursor, "", nil, http.StatusOK))
	if len(secondPage.Items) != 1 || secondPage.Items[0].Refund.Revision != 1 || secondPage.NextCursor != nil {
		t.Fatalf("second refund history page=%+v", secondPage)
	}
	other := f.client(f.q)
	other.call(http.MethodGet, "/transactions/"+refundID+"/refund-history?limit=1&cursor="+*firstPage.NextCursor, "", nil, http.StatusBadRequest)
	if visible := decodeResponse[generated.RefundHistoryPage](t, other.call(http.MethodGet, "/transactions/"+refundID+"/refund-history", "", nil, http.StatusOK)); len(visible.Items) != 2 {
		t.Fatalf("household refund history=%+v", visible)
	}
	foreign := newFixture(t)
	foreign.client(foreign.p).call(http.MethodGet, "/transactions/"+refundID+"/refund-history", "", nil, http.StatusNotFound)
	client.call(http.MethodGet, "/transactions/"+purchase.Result.Id+"/refund-history", "", nil, http.StatusNotFound)
}

func TestWaitingMatchedRefundDoesNotReducePurchaseExpense(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	entry := ledger.Revision{OperationID: uuid.NewString(), Revision: 1, ActorID: f.p.UserID(), Reason: "Confirmed refund receipt", Type: ledger.Refund, State: ledger.Posted, OccurredAt: f.now, Origin: "manual", FeeKnowledge: ledger.KnownFees, PayerState: "not_applicable", Postings: []ledger.Posting{{AccountID: accountID, Money: cash("400", money.RUB), Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}, RecordedAt: f.now}
	entry, err := entry.InTimezone(zone)
	if err != nil {
		t.Fatal(err)
	}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, entry, 0) }); err != nil {
		t.Fatal(err)
	}
	refund := entry.Clone()
	refund.OperationID = uuid.NewString()
	refund.Reason, refund.Type, refund.Origin = "Possible duplicate refund", ledger.Refund, "source"
	matcher := matching.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return matcher.Append(ctx, f.p, refund, 0) }); err != nil {
		t.Fatal(err)
	}
	stored, found, err := f.store.CurrentLedgerRevision(testContext, f.p, refund.OperationID)
	if err != nil || !found || stored.Participation.State != "waiting" || f.available(accountID, f.p) != "4400" {
		t.Fatalf("waiting refund=%+v found=%v err=%v", stored, found, err)
	}
	linked := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+refund.OperationID+"/links", uuid.NewString(), map[string]any{
		"kind": "refund", "reason": "Check possible duplicate against purchase",
		"expectedRevisions": []any{
			map[string]any{"transactionId": refund.OperationID, "expectedRevision": 1},
			map[string]any{"transactionId": purchase.Result.Id, "expectedRevision": 1},
		},
		"refund": map[string]any{"purchaseId": purchase.Result.Id, "expectedRevision": 0, "returnedItems": []any{}},
	}, http.StatusAccepted))
	view := readTransaction(t, client, purchase.Result.Id)
	if linked.Status != "succeeded" || len(view.Refunds) != 1 || view.Refunds[0].State != "inactive" || view.Refunds[0].Remaining.Amount != "1000" || f.available(accountID, f.p) != "4400" {
		t.Fatalf("waiting refund incorrectly applied: %+v %+v", linked, view.Refunds)
	}
	replacement := createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "1000", uuid.NewString())
	if replacement.Status != "succeeded" || f.available(accountID, f.p) != "5400" {
		t.Fatalf("waiting refund consumed capacity: %+v", replacement)
	}
}

func TestManualRefundAndImportedReceiptDoNotCreditAccountTwice(t *testing.T) {
	for _, importedFirst := range []bool{false, true} {
		t.Run(map[bool]string{false: "manual first", true: "import first"}[importedFirst], func(t *testing.T) {
			f := newFixture(t)
			client := f.client(f.p)
			accountID := f.account(money.RUB, "5000")
			purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
			var manual generated.CommandSucceeded
			if !importedFirst {
				manual = createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "400", uuid.NewString())
			}
			zone, _ := calendar.ParseTimezone("Europe/Moscow")
			imported := ledger.Revision{OperationID: uuid.NewString(), Revision: 1, ActorID: f.p.UserID(), Reason: "Imported refund receipt", Type: ledger.Refund, State: ledger.Posted, OccurredAt: f.now, Origin: "source", FeeKnowledge: ledger.KnownFees, PayerState: "not_applicable", Postings: []ledger.Posting{{AccountID: accountID, Money: cash("400", money.RUB), Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}, RecordedAt: f.now}
			imported, err := imported.InTimezone(zone)
			if err != nil {
				t.Fatal(err)
			}
			matcher := matching.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
			if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return matcher.Append(ctx, f.p, imported, 0) }); err != nil {
				t.Fatal(err)
			}
			if importedFirst {
				manual = createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "400", uuid.NewString())
			}
			stored, found, err := f.store.CurrentLedgerRevision(testContext, f.p, imported.OperationID)
			if err != nil || !found || (!importedFirst && stored.Participation.State != "waiting") || f.available(accountID, f.p) != "4400" {
				t.Fatalf("imported refund duplicated cash: %+v found=%v err=%v", stored, found, err)
			}
			linked := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+imported.OperationID+"/links", uuid.NewString(), map[string]any{
				"kind": "refund", "reason": "Link imported receipt to purchase",
				"expectedRevisions": []any{
					map[string]any{"transactionId": imported.OperationID, "expectedRevision": 1},
					map[string]any{"transactionId": purchase.Result.Id, "expectedRevision": 1},
				},
				"refund": map[string]any{"purchaseId": purchase.Result.Id, "expectedRevision": 0, "returnedItems": []any{}},
			}, http.StatusAccepted))
			view := readTransaction(t, client, purchase.Result.Id)
			if linked.Status != "succeeded" || len(view.Refunds) != 2 || view.Refunds[0].State == view.Refunds[1].State || f.available(accountID, f.p) != "4400" {
				t.Fatalf("duplicate refund attribution: manual=%s imported=%s refunds=%+v", manual.Result.Id, imported.OperationID, view.Refunds)
			}
			if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
				_, linkErr := matcher.Link(ctx, f.p, matching.LinkInput{
					Kind: matchingdomain.Payment, PrimaryID: manual.Result.Id, Reason: "Same refund receipt",
					Members: []matchingdomain.Member{{OperationID: manual.Result.Id, Revision: 1}, {OperationID: imported.OperationID, Revision: 1}},
				})
				return linkErr
			}); err != nil {
				t.Fatal(err)
			}
			stored, found, err = f.store.CurrentLedgerRevision(testContext, f.p, imported.OperationID)
			if err != nil || !found || stored.Participation.State != "linked" || stored.Contributes(0) != importedFirst || f.available(accountID, f.p) != "4400" {
				t.Fatalf("linked imported refund duplicated cash: %+v found=%v err=%v", stored, found, err)
			}
			view = readTransaction(t, client, purchase.Result.Id)
			if len(view.Refunds) != 2 || view.Refunds[0].State == view.Refunds[1].State {
				t.Fatalf("linked imported refund duplicated attribution: %+v", view.Refunds)
			}
		})
	}
}

func TestRefundRoundTripsAllAssetsAndThirdAssetFee(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		accountID := f.account(asset, "1")
		purchase := createExpense(t, client, accountID, asset, "0.2", "0.1")
		created := createRefund(t, client, purchase.Result.Id, accountID, asset, "0.1", uuid.NewString())
		view := readTransaction(t, client, created.Result.Id)
		if len(view.Refunds) != 1 || view.Refunds[0].Amount.Asset != generated.Asset(asset) || view.Refunds[0].Amount.Amount != "0.1" {
			t.Fatalf("%s refund=%+v", asset, view.Refunds)
		}
	}
	rubles := f.account(money.RUB, "1000")
	feeAccount := f.account(money.USDT, "10")
	purchase := createExpense(t, client, rubles, money.RUB, "1000", "500")
	input := map[string]any{
		"purchaseId": purchase.Result.Id, "purchaseExpectedRevision": 1, "receivingAccountId": rubles,
		"occurredAt": "2026-09-07T12:00:00Z", "amount": map[string]any{"amount": "40", "asset": "RUB"},
		"returnedItems": []any{}, "reason": "Return with network fee",
		"fees": []any{map[string]any{"accountId": feeAccount, "amount": map[string]any{"amount": "1", "asset": "USDT"}}},
	}
	created := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/refunds", uuid.NewString(), input, http.StatusAccepted))
	if created.Status != "succeeded" || f.available(feeAccount, f.p) != "9" {
		t.Fatalf("third-asset fee=%+v balance=%s", created, f.available(feeAccount, f.p))
	}
	view := readTransaction(t, client, created.Result.Id)
	if len(view.EconomicComponents) != 1 || view.EconomicComponents[0].Kind != "fee" || view.EconomicComponents[0].Money.Asset != "USDT" {
		t.Fatalf("fee components=%+v", view.EconomicComponents)
	}
}

func TestItemRefundPreservesReceiptBasisAndCapsEachItemAfterCorrection(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "100")
	purchase := createExpense(t, client, accountID, money.RUB, "10", "5")
	firstItem, secondItem := uuid.NewString(), uuid.NewString()
	correctionResponse := client.call(http.MethodPost, "/transactions/"+purchase.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 1,
		"reason":           "Attach discounted receipt items",
		"receiptItems": map[string]any{
			"action":        "replace",
			"totalDiscount": map[string]any{"amount": "1", "asset": "RUB"},
			"items": []any{
				map[string]any{"id": firstItem, "name": "Shared item", "quantity": "1", "gross": map[string]any{"amount": "7", "asset": "RUB"}, "discount": map[string]any{"amount": "1", "asset": "RUB"}},
				map[string]any{"id": secondItem, "name": "Other item", "quantity": "1", "gross": map[string]any{"amount": "4", "asset": "RUB"}, "discount": map[string]any{"amount": "0", "asset": "RUB"}},
			},
		},
	}, http.StatusAccepted)
	correction := decodeResponse[generated.CommandSucceeded](t, correctionResponse)
	if correction.Result.Revision != 2 {
		t.Fatalf("receipt correction=%s", correctionResponse.Body.String())
	}

	response := client.call(http.MethodPost, "/refunds", uuid.NewString(), map[string]any{
		"purchaseId": purchase.Result.Id, "purchaseExpectedRevision": 2, "receivingAccountId": accountID,
		"occurredAt": "2026-09-07T12:00:00Z", "amount": map[string]any{"amount": "2", "asset": "RUB"},
		"returnedItems": []any{map[string]any{"itemId": firstItem, "amount": map[string]any{"amount": "2", "asset": "RUB"}}},
		"fees":          []any{}, "reason": "Partial item return",
	}, http.StatusAccepted)
	created := decodeResponse[generated.CommandSucceeded](t, response)
	if created.Result.Id == "" {
		t.Fatalf("item refund creation=%s", response.Body.String())
	}
	view := readTransaction(t, client, created.Result.Id)
	refund := view.Refunds[0]
	if refund.State != "applied" || len(refund.ReturnedItems) != 1 || refund.ReturnedItems[0].ItemId != firstItem || refund.ReturnedItems[0].Amount.Amount != "2" || len(refund.Members) != 2 {
		t.Fatalf("item refund=%+v", refund)
	}
	corrected := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+created.Result.Id+"/corrections", uuid.NewString(), map[string]any{
		"expectedRevision": 1, "reason": "Correct confirmed return amount",
		"principal": []any{map[string]any{"accountId": accountID, "money": map[string]any{"amount": "3", "asset": "RUB"}, "role": "principal", "funding": "own", "treatment": "movement"}},
	}, http.StatusAccepted))
	if corrected.Status != "succeeded" || corrected.Result.Revision != 2 || f.available(accountID, f.p) != "93" {
		t.Fatalf("corrected refund=%+v balance=%s", corrected, f.available(accountID, f.p))
	}
	unresolved := readTransaction(t, client, created.Result.Id).Refunds[0]
	if unresolved.State != "clarification" || unresolved.Amount.Amount != "3" || len(unresolved.ReturnedItems) != 0 || len(unresolved.Members) != 0 || len(unresolved.Categories) != 0 {
		t.Fatalf("corrected item attribution=%+v", unresolved)
	}
	linked := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/transactions/"+created.Result.Id+"/links", uuid.NewString(), map[string]any{
		"kind": "refund", "reason": "Confirm corrected returned item",
		"expectedRevisions": []any{map[string]any{"transactionId": created.Result.Id, "expectedRevision": 2}, map[string]any{"transactionId": purchase.Result.Id, "expectedRevision": 2}},
		"refund":            map[string]any{"purchaseId": purchase.Result.Id, "expectedRevision": unresolved.Revision, "returnedItems": []any{map[string]any{"itemId": firstItem, "amount": map[string]any{"amount": "3", "asset": "RUB"}}}},
	}, http.StatusAccepted))
	if linked.Status != "succeeded" || f.available(accountID, f.p) != "93" {
		t.Fatalf("relinked refund=%+v balance=%s", linked, f.available(accountID, f.p))
	}
	refund = readTransaction(t, client, created.Result.Id).Refunds[0]
	if refund.State != "applied" || len(refund.ReturnedItems) != 1 || refund.ReturnedItems[0].Amount.Amount != "3" {
		t.Fatalf("relinked item attribution=%+v", refund)
	}

	failed := decodeResponse[generated.CommandFailed](t, client.call(http.MethodPost, "/refunds", uuid.NewString(), map[string]any{
		"purchaseId": purchase.Result.Id, "purchaseExpectedRevision": 2, "receivingAccountId": accountID,
		"occurredAt": "2026-09-07T11:00:00Z", "amount": map[string]any{"amount": "5", "asset": "RUB"},
		"returnedItems": []any{map[string]any{"itemId": firstItem, "amount": map[string]any{"amount": "5", "asset": "RUB"}}},
		"fees":          []any{}, "reason": "Exceeds returned item remainder",
	}, http.StatusAccepted))
	if failed.Error.Code != "refund_exceeds_purchase" {
		t.Fatalf("item cap error=%+v", failed.Error)
	}

	clarified := decodeResponse[generated.CommandSucceeded](t, client.call(http.MethodPost, "/refunds", uuid.NewString(), map[string]any{
		"purchaseId": purchase.Result.Id, "purchaseExpectedRevision": 2, "receivingAccountId": accountID,
		"occurredAt": "2026-09-07T10:00:00Z", "amount": map[string]any{"amount": "1", "asset": "RUB"},
		"returnedItems": []any{}, "fees": []any{}, "reason": "Receipt position needs clarification",
	}, http.StatusAccepted))
	clarification := readTransaction(t, client, clarified.Result.Id).Refunds[0]
	if clarification.State != "clarification" || len(clarification.Members) != 0 || len(clarification.Categories) != 0 {
		t.Fatalf("clarification=%+v", clarification)
	}
}

func TestRefundRefusalsAreTerminalAndLeaveNoFinancialChange(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	created := createRefund(t, client, purchase.Result.Id, accountID, money.RUB, "400", uuid.NewString())
	for _, candidate := range []struct {
		path, code string
		input      map[string]any
	}{
		{"/refunds", "not_found", map[string]any{"purchaseId": purchase.Result.Id, "purchaseExpectedRevision": 1, "receivingAccountId": uuid.NewString(), "occurredAt": "2026-09-07T12:00:00Z", "amount": map[string]any{"amount": "1", "asset": "RUB"}, "returnedItems": []any{}, "fees": []any{}, "reason": "Unknown receiving account"}},
		{"/refunds", "invalid_transaction", map[string]any{"purchaseId": purchase.Result.Id, "purchaseExpectedRevision": 1, "receivingAccountId": accountID, "occurredAt": "2026-09-07T12:00:00Z", "amount": map[string]any{"amount": "1", "asset": "RUB"}, "returnedItems": []any{}, "fees": []any{map[string]any{"accountId": accountID, "amount": map[string]any{"amount": "0.1", "asset": "RUB"}, "funding": "credit"}}, "reason": "Invalid credit funding on cash account"}},
		{"/transactions/" + created.Result.Id + "/corrections", "refund_exceeds_purchase", map[string]any{"expectedRevision": 1, "reason": "Excessive refund", "principal": []any{map[string]any{"accountId": accountID, "money": map[string]any{"amount": "1100", "asset": "RUB"}, "role": "principal", "treatment": "movement", "funding": "own"}}}},
	} {
		key := uuid.NewString()
		first := decodeResponse[generated.CommandFailed](t, client.call(http.MethodPost, candidate.path, key, candidate.input, http.StatusAccepted))
		second := decodeResponse[generated.CommandFailed](t, client.call(http.MethodPost, candidate.path, key, candidate.input, http.StatusAccepted))
		if first.Status != "failed" || string(first.Error.Code) != candidate.code || string(second.Error.Code) != candidate.code || f.available(accountID, f.p) != "4400" || readTransaction(t, client, created.Result.Id).Revision != 1 {
			t.Fatalf("refusal=%+v replay=%+v", first, second)
		}
	}
}
