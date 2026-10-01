//go:build integration

package collector_test

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connectionaccess "github.com/pchkauu/want-keep/backend/internal/connections/access"
	admission "github.com/pchkauu/want-keep/backend/internal/connections/admission"
	collector "github.com/pchkauu/want-keep/backend/internal/connections/collector"
	"github.com/pchkauu/want-keep/backend/internal/connections/credentials"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	integrations "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	jobsapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var (
	databaseURL *url.URL
	cluster     *pgxpool.Pool
	testContext = context.Background()
)

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Collector integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
		os.Exit(2)
	}
	databaseURL = u
	cluster, err = pgxpool.New(testContext, u.String())
	if err != nil || cluster.Ping(testContext) != nil {
		fmt.Fprintln(os.Stderr, "Isolated PostgreSQL is unavailable")
		os.Exit(2)
	}
	_, err = cluster.Exec(testContext, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_app') THEN CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; END IF; IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_maintenance') THEN CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance'; END IF; END $$;`)
	if err != nil {
		fmt.Fprintln(os.Stderr, "Cannot prepare isolated test roles")
		os.Exit(2)
	}
	code := m.Run()
	cluster.Close()
	os.Exit(code)
}

func TestEncryptedEvidenceLifecycleSurvivesRestart(t *testing.T) {
	store, admin, principal, job, keys, keyPath := fixture(t)
	evidence, err := collector.NewEvidenceStore(store, keys)
	if err != nil {
		t.Fatal(err)
	}
	at, _ := calendar.ParseInstant("2026-09-14T10:00:00.123456789Z")
	raw := []byte(`{"balance":"5000","description":"synthetic only"}`)
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	batch := ingestion.EvidenceBatch{
		HouseholdID: string(principal.HouseholdID()), JobID: job.ID, PageReference: "evidence:page:" + uuid.NewString(),
		FetchedAt: at, Disposition: ingestion.EvidenceStaged,
		Items: []ingestion.StoredEvidence{{Reference: "evidence:raw:" + uuid.NewString(), Raw: ingestion.Evidence{ID: "account-12345", MediaType: "application/json", Digest: digest, Locator: "synthetic:page:1", Data: raw}}},
	}
	if err = evidence.Save(testContext, batch); err != nil {
		t.Fatal(err)
	}
	var ciphertext []byte
	var disposition string
	var fetched time.Time
	var fetchedNS int16
	if err = admin.QueryRow(testContext, `SELECT i.ciphertext,b.disposition,b.fetched_at,b.fetched_ns FROM want_keep.collector_evidence_items i JOIN want_keep.collector_evidence_batches b USING(household_id,page_reference) WHERE i.household_id=$1 AND i.page_reference=$2`, batch.HouseholdID, batch.PageReference).Scan(&ciphertext, &disposition, &fetched, &fetchedNS); err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(ciphertext, raw) || disposition != "staged" || !fetched.UTC().Add(time.Duration(fetchedNS)).Equal(at.Time()) {
		t.Fatal("encrypted evidence metadata mismatch")
	}
	var storedMetadata string
	if err = admin.QueryRow(testContext, `SELECT to_jsonb(i)::text FROM want_keep.collector_evidence_items i WHERE i.household_id=$1 AND i.page_reference=$2`, batch.HouseholdID, batch.PageReference).Scan(&storedMetadata); err != nil || strings.Contains(storedMetadata, "account-12345") {
		t.Fatal("provider evidence ID leaked into metadata", err)
	}
	restarted, err := cryptobox.Load(keyPath, "connections")
	if err != nil {
		t.Fatal(err)
	}
	aad, _ := json.Marshal(struct {
		Version       int    `json:"version"`
		Purpose       string `json:"purpose"`
		HouseholdID   string `json:"householdId"`
		JobID         string `json:"jobId"`
		PageReference string `json:"pageReference"`
		ItemReference string `json:"itemReference"`
	}{1, "collector-evidence", batch.HouseholdID, batch.JobID, batch.PageReference, batch.Items[0].Reference})
	plain, err := restarted.Open(ciphertext, aad)
	if err != nil {
		t.Fatal(err)
	}
	var decoded ingestion.Evidence
	if json.Unmarshal(plain, &decoded) != nil || !bytes.Equal(decoded.Data, raw) {
		t.Fatal("restart could not read encrypted evidence")
	}
	if _, err = admin.Exec(testContext, `INSERT INTO want_keep.ingestion_result_receipts(household_id,id,job_id,lease_token,attempt,input_cursor,evidence_ref,kind) VALUES($1,$2,$3,$4,$5,$6,$7,'page')`, principal.HouseholdID(), uuid.NewString(), job.ID, job.LeaseToken, job.Attempt, job.Cursor, batch.PageReference); err != nil {
		t.Fatal(err)
	}
	staged, err := evidence.Staged(testContext, batch.HouseholdID, 10)
	if err != nil || len(staged) != 1 || staged[0].PageReference != batch.PageReference {
		t.Fatal("staged evidence not recoverable", staged, err)
	}
	if foreign, err := evidence.Staged(testContext, uuid.NewString(), 10); err != nil || len(foreign) != 0 {
		t.Fatal("household isolation failed", foreign, err)
	}
	done := ingestion.EvidenceDisposition{HouseholdID: batch.HouseholdID, JobID: batch.JobID, PageReference: batch.PageReference, State: ingestion.EvidenceApplied}
	if err = evidence.SetDisposition(testContext, done); err != nil {
		t.Fatal(err)
	}
	if err = evidence.SetDisposition(testContext, done); err != nil {
		t.Fatal("idempotent disposition failed", err)
	}
	if _, err = admin.Exec(testContext, `UPDATE want_keep.collector_evidence_items SET ciphertext='changed' WHERE household_id=$1 AND page_reference=$2`, batch.HouseholdID, batch.PageReference); err == nil {
		t.Fatal("immutable evidence item changed")
	}
	if _, err = admin.Exec(testContext, `UPDATE want_keep.collector_evidence_batches SET disposition='stale_result' WHERE household_id=$1 AND page_reference=$2`, batch.HouseholdID, batch.PageReference); err == nil {
		t.Fatal("terminal disposition changed")
	}
}

func TestBusyRejectionClearsExternalMarkerUnderCurrentLease(t *testing.T) {
	store, admin, principal, job, _, _ := fixture(t)
	if err := store.BeginExternal(testContext, principal, job); err != nil {
		t.Fatal(err)
	}
	if err := store.WithinHousehold(testContext, principal, func(ctx context.Context) error {
		return store.AcknowledgeExternalResult(ctx, principal, job)
	}); err != nil {
		t.Fatal(err)
	}
	if err := store.SetJobOutcome(testContext, principal, job, jobs.Waiting, jobs.HandlerUnavailable, 0); err != nil {
		t.Fatal(err)
	}
	var state string
	var externalStarted bool
	var attempt int
	if err := admin.QueryRow(testContext, `SELECT state,external_started,attempt FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, principal.HouseholdID(), job.ID).Scan(&state, &externalStarted, &attempt); err != nil {
		t.Fatal(err)
	}
	if state != "waiting" || externalStarted || attempt != job.Attempt-1 {
		t.Fatal("busy rejection did not leave a retryable job", state, externalStarted, attempt)
	}
}

func TestCollectorBusyHandlerKeepsJobRetryable(t *testing.T) {
	store, admin, principal, job, keys, _ := fixture(t, false)
	ref := connections.SecretReference{HouseholdID: principal.HouseholdID(), ConnectionID: job.ConnectionID, Purpose: connections.BrowserSession, Generation: job.ConnectionGeneration, Revision: 1}
	aad, _ := json.Marshal(struct {
		Version   int
		Purpose   string
		Reference connections.SecretReference
	}{1, "connection-secret", ref})
	secret, err := keys.Seal([]byte(`{"cookies":[],"origins":[]}`), aad)
	if err != nil {
		t.Fatal(err)
	}
	if err = store.WithinHousehold(testContext, principal, func(ctx context.Context) error {
		return store.SaveEncryptedSecret(ctx, ref, secret)
	}); err != nil {
		t.Fatal(err)
	}
	manifest, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "collector", "contracts", "v10", "fixtures", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	socket := filepath.Join("/tmp", "wk-"+uuid.NewString()[:8]+".sock")
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/ready":
			_, _ = io.WriteString(response, "want-keep-browser-collector/1\n")
		case "/v1/capabilities":
			_, _ = response.Write(manifest)
		case "/v1/read":
			response.WriteHeader(http.StatusConflict)
			_, _ = io.WriteString(response, `{"code":"collector_busy"}`)
		}
	})}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() { _ = server.Close(); _ = os.Remove(socket) })
	gate := admission.NewService(store, store)
	nowValue, _ := calendar.ParseInstant("2026-09-14T10:00:00Z")
	now := func() calendar.Instant { return nowValue }
	evidence, err := collector.NewEvidenceStore(store, keys)
	if err != nil {
		t.Fatal(err)
	}
	accountService := accounts.NewService(store, store, now, uuid.NewString)
	accountImporter, err := integrations.NewAccountImporter(accountService, store)
	if err != nil {
		t.Fatal(err)
	}
	sources := ledger.NewSources(store, ledger.NewWriter(store, store), allocation.NewService(store, now, uuid.NewString))
	sourceWriter, err := integrations.NewSourceWriter(sources, store)
	if err != nil {
		t.Fatal(err)
	}
	service, err := integrations.NewService(gate, evidence, accountImporter, sourceWriter, now, uuid.NewString)
	if err != nil {
		t.Fatal(err)
	}
	handler := collector.Handler{Socket: socket, Vault: credentials.New(connectionaccess.NewService(nil, store, gate), store, keys), Service: service}
	worker := jobsapp.Worker{Admission: gate, Repository: store, Handler: handler, Config: jobsapp.DefaultWorkerConfig(jobs.Sync)}
	if err = worker.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var state string
	var externalStarted bool
	if err = admin.QueryRow(testContext, `SELECT state,external_started FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, principal.HouseholdID(), job.ID).Scan(&state, &externalStarted); err != nil || state != "waiting" || externalStarted {
		t.Fatal("collector busy stranded the job", state, externalStarted, err)
	}
}

func TestStagedReconciliationUsesTerminalReceiptAfterRestart(t *testing.T) {
	store, admin, principal, job, keys, _ := fixture(t)
	evidence, err := collector.NewEvidenceStore(store, keys)
	if err != nil {
		t.Fatal(err)
	}
	at, _ := calendar.ParseInstant("2026-09-14T10:00:00.123456789Z")
	raw := []byte(`{"balance":"5000"}`)
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	batch := ingestion.EvidenceBatch{
		HouseholdID: string(principal.HouseholdID()), JobID: job.ID, PageReference: "evidence:page:" + uuid.NewString(), FetchedAt: at, Disposition: ingestion.EvidenceStaged,
		Items: []ingestion.StoredEvidence{{Reference: "evidence:raw:" + uuid.NewString(), Raw: ingestion.Evidence{ID: "raw-restart", MediaType: "application/json", Digest: digest, Locator: "synthetic:restart", Data: raw}}},
	}
	if err = evidence.Save(testContext, batch); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(testContext, `INSERT INTO want_keep.ingestion_result_receipts(household_id,id,job_id,lease_token,attempt,input_cursor,evidence_ref,kind) VALUES($1,$2,$3,$4,$5,$6,$7,'page')`, principal.HouseholdID(), uuid.NewString(), job.ID, job.LeaseToken, job.Attempt, job.Cursor, batch.PageReference); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(testContext, `INSERT INTO want_keep.collector_evidence_batches(household_id,page_reference,job_id,fetched_at,fetched_ns,disposition) SELECT $1,'evidence:unresolved:'||lpad(g::text,3,'0'),$2,'2026-09-14T09:00:00Z',0,'staged' FROM generate_series(1,100) g`, principal.HouseholdID(), job.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(testContext, `UPDATE want_keep.memberships SET active=false WHERE household_id=$1 AND user_id=$2`, principal.HouseholdID(), principal.UserID()); err != nil {
		t.Fatal(err)
	}
	reconciler := collector.StagedReconciler{Reconcile: store.ReconcileStagedCollectorEvidence, Report: func(error) {}}
	if err = reconciler.Step(testContext); err != nil {
		t.Fatal(err)
	}
	var disposition string
	if err = admin.QueryRow(testContext, `SELECT disposition FROM want_keep.collector_evidence_batches WHERE household_id=$1 AND page_reference=$2`, principal.HouseholdID(), batch.PageReference).Scan(&disposition); err != nil || disposition != "applied" {
		t.Fatal("staged receipt was not reconciled", disposition, err)
	}
	var unresolved int
	if err = admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.collector_evidence_batches WHERE household_id=$1 AND disposition='staged'`, principal.HouseholdID()).Scan(&unresolved); err != nil || unresolved != 100 {
		t.Fatal("unproven evidence changed", unresolved, err)
	}
}

func fixture(t *testing.T, claim ...bool) (*storage.Store, *pgxpool.Pool, household.Principal, jobs.Job, *cryptobox.Keyring, string) {
	t.Helper()
	name := "wk_collector_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _, _ = cluster.Exec(testContext, `DROP DATABASE `+name+` WITH (FORCE)`) })
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(testContext, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(admin.Close)
	if err = storage.Migrate(testContext, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	family := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Synthetic member"}
	membership := household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: family.ID, UserID: user.ID, Active: true}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = store.InitializeHousehold(testContext, family, []household.User{user}, []household.Membership{membership}, zone, 2); err != nil {
		t.Fatal(err)
	}
	principal, _ := membership.Principal()
	binding := connections.Binding{Provider: "bybit", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("0", 64), CollectorImageDigest: "sha256:" + strings.Repeat("1", 64), ContractVersion: "10", AllowlistRevision: "allowlist-1", NonSecretConfigRevision: "config-1", OperatorPermissionRevision: "permission-1"}
	connectionID := uuid.NewString()
	gate := admission.NewService(store, store)
	if err = store.WithinHousehold(testContext, principal, func(ctx context.Context) error {
		return store.CreateConnection(ctx, admission.Connection{HouseholdID: family.ID, ID: connectionID, Provider: "bybit", Owner: user.ID, Generation: 1, Authorized: true, SecretPurpose: connections.BrowserSession})
	}); err != nil {
		t.Fatal(err)
	}
	now, _ := calendar.ParseInstant("2026-09-14T09:00:00Z")
	for _, kind := range []connections.CheckKind{connections.ProviderCheck, connections.HostCheck} {
		if _, err = gate.RecordCheck(testContext, connections.Check{Kind: kind, Binding: binding, Result: connections.CheckPassed, At: now}); err != nil {
			t.Fatal(err)
		}
	}
	requested, err := gate.RequestSync(testContext, principal, connectionID, binding, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	var job jobs.Job
	if len(claim) > 0 && !claim[0] {
		job, err = store.Job(testContext, principal, requested.ID)
	} else {
		var claimed []jobs.Job
		claimed, err = store.ClaimJobs(testContext, "sync", 10, time.Minute)
		for _, candidate := range claimed {
			if candidate.ID == requested.ID {
				job = candidate
			}
		}
	}
	if err != nil || job.ID == "" {
		t.Fatal("job not available", err)
	}
	keyPath := filepath.Join(t.TempDir(), "connection-keyring")
	if err = cryptobox.Generate(keyPath, "connections"); err != nil {
		t.Fatal(err)
	}
	keys, err := cryptobox.Load(keyPath, "connections")
	if err != nil {
		t.Fatal(err)
	}
	return store, admin, principal, job, keys, keyPath
}
