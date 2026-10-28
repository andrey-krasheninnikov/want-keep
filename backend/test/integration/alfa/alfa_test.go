//go:build integration

package alfa_test

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/connections/access"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connectionapp "github.com/pchkauu/want-keep/backend/internal/connections/application"
	"github.com/pchkauu/want-keep/backend/internal/connections/collector"
	"github.com/pchkauu/want-keep/backend/internal/connections/credentials"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	delivery "github.com/pchkauu/want-keep/backend/internal/delivery/connections"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identitydomain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	integration "github.com/pchkauu/want-keep/backend/internal/integrations/application"

	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobdomain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var cluster *pgxpool.Pool
var databaseURL *url.URL
var ctx = context.Background()

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Alfa integration requires isolated loopback want_keep_test PostgreSQL.")
		os.Exit(2)
	}
	databaseURL = u
	cluster, err = pgxpool.New(ctx, raw)
	if err != nil || cluster.Ping(ctx) != nil {
		fmt.Fprintln(os.Stderr, "Isolated PostgreSQL unavailable")
		os.Exit(2)
	}
	_, err = cluster.Exec(ctx, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_app') THEN CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; END IF; IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_maintenance') THEN CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance'; END IF; END $$;`)
	if err != nil {
		os.Exit(2)
	}
	code := m.Run()
	cluster.Close()
	os.Exit(code)
}

type fixture struct {
	t          *testing.T
	db         *storage.Store
	admin      *pgxpool.Pool
	p, q       household.Principal
	gate       *admission.Service
	binding    connections.Binding
	connection string
	vault      *credentials.Vault
	ingestion  *integration.Service
	sessions   *identity.Service
	api        *delivery.Server
	service    *connectionapp.Service
	tokens     []identitydomain.Token
	now        calendar.Instant
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_alfa_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := cluster.Exec(ctx, `DROP DATABASE `+name+` WITH (FORCE)`); err != nil {
			t.Error(err)
		}
	})
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(ctx, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err = storage.Migrate(ctx, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	db, err := storage.Open(ctx, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(db.Close)
	f := &fixture{t: t, db: db, admin: admin, connection: uuid.NewString()}
	now := time.Now().UTC().Truncate(time.Microsecond)
	f.now, _ = calendar.ParseInstant(now.Format(time.RFC3339Nano))
	nowInstant := func() calendar.Instant { return f.now }
	h := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic family"}
	users := []household.User{{ID: household.UserID(uuid.NewString()), Name: "Member A"}, {ID: household.UserID(uuid.NewString()), Name: "Member B"}}
	members := []household.Membership{}
	for _, u := range users {
		members = append(members, household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: h.ID, UserID: u.ID, Active: true})
	}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = db.InitializeHousehold(ctx, h, users, members, zone, 2); err != nil {
		t.Fatal(err)
	}
	f.p, _ = members[0].Principal()
	f.q, _ = members[1].Principal()
	verifier, err := webauthn.New("localhost", "http://localhost:3000")
	if err != nil {
		t.Fatal(err)
	}
	f.sessions, err = identity.NewService(db, db, verifier, "localhost", "http://localhost:3000", 2, time.Now)
	if err != nil {
		t.Fatal(err)
	}
	for i, user := range users {
		profile := identitydomain.Profile{UserID: user.ID, HouseholdID: h.ID, MembershipID: members[i].ID, Name: user.Name, Locale: "en", ReportingAsset: "RUB", Generation: 1, Handle: make([]byte, 32)}
		if _, err = rand.Read(profile.Handle); err != nil {
			t.Fatal(err)
		}
		token, err := identitydomain.NewToken(32)
		if err != nil {
			t.Fatal(err)
		}
		f.tokens = append(f.tokens, token)
		err = db.WithinIdentity(ctx, func(ctx context.Context) error {
			if err := db.CreateIdentityProfile(ctx, profile); err != nil {
				return err
			}
			credential := identitydomain.Credential{ID: uuid.NewString(), UserID: user.ID, Name: "Synthetic session fixture", RPID: "localhost", RawID: []byte(uuid.NewString()), PublicKey: []byte("synthetic-public-key"), AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, AttestationFormat: "none", Transports: []string{}, CreatedAt: now}
			if err := db.SaveIdentityCredential(ctx, credential); err != nil {
				return err
			}
			return db.SaveIdentitySession(ctx, identitydomain.Session{ID: uuid.NewString(), UserID: user.ID, TokenHash: token.Hash(), CredentialID: credential.ID, CreatedAt: now, AuthenticatedAt: now, LastActivityAt: now})
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	f.gate = admission.NewService(db, db)
	f.binding = connections.Binding{Provider: "alfa", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("0", 64), CollectorImageDigest: "sha256:" + strings.Repeat("1", 64), ContractVersion: "10", AllowlistRevision: "alfa-read-v1", NonSecretConfigRevision: "test-v1", OperatorPermissionRevision: "synthetic-v1"}
	keyFile := filepath.Join(t.TempDir(), "connections-keyring.json")
	if err = cryptobox.Generate(keyFile, "connections"); err != nil {
		t.Fatal(err)
	}
	keys, err := cryptobox.Load(keyFile, "connections")
	if err != nil {
		t.Fatal(err)
	}
	f.vault = credentials.New(access.NewService(f.sessions, db, f.gate), db, keys)
	evidence, err := collector.NewEvidenceStore(db, keys)
	if err != nil {
		t.Fatal(err)
	}
	accountImporter, err := integration.NewAccountImporter(accounts.NewService(db, db, nowInstant, uuid.NewString), db)
	if err != nil {
		t.Fatal(err)
	}
	sources, err := integration.NewSourceWriter(ledger.NewSources(db, ledger.NewWriter(db, db), allocation.NewService(db, nowInstant, uuid.NewString)), db)
	if err != nil {
		t.Fatal(err)
	}
	f.ingestion, err = integration.NewService(f.gate, evidence, accountImporter, sources, nowInstant, uuid.NewString)
	if err != nil {
		t.Fatal(err)
	}
	f.service = &connectionapp.Service{Repository: db, Sessions: f.sessions, Executor: application.NewExecutor(db, db, nowInstant), Admission: f.gate, Vault: f.vault, Bindings: map[string]connections.Binding{"alfa": f.binding}, NewID: uuid.NewString}
	f.api, err = delivery.New(f.service, f.sessions, application.NewQueries(db, db.AuthorizeCommandResult), security.Config{Environment: "test", Origin: "http://localhost:3000"}, nowInstant)
	if err != nil {
		t.Fatal(err)
	}
	return f
}
func (f *fixture) request(member int, method, path, body, key string) *httptest.ResponseRecorder {
	r := httptest.NewRequest(method, "http://localhost:3000"+path, strings.NewReader(body))
	r.AddCookie(&http.Cookie{Name: "want_keep_session", Value: string(f.tokens[member])})
	if method != "GET" {
		r.Header.Set("Origin", "http://localhost:3000")
		r.Header.Set("X-CSRF-Token", f.tokens[member].CSRF())
		r.Header.Set("Content-Type", "application/json")
	}
	if key != "" {
		r.Header.Set("Idempotency-Key", key)
	}
	w := httptest.NewRecorder()
	f.api.ServeHTTP(w, r)
	return w
}

func (f *fixture) create(owner int) {
	f.t.Helper()
	from := time.Now().Add(-48 * time.Hour).Format(time.DateOnly)
	body := fmt.Sprintf(`{"provider":"alfa","externalAccountOwnerId":%q,"historyFrom":%q,"products":["current","debit_card"]}`, []household.Principal{f.p, f.q}[owner].UserID(), from)
	key := uuid.NewString()
	for range 2 {
		if w := f.request(1-owner, "POST", "/api/v1/connections", body, key); w.Code != 202 {
			f.t.Fatal(w.Code, w.Body.String())
		}
	}
	if err := f.admin.QueryRow(ctx, `SELECT connection_id FROM want_keep.connection_events WHERE command_id=$1 AND event_type='created'`, key).Scan(&f.connection); err != nil {
		f.t.Fatal(err)
	}
}
func (f *fixture) enroll(owner int) {
	f.t.Helper()
	c, err := f.connectionRecord()
	if err != nil {
		f.t.Fatal(err)
	}
	auth := f.request(owner, "POST", "/api/v1/connections/"+f.connection+"/authorization", fmt.Sprintf(`{"expectedRevision":%d}`, c.Revision), "")
	if auth.Code != 200 {
		f.t.Fatal(auth.Code, auth.Body.String())
	}
	var grant struct{ ID, Method string }
	if json.Unmarshal(auth.Body.Bytes(), &grant) != nil || grant.Method != "browser_session" {
		f.t.Fatal("browser grant missing")
	}
	session := []byte(`{"cookies":[],"origins":[],"headers":{"x-xsrf-token":"synthetic-xsrf","x-security-request":"synthetic-security","device-app-id":"synthetic-device","ao-origin-application-id":"synthetic-app","zone-offset":"180"}}`)
	body := fmt.Sprintf(`{"grantId":%q,"expectedRevision":%d,"session":%q}`, grant.ID, c.Revision, base64.StdEncoding.EncodeToString(session))
	w := f.request(owner, "POST", "/api/v1/connections/"+f.connection+"/browser-session", body, "")
	if w.Code != 204 {
		f.t.Fatal(w.Code, w.Body.String())
	}
	w = f.request(owner, "POST", "/api/v1/connections/"+f.connection+"/browser-session", body, "")
	if w.Code != 409 {
		f.t.Fatal("grant replay accepted", w.Code)
	}
}

func (f *fixture) admit() {
	f.t.Helper()
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := f.gate.RecordCheck(ctx, connections.Check{Kind: kind, Binding: f.binding, Result: connections.CheckPassed, At: f.now}); err != nil {
			f.t.Fatal(err)
		}
	}
}
func (f *fixture) sync() {
	f.t.Helper()
	c, err := f.connectionRecord()
	if err != nil {
		f.t.Fatal(err)
	}
	w := f.request(0, "POST", "/api/v1/connections/"+f.connection+"/sync", fmt.Sprintf(`{"expectedRevision":%d}`, c.Revision), uuid.NewString())
	if w.Code != 202 {
		f.t.Fatal(w.Code, w.Body.String())
	}
}
func bankFixture(t *testing.T, name string) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../../collector/contracts/alfa/fixtures", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func (f *fixture) collector(afterRead func(), externalAccount, accountsFixture string) string {
	f.t.Helper()
	directory, err := os.MkdirTemp("/tmp", "wk-alfa-")
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(func() { _ = os.RemoveAll(directory) })
	socket := filepath.Join(directory, "collector.sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		f.t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /ready", func(w http.ResponseWriter, r *http.Request) { fmt.Fprint(w, "want-keep-browser-collector/1\n") })
	mux.HandleFunc("POST /v1/capabilities", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprint(w, `{"provider":"alfa","contractVersion":"10","actions":["read_accounts","read_transactions","read_history"],"products":["current"],"logs":[{"product":"current","namespace":"alfa-current-v1","recordKinds":["account","transaction"]}],"history":{"paginated":true}}`)
	})
	mux.HandleFunc("POST /v1/read", func(w http.ResponseWriter, r *http.Request) {
		var in struct {
			SyncRequest    map[string]any
			StorageState   json.RawMessage
			SessionHeaders map[string]string
		}
		if json.NewDecoder(r.Body).Decode(&in) != nil || len(in.SessionHeaders) != 5 || strings.Contains(string(in.StorageState), "synthetic-xsrf") {
			http.Error(w, "invalid_envelope", 400)
			return
		}
		b := map[string]any{"version": 1, "accounts": strings.ReplaceAll(bankFixture(f.t, accountsFixture), "synthetic-current-a", externalAccount), "cards": strings.ReplaceAll(bankFixture(f.t, "cards"), "synthetic-current-a", externalAccount), "operations": bankFixture(f.t, "history"), "loyalty": bankFixture(f.t, "loyalty"), "summaries": []string{bankFixture(f.t, "summary")}, "account": externalAccount}
		raw, _ := json.Marshal(b)
		hash := sha256.Sum256(raw)
		page := in.SyncRequest
		delete(page, "connectionId")
		delete(page, "replayRange")
		page["cursor"] = ""
		page["complete"] = true
		page["coverage"] = map[string]any{"state": "partial", "gaps": []string{"source_partial"}}
		page["records"] = []any{}
		page["evidence"] = []any{map[string]any{"id": "alfa-page", "mediaType": "application/json", "data": base64.StdEncoding.EncodeToString(raw), "sha256": hex.EncodeToString(hash[:]), "locator": "alfa:structured-read:v1"}}
		if afterRead != nil {
			afterRead()
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"outcome": "page", "page": page})
	})
	server := &http.Server{Handler: mux, ReadHeaderTimeout: time.Second}
	go func() { _ = server.Serve(listener) }()
	f.t.Cleanup(func() { _ = server.Close() })
	return socket
}
func (f *fixture) run(socket string) error {
	f.t.Helper()
	handler := collector.Handler{Socket: socket, Vault: f.vault, Service: f.ingestion, Repository: f.db}
	worker := jobs.Worker{Repository: f.db, Admission: f.gate, Handler: handler, Config: jobs.DefaultWorkerConfig(jobdomain.Sync)}
	return worker.Step(ctx)
}
func count(t *testing.T, db *pgxpool.Pool, table string) int {
	t.Helper()
	var n int
	if err := db.QueryRow(ctx, `SELECT count(*) FROM want_keep.`+table).Scan(&n); err != nil {
		t.Fatal(err)
	}
	return n
}

func TestReadConnectorReplayAndOwnerSession(t *testing.T) {
	f := newFixture(t)
	f.create(0)
	a := access.NewService(f.sessions, f.db, f.gate)
	if _, err := a.Begin(ctx, f.tokens[1], f.connection, connections.BrowserSession); err == nil {
		t.Fatal("partner obtained owner session grant")
	}
	f.enroll(0)
	c, err := f.connectionRecord()
	if err != nil || c.State != "connected" || c.Generation != 2 {
		t.Fatal("session did not authorize and fence connection", c, err)
	}
	f.sync()
	var syncJobs int
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.jobs WHERE kind='sync'`).Scan(&syncJobs); err != nil {
		t.Fatal(err)
	}
	if syncJobs != 0 {
		t.Fatal("unadmitted sync created job")
	}
	f.admit()
	socket := f.collector(nil, "synthetic-current-a", "accounts")
	for range 2 {
		f.sync()
		if err := f.run(socket); err != nil {
			t.Fatal(err)
		}
	}
	for table, want := range map[string]int{"accounts": 1, "card_aliases": 1, "source_records": 1, "source_revisions": 1, "account_observations": 0} {
		if got := count(t, f.admin, table); got != want {
			t.Fatalf("%s=%d want %d", table, got, want)
		}
	}
	var nonOpening int
	if err = f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.operation_revisions WHERE economic_type!='opening'`).Scan(&nonOpening); err != nil || nonOpening != 0 {
		t.Fatal("unverified source created money", err, nonOpening)
	}
	var ciphertext []byte
	if err = f.admin.QueryRow(ctx, `SELECT ciphertext FROM want_keep.collector_evidence_items LIMIT 1`).Scan(&ciphertext); err != nil || strings.Contains(string(ciphertext), "synthetic-current-a") {
		t.Fatal("evidence plaintext persisted", err)
	}
	var coverage string
	if err = f.admin.QueryRow(ctx, `SELECT coverage FROM want_keep.sync_progress WHERE connection_id=$1`, f.connection).Scan(&coverage); err != nil || coverage != "partial" {
		t.Fatal("coverage overstated", coverage, err)
	}
}
func TestSessionReplacementQuarantinesOldResult(t *testing.T) {
	f := newFixture(t)
	f.create(0)
	f.enroll(0)
	f.admit()
	f.sync()
	socket := f.collector(func() { f.enroll(0) }, "synthetic-current-a", "accounts")
	if err := f.run(socket); err != nil && !errors.Is(err, jobdomain.ErrStaleAttempt) {
		t.Fatal(err)
	}
	if count(t, f.admin, "accounts") != 0 || count(t, f.admin, "source_records") != 0 {
		t.Fatal("old session result applied")
	}
	var state string
	if err := f.admin.QueryRow(ctx, `SELECT disposition FROM want_keep.collector_evidence_batches LIMIT 1`).Scan(&state); err != nil || state != "stale_result" {
		t.Fatal("stale evidence not quarantined", state, err)
	}
}
func TestDisconnectQuarantinesAndInvalidatesVault(t *testing.T) {
	f := newFixture(t)
	f.create(0)
	f.enroll(0)
	f.admit()
	f.sync()
	socket := f.collector(func() {
		if err := f.db.WithinHousehold(ctx, f.q, func(ctx context.Context) error { return f.db.Disconnect(ctx, f.connection) }); err != nil {
			t.Error(err)
		}
	}, "synthetic-current-a", "accounts")
	if err := f.run(socket); err != nil && !errors.Is(err, jobdomain.ErrStaleAttempt) {
		t.Fatal(err)
	}
	if count(t, f.admin, "source_records") != 0 {
		t.Fatal("disconnected result applied")
	}
	var revoked bool
	if err := f.admin.QueryRow(ctx, `SELECT revoked FROM want_keep.connection_secrets WHERE connection_id=$1`, f.connection).Scan(&revoked); err != nil || !revoked {
		t.Fatal("secret not revoked", err)
	}
}

func (f *fixture) connectionRecord() (connections.ConnectionRecord, error) {
	var c connections.ConnectionRecord
	err := f.db.WithinFinancialRead(ctx, f.p, func(ctx context.Context) error {
		var err error
		c, err = f.service.Read(ctx, f.p, f.connection)
		return err
	})
	return c, err
}

func TestDistinctExternalAccountsAndOwnerAuthorization(t *testing.T) {
	f := newFixture(t)
	f.create(0)
	first := f.connection
	if w := f.request(1, "POST", "/api/v1/connections/"+first+"/authorization", `{"expectedRevision":1}`, ""); w.Code != 403 {
		t.Fatal("partner issued owner grant", w.Code)
	}
	f.enroll(0)
	f.admit()
	f.sync()
	if err := f.run(f.collector(nil, "synthetic-current-a", "accounts")); err != nil {
		t.Fatal(err)
	}
	f.create(1)
	second := f.connection
	if first == second {
		t.Fatal("connections collapsed")
	}
	f.enroll(1)
	f.sync()
	if err := f.run(f.collector(nil, "synthetic-current-b", "accounts")); err != nil {
		t.Fatal(err)
	}
	if count(t, f.admin, "accounts") != 2 || count(t, f.admin, "source_records") != 2 {
		t.Fatal("different source accounts merged")
	}
	w := f.request(0, "GET", "/api/v1/connections", "", "")
	if w.Code != 200 || !strings.Contains(w.Body.String(), "debit_card") || strings.Contains(w.Body.String(), "synthetic-xsrf") {
		t.Fatal("connection metadata leaked secret or lost products", w.Code)
	}
}

func TestUnsupportedAssetDoesNotRejectSupportedAccount(t *testing.T) {
	f := newFixture(t)
	f.create(0)
	f.enroll(0)
	f.admit()
	f.sync()
	if err := f.run(f.collector(nil, "synthetic-current-a", "accounts-mixed")); err != nil {
		t.Fatal(err)
	}
	if count(t, f.admin, "accounts") != 1 || count(t, f.admin, "source_records") != 1 {
		t.Fatal("unsupported asset rejected the supported page")
	}
	var partial bool
	if err := f.admin.QueryRow(ctx, `SELECT coverage='partial' AND 'unsupported_asset'=ANY(gaps) FROM want_keep.sync_progress WHERE connection_id=$1`, f.connection).Scan(&partial); err != nil || !partial {
		t.Fatal("unsupported asset gap missing", err)
	}
}
