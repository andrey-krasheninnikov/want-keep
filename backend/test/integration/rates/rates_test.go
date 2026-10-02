//go:build integration

package rates_test

import (
	"context"
	"net"
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
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
	"github.com/pchkauu/want-keep/backend/migrations"
)

func testDatabase(t *testing.T) (*storage.Store, *pgxpool.Pool) {
	t.Helper()
	raw := os.Getenv("WANT_KEEP_TEST_DATABASE_URL")
	u, err := url.Parse(raw)
	if err != nil || raw == "" || u.Path != "/want_keep_test" || (u.Hostname() != "localhost" && (net.ParseIP(u.Hostname()) == nil || !net.ParseIP(u.Hostname()).IsLoopback())) {
		t.Fatal("rates integration requires an isolated loopback database named want_keep_test")
	}
	ctx := context.Background()
	cluster, err := pgxpool.New(ctx, raw)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(cluster.Close)
	if err = cluster.Ping(ctx); err != nil {
		t.Fatal("isolated PostgreSQL is unavailable")
	}
	_, err = cluster.Exec(ctx, `DO $$ BEGIN IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_app') THEN CREATE ROLE want_keep_app LOGIN PASSWORD 'synthetic-app'; END IF; IF NOT EXISTS(SELECT 1 FROM pg_roles WHERE rolname='want_keep_maintenance') THEN CREATE ROLE want_keep_maintenance LOGIN PASSWORD 'synthetic-maintenance'; END IF; END $$;`)
	if err != nil {
		t.Fatal(err)
	}
	name := "wk_rates_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = cluster.Exec(ctx, `CREATE DATABASE `+name); err != nil {
		t.Fatal(err)
	}
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
	store, err := storage.Open(ctx, storage.Config{DSN: u.String(), Environment: "test", MaxConnections: 8})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(store.Close)
	return store, admin
}

func instant(t *testing.T, value string) calendar.Instant {
	t.Helper()
	at, err := calendar.ParseInstant(value)
	if err != nil {
		t.Fatal(err)
	}
	return at
}

func TestImmutableRatesQuotaAndHouseholdQuotes(t *testing.T) {
	ctx := context.Background()
	store, admin := testDatabase(t)
	date, _ := calendar.ParseDate("2026-09-01")
	rate, _ := money.NewRate(money.USDC, money.USD, "0.999999999999999999999999")
	at := instant(t, "2026-09-01T12:00:00.123456789Z")
	observation := valuation.Observation{ID: uuid.NewString(), ProviderAssetID: "usd-coin", Source: "coingecko", Transport: "demo_history", Revision: 1, RequestedDate: date, EffectiveAt: at, FetchedAt: at, Granularity: "daily", Rate: rate}
	first, err := store.SaveRateObservation(ctx, observation)
	if err != nil {
		t.Fatal(err)
	}
	second, err := store.SaveRateObservation(ctx, observation)
	if err != nil || second.Revision != 2 || second.ID == first.ID {
		t.Fatalf("revision: %+v %v", second, err)
	}
	known, err := store.RateObservations(ctx, money.USDC, money.USD, date, false)
	if err != nil || len(known) != 1 || known[0].ID != second.ID || known[0].Rate.Value() != rate.Value() || known[0].EffectiveAt != at {
		t.Fatalf("round trip: %+v %v", known, err)
	}
	if _, err = admin.Exec(ctx, `UPDATE want_keep.rate_observations SET rate_value=1 WHERE id=$1`, first.ID); err == nil {
		t.Fatal("historical rate changed")
	}
	window := time.Date(2026, 9, 1, 12, 0, 0, 0, time.UTC)
	for i := 0; i < 25; i++ {
		if err = store.ReserveRateCall(ctx, window); err != nil {
			t.Fatal(err)
		}
	}
	if err = store.ReserveRateCall(ctx, window); err != valuation.ErrQuotaExceeded {
		t.Fatalf("quota: %v", err)
	}
	if err = store.ReserveRateCall(ctx, window.Add(time.Minute)); err != nil {
		t.Fatal(err)
	}

	family := household.Household{ID: household.HouseholdID(uuid.NewString()), Name: "Synthetic family"}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Member A"}
	member := household.Membership{ID: household.MembershipID(uuid.NewString()), HouseholdID: family.ID, UserID: user.ID, Active: true}
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	if err = store.InitializeHousehold(ctx, family, []household.User{user}, []household.Membership{member}, zone, 2); err != nil {
		t.Fatal(err)
	}
	p, _ := member.Principal()
	amount, _ := money.NewMoney("100", money.USDT)
	quoteRate, _ := money.NewRate(money.USDT, money.RUB, "80.5")
	fee, _ := money.NewMoney("0.1", money.USDC)
	quote := valuation.PlatformQuote{ID: uuid.NewString(), Provider: "bybit", EvidenceRef: "synthetic-reference-1", Direction: valuation.SellBase, Amount: amount, Rate: quoteRate, Fees: []money.Money{fee}, FeeCoverage: "included", SpreadCoverage: "included", ObservedAt: at, FetchedAt: at}
	if err = store.WithinHousehold(ctx, p, func(ctx context.Context) error { return store.RecordPlatformQuote(ctx, p, quote) }); err != nil {
		t.Fatal(err)
	}
	got, found, err := store.LatestPlatformQuote(ctx, p, "bybit", money.USDT, money.RUB, valuation.SellBase, amount)
	if err != nil || !found || got.ID != quote.ID || len(got.Fees) != 1 || got.Fees[0].Amount() != "0.1" {
		t.Fatalf("quote: %+v %v", got, err)
	}
	other, _ := money.NewMoney("101", money.USDT)
	if _, found, err = store.LatestPlatformQuote(ctx, p, "bybit", money.USDT, money.RUB, valuation.SellBase, other); err != nil || found {
		t.Fatalf("amount-specific quote: %v %v", found, err)
	}
	var count int
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.platform_quotes WHERE household_id=$1`, family.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("quote persisted: %d %v", count, err)
	}
	accountID := uuid.NewString()
	owner, _ := household.NewOwnership(family.ID, household.Shared, "")
	if err = store.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if createErr := store.CreateAccount(ctx, account.Account{ID: accountID, Name: "Cash", Product: "cash", Ownership: owner, Asset: money.USDC, Revision: 1, OpeningDate: date}); createErr != nil {
			return createErr
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		for _, field := range []string{"owned", "available", "locked", "debt"} {
			value := "1000"
			if field == "locked" || field == "debt" {
				value = "0"
			}
			cash, _ := money.NewMoney(value, money.USDC)
			known, _ := reporting.KnownAmount(cash)
			if recordErr := store.RecordBalance(ctx, account.Balance{AccountID: accountID, Field: field, Amount: known, Coverage: coverage, Freshness: reporting.Fresh, ObservedAt: at}); recordErr != nil {
				return recordErr
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	month, _ := calendar.ParseMonth("2026-09")
	spent, _ := money.NewMoney("-10.25", money.USDC)
	operationID := uuid.NewString()
	entry := ledger.Revision{OperationID: operationID, Revision: 1, ActorID: user.ID, Reason: "Synthetic purchase", Type: ledger.Expense, State: ledger.Posted, OccurredAt: at, CashDate: date, ExpenseMonth: month, PayerState: "known", PayerMemberID: member.ID, Postings: []ledger.Posting{{AccountID: accountID, Money: spent, Role: ledger.Principal}}}
	executor := commands.NewExecutor(store, store, func() calendar.Instant { return at })
	_, err = executor.Execute(ctx, p, commands.Request{ID: uuid.NewString(), Kind: "transaction.create", PayloadHash: strings.Repeat("a", 64)}, func(ctx context.Context) (command.Result, error) {
		writeErr := journal.NewWriter(store, store).Append(ctx, p, entry, 0)
		return command.Result{ResourceType: "transaction", ResourceID: operationID, Revision: 1}, writeErr
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = admin.Exec(ctx, `UPDATE want_keep.jobs SET state='waiting',reason='gateway_unavailable',available_at=clock_timestamp()+INTERVAL '1 hour' WHERE household_id=$1 AND kind='outbox'`, family.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.ResumeWaiting(ctx, jobs.Outbox, jobs.GatewayUnavailable); err != nil {
		t.Fatal(err)
	}
	var jobState string
	if err = admin.QueryRow(ctx, `SELECT state FROM want_keep.jobs WHERE household_id=$1 AND kind='outbox'`, family.ID).Scan(&jobState); err != nil || jobState != "waiting" {
		t.Fatalf("early gateway resume: %s %v", jobState, err)
	}
	if _, err = admin.Exec(ctx, `UPDATE want_keep.jobs SET available_at=clock_timestamp()-INTERVAL '1 second' WHERE household_id=$1 AND kind='outbox'`, family.ID); err != nil {
		t.Fatal(err)
	}
	if err = store.ResumeWaiting(ctx, jobs.Outbox, jobs.GatewayUnavailable); err != nil {
		t.Fatal(err)
	}
	if err = admin.QueryRow(ctx, `SELECT state FROM want_keep.jobs WHERE household_id=$1 AND kind='outbox'`, family.ID).Scan(&jobState); err != nil || jobState != "ready" {
		t.Fatalf("due gateway resume: %s %v", jobState, err)
	}
	converted, _, legs, err := valuation.Convert(spent, money.USD, []valuation.Observation{first})
	if err != nil {
		t.Fatal(err)
	}
	snapshot := valuation.Snapshot{OperationID: operationID, OperationRevision: 1, ComponentIndex: 0, ComponentKind: "expense", ValuationRevision: 1, Native: spent, Reporting: &converted, Target: money.USD, RequestedDate: date, CoverageReasons: []string{"historical_fallback"}, Freshness: reporting.Stale, Legs: legs, RecordedAt: at}
	if err = store.WithinHousehold(ctx, p, func(ctx context.Context) error { return store.SaveValuationSnapshot(ctx, p, snapshot) }); err != nil {
		t.Fatal(err)
	}
	loaded, found, err := store.ValuationSnapshot(ctx, p, operationID, 1, 0, money.USD)
	if err != nil || !found || loaded.Reporting == nil || loaded.Reporting.Amount() != converted.Amount() || len(loaded.Legs) != 1 || loaded.Legs[0].ID != first.ID || loaded.Freshness != reporting.Stale || len(loaded.CoverageReasons) != 1 || loaded.CoverageReasons[0] != "historical_fallback" {
		t.Fatalf("pinned snapshot: %+v %v", loaded, err)
	}
	var status string
	if err = admin.QueryRow(ctx, `SELECT status FROM want_keep.valuation_snapshots WHERE household_id=$1`, family.ID).Scan(&status); err != nil || status != "partial" {
		t.Fatalf("partial snapshot: %q %v", status, err)
	}
	if err = store.WithinHousehold(ctx, p, func(ctx context.Context) error { return store.SaveValuationSnapshot(ctx, p, snapshot) }); err != nil {
		t.Fatal(err)
	}
	if err = admin.QueryRow(ctx, `SELECT count(*) FROM want_keep.valuation_snapshots WHERE household_id=$1`, family.ID).Scan(&count); err != nil || count != 1 {
		t.Fatalf("idempotent snapshot: %d %v", count, err)
	}
}
