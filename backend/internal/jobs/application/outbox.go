package application

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

type Event struct {
	ResourceType, ResourceID, Type string
	Revision                       uint64
}
type OutboxRepository interface {
	OutboxEvent(context.Context, household.Principal, string) (Event, error)
	EnqueueReview(context.Context, household.Principal, Event) error
}

type ReconciliationDispatcher interface {
	DispatchReplayRevision(context.Context, household.Principal, string, uint64) error
}

type ValuationDispatcher interface {
	PrepareRevision(context.Context, household.Principal, string, uint64) ([]valuation.Snapshot, error)
	Save(context.Context, household.Principal, []valuation.Snapshot) error
}

type OutboxHandler struct {
	Repository     OutboxRepository
	Reconciliation ReconciliationDispatcher
	Valuation      ValuationDispatcher
}

func (h OutboxHandler) Prepare(ctx context.Context, x Execution) (Result, error) {
	event, err := h.Repository.OutboxEvent(ctx, x.Principal, x.Job.ID)
	if err != nil {
		return Result{}, err
	}
	if event.Type == "reconciliation.changed" && event.ResourceType == "reconciliation" {
		if h.Reconciliation == nil {
			return Result{State: jobs.Waiting, Reason: jobs.ConsumerUnavailable}, nil
		}
		if err = h.Reconciliation.DispatchReplayRevision(ctx, x.Principal, event.ResourceID, event.Revision); err != nil {
			return Result{}, err
		}
		return Result{State: jobs.Succeeded}, nil
	}
	if event.Type != "transaction.changed" || event.ResourceType != "transaction" {
		return Result{State: jobs.Waiting, Reason: jobs.ConsumerUnavailable}, nil
	}
	var snapshots []valuation.Snapshot
	if h.Valuation != nil {
		snapshots, err = h.Valuation.PrepareRevision(ctx, x.Principal, event.ResourceID, event.Revision)
		if err != nil {
			return Result{}, err
		}
	}
	return Result{State: jobs.Succeeded, Apply: func(ctx context.Context, p household.Principal) error {
		if err := h.Repository.EnqueueReview(ctx, p, event); err != nil {
			return err
		}
		if h.Valuation != nil {
			return h.Valuation.Save(ctx, p, snapshots)
		}
		return nil
	}}, nil
}
