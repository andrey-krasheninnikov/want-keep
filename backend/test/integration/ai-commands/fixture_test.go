//go:build integration

package aicommands_test

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
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
		fmt.Fprintln(os.Stderr, "Transaction review integration requires an isolated loopback PostgreSQL database named want_keep_test via WANT_KEEP_TEST_DATABASE_URL.")
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
	t          *testing.T
	store      *storage.Store
	admin      *pgxpool.Pool
	dsn        string
	family     household.Household
	p          household.Principal
	membership household.Membership
	writer     *journal.Writer
	executor   *commands.Executor
	now        calendar.Instant
}

func newFixture(t *testing.T) *fixture {
	t.Helper()
	name := "wk_test_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := cluster.Exec(testContext, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
	u := *databaseURL
	u.Path = "/" + name
	admin, err := pgxpool.New(testContext, u.String())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { admin.Close(); _, _ = cluster.Exec(testContext, `DROP DATABASE `+name) })
	if err = storage.Migrate(testContext, admin, migrations.Files); err != nil {
		t.Fatal(err)
	}
	u.User = url.UserPassword("want_keep_app", "synthetic-app")
	store, err := storage.Open(testContext, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 12})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	f := &fixture{
		t: t, store: store, admin: admin, dsn: u.String(),
		family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic household"},
		now:    mustInstant("2026-09-30T23:59:00.123456789Z"),
	}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Member A"}
	f.membership = household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: f.family.ID, UserID: user.ID, Active: true}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = store.InitializeHousehold(testContext, f.family, []household.User{user}, []household.Membership{f.membership}, zone, 2); err != nil {
		t.Fatal(err)
	}
	f.p, _ = f.membership.Principal()
	f.writer = journal.NewWriter(store, store)
	f.executor = commands.NewExecutor(store, store, func() calendar.Instant { return f.now })
	return f
}

func (f *fixture) anotherHousehold() *fixture {
	f.t.Helper()
	other := &fixture{
		t: f.t, store: f.store, admin: f.admin, dsn: f.dsn,
		family: household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Other synthetic household"}, now: f.now,
	}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Other member"}
	other.membership = household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: other.family.ID, UserID: user.ID, Active: true}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err := f.store.InitializeHousehold(testContext, other.family, []household.User{user}, []household.Membership{other.membership}, zone, 2); err != nil {
		f.t.Fatal(err)
	}
	other.p, _ = other.membership.Principal()
	other.writer = journal.NewWriter(other.store, other.store)
	other.executor = commands.NewExecutor(other.store, other.store, func() calendar.Instant { return other.now })
	return other
}

func (f *fixture) enqueueReviewJobs(count int) {
	f.t.Helper()
	accountID := f.account("100000")
	for range count {
		r := f.revision(uuid.NewString(), accountID)
		request := commands.Request{ID: uuid.NewString(), Kind: "transaction.create", PayloadHash: strings.Repeat("a", 64)}
		if _, err := f.executor.Execute(testContext, f.p, request, func(ctx context.Context) (command.Result, error) {
			err := f.writer.Append(ctx, f.p, r, 0)
			return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: 1}, err
		}); err != nil {
			f.t.Fatal(err)
		}
	}
	outbox := jobapp.Worker{Repository: f.store, Handler: jobapp.OutboxHandler{Repository: f.store}, Config: jobapp.DefaultWorkerConfig(jobs.Outbox)}
	for range count {
		if err := outbox.Step(testContext); err != nil {
			f.t.Fatal(err)
		}
	}
}

func (f *fixture) account(balance string) string {
	f.t.Helper()
	id := uuid.NewString()
	ownership, _ := household.NewOwnership(f.family.ID, household.Shared, "")
	date, _ := calendar.ParseDate("2026-09-01")
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		if err := f.store.CreateAccount(ctx, account.Account{ID: id, Name: "Synthetic cash", Product: "cash", Ownership: ownership, Asset: money.RUB, Revision: 1, OpeningDate: date}); err != nil {
			return err
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, []string{})
		for _, field := range []string{"owned", "available", "locked", "debt"} {
			value := balance
			if field == "locked" || field == "debt" {
				value = "0"
			}
			amount, _ := reporting.KnownAmount(mustMoney(value))
			if err := f.store.RecordBalance(ctx, account.Balance{AccountID: id, Field: field, Amount: amount, Coverage: coverage, Freshness: reporting.Fresh, ObservedAt: f.now}); err != nil {
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

func (f *fixture) revision(id, accountID string) ledger.Revision {
	date, _ := calendar.ParseDate("2026-09-30")
	month, _ := calendar.ParseMonth("2026-09")
	return ledger.Revision{
		OperationID: id, Revision: 1, ActorID: f.p.UserID(), Reason: "Synthetic expense",
		Type: "expense", State: "posted", OccurredAt: f.now, CashDate: date, ExpenseMonth: month,
		PayerState: "known", PayerMemberID: f.membership.ID,
		Postings: []ledger.Posting{{AccountID: accountID, Money: mustMoney("-1"), Role: "principal"}},
	}
}

func (f *fixture) maintenancePool() *pgxpool.Pool {
	f.t.Helper()
	u, _ := url.Parse(f.dsn)
	u.User = url.UserPassword("want_keep_maintenance", "synthetic-maintenance")
	pool, err := pgxpool.New(testContext, u.String())
	if err != nil {
		f.t.Fatal(err)
	}
	f.t.Cleanup(pool.Close)
	return pool
}

func mustMoney(value string) money.Money {
	amount, err := money.NewMoney(value, money.RUB)
	if err != nil {
		panic(err)
	}
	return amount
}

func mustInstant(value string) calendar.Instant {
	instant, err := calendar.ParseInstant(value)
	if err != nil {
		panic(err)
	}
	return instant
}
