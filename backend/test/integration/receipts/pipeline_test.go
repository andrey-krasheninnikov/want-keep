//go:build integration

package receipts_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

const extracted = `{"document":"receipt","kind":"expense","paid":true,"date":"2026-08-31","time":"14:30:00","asset":"RUB","total":"900","discount":"100","merchant":"Synthetic shop","reference":null,"items":[{"name":"Food","quantity":"1","gross":"600","discount":null,"category":null},{"name":"Home","quantity":"1","gross":"400","discount":null,"category":null}],"reason":"Visible payment receipt."}`

func TestOnePaymentThroughHTTPStorageAndProvider(t *testing.T) {
	f := newFixture(t)
	a := f.upload(f.a, 1)
	id := f.start(a, f.ta)
	client := f.provider(extracted, 200)
	f.process(client)
	r := f.read(id)
	if r.State != receipt.Recorded || r.TransactionID == "" || r.Facts == nil {
		t.Fatalf("not recorded: %+v", r)
	}
	w := f.request(f.tb, "GET", "/receipts/"+id, "", nil)
	if w.Code != 200 {
		t.Fatalf("family read: %d %s", w.Code, w.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["state"] != "recorded" || len(response["items"].([]any)) != 2 {
		t.Fatal("receipt response lost items")
	}
	duplicate := f.upload(f.b, 1)
	if second := f.start(duplicate, f.tb); second != id {
		t.Fatal("same receipt created another payment")
	}
	if f.calls.Load() != 1 {
		t.Fatal("duplicate provider call")
	}
	var count int
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.operation_revisions WHERE household_id=$1 AND economic_type='expense'`, f.a.HouseholdID()).Scan(&count); err != nil || count != 1 {
		t.Fatalf("effects=%d %v", count, err)
	}
	foreign := f.request(f.tf, "GET", "/receipts/"+id, "", nil)
	missing := f.request(f.tf, "GET", "/receipts/"+uuid.NewString(), "", nil)
	if foreign.Code != 404 || missing.Code != 404 {
		t.Fatalf("foreign existence leak: %d %d", foreign.Code, missing.Code)
	}
}
func TestIrrelevantAndPartialNeverPost(t *testing.T) {
	for _, test := range []struct {
		name, output string
		state        receipt.State
	}{{"irrelevant", `{"document":"irrelevant","kind":"unknown","paid":null,"date":null,"time":null,"asset":null,"total":null,"discount":null,"merchant":null,"reference":null,"items":[],"reason":"Menu is not a receipt."}`, receipt.Skipped}, {"partial", `{"document":"receipt","kind":"expense","paid":true,"date":"2026-08-31","time":null,"asset":"RUB","total":null,"discount":null,"merchant":null,"reference":null,"items":[],"reason":"Total is unreadable."}`, receipt.Clarification}} {
		t.Run(test.name, func(t *testing.T) {
			f := newFixture(t)
			id := f.start(f.upload(f.a, 1), f.ta)
			f.process(f.provider(test.output, 200))
			r := f.read(id)
			if r.State != test.state || r.TransactionID != "" {
				t.Fatalf("unexpected effect: %+v", r)
			}
		})
	}
}
func TestUnknownProviderOutcomeIsNotRetried(t *testing.T) {
	f := newFixture(t)
	id := f.start(f.upload(f.a, 2), f.ta)
	f.process(f.provider(extracted, 500))
	r := f.read(id)
	if r.State != receipt.WaitingAI || r.TransactionID != "" {
		t.Fatalf("unknown outcome lost: %+v", r)
	}
	if f.calls.Load() != 1 {
		t.Fatal("provider retry")
	}
}

func TestReceiptStartTransaction(t *testing.T) {
	f := newFixture(t)
	attachmentID := f.upload(f.a, 7)
	err := f.store.WithinHousehold(ctx, f.a, func(tx context.Context) error {
		_, err := f.service.Start(tx, f.a, attachmentID, f.account)
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
}

func TestReceiptReplayAndConcurrentPartnerUploads(t *testing.T) {
	f := newFixture(t)
	a := f.upload(f.a, 3)
	b := f.upload(f.b, 3)
	start := make(chan struct{})
	ids := make(chan string, 2)
	errors := make(chan string, 2)
	for _, input := range []struct {
		attachment string
		token      identity.Token
	}{{a, f.ta}, {b, f.tb}} {
		go func() {
			<-start
			w := f.request(input.token, "POST", "/receipts", uuid.NewString(), map[string]any{"attachmentId": input.attachment, "accountId": f.account})
			if w.Code != 202 {
				errors <- w.Body.String()
				return
			}
			var value struct {
				Result struct {
					ID string `json:"id"`
				} `json:"result"`
			}
			_ = json.Unmarshal(w.Body.Bytes(), &value)
			ids <- value.Result.ID
		}()
	}
	close(start)
	var first string
	for range 2 {
		select {
		case failure := <-errors:
			t.Fatal(failure)
		case id := <-ids:
			if first != "" && first != id {
				t.Fatal("two receipts for one file")
			}
			first = id
		}
	}
	f.process(f.provider(extracted, 200))
	if f.calls.Load() != 1 {
		t.Fatal("duplicate extraction")
	}
	key := uuid.NewString()
	input := map[string]any{"attachmentId": a, "accountId": f.account}
	one := f.request(f.ta, "POST", "/receipts", key, input)
	two := f.request(f.ta, "POST", "/receipts", key, input)
	if one.Code != 202 || two.Code != 202 || one.Body.String() != two.Body.String() {
		t.Fatal("result replay changed")
	}
	input["accountId"] = uuid.NewString()
	changed := f.request(f.ta, "POST", "/receipts", key, input)
	if changed.Code != 409 {
		t.Fatalf("changed payload: %d", changed.Code)
	}
}
func TestMissingAccountAndOriginCannotStartProcessing(t *testing.T) {
	f := newFixture(t)
	attachmentID := f.upload(f.a, 4)
	key := uuid.NewString()
	missing := f.request(f.ta, "POST", "/receipts", key, map[string]any{"attachmentId": attachmentID})
	if missing.Code != 400 {
		t.Fatalf("missing selected account: %d", missing.Code)
	}
	data, _ := json.Marshal(map[string]any{"attachmentId": attachmentID, "accountId": f.account})
	request := httptest.NewRequest("POST", "http://localhost/api/v1/receipts", bytes.NewReader(data))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "https://untrusted.example")
	request.Header.Set("Idempotency-Key", uuid.NewString())
	request.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(f.ta)})
	request.Header.Set("X-CSRF-Token", f.ta.CSRF())
	response := httptest.NewRecorder()
	f.handler.ServeHTTP(response, request)
	if response.Code != 401 {
		t.Fatalf("cross origin: %d", response.Code)
	}
}
func TestClarificationAnswerAndVersionConflict(t *testing.T) {
	f := newFixture(t)
	id := f.start(f.upload(f.a, 5), f.ta)
	output := strings.Replace(extracted, `"total":"900"`, `"total":null`, 1)
	f.process(f.provider(output, 200))
	r := f.read(id)
	if r.State != receipt.Clarification {
		t.Fatal(r.State)
	}
	facts := map[string]any{}
	_ = json.Unmarshal([]byte(extracted), &facts)
	input := map[string]any{"expectedRevision": r.Revision, "facts": facts}
	ambiguous := f.request(f.ta, "POST", "/receipts/"+id+"/answers", uuid.NewString(), map[string]any{"expectedRevision": r.Revision, "distinct": true, "purchase": map[string]any{"id": uuid.NewString(), "revision": 1}})
	if ambiguous.Code != 400 {
		t.Fatal("ambiguous answer variants accepted")
	}
	w := f.request(f.tb, "POST", "/receipts/"+id+"/answers", uuid.NewString(), input)
	if w.Code != 400 {
		t.Fatalf("provider-only category fields must not cross user schema: %d %s", w.Code, w.Body.String())
	}
	for _, item := range facts["items"].([]any) {
		delete(item.(map[string]any), "category")
	}
	// Omitted optional user fields stay absent rather than JSON null.
	for key, value := range facts {
		if value == nil {
			delete(facts, key)
		}
	}
	for _, item := range facts["items"].([]any) {
		if item.(map[string]any)["discount"] == nil {
			delete(item.(map[string]any), "discount")
		}
	}
	w = f.request(f.tb, "POST", "/receipts/"+id+"/answers", uuid.NewString(), input)
	if w.Code != 202 {
		t.Fatalf("answer: %d %s", w.Code, w.Body.String())
	}
	r = f.read(id)
	if r.State != receipt.Recorded {
		t.Fatalf("answer not applied: %+v", r)
	}
	stale := f.request(f.ta, "POST", "/receipts/"+id+"/answers", uuid.NewString(), input)
	if stale.Code != 409 || !strings.Contains(stale.Body.String(), "version_conflict") {
		t.Fatalf("stale answer: %d %s", stale.Code, stale.Body.String())
	}
}

func TestExactAssetsThroughReceiptHTTP(t *testing.T) {
	for _, tc := range []struct{ asset, total string }{{"RUB", "0.01"}, {"USD", "0.01"}, {"USDT", "0.000001"}, {"USDC", "0.000001"}, {"BTC", "0.000000001"}, {"ETH", "0.000000000000000001"}} {
		t.Run(tc.asset, func(t *testing.T) {
			f := newFixture(t)
			if tc.asset != "RUB" {
				f.account = f.createAccount(tc.asset)
			}
			output := map[string]any{}
			_ = json.Unmarshal([]byte(extracted), &output)
			output["asset"] = tc.asset
			output["total"] = tc.total
			output["discount"] = nil
			output["items"] = []any{}
			data, _ := json.Marshal(output)
			id := f.start(f.upload(f.a, 8), f.ta)
			f.process(f.provider(string(data), 200))
			r := f.read(id)
			if r.State != receipt.Recorded || r.Facts.Total.Amount() != tc.total {
				t.Fatal("native amount changed")
			}
			w := f.request(f.ta, "GET", "/receipts/"+id, "", nil)
			if w.Code != 200 || !strings.Contains(w.Body.String(), `"amount":"`+tc.total+`"`) {
				t.Fatal("HTTP lost decimal string")
			}
		})
	}
}
func TestMatchingRequiresConfirmationAndKeepsOneEffect(t *testing.T) {
	f := newFixture(t)
	at, _ := calendar.ParseInstant("2026-08-31T11:30:00Z")
	amount, _ := money.NewMoney("900", money.RUB)
	var existing command.Result
	err := f.store.WithinHousehold(ctx, f.a, func(ctx context.Context) error {
		var err error
		existing, err = f.ledger.Create(ctx, f.a, journal.CreateInput{Type: ledger.Expense, AccountID: f.account, At: at, Amount: amount, PayerState: "unknown", Allocation: ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: "unresolved"}})
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	id := f.start(f.upload(f.a, 9), f.ta)
	f.process(f.provider(extracted, 200))
	r := f.read(id)
	if r.State != receipt.Clarification || len(r.Candidates) != 1 || r.Candidates[0].ID != existing.ResourceID {
		t.Fatalf("unproven match: %+v", r)
	}
	w := f.request(f.tb, "POST", "/receipts/"+id+"/answers", uuid.NewString(), map[string]any{"expectedRevision": r.Revision, "match": map[string]any{"id": existing.ResourceID, "revision": existing.Revision}})
	if w.Code != 202 {
		t.Fatalf("confirm: %d %s", w.Code, w.Body.String())
	}
	r = f.read(id)
	if r.State != receipt.Linked || r.TransactionID != existing.ResourceID {
		t.Fatalf("not linked: %+v", r)
	}
	var count int
	if err = f.admin.QueryRow(ctx, `SELECT count(DISTINCT operation_id) FROM want_keep.operation_revisions WHERE household_id=$1 AND economic_type='expense'`, f.a.HouseholdID()).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate expense", count, err)
	}
}
func TestMixedItemsAndHistoricalRefund(t *testing.T) {
	f := newFixture(t)
	output := strings.Replace(extracted, `"total":"900","discount":"100"`, `"total":"1000","discount":"0"`, 1)
	output = strings.Replace(output, `{"name":"Home","quantity":"1","gross":"400","discount":null,"category":null}`, `{"name":"A","quantity":"1","gross":"100","discount":null,"category":null},{"name":"B","quantity":"1","gross":"300","discount":null,"category":null}`, 1)
	id := f.start(f.upload(f.a, 10), f.ta)
	f.process(f.provider(output, 200))
	r := f.read(id)
	if r.State != receipt.Recorded {
		t.Fatal(r.State)
	}
	a, _ := f.store.Membership(ctx, f.a.HouseholdID(), f.a.UserID())
	b, _ := f.store.Membership(ctx, f.b.HouseholdID(), f.b.UserID())
	shared := map[string]any{"mode": "equal", "purpose": "shared"}
	personal := func(member household.MembershipID, amount string) any {
		return map[string]any{"mode": "amounts", "purpose": "personal", "members": []any{map[string]any{"memberId": member, "amount": map[string]any{"amount": amount, "asset": "RUB"}}}}
	}
	input := map[string]any{"expectedRevision": r.Revision, "subjectExpectedRevision": r.TransactionRevision, "allocation": shared, "items": []any{map[string]any{"itemId": r.Facts.Items[0].ID, "allocation": shared}, map[string]any{"itemId": r.Facts.Items[1].ID, "allocation": personal(a.ID, "100")}, map[string]any{"itemId": r.Facts.Items[2].ID, "allocation": personal(b.ID, "300")}}, "reason": "Confirmed item purposes"}
	w := f.request(f.tb, "POST", "/receipts/"+id+"/allocations", uuid.NewString(), input)
	if w.Code != 202 || strings.Contains(w.Body.String(), `"status":"failed"`) {
		t.Fatalf("allocation: %d %s", w.Code, w.Body.String())
	}
	r = f.read(id)
	current, _, err := f.store.CurrentLedgerRevision(ctx, f.a, r.TransactionID)
	if err != nil {
		t.Fatal(err)
	}
	actual := map[household.MembershipID]string{}
	for _, member := range current.Allocation.Members {
		actual[member.MemberID] = member.Money.Amount()
	}
	if actual[a.ID] != "400" || actual[b.ID] != "600" {
		t.Fatalf("mixed allocation: %v", actual)
	}
	refund := `{"document":"receipt","kind":"refund","paid":true,"date":"2026-09-07","time":"12:00:00","asset":"RUB","total":"200","discount":null,"merchant":"Synthetic shop","reference":null,"items":[],"reason":"Confirmed returned payment."}`
	refundID := f.start(f.upload(f.a, 11), f.ta)
	f.process(f.provider(refund, 200))
	rr := f.read(refundID)
	if rr.State != receipt.Clarification {
		t.Fatal(rr.State)
	}
	w = f.request(f.ta, "POST", "/receipts/"+refundID+"/answers", uuid.NewString(), map[string]any{"expectedRevision": rr.Revision, "purchase": map[string]any{"id": r.TransactionID, "revision": current.Revision}, "refundItems": []any{map[string]any{"itemId": r.Facts.Items[0].ID, "amount": "200"}}})
	if w.Code != 202 {
		t.Fatalf("refund: %d %s", w.Code, w.Body.String())
	}
	rr = f.read(refundID)
	if rr.State != receipt.Recorded {
		t.Fatalf("refund not recorded: %+v %s", rr, w.Body.String())
	}
	facts, found, err := f.store.Refund(ctx, f.a, rr.TransactionID)
	if err != nil || !found {
		t.Fatal(err)
	}
	if facts.ExpenseMonth.String() != "2026-08" || facts.CashDate.String() != "2026-09-07" || len(facts.Members) != 2 {
		t.Fatalf("refund historical basis: %+v", facts)
	}
	for _, m := range facts.Members {
		expected, _ := money.NewMoney("100", money.RUB)
		if n, e := m.Amount.Compare(expected); e != nil || n != 0 {
			t.Fatal("historical member split changed")
		}
	}
}
func TestRollbackDoesNotLeaveReceiptOrJobs(t *testing.T) {
	f := newFixture(t)
	attachmentID := f.upload(f.a, 12)
	stop := commands.Rejection{Code: "synthetic_rollback"}
	err := f.store.WithinHousehold(ctx, f.a, func(ctx context.Context) error {
		if _, err := f.service.Start(ctx, f.a, attachmentID, f.account); err != nil {
			return err
		}
		return stop
	})
	if err == nil {
		t.Fatal("rollback not exercised")
	}
	var count int
	err = f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.receipts WHERE household_id=$1`, f.a.HouseholdID()).Scan(&count)
	if err != nil || count != 0 {
		t.Fatal("partial receipt after rollback", err)
	}
}

func TestProviderPayloadDoesNotPersistImagesOrSecrets(t *testing.T) {
	f := newFixture(t)
	id := f.start(f.upload(f.a, 15), f.ta)
	f.process(f.provider(extracted, 200))
	if f.read(id).State != receipt.Recorded {
		t.Fatal("receipt not recorded")
	}
	var input string
	if err := f.admin.QueryRow(ctx, `SELECT allowed_input::text FROM want_keep.ai_attempts WHERE household_id=$1 AND purpose='receipt_page'`, f.a.HouseholdID()).Scan(&input); err != nil {
		t.Fatal(err)
	}
	for _, secret := range []string{"image_url", "base64", string(f.ta), string(f.a.HouseholdID()), string(f.a.UserID()), "synthetic-not-a-real-key"} {
		if strings.Contains(input, secret) {
			t.Fatal("unrelated or binary input persisted")
		}
	}
}
func TestTerminalExtractionFailureCanBeClarified(t *testing.T) {
	f := newFixture(t)
	id := f.start(f.upload(f.a, 16), f.ta)
	if _, err := f.admin.Exec(ctx, `UPDATE want_keep.jobs SET state='failed',reason='permanent_failure' WHERE household_id=$1 AND kind='receipt_extract'`, f.a.HouseholdID()); err != nil {
		t.Fatal(err)
	}
	r := f.read(id)
	if r.State != receipt.Clarification {
		t.Fatal("terminal failure stuck processing")
	}
	var facts map[string]any
	_ = json.Unmarshal([]byte(extracted), &facts)
	for key, value := range facts {
		if value == nil {
			delete(facts, key)
		}
	}
	for _, item := range facts["items"].([]any) {
		m := item.(map[string]any)
		delete(m, "category")
		delete(m, "discount")
	}
	w := f.request(f.tb, "POST", "/receipts/"+id+"/answers", uuid.NewString(), map[string]any{"expectedRevision": r.Revision, "facts": facts})
	if w.Code != 202 || f.read(id).State != receipt.Recorded {
		t.Fatalf("failed extraction clarification: %d %s", w.Code, w.Body.String())
	}
}
