//go:build integration

package aicommands_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/review"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func TestHTTPClarificationRightsCSRFAndRecovery(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	question := "Is the purchase shared?"
	g := &gateway{output: output(ai.ReviewCommand{Kind: "clarify", Question: &question, Evidence: []string{"ledger_revision"}, Reason: "Purpose is unknown."})}
	f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	service := f.service()
	f.step(jobs.AIValidation, service)
	token, err := identity.NewToken(32)
	if err != nil {
		t.Fatal(err)
	}
	handle := make([]byte, 32)
	if _, err = rand.Read(handle); err != nil {
		t.Fatal(err)
	}
	credentialID := uuid.NewString()
	err = f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		profile := identity.Profile{UserID: f.p.UserID(), HouseholdID: f.p.HouseholdID(), MembershipID: f.membership.ID, Name: "Synthetic member", Locale: "en", ReportingAsset: "RUB", Handle: handle, Generation: 1}
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return err
		}
		credential := identity.Credential{ID: credentialID, UserID: f.p.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{}, UserVerified: true, CreatedAt: f.now.Time()}
		if err := f.store.SaveIdentityCredential(ctx, credential); err != nil {
			return err
		}
		return f.store.SaveIdentitySession(ctx, identity.Session{ID: uuid.NewString(), TokenHash: token.Hash(), CredentialID: credentialID, Name: "Synthetic browser", UserID: f.p.UserID(), CreatedAt: f.now.Time(), AuthenticatedAt: f.now.Time(), LastActivityAt: f.now.Time()})
	})
	if err != nil {
		t.Fatal(err)
	}
	verifier, err := webauthn.New("localhost", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	sessions, err := identityapp.NewService(f.store, f.store, verifier, "localhost", "http://localhost", 2, func() time.Time { return f.now.Time() })
	if err != nil {
		t.Fatal(err)
	}
	handler, err := delivery.New(service, f.executor, commands.NewQueries(f.store, f.store.AuthorizeCommandResult), sessions, f.store, security.Config{Environment: "test", Origin: "http://localhost"}, func() calendar.Instant { return f.now })
	if err != nil {
		t.Fatal(err)
	}
	call := func(method, path, key, body, csrf string, expected int) *httptest.ResponseRecorder {
		t.Helper()
		r := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
		r.Header.Set("Origin", "http://localhost")
		r.Header.Set("X-CSRF-Token", csrf)
		r.Header.Set("Idempotency-Key", key)
		r.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(token)})
		result := httptest.NewRecorder()
		handler.ServeHTTP(result, r)
		if result.Code != expected {
			t.Fatalf("%s %s: %d %s", method, path, result.Code, result.Body.String())
		}
		if result.Header().Get("Cache-Control") != "no-store" {
			t.Fatal("cacheable review response")
		}
		return result
	}
	list := call("GET", "/clarifications", "", "", "", 200)
	var page struct {
		Items []ai.Clarification `json:"items"`
	}
	if err = json.Unmarshal(list.Body.Bytes(), &page); err != nil || len(page.Items) != 1 {
		t.Fatalf("question list: %v", err)
	}
	value := page.Items[0]
	call("GET", "/proposals/"+uuid.NewString(), "", "", "", 404)
	call("GET", "/clarifications?limit=101", "", "", "", 400)
	call("GET", "/clarifications?cursor=invalid", "", "", "", 400)
	key := uuid.NewString()
	body := `{"expectedRevision":1,"subjectExpectedRevision":1,"choiceId":"keep"}`
	path := "/clarifications/" + value.ID + "/answers"
	call("POST", path, key, body, "", 401)
	call("POST", path, key, `{"expectedRevision":1,"subjectExpectedRevision":1,"choiceId":"keep","actor":"partner"}`, token.CSRF(), 400)
	first := call("POST", path, key, body, token.CSRF(), 202)
	repeated := call("POST", path, key, body, token.CSRF(), 202)
	var a, b struct {
		Id     string `json:"id"`
		Status string `json:"status"`
	}
	if json.Unmarshal(first.Body.Bytes(), &a) != nil || json.Unmarshal(repeated.Body.Bytes(), &b) != nil || a.Id != b.Id || a.Status != "succeeded" {
		t.Fatalf("command replay: %s / %s", first.Body.String(), repeated.Body.String())
	}
	call("GET", "/proposals/"+value.ProposalID, "", "", "", 200)
	remaining := call("GET", "/clarifications", "", "", "", 200)
	if err = json.Unmarshal(remaining.Body.Bytes(), &page); err != nil || len(page.Items) != 0 {
		t.Fatal("answered clarification remained actionable")
	}
	err = f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		status, _, err := f.store.TransactionReviewStatus(ctx, f.p, value.OperationID, value.OperationRevision)
		if err == nil && status != "reviewed" {
			t.Fatalf("transaction status: %s", status)
		}
		return err
	})
	if err != nil {
		t.Fatal(err)
	}
	var history int
	if err = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.review_state_history`).Scan(&history); err != nil || history != 4 {
		t.Fatalf("immutable transitions: %d %v", history, err)
	}
}
