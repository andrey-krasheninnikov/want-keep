package application

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

var (
	ErrBudgetExhausted     = errors.New("AI budget exhausted")
	ErrBudgetBlocked       = errors.New("AI budget requires reconciliation")
	ErrConcurrency         = errors.New("AI family concurrency exhausted")
	ErrRetryExhausted      = errors.New("AI provider retry exhausted")
	ErrInvalidBudgetQueue  = errors.New("invalid AI budget queue")
	ErrInvalidGatewayQueue = errors.New("invalid AI gateway queue")
)

type Settlement struct {
	Result              ai.Result
	Reservation         ai.Cost
	Actual              *ai.Cost
	Conservative        bool
	NeedsReconciliation bool
	Terminal            bool
}

func (s Settlement) Validate() error {
	if err := s.Result.Validate(); err != nil {
		return err
	}
	if err := s.Reservation.Validate(); err != nil {
		return err
	}
	if s.Actual != nil {
		if err := s.Actual.Validate(); err != nil {
			return err
		}
	}
	if s.Result.State != ai.Unknown && s.Result.State != ai.KnownRejection && s.Actual == nil {
		return ai.ErrInvalidAttempt
	}
	return nil
}

type Repository interface {
	ReviewInput(context.Context, household.Principal, jobs.Job) (json.RawMessage, error)
	StartAIAttempt(context.Context, household.Principal, jobs.Job, ai.Request, string, time.Time) error
	ReserveAIAttempt(context.Context, household.Principal, jobs.Job, string, int64, ai.Cost, time.Time) error
	BeginAIGeneration(context.Context, household.Principal, jobs.Job, string, time.Time) error
	AIOutcomeRetryAllowed(context.Context, household.Principal, jobs.Job) (bool, error)
	SaveAIOutcome(context.Context, household.Principal, jobs.Job, string, Settlement, time.Time) error
	SaveAICompletion(context.Context, household.Principal, jobs.Job, string, Settlement, time.Time) error
	MarkAIUnknown(context.Context, household.Principal, jobs.Job, string, string, ProviderObservation, time.Time) error
}

type BudgetQueueRepository interface {
	ResumeAIBudgetWaiting(context.Context, time.Time) (int64, error)
}

type GatewayQueueRepository interface {
	ResumeAIGatewayWaiting(context.Context) error
}

type WaitingHandler struct{}

func (WaitingHandler) Prepare(context.Context, jobapp.Execution) (jobapp.Result, error) {
	return jobapp.Result{State: jobs.Waiting, Reason: jobs.GatewayUnavailable}, nil
}
