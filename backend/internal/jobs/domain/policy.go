package domain

import "time"

type Kind string

const (
	Sync         Kind = "sync"
	Outbox       Kind = "outbox"
	AI           Kind = "ai"
	AIAnswer     Kind = "ai_answer"
	AIValidation Kind = "ai_validation"
)

func (k Kind) Valid() bool {
	return k == Sync || k == Outbox || k == AI || k == AIAnswer || k == AIValidation
}

type State string

const (
	Ready      State = "ready"
	Running    State = "running"
	Waiting    State = "waiting"
	Succeeded  State = "succeeded"
	Failed     State = "failed"
	Canceled   State = "canceled"
	Unresolved State = "unresolved"
)

type Reason string

const (
	HandlerUnavailable  Reason = "handler_unavailable"
	ConsumerUnavailable Reason = "consumer_unavailable"
	GatewayUnavailable  Reason = "gateway_unavailable"
	BudgetWait          Reason = "budget_wait"
	ProviderNotAdmitted Reason = "provider_not_admitted"
	ReauthRequired      Reason = "reauth_required"
	TemporaryFailure    Reason = "temporary_failure"
	PermanentFailure    Reason = "permanent_failure"
	ExternalUnknown     Reason = "external_unknown"
)

func (r Reason) Waiting() bool {
	return r == HandlerUnavailable || r == ConsumerUnavailable || r == GatewayUnavailable || r == BudgetWait || r == ProviderNotAdmitted || r == ReauthRequired
}

type RetryPolicy struct{ Base, Maximum time.Duration }

func DefaultRetryPolicy() RetryPolicy { return RetryPolicy{5 * time.Second, 5 * time.Minute} }

const MaxRetryDelay = 24 * time.Hour

func (p RetryPolicy) Delay(attempt int, jitter float64) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	if jitter < 0 {
		jitter = 0
	}
	if jitter > 1 {
		jitter = 1
	}
	delay := p.Base
	for i := 1; i < attempt && delay < p.Maximum; i++ {
		if delay > p.Maximum/2 {
			delay = p.Maximum
			break
		}
		delay *= 2
	}
	if delay > p.Maximum {
		delay = p.Maximum
	}
	// Jitter only reduces the capped duration, keeping retry bounds strict.
	return time.Duration(float64(delay) * (0.8 + 0.2*jitter))
}

type SyncProgress struct {
	Cursor, Coverage string
	Gaps             []string
	LastSuccessAt    *time.Time
	Completed        bool
}

func (k Kind) ProviderCall() bool { return k == AI || k == AIAnswer }
