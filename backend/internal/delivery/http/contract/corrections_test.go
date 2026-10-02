package contract_test

import (
	"encoding/json"
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func TestCorrectionFailuresRemainRecoverableAfterExpiration(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const id = "10000000-0000-4000-8000-000000000001"
	p, _ := (household.Membership{ID: "membership", UserID: "actor", HouseholdID: "family", Active: true}).Principal()
	at, _ := calendar.ParseInstant("2026-09-07T00:00:00Z")
	for _, code := range []string{"decision_conflict", "no_change"} {
		c, err := command.NewCommand(id, "transactions.corrections", strings.Repeat("a", 64), p, at)
		if err != nil {
			t.Fatal(err)
		}
		c, err = c.Fail(code, 0, at)
		if err != nil {
			t.Fatal(err)
		}
		live, err := b.CommandToDTO(c, id)
		if err != nil {
			t.Fatal(err)
		}
		expired, err := b.ExpiredCommandToDTO(id, &command.Outcome{CommandID: id, Status: command.Failed, FailureCode: code})
		if err != nil {
			t.Fatal(err)
		}
		for schema, dto := range map[string]any{"CommandStatus": live, "ExpiredCommand": expired} {
			data, err := json.Marshal(dto)
			if err != nil {
				t.Fatal(err)
			}
			var decoded json.RawMessage
			if err := b.Decode(schema, data, &decoded); err != nil {
				t.Fatalf("%s %s: %v", code, schema, err)
			}
		}
	}
}

func TestVersionConflictCarriesAuthorizedCurrentRevision(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const id = "10000000-0000-4000-8000-000000000001"
	p, _ := (household.Membership{ID: "membership", UserID: "actor", HouseholdID: "family", Active: true}).Principal()
	at, _ := calendar.ParseInstant("2026-09-07T00:00:00Z")
	c, err := command.NewCommand(id, "reconciliations.resolve", strings.Repeat("a", 64), p, at)
	if err != nil {
		t.Fatal(err)
	}
	c, err = c.Fail("version_conflict", 7, at)
	if err != nil {
		t.Fatal(err)
	}
	live, err := b.CommandToDTO(c, id)
	if err != nil {
		t.Fatal(err)
	}
	failed, err := live.AsCommandFailed()
	if err != nil || failed.Error.CurrentRevision == nil || int64(*failed.Error.CurrentRevision) != 7 {
		t.Fatalf("live conflict lost current revision: %#v %v", failed, err)
	}
	expired, err := b.ExpiredCommandToDTO(id, &command.Outcome{CommandID: id, Status: command.Failed, FailureCode: "version_conflict", CurrentRevision: 7})
	if err != nil || expired.CurrentRevision == nil || int64(*expired.CurrentRevision) != 7 {
		t.Fatalf("expired conflict lost current revision: %#v %v", expired, err)
	}
}

func TestReimbursementContractsAreTypedAndLinkCommandsStayClosed(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const id = "10000000-0000-4000-8000-000000000001"
	reimbursement := map[string]any{"id": id, "revision": 1, "decisionId": id, "state": "open", "creditorMemberId": id, "debtorMemberId": id, "principal": map[string]any{"amount": "100", "asset": "RUB"}, "outstanding": map[string]any{"amount": "100", "asset": "RUB"}, "reason": "Explicit debt", "actorId": id, "recordedAt": "2026-09-14T00:00:00Z", "settlements": []any{}}
	data, _ := json.Marshal(reimbursement)
	var out json.RawMessage
	if err := b.Decode("Reimbursement", data, &out); err != nil {
		t.Fatal(err)
	}
	for schema, value := range map[string]any{
		"ReimbursementCorrection": map[string]any{"expectedRevision": 1, "amount": map[string]any{"amount": "200", "asset": "RUB"}, "reason": "Correct amount"},
		"ReimbursementUndo":       map[string]any{"expectedRevision": 2, "decisionId": id, "reason": "Undo correction"},
		"SettlementCreate":        map[string]any{"expectedRevision": 2, "transferId": id, "transferExpectedRevision": 1, "transferAmount": map[string]any{"amount": "100", "asset": "RUB"}, "settledAmount": map[string]any{"amount": "100", "asset": "RUB"}},
	} {
		data, _ := json.Marshal(value)
		if err := b.Decode(schema, data, &out); err != nil {
			t.Fatalf("%s rejected: %v", schema, err)
		}
	}
	for field, value := range map[string]any{"accountingState": "excluded", "decisionId": id, "protectedFields": []any{}, "sourceConflict": false} {
		link := map[string]any{"transactionId": id, "expectedRevision": 1, field: value}
		data, _ := json.Marshal(link)
		if b.Decode("ExistingTransaction", data, &out) == nil {
			t.Fatalf("server field %s accepted in link command", field)
		}
	}
}
