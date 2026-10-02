//go:build integration

package familyreimbursements_test

import (
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

func TestReimbursementHistoryIsImmutableAndLeastPrivilege(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	key := uuid.NewString()
	client.call(http.MethodPost, "/reimbursements", key, map[string]any{"creditorMemberId": f.members[0].ID, "debtorMemberId": f.members[1].ID, "amount": map[string]any{"amount": "10", "asset": "RUB"}, "reason": "Immutable debt"}, http.StatusAccepted)
	id := client.result(key).ResourceID
	if _, err := f.admin.Exec(testContext, `UPDATE want_keep.reimbursement_revisions SET reason='rewritten' WHERE household_id=$1 AND reimbursement_id=$2`, f.family.ID, id); err == nil {
		t.Fatal("immutable reimbursement history was updated")
	}
	for table := range map[string]bool{"reimbursements": false, "reimbursement_revisions": true, "reimbursement_decisions": true, "reimbursement_settlements": true, "reimbursement_audit": true} {
		var insert, update, remove bool
		if err := f.admin.QueryRow(testContext, `SELECT has_table_privilege('want_keep_app',$1,'INSERT'),has_table_privilege('want_keep_app',$1,'UPDATE'),has_table_privilege('want_keep_app',$1,'DELETE')`, "want_keep."+table).Scan(&insert, &update, &remove); err != nil {
			t.Fatal(err)
		}
		if !insert || remove || table != "reimbursements" && update {
			t.Fatalf("unsafe grants for %s: insert=%v update=%v delete=%v", table, insert, update, remove)
		}
	}
	history := decode[generated.ReimbursementPage](t, client.call(http.MethodGet, "/reimbursements/"+id+"/history?limit=1", "", nil, http.StatusOK))
	if len(history.Items) != 1 || history.Items[0].DecisionId == "" || history.Items[0].ActorId == "" || history.Items[0].RecordedAt == "" {
		t.Fatalf("history lost revision provenance: %#v", history)
	}
}
