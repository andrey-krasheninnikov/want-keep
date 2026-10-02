//go:build integration

package refunds_test

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
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	ledgerdelivery "github.com/pchkauu/want-keep/backend/internal/delivery/ledger"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	"github.com/pchkauu/want-keep/backend/internal/identity/webauthn"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/application"
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
		fmt.Fprintln(os.Stderr, "Refund integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
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
	t        *testing.T
	store    *storage.Store
	admin    *pgxpool.Pool
	family   household.Household
	p, q     household.Principal
	members  []household.Membership
	now      calendar.Instant
	writer   *journal.Writer
	executor *commands.Executor
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_refunds_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if _, err := cluster.Exec(testContext, `DROP DATABASE `+name+` WITH (FORCE)`); err != nil {
			t.Errorf("drop synthetic refund database: %v", err)
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
	store, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 12})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	f := &fixture{t: t, store: store, admin: admin, family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"}, now: instant("2026-09-07T12:00:00.123456789Z")}
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
	f.writer = journal.NewWriterWithRefundsAndReimbursements(store, store, nil, expenses.NewProjector(store), journal.NewReimbursementService(store, func() calendar.Instant { return f.now }, uuid.NewString))
	f.executor = commands.NewExecutor(store, store, func() calendar.Instant { return f.now })
	return f
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

func (f *fixture) account(asset money.Asset, balance string) string {
	return f.accountWithProduct(asset, balance, "cash")
}

func (f *fixture) accountWithProduct(asset money.Asset, balance, product string) string {
	f.t.Helper()
	id := uuid.NewString()
	ownership, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	date, _ := calendar.ParseDate("2026-08-01")
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		if err := f.store.CreateAccount(ctx, account.Account{ID: id, Name: product, Product: product, Ownership: ownership, Asset: asset, Revision: 1, OpeningDate: date}); err != nil {
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

func (f *fixture) available(accountID string, principal household.Principal) string {
	f.t.Helper()
	value, err := f.store.Balance(testContext, principal, accountID, "available")
	if err != nil {
		f.t.Fatal(err)
	}
	amount, known := value.Amount.Value()
	if !known {
		f.t.Fatal("available balance is unknown")
	}
	return amount.Amount()
}

func (f *fixture) postRefund(accountID, amount string) string {
	f.t.Helper()
	id := uuid.NewString()
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	revision := ledger.Revision{OperationID: id, Revision: 1, ActorID: f.p.UserID(), Reason: "Imported refund", Type: ledger.Refund, State: ledger.Posted, OccurredAt: f.now, Origin: "source", FeeKnowledge: ledger.KnownFees, PayerState: "not_applicable", Postings: []ledger.Posting{{AccountID: accountID, Money: cash(amount, money.RUB), Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}, Allocation: ledger.NotApplicableAllocation(), RecordedAt: f.now}
	var err error
	revision, err = revision.InTimezone(zone)
	if err != nil {
		f.t.Fatal(err)
	}
	if err = f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.writer.Append(ctx, f.p, revision, 0) }); err != nil {
		f.t.Fatal(err)
	}
	return id
}

func (f *fixture) transition(operationID string, state ledger.State) {
	f.t.Helper()
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		current, found, err := f.store.CurrentLedgerRevision(ctx, f.p, operationID)
		if err != nil {
			return err
		}
		if !found {
			return ledger.ErrNotFound
		}
		next := current.Clone()
		next.Revision++
		next.State = state
		next.DecisionID = ""
		next.RecordedAt = instant(f.now.Time().Add(time.Duration(next.Revision) * time.Second).Format(time.RFC3339Nano))
		return f.writer.Append(ctx, f.p, next, current.Revision)
	})
	if err != nil {
		f.t.Fatal(err)
	}
}

type client struct {
	f       *fixture
	handler http.Handler
	token   identity.Token
}

func (f *fixture) client(principal household.Principal) *client {
	f.t.Helper()
	token, _ := identity.NewToken(32)
	var member household.Membership
	for _, candidate := range f.members {
		if candidate.UserID == principal.UserID() {
			member = candidate
		}
	}
	handle := make([]byte, 32)
	_, _ = rand.Read(handle)
	credentialID := uuid.NewString()
	err := f.store.WithinIdentity(testContext, func(ctx context.Context) error {
		profile := identity.Profile{UserID: principal.UserID(), HouseholdID: principal.HouseholdID(), MembershipID: member.ID, Name: "Synthetic member", Locale: "en", ReportingAsset: "RUB", Handle: handle, Generation: 1}
		if err := f.store.CreateIdentityProfile(ctx, profile); err != nil {
			return err
		}
		credential := identity.Credential{ID: credentialID, UserID: principal.UserID(), Name: "Synthetic credential", RPID: "localhost", RawID: handle, PublicKey: []byte{}, AAGUID: []byte{}, AttestationObject: []byte{}, AttestationClientData: []byte{}, AttestationClientHash: []byte{}, Transports: []string{}, UserVerified: true, CreatedAt: f.now.Time()}
		if err := f.store.SaveIdentityCredential(ctx, credential); err != nil {
			return err
		}
		return f.store.SaveIdentitySession(ctx, identity.Session{ID: uuid.NewString(), TokenHash: token.Hash(), CredentialID: credentialID, Name: "Synthetic browser", UserID: principal.UserID(), CreatedAt: f.now.Time(), AuthenticatedAt: f.now.Time(), LastActivityAt: f.now.Time()})
	})
	if err != nil {
		f.t.Fatal(err)
	}
	verifier, _ := webauthn.New("localhost", "http://localhost")
	sessions, err := identityapp.NewService(f.store, f.store, verifier, "localhost", "http://localhost", 2, func() time.Time { return f.now.Time() })
	if err != nil {
		f.t.Fatal(err)
	}
	now := func() calendar.Instant { return f.now }
	commandQueries := commands.NewQueries(f.store, f.store.AuthorizeCommandResult)
	allocationService := allocation.NewService(f.store, now, uuid.NewString)
	matcher := matching.NewService(f.store, f.writer, now, uuid.NewString)
	ledgerService := journal.NewServiceWithAllocations(f.store, matcher, allocationService, now, uuid.NewString)
	refundService := expenses.NewService(f.store, matcher, now, uuid.NewString)
	handler, err := ledgerdelivery.NewWithRefunds(ledgerService, matcher, refundService, journal.NewQueries(f.store), f.executor, commandQueries, sessions, f.store, security.Config{Environment: "test", Origin: "http://localhost"}, now)
	if err != nil {
		f.t.Fatal(err)
	}
	mux := http.NewServeMux()
	mux.Handle("/api/v1/transactions", handler)
	mux.Handle("/api/v1/transactions/", handler)
	mux.Handle("/api/v1/refunds", handler)
	return &client{f: f, handler: mux, token: token}
}

func (c *client) call(method, path, key string, input any, expected int) *httptest.ResponseRecorder {
	c.f.t.Helper()
	var raw []byte
	if input != nil {
		var err error
		raw, err = json.Marshal(input)
		if err != nil {
			c.f.t.Fatal(err)
		}
	}
	req := httptest.NewRequest(method, "http://localhost/api/v1"+path, bytes.NewReader(raw))
	if input != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Origin", "http://localhost")
	req.Header.Set("X-CSRF-Token", c.token.CSRF())
	if key != "" {
		req.Header.Set("Idempotency-Key", key)
	}
	req.AddCookie(&http.Cookie{Name: security.SessionCookie, Value: string(c.token)})
	response := httptest.NewRecorder()
	c.handler.ServeHTTP(response, req)
	if response.Code != expected {
		c.f.t.Fatalf("%s %s => %d want %d: %s", method, path, response.Code, expected, response.Body.String())
	}
	return response
}

func decode[T any](t *testing.T, response *httptest.ResponseRecorder) T {
	t.Helper()
	var value T
	if err := json.Unmarshal(response.Body.Bytes(), &value); err != nil {
		t.Fatal(err)
	}
	return value
}
