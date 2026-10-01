//go:build integration

package familyreimbursements_test

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	reimbursementdelivery "github.com/pchkauu/want-keep/backend/internal/delivery/reimbursements"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/migrations"
)

var databaseURL *url.URL
var cluster *pgxpool.Pool
var testContext = context.Background()

func TestMain(m *testing.M) {
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		fmt.Fprintln(os.Stderr, "Family reimbursement integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
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

type fixture struct {
	t              *testing.T
	store          *storage.Store
	admin          *pgxpool.Pool
	dsn            string
	family         household.Household
	p, q           household.Principal
	members        []household.Membership
	now            calendar.Instant
	executor       *commands.Executor
	reimbursements *journal.ReimbursementService
	writer         *journal.Writer
	ledger         *journal.Service
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_family_reimbursement_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := cluster.Exec(testContext, `DROP DATABASE `+name+` WITH (FORCE)`); err != nil {
			t.Errorf("drop test database: %v", err)
		}
	})
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
	f := &fixture{t: t, store: store, admin: admin, dsn: u.String(), family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}, now: instant("2026-09-14T12:00:00.123456789Z")}
	users := []household.User{{ID: household.UserID(uuid.NewString()), Name: "Member A"}, {ID: household.UserID(uuid.NewString()), Name: "Member B"}}
	for _, user := range users {
		f.members = append(f.members, household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: f.family.ID, UserID: user.ID, Active: true})
	}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = store.InitializeHousehold(testContext, f.family, users, f.members, zone, 2); err != nil {
		t.Fatal(err)
	}
	f.p, _ = f.members[0].Principal()
	f.q, _ = f.members[1].Principal()
	f.rebuild()
	return f
}

func (f *fixture) rebuild() {
	f.executor = commands.NewExecutor(f.store, f.store, func() calendar.Instant { return f.now })
	f.reimbursements = journal.NewReimbursementService(f.store, func() calendar.Instant { return f.now }, uuid.NewString)
	f.writer = journal.NewWriterWithReconciliationAndReimbursements(f.store, f.store, nil, f.reimbursements)
	f.ledger = journal.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
}

func (f *fixture) restart() {
	f.t.Helper()
	f.store.Close()
	store, err := storage.Open(testContext, storage.Config{DSN: f.dsn, Environment: "test", MaxConnections: 8})
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(store.Close)
	f.store = store
	f.rebuild()
}

func instant(value string) calendar.Instant {
	result, err := calendar.ParseInstant(value)
	if err != nil {
		panic(err)
	}
	return result
}

func cash(value string, asset money.Asset) money.Money {
	result, err := money.NewMoney(value, asset)
	if err != nil {
		panic(err)
	}
	return result
}

func (f *fixture) personalAccount(member int, asset money.Asset, balance string) string {
	f.t.Helper()
	id := uuid.NewString()
	owner := household.UserID("")
	scope := household.Shared
	if member >= 0 {
		owner = f.members[member].UserID
		scope = household.Personal
	}
	ownership, _ := household.NewOwnership(f.family.ID, scope, owner)
	date, _ := calendar.ParseDate("2026-09-01")
	principal := f.p
	if member == 1 {
		principal = f.q
	}
	err := f.store.WithinHousehold(testContext, principal, func(ctx context.Context) error {
		if err := f.store.CreateAccount(ctx, account.Account{ID: id, Name: "Personal cash", Product: "cash", Ownership: ownership, Asset: asset, Revision: 1, OpeningDate: date}); err != nil {
			return err
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		for _, field := range []string{"owned", "available", "locked", "debt"} {
			value := balance
			if field == "locked" || field == "debt" {
				value = "0"
			}
			known, _ := reporting.KnownAmount(cash(value, asset))
			if err := f.store.RecordBalance(ctx, account.Balance{AccountID: id, Field: field, Amount: known, Coverage: coverage, Freshness: reporting.Fresh, ObservedAt: f.now}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) execute(p household.Principal, kind string, apply func(context.Context) (command.Result, error)) command.Result {
	f.t.Helper()
	key := commands.Request{ID: uuid.NewString(), Kind: kind, PayloadHash: strings.Repeat("a", 64)}
	value, err := f.executor.Execute(testContext, p, key, apply)
	if err != nil {
		f.t.Fatal(err)
	}
	result, ok := value.Result()
	if !ok {
		f.t.Fatal("command did not succeed")
	}
	return result
}

func (f *fixture) expense(accountID, amount string, asset money.Asset) string {
	result := f.execute(f.p, "transactions.create", func(ctx context.Context) (command.Result, error) {
		return f.ledger.Create(ctx, f.p, journal.CreateInput{Type: ledger.Expense, AccountID: accountID, At: f.now, Amount: cash(amount, asset), PayerState: "known", PayerMemberID: f.members[0].ID})
	})
	return result.ResourceID
}

func (f *fixture) transfer(from, to string, sent money.Money, received money.Money) string {
	result := f.execute(f.p, "transfers.create", func(ctx context.Context) (command.Result, error) {
		return f.ledger.Transfer(ctx, f.p, journal.TransferInput{FromAccountID: from, ToAccountID: to, At: f.now, Sent: sent, Received: received})
	})
	return result.ResourceID
}

type client struct {
	f       *fixture
	p       household.Principal
	handler http.Handler
	token   identity.Token
}

func (f *fixture) client(p household.Principal) *client {
	f.t.Helper()
	token, _ := identity.NewToken(32)
	var member household.Membership
	for _, candidate := range f.members {
		if candidate.UserID == p.UserID() {
			member = candidate
		}
	}
	handle := make([]byte, 32)
	_, _ = rand.Read(handle)
	credentialID := uuid.NewString()
	err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		profile := identity.Profile{UserID: p.UserID(), HouseholdID: p.HouseholdID(), MembershipID: member.ID, Name: "Synthetic member", Locale: "en", ReportingAsset: "RUB", Handle: handle, Generation: 1}
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return err
		}
		credential := identity.Credential{ID: credentialID, UserID: p.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{}, UserVerified: true, CreatedAt: f.now.Time()}
		if err := f.store.SaveIdentityCredential(ctx, credential); err != nil {
			return err
		}
		return f.store.SaveIdentitySession(ctx, identity.Session{ID: uuid.NewString(), TokenHash: token.Hash(), CredentialID: credentialID, Name: "Synthetic browser", UserID: p.UserID(), CreatedAt: f.now.Time(), AuthenticatedAt: f.now.Time(), LastActivityAt: f.now.Time()})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	verifier, _ := webauthn.New("localhost", "http://localhost")
	sessions, err := identityapp.NewService(f.store, f.store, verifier, "localhost", "http://localhost", 2, func() time.Time { return f.now.Time() })
	if err != nil {
		f.t.Fatal(err)
	}
	commandQueries := commands.NewQueries(f.store, f.store.AuthorizeCommandResult)
	handler, err := reimbursementdelivery.New(f.reimbursements, f.executor, commandQueries, sessions, f.store, security.Config{Environment: "test", Origin: "http://localhost"}, func() calendar.Instant { return f.now })
	if err != nil {
		f.t.Fatal(err)
	}
	return &client{f: f, p: p, handler: handler, token: token}
}

func (c *client) call(method, path, key string, input any, expected int) *httptest.ResponseRecorder {
	c.f.t.Helper()
	raw, err := json.Marshal(input)
	if err != nil {
		c.f.t.Fatal(err)
	}
	request := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(raw))
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("Origin", "http://localhost")
	request.Header.Set("X-CSRF-Token", c.token.CSRF())
	if key != "" {
		request.Header.Set("Idempotency-Key", key)
	}
	request.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, request)
	if response.Code != expected {
		commandState := ""
		if key != "" {
			if value, loadErr := c.f.store.LoadCommand(testContext, c.p, key); loadErr == nil {
				snapshot := value.Snapshot()
				commandState = " command=" + string(snapshot.Status) + "/" + snapshot.ErrorCode
			}
		}
		c.f.t.Fatalf("%s %s => %d want %d: %s%s", method, path, response.Code, expected, response.Body.String(), commandState)
	}
	if response.Header().Get("Cache-Control") != "no-store" {
		c.f.t.Fatal("missing no-store")
	}
	return response
}

func (c *client) result(key string) command.Result {
	c.f.t.Helper()
	value, err := c.f.store.LoadCommand(testContext, c.p, key)
	if err != nil {
		c.f.t.Fatal(err)
	}
	return value.Snapshot().Result
}

func (c *client) command(key string) command.Snapshot {
	c.f.t.Helper()
	value, err := c.f.store.LoadCommand(testContext, c.p, key)
	if err != nil {
		c.f.t.Fatal(err)
	}
	return value.Snapshot()
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
