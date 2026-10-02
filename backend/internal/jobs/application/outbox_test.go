package application

import (
	"context"
	"testing"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

type waitingOutbox struct{ reviews int }

func (w *waitingOutbox) OutboxEvent(context.Context, household.Principal, string) (Event, error) {
	return Event{ResourceType: "transaction", ResourceID: "transaction-1", Type: "transaction.changed", Revision: 1}, nil
}
func (w *waitingOutbox) EnqueueReview(context.Context, household.Principal, Event) error {
	w.reviews++
	return nil
}

type unavailableValuation struct{}

func (unavailableValuation) PrepareRevision(context.Context, household.Principal, string, uint64) ([]valuation.Snapshot, error) {
	return nil, valuation.ErrRateUnavailable
}
func (unavailableValuation) Save(context.Context, household.Principal, []valuation.Snapshot) error {
	return nil
}

func TestOutboxWaitsForRateWithoutDelayingReview(t *testing.T) {
	repo := &waitingOutbox{}
	result, err := (OutboxHandler{Repository: repo, Valuation: unavailableValuation{}}).Prepare(context.Background(), Execution{Job: jobs.Job{ID: "event-1"}})
	if err != nil || result.State != jobs.Waiting || result.Reason != jobs.GatewayUnavailable || result.Apply == nil {
		t.Fatalf("temporary rate failure outcome: %+v %v", result, err)
	}
	if err := result.Apply(context.Background(), household.Principal{}); err != nil || repo.reviews != 1 {
		t.Fatalf("review not enqueued: %d %v", repo.reviews, err)
	}
}
