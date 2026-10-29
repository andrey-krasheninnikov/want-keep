//go:build integration

package receipts_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	attachments "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/attachments/files"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/receipts"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	"github.com/pchkauu/want-keep/backend/internal/gateways/openai"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
	receiptapp "github.com/pchkauu/want-keep/backend/internal/receipts/application"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var ctx = context.Background()
var cluster *pgxpool.Pool
var databaseURL *url.URL

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Receipts integration requires isolated loopback PostgreSQL want_keep_test.")
		os.Exit(2)
	}
	databaseURL = u
	cluster, err = pgxpool.New(ctx, raw)
	if err != nil || cluster.Ping(ctx) != nil {
		fmt.Fprintln(os.Stderr, "Receipt test PostgreSQL unavailable.")
		os.Exit(2)
	}
	_, err = cluster.Exec(ctx, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_app') THEN CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; END IF; IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_maintenance') THEN CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance'; END IF; END $$;`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot prepare test roles.")
		os.Exit(2)
	}
	code := m.Run()
	cluster.Close()
	os.Exit(code)
}

type fixture struct {
	t             *testing.T
	store         *storage.Store
	admin         *pgxpool.Pool
	service       *receiptapp.Service
	ledger        *journal.Service
	matcher       *matching.Service
	documents     *attachments.Service
	blobs         *files.Store
	handler       http.Handler
	identity      *identityapp.Service
	a, b, foreign household.Principal
	ta, tb, tf    identity.Token
	account       string
	calls         atomic.Int32
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_receipts_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(ctx, "CREATE DATABASE "+name); err != nil {
		t.Fatal(err)
	}
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		admin.Close()
		_, err := cluster.Exec(ctx, "DROP DATABASE "+name+" WITH (FORCE)")
		if err != nil {
			t.Errorf("test database cleanup: %v", err)
		}
	})
	if err = storage.Migrate(ctx, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(ctx, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	f := &fixture{t: t, store: store, admin: admin}
	f.a, f.b = f.family()
	f.foreign, _ = f.family()
	verifier, err := webauthn.New("localhost", "http://localhost")
	if err != nil {
		t.Fatal(err)
	}
	f.identity, err = identityapp.NewService(store, store, verifier, "localhost", "http://localhost", 2, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	f.ta = f.session(f.a)
	f.tb = f.session(f.b)
	f.tf = f.session(f.foreign)
	now := func() calendar.Instant {
		at, _ := calendar.ParseInstant(time.Now().UTC().Format(time.RFC3339Nano))
		return at
	}
	matcher := matching.NewService(store, journal.NewWriter(store, store), now, uuid.NewString)
	f.matcher = matcher
	f.ledger = journal.NewServiceWithAllocations(store, matcher, allocation.NewService(store, now, uuid.NewString), now, uuid.NewString)
	f.service = receiptapp.NewService(store, f.ledger, expenses.NewService(store, matcher, now, uuid.NewString), matcher, time.Now, uuid.NewString)
	executor := commands.NewExecutor(store, store, now)
	f.handler, err = delivery.New(f.service, executor, commands.NewQueries(store, store.AuthorizeCommandResult), f.identity, store, security.Config{Environment: "test", Origin: "http://localhost"}, now)
	if err != nil {
		t.Fatal(err)
	}
	directory := t.TempDir()
	if err = os.Chmod(directory, 0700); err != nil {
		t.Fatal(err)
	}
	key := filepath.Join(t.TempDir(), "attachment-key")
	if err = cryptobox.Generate(key, "attachments"); err != nil {
		t.Fatal(err)
	}
	ring, err := cryptobox.Load(key, "attachments")
	if err != nil {
		t.Fatal(err)
	}
	blobs, err := files.Open(directory, ring)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = blobs.Close() })
	f.blobs = blobs
	f.documents = attachments.NewService(store, blobs, syntheticProcessor{})
	owner, _ := household.NewOwnership(f.a.HouseholdID(), household.Shared, "")
	date, _ := calendar.ParseDate("2026-01-01")
	balance, _ := money.NewMoney("5000", money.RUB)
	err = store.WithinHousehold(ctx, f.a, func(ctx context.Context) error {
		r, e := accounts.NewService(store, store, now, uuid.NewString).Create(ctx, f.a, accounts.CreateInput{Name: "Synthetic cash", Ownership: owner, Asset: money.RUB, Date: date, Balance: balance})
		f.account = r.ResourceID
		return e
	})
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *fixture) family() (household.Principal, household.Principal) {
	f.t.Helper()
	h := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic family"}
	users := []household.User{{ID: household.UserID(uuid.NewString()), Name: "Member A"}, {ID: household.UserID(uuid.NewString()), Name: "Member B"}}
	members := []household.Membership{{ID: household.MembershipID(uuid.NewString()), HouseholdID: h.ID, UserID: users[0].ID, Active: true}, {ID: household.MembershipID(uuid.NewString()), HouseholdID: h.ID, UserID: users[1].ID, Active: true}}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err := f.store.InitializeHousehold(ctx, h, users, members, zone, 2); err != nil {
		f.t.Fatal(err)
	}
	a, _ := members[0].Principal()
	b, _ := members[1].Principal()
	return a, b
}
func (f *fixture) session(p household.Principal) identity.Token {
	f.t.Helper()
	member, err := f.store.Membership(ctx, p.HouseholdID(), p.UserID())
	if err != nil {
		f.t.Fatal(err)
	}
	handle := make([]byte, 32)
	if _, err = rand.Read(handle); err != nil {
		f.t.Fatal(err)
	}
	token, err := identity.NewToken(32)
	if err != nil {
		f.t.Fatal(err)
	}
	now := time.Now().UTC()
	profile := identity.Profile{UserID: p.UserID(), HouseholdID: p.HouseholdID(), MembershipID: member.ID, Name: "Synthetic member", Locale: "ru", ReportingAsset: "RUB", Generation: 1, Handle: handle}
	c := identity.Credential{ID: uuid.NewString(), UserID: p.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{1}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{"internal"}, CreatedAt: now, UserVerified: true}
	session := identity.Session{ID: uuid.NewString(), UserID: p.UserID(), TokenHash: token.Hash(), CredentialID: c.ID, Name: "Synthetic session", CreatedAt: now, AuthenticatedAt: now, LastActivityAt: now}
	if err = f.store.WithinIdentity(ctx, func(ctx context.Context) error {
		if e := f.store.CreateIdentityProfile(ctx, profile); e != nil {
			return e
		}
		if e := f.store.SaveIdentityCredential(ctx, c); e != nil {
			return e
		}
		return f.store.SaveIdentitySession(ctx, session)
	}); err != nil {
		f.t.Fatal(err)
	}
	return token
}

type syntheticProcessor struct{}

func (syntheticProcessor) Ready(context.Context) bool { return true }
func (syntheticProcessor) Inspect(_ context.Context, media string, data []byte) (attachments.Inspection, error) {
	config, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || media != "image/png" {
		return attachments.Inspection{Reason: attachment.InvalidDocument}, nil
	}
	return attachments.Inspection{Pages: []attachments.Preview{{Width: config.Width, Height: config.Height, PNG: data}}}, nil
}
func (f *fixture) upload(p household.Principal, seed uint8) string {
	f.t.Helper()
	image := image.NewNRGBA(image.Rect(0, 0, 8, 8))
	image.SetNRGBA(0, 0, color.NRGBA{R: seed, A: 255})
	var output bytes.Buffer
	if err := png.Encode(&output, image); err != nil {
		f.t.Fatal(err)
	}
	data := output.Bytes()
	id := uuid.NewString()
	if _, err := f.documents.Upload(ctx, p, attachment.Upload{ID: id, AccountID: f.account, Name: "synthetic.png", MediaType: "image/png", Size: int64(len(data)), Hash: attachment.Hash(data)}, data); err != nil {
		f.t.Fatal(err)
	}
	if err := f.documents.ProcessNext(ctx); err != nil {
		f.t.Fatal(err)
	}
	return id
}
func (f *fixture) request(token identity.Token, method, path, key string, input any) *httptest.ResponseRecorder {
	f.t.Helper()
	data, err := json.Marshal(input)
	if err != nil {
		f.t.Fatal(err)
	}
	r := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(data))
	r.Header.Set("Origin", "http://localhost")
	r.Header.Set("Content-Type", "application/json")
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	if token != "" {
		r.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(token)})
		r.Header.Set("X-CSRF-Token", token.CSRF())
	}
	w := httptest.NewRecorder()
	f.handler.ServeHTTP(w, r)
	return w
}
func (f *fixture) start(id string, token identity.Token) string {
	f.t.Helper()
	w := f.request(token, "POST", "/receipts", uuid.NewString(), map[string]any{"attachmentId": id, "accountId": f.account})
	if w.Code != 202 {
		f.t.Fatalf("start %d %s", w.Code, w.Body.String())
	}
	var body struct {
		Result struct {
			ID string `json:"id"`
		} `json:"result"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil || body.Result.ID == "" {
		f.t.Fatalf("start response: %s %v", w.Body.String(), err)
	}
	return body.Result.ID
}
func (f *fixture) provider(output string, status int) *openai.Client {
	f.t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		var input map[string]any
		if json.NewDecoder(r.Body).Decode(&input) != nil {
			f.t.Error("invalid request")
		}
		messages, ok := input["input"].([]any)
		if !ok || len(messages) != 1 {
			f.t.Error("missing isolated image message")
		} else {
			contents := messages[0].(map[string]any)["content"].([]any)
			image := contents[1].(map[string]any)
			if image["type"] != "input_image" || image["detail"] != "high" || !strings.HasPrefix(image["image_url"].(string), "data:image/png;base64,") {
				f.t.Error("unsafe image delivery")
			}
		}
		if input["model"] != "gpt-5.6-terra" || input["reasoning"].(map[string]any)["effort"] != "xhigh" || input["tools"] != nil {
			f.t.Error("unsafe provider capability")
		}
		if r.URL.Path == "/responses/input_tokens" {
			_, _ = w.Write([]byte(`{"input_tokens":100}`))
			return
		}
		f.calls.Add(1)
		if status != 200 {
			w.WriteHeader(status)
			_, _ = w.Write([]byte(`{"error":{"type":"server_error","message":"synthetic"}}`))
			return
		}
		if input["store"] != false || input["background"] != false || input["max_output_tokens"] != float64(4096) {
			f.t.Error("unsafe generation settings")
		}
		body := map[string]any{"id": "response-synthetic", "model": "gpt-5.6-terra", "status": "completed", "output": []any{map[string]any{"type": "message", "id": "msg", "role": "assistant", "status": "completed", "content": []any{map[string]any{"type": "output_text", "text": output}}}}, "usage": map[string]any{"input_tokens": 100, "input_tokens_details": map[string]any{"cached_tokens": 0, "cache_write_tokens": 0}, "output_tokens": 100, "output_tokens_details": map[string]any{"reasoning_tokens": 0}, "total_tokens": 200}}
		_ = json.NewEncoder(w).Encode(body)
	}))
	f.t.Cleanup(server.Close)
	key := filepath.Join(f.t.TempDir(), "provider-key")
	if err := os.WriteFile(key, []byte("synthetic-not-a-real-key"), 0600); err != nil {
		f.t.Fatal(err)
	}
	client, err := openai.New(openai.Config{APIKeyFile: key, Environment: "test", BaseURL: server.URL, HTTPClient: server.Client()})
	if err != nil {
		f.t.Fatal(err)
	}
	receiptClient, err := client.ReceiptPages()
	if err != nil {
		f.t.Fatal(err)
	}
	return receiptClient
}
func (f *fixture) process(client *openai.Client) {
	f.t.Helper()
	handler := aiapp.NewPageHandler(f.store, client, receiptapp.Pages{Repository: f.store, Documents: f.documents}, time.Now, uuid.NewString)
	for _, pair := range []struct {
		kind    jobs.Kind
		handler jobapp.Handler
	}{{jobs.ReceiptExtract, handler}, {jobs.ReceiptValidation, f.service}} {
		worker := jobapp.Worker{Repository: f.store, Handler: pair.handler, Config: jobapp.DefaultWorkerConfig(pair.kind)}
		if err := worker.Step(ctx); err != nil {
			f.t.Fatal(err)
		}
	}
}
func (f *fixture) read(id string) receipt.Receipt {
	f.t.Helper()
	var value receipt.Receipt
	err := f.store.WithinFinancialRead(ctx, f.a, func(ctx context.Context) error { var e error; value, e = f.service.Read(ctx, f.a, id); return e })
	if err != nil {
		f.t.Fatal(err)
	}
	return value
}

func (f *fixture) createAccount(asset string) string {
	f.t.Helper()
	owner, _ := household.NewOwnership(f.a.HouseholdID(), household.Shared, "")
	date, _ := calendar.ParseDate("2026-01-01")
	balance, err := money.NewMoney("5000", money.Asset(asset))
	if err != nil {
		f.t.Fatal(err)
	}
	now := func() calendar.Instant {
		v, _ := calendar.ParseInstant(time.Now().UTC().Format(time.RFC3339Nano))
		return v
	}
	var id string
	err = f.store.WithinHousehold(ctx, f.a, func(ctx context.Context) error {
		r, e := accounts.NewService(f.store, f.store, now, uuid.NewString).Create(ctx, f.a, accounts.CreateInput{Name: "Synthetic " + asset, Ownership: owner, Asset: money.Asset(asset), Date: date, Balance: balance})
		id = r.ResourceID
		return e
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}
