//go:build integration

package matching_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (c *client) transaction(id string) generated.Transaction {
	return decode[generated.Transaction](c.f.t, c.call("GET", "/transactions/"+id, "", nil, 200))
}
func (c *client) result(path string, input any) generated.CommandStatus {
	c.f.t.Helper()
	out := decode[generated.CommandStatus](c.f.t, c.call("POST", path, uuid.NewString(), input, 202))
	value, err := out.AsCommandSucceeded()
	if err != nil || value.Status != "succeeded" {
		c.f.t.Fatalf("command failed: %+v", out)
	}
	return out
}
func (f *fixture) versions(ids ...string) []generated.DecisionRevision {
	out := []generated.DecisionRevision{}
	for _, id := range ids {
		r := f.current(id)
		out = append(out, generated.DecisionRevision{TransactionId: id, ExpectedRevision: int64(r.Revision)})
	}
	return out
}
func TestHTTPMatchingQueueResolutionAndIsolation(t *testing.T) {
	f := newFixture(t)
	account := f.create(money.RUB, "5000")
	c := f.client(f.p)
	input := map[string]any{"type": "expense", "accountId": account, "amount": map[string]string{"asset": "RUB", "amount": "300"}, "occurredAt": f.now.String(), "payer": map[string]string{"state": "unknown"}, "allocation": map[string]string{"mode": "unresolved", "reason": "not clarified"}}
	c.result("/transactions", input)
	c.result("/transactions", input)
	queue := decode[generated.MatchingPage](t, c.call("GET", "/matching?state=clarification", "", nil, 200))
	if len(queue.Items) != 1 {
		t.Fatal(queue)
	}
	g := queue.Items[0]
	waiting := c.transaction(g.PrimaryId)
	if waiting.Participation == nil || waiting.Participation.State != "waiting" || len(waiting.BalanceEffects) != 0 {
		t.Fatal("missing waiting state", waiting)
	}
	quality, err := waiting.Quality.Coverage.AsIncompleteCoverage()
	if err != nil || quality.State != "partial" {
		t.Fatal("waiting quality claims complete")
	}
	current := decode[generated.MatchingCase](t, c.call("GET", "/matching/"+g.Id, "", nil, 200))
	if current.Revision != g.Revision {
		t.Fatal("case changed on read")
	}
	bankID := g.Candidates[0].TransactionId
	body := map[string]any{"expectedRevision": g.Revision, "decision": "link", "kind": "payment", "primaryId": bankID, "expectedRevisions": f.versions(g.PrimaryId, bankID), "reason": "Same payment"}
	c.result("/matching/"+g.Id+"/resolve", body)
	f.balance(account, "owned", "4700")
	linked := c.transaction(bankID)
	history := decode[generated.TransactionHistoryPage](t, c.call("GET", "/transactions/"+bankID+"/history", "", nil, 200))
	if history.Items[0].DecisionKind == nil || *history.Items[0].DecisionKind != "matching" {
		t.Fatal("matching decision missing")
	}
	c.result("/transactions/"+bankID+"/undo", map[string]any{"decisionId": *linked.DecisionId, "reason": "Undo matching", "expectedRevisions": f.versions(bankID, g.PrimaryId)})
	f.balance(account, "owned", "4700")
	outsider := f.otherFamily()
	otherClient := outsider.client(outsider.p)
	otherClient.call("GET", "/matching/"+g.Id, "", nil, 404)
	otherClient.call("GET", "/transactions/"+bankID, "", nil, 404)
	c.call("POST", "/transactions/"+bankID+"/links", uuid.NewString(), map[string]any{"kind": "refund", "expectedRevisions": f.versions(bankID, g.PrimaryId), "reason": "Missing refund details"}, 400)
	body["actorId"] = string(f.q.UserID())
	c.call("POST", "/matching/"+g.Id+"/resolve", uuid.NewString(), body, 400)
}
func TestManualTransferMatchesTwoBankCasesAndUndoKeepsBothWaiting(t *testing.T) {
	f := newFixture(t)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	c := f.client(f.p)
	var original string
	_, err := f.executor.Execute(testContext, f.p, request(), func(ctx context.Context) (command.Result, error) {
		r, err := f.ledgerService().Transfer(ctx, f.p, journal.TransferInput{FromAccountID: from, ToAccountID: to, At: f.now, Sent: cash("1000", money.RUB), Received: cash("1000", money.RUB)})
		original = r.ResourceID
		return r, err
	})
	if err != nil {
		t.Fatal(err)
	}
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	b.Type = ledger.Income
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
	for _, r := range []ledger.Revision{a, b} {
		if f.current(r.OperationID).Participation.State != "waiting" {
			t.Fatal("imported side duplicated manual movement")
		}
	}
	ids := []string{original, a.OperationID, b.OperationID}
	c.result("/transactions/"+original+"/links", map[string]any{"kind": "transfer", "reason": "Manual transfer and bank sides", "expectedRevisions": f.versions(ids...)})
	linked := c.transaction(original)
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
	c.result("/transactions/"+original+"/undo", map[string]any{"decisionId": *linked.DecisionId, "reason": "Restore unresolved evidence", "expectedRevisions": f.versions(ids...)})
	for _, r := range []ledger.Revision{a, b} {
		g, found, err := f.store.MatchingForOperation(testContext, f.p, r.OperationID)
		if err != nil || !found || g.State != matching.Clarification {
			t.Fatal(g, err)
		}
	}
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
}
func TestLinkedCorrectionsRequireCompleteVersionsAndPreserveMetadata(t *testing.T) {
	f := newFixture(t)
	from, to := f.create(money.RUB, "5000"), f.create(money.RUB, "0")
	c := f.client(f.p)
	a, b := f.revision(uuid.NewString(), from, "-1000", money.RUB, 1), f.revision(uuid.NewString(), to, "1000", money.RUB, 1)
	b.Type = ledger.Income
	for _, r := range []ledger.Revision{a, b} {
		if _, err := f.write(r, request()); err != nil {
			t.Fatal(err)
		}
	}
	f.link(matching.Transfer, a.OperationID, a.OperationID, b.OperationID)
	av, bv := c.transaction(a.OperationID), c.transaction(b.OperationID)
	ap, bp := av.Postings, bv.Postings
	ap[0].Money.Amount = "-1200"
	bp[0].Money.Amount = "1200"
	payload := map[string]any{"expectedRevision": av.Revision, "reason": "Correct principal", "principal": ap}
	failed := decode[generated.CommandStatus](t, c.call("POST", "/transactions/"+a.OperationID+"/corrections", uuid.NewString(), payload, 202))
	failure, err := failed.AsCommandFailed()
	if err != nil || failure.Status != "failed" {
		t.Fatal("unilateral monetary correction allowed", failed)
	}
	f.balance(from, "owned", "4000")
	payload["relatedChanges"] = []map[string]any{{"transactionId": b.OperationID, "expectedRevision": bv.Revision, "principal": bp}}
	c.result("/transactions/"+a.OperationID+"/corrections", payload)
	f.balance(from, "owned", "3800")
	f.balance(to, "owned", "1200")
	corrected := c.transaction(a.OperationID)
	decision := *corrected.DecisionId
	c.result("/transactions/"+b.OperationID+"/corrections", map[string]any{"expectedRevision": f.current(b.OperationID).Revision, "reason": "Independent note", "note": "Partner's note"})
	c.result("/transactions/"+a.OperationID+"/undo", map[string]any{"decisionId": decision, "reason": "Undo amount only", "expectedRevisions": f.versions(a.OperationID, b.OperationID)})
	f.balance(from, "owned", "4000")
	f.balance(to, "owned", "1000")
	if v := c.transaction(b.OperationID); v.Note == nil || *v.Note != "Partner's note" {
		t.Fatal("independent change overwritten")
	}
}
