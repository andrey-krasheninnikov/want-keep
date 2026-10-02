package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Handler struct {
	repository Repository
	gateway    Gateway
	now        func() time.Time
	newID      func() string
}

func NewHandler(repository Repository, gateway Gateway, now func() time.Time, newID func() string) *Handler {
	if now == nil {
		now = time.Now
	}
	h := &Handler{repository: repository, gateway: gateway, now: now, newID: newID}
	return h
}

func (h *Handler) Prepare(ctx context.Context, execution jobapp.Execution) (jobapp.Result, error) {
	job := execution.Job
	if h.repository == nil || h.gateway == nil || h.newID == nil || !job.Kind.ProviderCall() || job.ResourceID == "" || job.ResourceRevision < 1 {
		return jobapp.Result{}, jobs.ErrInvalidJob
	}
	input, err := h.repository.ReviewInput(ctx, execution.Principal, job)
	if err != nil {
		return jobapp.Result{}, err
	}
	contract := h.gateway.Contract()
	request := ai.Request{
		ID: h.newID(), JobID: job.ID, ResourceID: job.ResourceID,
		HouseholdID: job.HouseholdID, ActorID: job.ActorID, ResourceRevision: job.ResourceRevision,
		Purpose: ai.TransactionReview, Model: contract.Model, Qualification: contract.Qualification,
		PromptFingerprint: contract.PromptFingerprint, SchemaFingerprint: contract.SchemaFingerprint,
		ConfigFingerprint: contract.ConfigFingerprint, Input: input, MaximumOutputTokens: 2048,
	}
	if job.Kind == jobs.AIAnswer {
		request.Purpose = ai.ComplexClarification
		request.MaximumOutputTokens = 8192
	}
	if err = request.Validate(); err != nil {
		return jobapp.Result{}, err
	}
	now := h.now().UTC()
	if err = h.repository.StartAIAttempt(ctx, execution.Principal, job, request, requestFingerprint(request), now); err != nil {
		return h.waitOrFail(err)
	}
	counted, err := h.gateway.Count(ctx, request)
	if err != nil {
		return h.handleGatewayFailure(ctx, execution, request.ID, ai.MustCost("0"), false, err)
	}
	reservation, err := ai.TerraPricing().Reservation(counted, request.MaximumOutputTokens)
	if err != nil {
		return h.recordKnownFailure(ctx, execution, request.ID, ai.MustCost("0"), "reservation_invalid", false, 0)
	}
	if err = h.repository.ReserveAIAttempt(ctx, execution.Principal, job, request.ID, counted, reservation, h.now().UTC()); err != nil {
		return h.waitOrFail(err)
	}
	if err = h.repository.BeginAIGeneration(ctx, execution.Principal, job, request.ID, h.now().UTC()); err != nil {
		return h.waitOrFail(err)
	}
	providerResult, err := h.gateway.Generate(ctx, request)
	if err != nil {
		return h.handleGatewayFailure(ctx, execution, request.ID, reservation, true, err)
	}
	if providerResult.Usage.InputTokens < counted || providerResult.Usage.InputTokens > counted+ai.InputReservationMargin {
		return h.markUnknown(execution, request.ID, "input_count_mismatch", ProviderObservation{ID: providerResult.ProviderID, Model: providerResult.ProviderModel, Usage: ObserveUsage(providerResult.Usage)})
	}
	if providerResult.Usage.OutputTokens < 1 || providerResult.Usage.OutputTokens > request.MaximumOutputTokens {
		return h.markUnknown(execution, request.ID, "output_count_mismatch", ProviderObservation{ID: providerResult.ProviderID, Model: providerResult.ProviderModel, Usage: ObserveUsage(providerResult.Usage)})
	}
	actual, conservative, err := ai.TerraPricing().Actual(providerResult.Usage)
	if err != nil {
		return h.markUnknown(execution, request.ID, "usage_invalid", ProviderObservation{ID: providerResult.ProviderID, Model: providerResult.ProviderModel})
	}
	comparison, err := actual.Compare(reservation)
	if err != nil {
		return h.markUnknown(execution, request.ID, "cost_invalid", ProviderObservation{ID: providerResult.ProviderID, Model: providerResult.ProviderModel, Usage: ObserveUsage(providerResult.Usage)})
	}
	settlement := Settlement{Result: providerResult, Reservation: reservation, Actual: &actual, Conservative: conservative, NeedsReconciliation: comparison > 0}
	if providerResult.State == ai.Completed {
		return jobapp.Result{State: jobs.Succeeded, Apply: func(ctx context.Context, _ household.Principal) error {
			return h.repository.SaveAICompletion(ctx, execution.Principal, job, request.ID, settlement, h.now().UTC())
		}}, nil
	}
	return jobapp.Result{State: jobs.Failed, Reason: jobs.PermanentFailure, Apply: func(ctx context.Context, _ household.Principal) error {
		return h.repository.SaveAIOutcome(ctx, execution.Principal, job, request.ID, settlement, h.now().UTC())
	}}, nil
}

func (h *Handler) handleGatewayFailure(ctx context.Context, execution jobapp.Execution, attemptID string, reservation ai.Cost, generation bool, err error) (jobapp.Result, error) {
	var failure GatewayFailure
	if !errors.As(err, &failure) {
		if generation {
			return h.markUnknown(execution, attemptID, "gateway_unknown", ProviderObservation{})
		}
		failure = GatewayFailure{Code: "gateway_unavailable", Retryable: true}
	}
	if failure.OutcomeUnknown {
		return h.markUnknown(execution, attemptID, failure.Code, failure.Observation)
	}
	if generation && !failure.ConfirmedNoCharge {
		return h.markUnknown(execution, attemptID, "provider_charge_unknown", failure.Observation)
	}
	if !generation && failure.Retryable {
		if failure.RetryAfter > 0 {
			return h.recordProviderRetry(execution, attemptID, failure.Code, failure.RetryAfter)
		}
		return h.recordProviderWait(execution, attemptID, failure.Code)
	}
	result := ai.Result{State: ai.KnownRejection, Code: failure.Code}
	return h.recordKnownFailure(ctx, execution, attemptID, reservation, result.Code, failure.Retryable, failure.RetryAfter)
}

func (h *Handler) recordProviderRetry(execution jobapp.Execution, attemptID, code string, retryAfter time.Duration) (jobapp.Result, error) {
	retryAfter = min(max(retryAfter, 0), jobs.DefaultRetryPolicy().Maximum)
	settlement := Settlement{Result: ai.Result{State: ai.KnownRejection, Code: code}, Reservation: ai.MustCost("0")}
	return jobapp.Result{State: jobs.Waiting, Reason: jobs.GatewayUnavailable, MinimumDelay: retryAfter, Apply: func(ctx context.Context, _ household.Principal) error {
		return h.repository.SaveAIOutcome(ctx, execution.Principal, execution.Job, attemptID, settlement, h.now().UTC())
	}}, nil
}

func (h *Handler) recordProviderWait(execution jobapp.Execution, attemptID, code string) (jobapp.Result, error) {
	settlement := Settlement{Result: ai.Result{State: ai.KnownRejection, Code: code}, Reservation: ai.MustCost("0")}
	return jobapp.Result{State: jobs.Waiting, Reason: jobs.GatewayUnavailable, MinimumDelay: jobs.DefaultRetryPolicy().Maximum, Apply: func(ctx context.Context, _ household.Principal) error {
		return h.repository.SaveAIOutcome(ctx, execution.Principal, execution.Job, attemptID, settlement, h.now().UTC())
	}}, nil
}

func (h *Handler) recordKnownFailure(ctx context.Context, execution jobapp.Execution, attemptID string, reservation ai.Cost, code string, retryable bool, retryAfter time.Duration) (jobapp.Result, error) {
	retryAfter = min(max(retryAfter, 0), jobs.DefaultRetryPolicy().Maximum)
	settlement := Settlement{Result: ai.Result{State: ai.KnownRejection, Code: code}, Reservation: reservation}
	canRetry := false
	if retryable {
		var err error
		canRetry, err = h.repository.AIOutcomeRetryAllowed(ctx, execution.Principal, execution.Job)
		if err != nil {
			return jobapp.Result{}, err
		}
	}
	result := jobapp.Result{State: jobs.Failed, Reason: jobs.PermanentFailure}
	if retryable && canRetry {
		result.State, result.Reason = jobs.Ready, jobs.TemporaryFailure
		result.MinimumDelay = retryAfter
	}
	settlement.Terminal = result.State == jobs.Failed
	result.Apply = func(ctx context.Context, _ household.Principal) error {
		return h.repository.SaveAIOutcome(ctx, execution.Principal, execution.Job, attemptID, settlement, h.now().UTC())
	}
	return result, nil
}

func (h *Handler) markUnknown(execution jobapp.Execution, attemptID, code string, observation ProviderObservation) (jobapp.Result, error) {
	return jobapp.Result{State: jobs.Unresolved, Reason: jobs.ExternalUnknown, Apply: func(ctx context.Context, _ household.Principal) error {
		return h.repository.MarkAIUnknown(ctx, execution.Principal, execution.Job, attemptID, code, observation, h.now().UTC())
	}}, nil
}

func (h *Handler) waitOrFail(err error) (jobapp.Result, error) {
	switch {
	case errors.Is(err, ErrBudgetExhausted), errors.Is(err, ErrBudgetBlocked), errors.Is(err, ErrConcurrency):
		return jobapp.Result{State: jobs.Waiting, Reason: jobs.BudgetWait}, nil
	case errors.Is(err, ErrRetryExhausted), errors.Is(err, ai.ErrInvalidAttempt):
		return jobapp.Result{State: jobs.Failed, Reason: jobs.PermanentFailure}, nil
	default:
		return jobapp.Result{}, err
	}
}

func requestFingerprint(request ai.Request) string {
	encoded, _ := json.Marshal(struct {
		Purpose, Model, Qualification, Prompt, Schema, Config string
		ResourceID                                            string
		ResourceRevision                                      uint64
		Input                                                 json.RawMessage
	}{string(request.Purpose), string(request.Model), string(request.Qualification), request.PromptFingerprint, request.SchemaFingerprint, request.ConfigFingerprint, request.ResourceID, request.ResourceRevision, request.Input})
	sum := sha256.Sum256(encoded)
	return hex.EncodeToString(sum[:])
}
