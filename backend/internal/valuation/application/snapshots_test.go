package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

type snapshotFact struct{ revision ledger.Revision }

func (s snapshotFact) LedgerRevision(context.Context, household.Principal, string, uint64) (ledger.Revision, error) {
	return s.revision, nil
}
func (snapshotFact) SaveValuationSnapshot(context.Context, household.Principal, valuation.Snapshot) error {
	return nil
}

func TestTemporaryRateFailureDoesNotBecomePinnedSnapshot(t *testing.T) {
	date, _ := calendar.ParseDate("2026-09-01")
	month, _ := calendar.ParseMonth("2026-09")
	at, _ := calendar.ParseInstant("2026-09-01T12:00:00Z")
	amount, _ := money.NewMoney("-10", money.USDC)
	fact := ledger.Revision{OperationID: uuid.NewString(), Revision: 1, ActorID: household.UserID(uuid.NewString()), Reason: "Purchase", Type: ledger.Expense, State: ledger.Posted, OccurredAt: at, CashDate: date, ExpenseMonth: month, PayerState: "unknown", Postings: []ledger.Posting{{AccountID: uuid.NewString(), Money: amount, Role: ledger.Principal}}}
	service := SnapshotPreparer{Rates: Service{Repository: &memoryRates{}, Sources: fixedSources{}, Now: func() time.Time { return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC) }}, Repository: snapshotFact{revision: fact}, Now: func() time.Time { return time.Date(2026, 9, 2, 12, 0, 0, 0, time.UTC) }}
	values, err := service.PrepareRevision(context.Background(), household.Principal{}, fact.OperationID, 1)
	if !errors.Is(err, valuation.ErrRateUnavailable) || len(values) != 0 {
		t.Fatalf("transient failure was pinned: %v %+v", err, values)
	}
}
