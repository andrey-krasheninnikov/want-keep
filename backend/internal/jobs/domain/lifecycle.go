package domain

import "time"

const (
	DeadlineExceeded  Reason = "deadline_exceeded"
	AttemptsExhausted Reason = "attempts_exhausted"
	Cancellation      Reason = "canceled"
	MembershipRevoked Reason = "membership_revoked"
)

type Outcome struct {
	State   State
	Reason  Reason
	Attempt int
}

// Complete determines the result of a fenced execution attempt.
func (j Job) Complete(state State, reason Reason) (Outcome, error) {
	if state != Ready && state != Waiting && state != Succeeded && state != Failed && state != Unresolved {
		return Outcome{}, ErrInvalidJob
	}
	if state == Waiting && !reason.Waiting() {
		return Outcome{}, ErrInvalidJob
	}
	out := Outcome{state, reason, j.Attempt}
	switch {
	case state == Succeeded:
		out.Reason = ""
	case j.ExternalStarted || state == Unresolved:
		out.State, out.Reason = Unresolved, ExternalUnknown
	case j.CancelRequested:
		out.State, out.Reason = Canceled, Cancellation
	case state == Ready && j.Attempt >= j.MaxAttempts:
		out.State, out.Reason = Failed, AttemptsExhausted
	case state == Waiting:
		if j.State == Running && out.Attempt > 0 {
			out.Attempt--
		}
	case state == Ready && reason == "":
		out.Reason = TemporaryFailure
	case state == Failed && reason == "":
		out.Reason = PermanentFailure
	}
	return out, nil
}

type Resolution string

const (
	EffectConfirmed Resolution = "confirmed"
	PageConfirmed   Resolution = "page_confirmed"
	EffectAbsent    Resolution = "absent"
)

// Cancel retains uncertainty until trusted reconciliation resolves the effect.
func (j Job) Cancel() Outcome {
	out := Outcome{j.State, j.Reason, j.Attempt}
	switch {
	case j.ExternalStarted || j.State == Unresolved:
		out.State, out.Reason = Unresolved, ExternalUnknown
	case j.State == Ready || j.State == Running || j.State == Waiting:
		out.State, out.Reason = Canceled, Cancellation
	}
	return out
}

// Reconcile is called only after trusted evidence has resolved the external action.
func (j Job) Reconcile(resolution Resolution, replacementExists bool) (Outcome, error) {
	if resolution != EffectConfirmed && resolution != PageConfirmed && resolution != EffectAbsent {
		return Outcome{}, ErrInvalidJob
	}
	j.ExternalStarted = false
	j.State = Ready
	state := Succeeded
	if resolution != EffectConfirmed {
		state = Ready
	}
	out, err := j.Complete(state, "")
	if err == nil && resolution == EffectAbsent && j.Kind == Sync && replacementExists && out.State == Ready {
		out = j.Cancel()
	}
	return out, err
}

// Recover retains unstarted dependency work even if the worker was offline past its deadline.
func (j Job) Recover(now time.Time, membershipActive bool, dependency Reason) (Outcome, bool) {
	unchanged := Outcome{j.State, j.Reason, j.Attempt}
	if j.State != Ready && j.State != Running {
		return unchanged, false
	}
	if j.State == Running && now.Before(j.LeaseUntil) && now.Before(j.Deadline) && !j.CancelRequested {
		return unchanged, false
	}
	out := Outcome{Failed, "", j.Attempt}
	switch {
	case j.ExternalStarted:
		out.State, out.Reason = Unresolved, ExternalUnknown
	case j.CancelRequested:
		out.State, out.Reason = Canceled, Cancellation
	case !membershipActive:
		out.Reason = MembershipRevoked
	case j.State == Ready && j.Attempt == 0 && membershipActive && dependency.Waiting():
		out.State, out.Reason = Waiting, dependency
	case !now.Before(j.Deadline):
		out.Reason = DeadlineExceeded
	case j.Kind.ProviderCall() && j.State == Running && !j.ExternalStarted:
		out.State, out.Reason = Ready, TemporaryFailure
		if out.Attempt > 0 {
			out.Attempt--
		}
	case j.Attempt >= j.MaxAttempts:
		out.Reason = AttemptsExhausted
	case dependency.Waiting():
		out.State, out.Reason = Waiting, dependency
	case j.State == Running:
		out.State, out.Reason = Ready, TemporaryFailure
	default:
		return unchanged, false
	}
	return out, true
}
