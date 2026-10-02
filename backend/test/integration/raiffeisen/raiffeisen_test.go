//go:build integration

package raiffeisen_test

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
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
	"github.com/pchkauu/want-keep/backend/internal/integrations/raiffeisen"
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
		fmt.Fprintln(os.Stderr, "Raiffeisen integration requires isolated loopback want_keep_test PostgreSQL.")
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
	name := "wk_raif_" + strings.ReplaceAll(uuid.NewString(), "-", "")
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
	f.binding = connections.Binding{Provider: "raiffeisen", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("0", 64), CollectorImageDigest: "sha256:" + strings.Repeat("1", 64), ContractVersion: "10", AllowlistRevision: "rbo-v1", NonSecretConfigRevision: "test-v1", OperatorPermissionRevision: "synthetic-v1"}
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
	f.service = &connectionapp.Service{Repository: db, Sessions: f.sessions, Executor: application.NewExecutor(db, db, nowInstant), Admission: f.gate, Vault: f.vault, Bindings: map[string]connections.Binding{"raiffeisen": f.binding}, NewID: uuid.NewString}
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

func TestConnectionCommandsAndOAuthClaim(t *testing.T) {
	f := newFixture(t)
	from := time.Now().Add(-48 * time.Hour).In(time.FixedZone("Moscow", 3*3600)).Format(time.DateOnly)
	body := fmt.Sprintf(`{"provider":"raiffeisen","externalAccountOwnerId":%q,"historyFrom":%q,"products":["current"]}`, f.p.UserID(), from)
	key := uuid.NewString()
	for i := 0; i < 2; i++ {
		w := f.request(1, "POST", "/api/v1/connections", body, key)
		if w.Code != 202 {
			t.Fatalf("create/replay %d: %s", w.Code, w.Body.String())
		}
	}
	var count int
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.connections`).Scan(&count); err != nil || count != 1 {
		t.Fatal("duplicate connection", err)
	}
	if err := f.admin.QueryRow(ctx, `SELECT id FROM want_keep.connections`).Scan(&f.connection); err != nil {
		t.Fatal(err)
	}
	w := f.request(0, "POST", "/api/v1/connections/"+f.connection+"/sync", `{"expectedRevision":1}`, uuid.NewString())
	if w.Code != 409 && w.Code != 202 {
		t.Fatalf("gate refusal: %d %s", w.Code, w.Body.String())
	}
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.jobs WHERE kind='sync'`).Scan(&count); err != nil || count != 0 {
		t.Fatal("unadmitted job created", err)
	}
	w = f.request(1, "GET", "/api/v1/connections", "", "")
	if w.Code != 200 || strings.Contains(w.Body.String(), "ciphertext") {
		t.Fatalf("list: %d %s", w.Code, w.Body.String())
	}
	// Claim persists before provider IO; a competing callback cannot exchange the same code.
	var a connections.OAuthAttempt
	if err := f.db.WithinHousehold(ctx, f.p, func(ctx context.Context) error {
		a = connections.OAuthAttempt{ID: uuid.NewString(), ConnectionID: f.connection, HouseholdID: f.p.HouseholdID(), OwnerID: f.p.UserID(), SessionHash: f.tokens[0].Hash(), StateHash: strings.Repeat("a", 64), Generation: 1, Revision: 1, ExpiresAt: time.Now().Add(time.Minute), Phase: "ready", Ciphertext: []byte("synthetic-encrypted-state")}
		return f.db.SaveOAuthAttempt(ctx, a)
	}); err != nil {
		t.Fatal(err)
	}
	start := make(chan struct{})
	results := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			results <- f.db.WithinHousehold(ctx, f.p, func(ctx context.Context) error { return f.db.OAuthPhase(ctx, a, "ready", "claimed") })
		}()
	}
	close(start)
	wg.Wait()
	close(results)
	success := 0
	for err := range results {
		if err == nil {
			success++
		}
	}
	if success != 1 {
		t.Fatal("OAuth code claimed twice")
	}
	w = f.request(1, "POST", "/api/v1/connections/"+f.connection+"/reauth", `{"expectedRevision":1}`, uuid.NewString())
	if w.Code != 202 {
		t.Fatalf("partner reauth: %d %s", w.Code, w.Body.String())
	}
	if err := f.db.WithinFinancialRead(ctx, f.p, func(ctx context.Context) error {
		c, err := f.db.ConnectionRecord(ctx, f.p, f.connection)
		if err == nil && (c.Generation != 2 || c.Revision != 2 || c.State != "reauth_required") {
			t.Fatal("generation fence lost")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
}

type fakeBank struct {
	variant    int
	mu         sync.Mutex
	statements map[string]string
	creates    int
	afterFile  func()
}

func (b *fakeBank) RoundTrip(r *http.Request) (*http.Response, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	status, data := 200, ""
	switch {
	case strings.HasSuffix(r.URL.Path, "/accounts"):
		data = `[{"id":"00000000-0000-4000-8000-000000000001","number":"00000000000000000000","name":"Synthetic account","organizationName":"Synthetic organization","currency":"RUR"}]`
	case r.Method == "POST":
		var request struct{ From, To string }
		body, _ := io.ReadAll(r.Body)
		if json.Unmarshal(body, &request) != nil {
			return nil, fmt.Errorf("synthetic request invalid")
		}
		b.creates++
		id := uuid.NewString()
		sample, err := os.ReadFile("../../../../spec/001-want-keep-mvp/evidence/raiffeisen.camt053.sample.xml")
		if err != nil {
			return nil, err
		}
		xml := string(sample)
		xml = strings.ReplaceAll(xml, "SYNTHETIC-MESSAGE", id)
		xml = strings.ReplaceAll(xml, "SYNTHETIC-STATEMENT", id)
		xml = strings.ReplaceAll(xml, "2026-08-01T00:00:00+03:00", request.From+"T00:00:00+03:00")
		xml = strings.ReplaceAll(xml, "2026-08-31T23:59:59+03:00", request.To+"T23:59:59+03:00")
		xml = strings.ReplaceAll(xml, "<RmtInf>", "<RltdPties><DbtrAcct><Id><Othr><Id>SYNTHETIC-PARTY</Id></Othr></Id></DbtrAcct></RltdPties><RmtInf>")
		xml = strings.ReplaceAll(xml, request.To+"T23:59:59+03:00", time.Now().UTC().Add(-time.Minute).Format(time.RFC3339))
		if b.variant > 0 {
			xml = strings.ReplaceAll(xml, "2026-09-01T06:00:00+03:00", time.Now().UTC().Format(time.RFC3339Nano))
			xml = strings.ReplaceAll(xml, "1000.00", "1100.00")
		}
		if b.variant > 1 {
			xml = strings.ReplaceAll(xml, "1100.00", "1200.00")
			xml = strings.ReplaceAll(xml, "SYNTHETIC-ENTRY-1", "DIFFERENT-ENTRY-1")
			xml = strings.ReplaceAll(xml, "SYNTHETIC-END-1", "DIFFERENT-END-1")
		}
		b.statements[id] = xml
		status, data = 202, fmt.Sprintf(`{"reportId":%q}`, id)
	case strings.HasSuffix(r.URL.Path, "/status"):
		data = `{"status":"completed"}`
	case strings.HasSuffix(r.URL.Path, "/file"):
		parts := strings.Split(r.URL.Path, "/")
		data = b.statements[parts[len(parts)-2]]
		if b.afterFile != nil {
			b.afterFile()
		}
	default:
		return nil, fmt.Errorf("unexpected synthetic bank route")
	}
	return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(data)), Header: http.Header{}}, nil
}

func TestNativeIngestionReplay(t *testing.T) {
	f := newFixture(t)
	from, _ := calendar.ParseDate(time.Now().In(time.FixedZone("Moscow", 3*3600)).Format(time.DateOnly))
	err := f.db.WithinHousehold(ctx, f.p, func(ctx context.Context) error {
		c := connections.ConnectionRecord{ID: f.connection, HouseholdID: f.p.HouseholdID(), OwnerID: f.p.UserID(), Provider: "raiffeisen", State: "pending", Revision: 1, Generation: 1, HistoryFrom: from}
		if err := f.db.CreateConnectionRecord(ctx, f.p, c); err != nil {
			return err
		}
		if err := f.db.AuthorizeConnection(ctx, f.p, c); err != nil {
			return err
		}
		ref := connections.SecretReference{HouseholdID: f.p.HouseholdID(), ConnectionID: f.connection, Purpose: connections.OAuthTokens, Generation: 1, Revision: 1}
		plain, _ := json.Marshal(connections.TokenSet{Access: "synthetic-access", ID: "synthetic-id", Refresh: "synthetic-refresh", Type: "bearer", Subject: "synthetic-owner", IssuedAt: time.Now()})
		ciphertext, err := f.vault.Seal(ref, plain)
		if err != nil {
			return err
		}
		return f.db.SaveEncryptedSecret(ctx, ref, ciphertext)
	})
	if err != nil {
		t.Fatal(err)
	}
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err := f.gate.RecordCheck(ctx, connections.Check{Kind: kind, Binding: f.binding, Result: connections.CheckPassed, At: f.now}); err != nil {
			t.Fatal(err)
		}
	}
	bank := &fakeBank{statements: map[string]string{}}
	for pass := 0; pass < 2; pass++ {
		if _, err := f.gate.RequestSync(ctx, f.p, f.connection, f.binding, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		handler := raiffeisen.Handler{Vault: f.vault, Service: f.ingestion, Gate: f.gate, Repository: f.db, Now: time.Now, TransportClient: func() *raiffeisen.Client { return raiffeisen.NewClient(bank) }}
		worker := jobs.Worker{Repository: f.db, Admission: f.gate, Handler: handler, Config: jobs.DefaultWorkerConfig(jobdomain.Sync)}
		if err := worker.Step(ctx); err != nil {
			t.Fatal(err)
		}

	}
	var operations, sources int
	if err = f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.operations`).Scan(&operations); err != nil {
		t.Fatal(err)
	}
	if err = f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.source_records`).Scan(&sources); err != nil {
		t.Fatal(err)
	}
	if operations != 3 || sources != 3 {
		t.Fatalf("overlap duplicated facts: operations=%d sources=%d", operations, sources)
	}
	var observations int
	if err = f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.account_observations`).Scan(&observations); err != nil {
		t.Fatal(err)
	}
	if observations < 1 {
		t.Fatal("closing observation lost")
	}
	for variant := 1; variant <= 2; variant++ {
		bank.variant = variant
		if _, err := f.gate.RequestSync(ctx, f.p, f.connection, f.binding, time.Now().Add(time.Hour)); err != nil {
			t.Fatal(err)
		}
		handler := raiffeisen.Handler{Vault: f.vault, Service: f.ingestion, Gate: f.gate, Repository: f.db, Now: time.Now, TransportClient: func() *raiffeisen.Client { return raiffeisen.NewClient(bank) }}
		if err := (jobs.Worker{Repository: f.db, Admission: f.gate, Handler: handler, Config: jobs.DefaultWorkerConfig(jobdomain.Sync)}).Step(ctx); err != nil {
			t.Fatal(err)
		}
		var amount string
		if err := f.admin.QueryRow(ctx, `SELECT p.amount::text FROM want_keep.postings p JOIN want_keep.operations o ON (o.household_id,o.id,o.revision)=(p.household_id,p.operation_id,p.revision) JOIN want_keep.operation_revisions r ON (r.household_id,r.operation_id,r.revision)=(p.household_id,p.operation_id,p.revision) WHERE r.economic_type='income'`).Scan(&amount); err != nil {
			t.Fatal(err)
		}
		if amount != "1100.00" {
			t.Fatalf("correction/collision changed effect %s", amount)
		}
	}
	bank.afterFile = func() {
		if _, err := f.gate.RecordCheck(ctx, connections.Check{Kind: connections.ProviderCheck, Binding: f.binding, Result: connections.CheckRevoked, At: instantNow()}); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.gate.RequestSync(ctx, f.p, f.connection, f.binding, time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	handler := raiffeisen.Handler{Vault: f.vault, Service: f.ingestion, Gate: f.gate, Repository: f.db, Now: time.Now, TransportClient: func() *raiffeisen.Client { return raiffeisen.NewClient(bank) }}
	if err := (jobs.Worker{Repository: f.db, Admission: f.gate, Handler: handler, Config: jobs.DefaultWorkerConfig(jobdomain.Sync)}).Step(ctx); err != nil && !errors.Is(err, jobdomain.ErrStaleAttempt) {
		t.Fatal(err)
	}
	var quarantined, after int
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.quarantine`).Scan(&quarantined); err != nil {
		t.Fatal(err)
	}
	if err := f.admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.operations`).Scan(&after); err != nil {
		t.Fatal(err)
	}
	if quarantined == 0 || after != operations {
		t.Fatal("stale result reached journal")
	}
}

type syntheticAuthorizer struct{ calls int }

func (*syntheticAuthorizer) Available() bool { return true }
func (*syntheticAuthorizer) URL(s connections.OAuthSecrets) (string, error) {
	return "https://sso.rbo.raiffeisen.ru/authorize?state=" + url.QueryEscape(s.State), nil
}
func (a *syntheticAuthorizer) Exchange(_ context.Context, _ string, _ connections.OAuthSecrets, at time.Time) (connections.TokenSet, error) {
	a.calls++
	return connections.TokenSet{Access: "synthetic-access", ID: "synthetic-id", Refresh: "synthetic-refresh", Type: "bearer", Subject: "synthetic-owner", IssuedAt: at}, nil
}
func TestOwnerOAuthAndCallbackReadback(t *testing.T) {
	f := newFixture(t)
	auth := &syntheticAuthorizer{}
	f.service.Authorizer = auth
	body := fmt.Sprintf(`{"provider":"raiffeisen","externalAccountOwnerId":%q,"historyFrom":"2026-08-01","products":["current"]}`, f.p.UserID())
	if w := f.request(1, "POST", "/api/v1/connections", body, uuid.NewString()); w.Code != 202 {
		t.Fatal(w.Body.String())
	}
	if err := f.admin.QueryRow(ctx, `SELECT id FROM want_keep.connections`).Scan(&f.connection); err != nil {
		t.Fatal(err)
	}
	path := "/api/v1/connections/" + f.connection + "/authorization"
	if w := f.request(1, "POST", path, `{"expectedRevision":1}`, ""); w.Code != 403 {
		t.Fatalf("partner authorization %d", w.Code)
	}
	w := f.request(0, "POST", path, `{"expectedRevision":1}`, "")
	if w.Code != 200 {
		t.Fatal(w.Body.String())
	}
	var attempt struct {
		URL string `json:"authorizationUrl"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &attempt); err != nil {
		t.Fatal(err)
	}
	u, _ := url.Parse(attempt.URL)
	callback := "/api/v1/connections/raiffeisen/callback?state=" + url.QueryEscape(u.Query().Get("state")) + "&code=synthetic-code"
	if w := f.request(1, "GET", callback, "", ""); w.Code == 303 {
		t.Fatal("partner consumed owner state")
	}
	for range 2 {
		if w := f.request(0, "GET", callback, "", ""); w.Code != 303 {
			t.Fatalf("callback %d %s", w.Code, w.Body.String())
		}
	}
	if auth.calls != 1 {
		t.Fatal("completed exchange repeated")
	}
	if w := f.request(0, "DELETE", "/api/v1/connections/"+f.connection, `{"expectedRevision":2}`, uuid.NewString()); w.Code != 202 {
		t.Fatal(w.Body.String())
	}
	if w := f.request(0, "GET", callback, "", ""); w.Code == 303 {
		t.Fatal("disconnected generation accepted")
	}
}

func instantNow() calendar.Instant {
	at, _ := calendar.ParseInstant(time.Now().UTC().Format(time.RFC3339Nano))
	return at
}
