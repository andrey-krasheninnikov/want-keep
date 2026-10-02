package storage

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type aiReviewPosting struct {
	AccountID string `json:"account_id"`
	Amount    string `json:"amount"`
	Asset     string `json:"asset"`
	Role      string `json:"role"`
	Funding   string `json:"funding"`
	Treatment string `json:"treatment"`
}

type aiReviewItem struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Quantity string `json:"quantity"`
	Gross    string `json:"gross"`
	Discount string `json:"discount"`
	Net      string `json:"net"`
	Asset    string `json:"asset"`
}

type aiReviewInput struct {
	EconomicType   string            `json:"economic_type"`
	BankState      string            `json:"bank_state"`
	OccurredAt     string            `json:"occurred_at"`
	PostedAt       string            `json:"posted_at,omitempty"`
	Revision       uint64            `json:"revision"`
	Timezone       string            `json:"timezone"`
	CashDate       string            `json:"cash_date"`
	ExpenseMonth   string            `json:"expense_month,omitempty"`
	Merchant       string            `json:"merchant,omitempty"`
	Note           string            `json:"note,omitempty"`
	PayerState     string            `json:"payer_state"`
	SourceConflict bool              `json:"source_conflict"`
	Postings       []aiReviewPosting `json:"postings"`
	ReceiptItems   []aiReviewItem    `json:"receipt_items"`
}

type aiReviewCase struct {
	ID      string   `json:"id"`
	Source  string   `json:"source"`
	Members []string `json:"members"`
	ActorID string   `json:"actor_id"`
	Text    string   `json:"text"`
}

// ReviewInput is the only storage-to-provider projection. It excludes credentials,
// sessions, connection secrets, source payloads and unrelated household data.
func (s *Store) buildReviewInput(ctx context.Context, p household.Principal, job jobs.Job) (json.RawMessage, error) {
	if job.HouseholdID != p.HouseholdID() || job.ActorID != p.UserID() || !job.Kind.ProviderCall() {
		return nil, household.ErrForbidden
	}
	revision, err := s.LedgerRevision(ctx, p, job.ResourceID, job.ResourceRevision)
	if err != nil {
		return nil, err
	}
	input := aiReviewInput{
		Revision: revision.Revision, EconomicType: string(revision.Type), BankState: string(revision.State),
		OccurredAt: revision.OccurredAt.String(), PostedAt: revision.PostedAt.String(),
		Timezone: revision.Timezone.String(), CashDate: revision.CashDate.String(), ExpenseMonth: revision.ExpenseMonth.String(),
		Merchant: revision.Merchant, Note: revision.Note, PayerState: revision.PayerState, SourceConflict: revision.SourceConflict,
		Postings: []aiReviewPosting{}, ReceiptItems: []aiReviewItem{},
	}
	accountLabels := map[string]string{}
	for _, posting := range revision.Postings {
		label, exists := accountLabels[posting.AccountID]
		if !exists {
			label = "account-" + strconv.Itoa(len(accountLabels)+1)
			accountLabels[posting.AccountID] = label
		}
		input.Postings = append(input.Postings, aiReviewPosting{
			AccountID: label, Amount: posting.Money.Amount(), Asset: string(posting.Money.Asset()),
			Role: string(posting.Role), Funding: string(posting.Funding), Treatment: string(posting.Treatment),
		})
	}
	for index, item := range revision.ReceiptItems {
		net, err := item.Net()
		if err != nil {
			return nil, err
		}
		input.ReceiptItems = append(input.ReceiptItems, aiReviewItem{
			ID: "item-" + strconv.Itoa(index+1), Name: item.Name, Quantity: item.Quantity, Gross: item.Gross.Amount(),
			Discount: item.Discount.Amount(), Net: net.Amount(), Asset: string(item.Gross.Asset()),
		})
	}
	text, err := json.Marshal(input)
	if err != nil {
		return nil, err
	}
	return json.Marshal([]aiReviewCase{{ID: "case-1", Source: "ledger_revision", Members: []string{"actor"}, ActorID: "actor", Text: string(text)}})
}

func (s *Store) StartAIAttempt(ctx context.Context, p household.Principal, job jobs.Job, request ai.Request, requestFingerprint string, now time.Time) error {
	if err := request.Validate(); err != nil || len(requestFingerprint) != 64 || request.HouseholdID != p.HouseholdID() || request.ActorID != p.UserID() || request.JobID != job.ID || request.ResourceID != job.ResourceID || request.ResourceRevision != job.ResourceRevision {
		return ai.ErrInvalidAttempt
	}
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, err := s.FenceJob(ctx, p, job); err != nil {
			return err
		}
		scope, _ := s.familyScope(ctx)
		if err := s.releaseSafeInterruptedAttempts(ctx, scope, p, job, now); err != nil {
			return err
		}
		blocked, active, err := s.aiBudgetGate(ctx, scope.tx, p.HouseholdID())
		if err != nil {
			return err
		}
		if blocked {
			return aiapp.ErrBudgetBlocked
		}
		if active >= 2 {
			return aiapp.ErrConcurrency
		}
		at, ns := splitAICallTime(now)
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ai_attempts(household_id,id,job_id,actor_id,resource_id,resource_revision,purpose,model,qualification,request_fingerprint,prompt_fingerprint,schema_fingerprint,config_fingerprint,allowed_input,maximum_output_tokens,created_at,created_ns) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17)`, p.HouseholdID(), request.ID, job.ID, p.UserID(), request.ResourceID, request.ResourceRevision, request.Purpose, request.Model, request.Qualification, requestFingerprint, request.PromptFingerprint, request.SchemaFingerprint, request.ConfigFingerprint, request.Input, request.MaximumOutputTokens, at, ns)
		if err != nil {
			return err
		}
		return s.insertAIState(ctx, scope.tx, p.HouseholdID(), request.ID, aiState{State: ai.Counting, RecordedAt: now})
	})
}

func (s *Store) ReserveAIAttempt(ctx context.Context, p household.Principal, job jobs.Job, attemptID string, counted int64, reservation ai.Cost, now time.Time) error {
	if counted < 1 || counted > 262144 || reservation.Validate() != nil {
		return ai.ErrInvalidAttempt
	}
	var gateErr error
	err := s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, err := s.FenceJob(ctx, p, job); err != nil {
			return err
		}
		scope, _ := s.familyScope(ctx)
		current, err := s.lockAIState(ctx, scope.tx, p.HouseholdID(), attemptID)
		if err != nil || current.State != ai.Counting {
			return errors.Join(err, ai.ErrInvalidAttempt)
		}
		budgetMonth := startOfUTCMonth(now)
		current.BudgetMonth = &budgetMonth
		blocked, _, err := s.aiBudgetGate(ctx, scope.tx, p.HouseholdID())
		if err != nil {
			return err
		}
		if blocked {
			gateErr = aiapp.ErrBudgetBlocked
			return s.refuseAIReservation(ctx, scope.tx, p.HouseholdID(), attemptID, current, reservation, "budget_blocked", now)
		}
		var generationAttempts int
		if err = scope.tx.QueryRow(ctx, `SELECT count(*) FROM want_keep.ai_attempts a WHERE a.household_id=$1 AND a.job_id=$2 AND EXISTS(SELECT 1 FROM want_keep.ai_attempt_states s WHERE s.household_id=a.household_id AND s.attempt_id=a.id AND s.external_started)`, p.HouseholdID(), job.ID).Scan(&generationAttempts); err != nil {
			return err
		}
		if generationAttempts >= 2 {
			gateErr = aiapp.ErrRetryExhausted
			return s.refuseAIReservation(ctx, scope.tx, p.HouseholdID(), attemptID, current, reservation, "retry_exhausted", now)
		}
		used, err := s.aiMonthUsage(ctx, scope.tx, p.HouseholdID(), now)
		if err != nil {
			return err
		}
		total, err := used.Add(reservation)
		if err != nil {
			return err
		}
		if comparison, _ := total.Compare(ai.MustCost("50")); comparison > 0 {
			gateErr = aiapp.ErrBudgetExhausted
			return s.refuseAIReservation(ctx, scope.tx, p.HouseholdID(), attemptID, current, reservation, "budget_exhausted", now)
		}
		return s.insertAIState(ctx, scope.tx, p.HouseholdID(), attemptID, aiState{Revision: current.Revision + 1, State: ai.Reserved, Counted: &counted, Reservation: &reservation, BudgetMonth: &budgetMonth, RecordedAt: now})
	})
	if err != nil {
		return err
	}
	return gateErr
}

func (s *Store) BeginAIGeneration(ctx context.Context, p household.Principal, job jobs.Job, attemptID string, now time.Time) error {
	var gateErr error
	err := s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, err := s.FenceJob(ctx, p, job); err != nil {
			return err
		}
		scope, _ := s.familyScope(ctx)
		current, err := s.lockAIState(ctx, scope.tx, p.HouseholdID(), attemptID)
		if err != nil || current.State != ai.Reserved || current.ExternalStarted || current.Reservation == nil || current.BudgetMonth == nil {
			return errors.Join(err, ai.ErrInvalidAttempt)
		}
		blocked, _, err := s.aiBudgetGate(ctx, scope.tx, p.HouseholdID())
		if err != nil {
			return err
		}
		if blocked {
			gateErr = aiapp.ErrBudgetBlocked
			return s.refuseAIReservation(ctx, scope.tx, p.HouseholdID(), attemptID, current, *current.Reservation, "budget_blocked", now)
		}
		generationMonth := startOfUTCMonth(now)
		used, usageErr := s.aiMonthUsage(ctx, scope.tx, p.HouseholdID(), now)
		if usageErr != nil {
			return usageErr
		}
		total := used
		if !current.BudgetMonth.Equal(generationMonth) {
			moved, addErr := used.Add(*current.Reservation)
			if addErr != nil {
				return addErr
			}
			total = moved
			current.BudgetMonth = &generationMonth
		}
		if comparison, _ := total.Compare(ai.MustCost("50")); comparison > 0 {
			gateErr = aiapp.ErrBudgetExhausted
			return s.refuseAIReservation(ctx, scope.tx, p.HouseholdID(), attemptID, current, *current.Reservation, "budget_exhausted", now)
		}
		current.Revision++
		current.ExternalStarted = true
		current.RecordedAt = now
		if err = s.insertAIState(ctx, scope.tx, p.HouseholdID(), attemptID, current); err != nil {
			return err
		}
		tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET external_started=true WHERE household_id=$1 AND id=$2 AND state='running' AND lease_token=$3 AND attempt=$4`, p.HouseholdID(), job.ID, job.LeaseToken, job.Attempt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
		return nil
	})
	if err != nil {
		return err
	}
	return gateErr
}

func (s *Store) AIOutcomeRetryAllowed(ctx context.Context, p household.Principal, job jobs.Job) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var retryableRejections int
	err = q.QueryRow(ctx, `SELECT count(*) FROM want_keep.ai_attempts a JOIN LATERAL(SELECT state,code,external_started FROM want_keep.ai_attempt_states s WHERE (s.household_id,s.attempt_id)=(a.household_id,a.id) ORDER BY revision DESC LIMIT 1) latest ON true WHERE a.household_id=$1 AND a.job_id=$2 AND latest.external_started AND latest.state='known_rejection' AND latest.code='provider_rate_limited'`, p.HouseholdID(), job.ID).Scan(&retryableRejections)
	return retryableRejections == 0, err
}

func (s *Store) SaveAIOutcome(ctx context.Context, p household.Principal, job jobs.Job, attemptID string, settlement aiapp.Settlement, now time.Time) error {
	if err := settlement.Validate(); err != nil {
		return err
	}
	if _, err := s.FenceJob(ctx, p, job); err != nil {
		return err
	}
	scope, _ := s.familyScope(ctx)
	current, err := s.lockAIState(ctx, scope.tx, p.HouseholdID(), attemptID)
	if err != nil || current.State.Terminal() {
		return errors.Join(err, ai.ErrInvalidAttempt)
	}
	if err = s.writeAISettlement(ctx, scope.tx, p.HouseholdID(), attemptID, current, settlement, now); err != nil {
		return err
	}
	if settlement.Result.State == ai.Refused || settlement.Result.State == ai.Incomplete || settlement.Result.State == ai.SchemaError || settlement.Terminal {
		if err = s.enqueueAIValidation(ctx, p, job, attemptID); err != nil {
			return err
		}
	}
	if !current.ExternalStarted {
		return nil
	}
	return s.clearAIJobExternal(ctx, scope.tx, p, job)
}

func (s *Store) clearAIJobExternal(ctx context.Context, tx pgx.Tx, p household.Principal, job jobs.Job) error {
	tag, err := tx.Exec(ctx, `UPDATE want_keep.jobs SET external_started=false WHERE household_id=$1 AND id=$2 AND state='running' AND lease_token=$3 AND attempt=$4`, p.HouseholdID(), job.ID, job.LeaseToken, job.Attempt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return jobs.ErrStaleAttempt
	}
	return nil
}

func (s *Store) SaveAICompletion(ctx context.Context, p household.Principal, job jobs.Job, attemptID string, settlement aiapp.Settlement, now time.Time) error {
	if err := settlement.Validate(); err != nil || settlement.Result.State != ai.Completed {
		return ai.ErrInvalidAttempt
	}
	if _, err := s.FenceJob(ctx, p, job); err != nil {
		return err
	}
	scope, _ := s.familyScope(ctx)
	current, err := s.lockAIState(ctx, scope.tx, p.HouseholdID(), attemptID)
	if err != nil || current.State != ai.Reserved || !current.ExternalStarted {
		return errors.Join(err, ai.ErrInvalidAttempt)
	}
	if err = s.writeAISettlement(ctx, scope.tx, p.HouseholdID(), attemptID, current, settlement, now); err != nil {
		return err
	}
	if err = s.enqueueAIValidation(ctx, p, job, attemptID); err != nil {
		return err
	}
	return s.clearAIJobExternal(ctx, scope.tx, p, job)
}

func (s *Store) MarkAIUnknown(ctx context.Context, p household.Principal, job jobs.Job, attemptID, code string, observation aiapp.ProviderObservation, now time.Time) error {
	if code == "" || len(code) > 100 || len(observation.ID) > 200 || len(observation.Model) > 200 {
		return ai.ErrInvalidAttempt
	}
	if observation.Usage != nil && observation.Usage.Validate() != nil {
		return ai.ErrInvalidAttempt
	}
	if _, err := s.FenceJob(ctx, p, job); err != nil {
		return err
	}
	scope, _ := s.familyScope(ctx)
	current, err := s.lockAIState(ctx, scope.tx, p.HouseholdID(), attemptID)
	if err != nil || current.State.Terminal() {
		return errors.Join(err, ai.ErrInvalidAttempt)
	}
	current.Revision++
	current.State, current.Code = ai.Unknown, code
	current.Reconciliation = "pending"
	current.ProviderID, current.ProviderModel, current.ObservedUsage = observation.ID, observation.Model, observation.Usage
	current.RecordedAt = now
	return s.insertAIState(ctx, scope.tx, p.HouseholdID(), attemptID, current)
}

type aiState struct {
	Revision        uint64
	State           ai.State
	Counted         *int64
	Reservation     *ai.Cost
	BudgetMonth     *time.Time
	ExternalStarted bool
	ProviderID      string
	ProviderModel   string
	Usage           *ai.Usage
	ObservedUsage   *aiapp.ObservedUsage
	Actual          *ai.Cost
	Conservative    bool
	Output          json.RawMessage
	Validation      string
	Code            string
	Reconciliation  string
	EvidenceRef     string
	RecordedAt      time.Time
}

func (s *Store) lockAIState(ctx context.Context, tx pgx.Tx, family household.HouseholdID, id string) (aiState, error) {
	return scanAIState(tx.QueryRow(ctx, `SELECT revision,state,counted_input_tokens,reservation_usd::text,budget_month,external_started,COALESCE(provider_id,''),COALESCE(provider_model,''),input_tokens,cached_tokens,cache_write_tokens,output_tokens,reasoning_tokens,observed_usage,actual_usd::text,conservative_cost,structured_output,validation_state,code,reconciliation_state,COALESCE(evidence_ref,''),recorded_at,recorded_ns FROM want_keep.ai_attempt_states WHERE household_id=$1 AND attempt_id=$2 ORDER BY revision DESC LIMIT 1`, family, id))
}

func scanAIState(row pgx.Row) (aiState, error) {
	var state aiState
	var reservation, actual *string
	var input, cached, writes, output, reasoning *int64
	var observed []byte
	var at time.Time
	var ns int16
	err := row.Scan(&state.Revision, &state.State, &state.Counted, &reservation, &state.BudgetMonth, &state.ExternalStarted, &state.ProviderID, &state.ProviderModel, &input, &cached, &writes, &output, &reasoning, &observed, &actual, &state.Conservative, &state.Output, &state.Validation, &state.Code, &state.Reconciliation, &state.EvidenceRef, &at, &ns)
	if err != nil {
		return state, err
	}
	state.RecordedAt = at.UTC().Add(time.Duration(ns))
	if reservation != nil {
		value, err := ai.NewCost(*reservation)
		if err != nil {
			return state, err
		}
		state.Reservation = &value
	}
	if actual != nil {
		value, err := ai.NewCost(*actual)
		if err != nil {
			return state, err
		}
		state.Actual = &value
	}
	if input != nil || cached != nil || writes != nil || output != nil || reasoning != nil {
		if input == nil || cached == nil || output == nil || reasoning == nil {
			return state, ai.ErrInvalidAttempt
		}
		state.Usage = &ai.Usage{InputTokens: *input, CachedTokens: *cached, CacheWriteTokens: writes, OutputTokens: *output, ReasoningTokens: *reasoning}
	}
	if len(observed) > 0 {
		var usage aiapp.ObservedUsage
		if len(observed) > 2048 || json.Unmarshal(observed, &usage) != nil || usage.Validate() != nil {
			return state, ai.ErrInvalidAttempt
		}
		state.ObservedUsage = &usage
	}
	return state, nil
}

func (s *Store) insertAIState(ctx context.Context, tx pgx.Tx, family household.HouseholdID, attemptID string, state aiState) error {
	if state.Revision == 0 {
		state.Revision = 1
	}
	at, ns := splitAICallTime(state.RecordedAt)
	var reservation, actual any
	if state.Reservation != nil {
		reservation = state.Reservation.String()
	}
	if state.Actual != nil {
		actual = state.Actual.String()
	}
	var input, cached, writes, output, reasoning any
	if state.Usage != nil {
		input, cached, output, reasoning = state.Usage.InputTokens, state.Usage.CachedTokens, state.Usage.OutputTokens, state.Usage.ReasoningTokens
		if state.Usage.CacheWriteTokens != nil {
			writes = *state.Usage.CacheWriteTokens
		}
	}
	var structured any
	if len(state.Output) > 0 {
		structured = state.Output
	}
	var observed any
	if state.ObservedUsage != nil {
		if state.ObservedUsage.Validate() != nil {
			return ai.ErrInvalidAttempt
		}
		encoded, err := json.Marshal(state.ObservedUsage)
		if err != nil || len(encoded) > 2048 {
			return ai.ErrInvalidAttempt
		}
		observed = encoded
	}
	_, err := tx.Exec(ctx, `INSERT INTO want_keep.ai_attempt_states(household_id,attempt_id,revision,state,counted_input_tokens,reservation_usd,budget_month,external_started,provider_id,provider_model,input_tokens,cached_tokens,cache_write_tokens,output_tokens,reasoning_tokens,observed_usage,actual_usd,conservative_cost,structured_output,validation_state,code,reconciliation_state,evidence_ref,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6::numeric,$7,$8,NULLIF($9,''),NULLIF($10,''),$11,$12,$13,$14,$15,$16,$17::numeric,$18,$19,$20,$21,$22,NULLIF($23,''),$24,$25)`, family, attemptID, state.Revision, state.State, state.Counted, reservation, state.BudgetMonth, state.ExternalStarted, state.ProviderID, state.ProviderModel, input, cached, writes, output, reasoning, observed, actual, state.Conservative, structured, state.Validation, state.Code, defaultReconciliation(state.Reconciliation), state.EvidenceRef, at, ns)
	return err
}

func defaultReconciliation(value string) string {
	if value == "" {
		return "none"
	}
	return value
}

func splitAICallTime(value time.Time) (time.Time, int16) {
	value = value.UTC()
	return value.Truncate(time.Microsecond), int16(value.Nanosecond() % 1000)
}

func startOfUTCMonth(value time.Time) time.Time {
	value = value.UTC()
	return time.Date(value.Year(), value.Month(), 1, 0, 0, 0, 0, time.UTC)
}

func (s *Store) writeAISettlement(ctx context.Context, tx pgx.Tx, family household.HouseholdID, attemptID string, current aiState, settlement aiapp.Settlement, now time.Time) error {
	next := aiState{
		Revision: current.Revision + 1, State: settlement.Result.State, Counted: current.Counted,
		Reservation: &settlement.Reservation, BudgetMonth: current.BudgetMonth, ExternalStarted: current.ExternalStarted,
		ProviderID: settlement.Result.ProviderID, ProviderModel: settlement.Result.ProviderModel, Usage: &settlement.Result.Usage,
		Actual: settlement.Actual, Conservative: settlement.Conservative, Output: settlement.Result.Output,
		Code: settlement.Result.Code, RecordedAt: now,
	}
	if settlement.Result.State == ai.Completed {
		next.Validation = "pending_validation"
	}
	if settlement.NeedsReconciliation {
		next.Reconciliation = "pending"
	} else if settlement.Conservative {
		next.Reconciliation = "conservative"
	}
	if settlement.Result.State == ai.KnownRejection {
		next.Usage = nil
	}
	return s.insertAIState(ctx, tx, family, attemptID, next)
}

func (s *Store) aiBudgetGate(ctx context.Context, tx pgx.Tx, family household.HouseholdID) (bool, int, error) {
	var blocked bool
	var active int
	err := tx.QueryRow(ctx, `WITH latest AS (SELECT DISTINCT ON(s.attempt_id) s.* FROM want_keep.ai_attempt_states s WHERE s.household_id=$1 ORDER BY s.attempt_id,s.revision DESC) SELECT COALESCE(bool_or(l.reconciliation_state='pending' OR l.external_started AND l.state='reserved' AND (j.state!='running' OR j.lease_until IS NULL OR j.lease_until<=clock_timestamp())),false),count(*) FILTER(WHERE l.state IN ('counting','reserved')) FROM latest l JOIN want_keep.ai_attempts a ON (a.household_id,a.id)=(l.household_id,l.attempt_id) JOIN want_keep.jobs j ON (j.household_id,j.id)=(a.household_id,a.job_id)`, family).Scan(&blocked, &active)
	return blocked, active, err
}

func (s *Store) aiMonthUsage(ctx context.Context, tx pgx.Tx, family household.HouseholdID, now time.Time) (ai.Cost, error) {
	var value string
	err := tx.QueryRow(ctx, `WITH latest AS (SELECT DISTINCT ON(s.attempt_id) s.* FROM want_keep.ai_attempt_states s WHERE s.household_id=$1 ORDER BY s.attempt_id,s.revision DESC) SELECT COALESCE(sum(CASE WHEN l.actual_usd IS NOT NULL THEN l.actual_usd WHEN l.state IN ('reserved','unknown') THEN l.reservation_usd ELSE 0 END),0)::text FROM latest l WHERE l.budget_month=date_trunc('month',$2::timestamptz)::date`, family, now.UTC()).Scan(&value)
	if err != nil {
		return ai.Cost{}, err
	}
	return ai.NewCost(value)
}

func (s *Store) refuseAIReservation(ctx context.Context, tx pgx.Tx, family household.HouseholdID, attemptID string, current aiState, reservation ai.Cost, code string, now time.Time) error {
	zero := ai.MustCost("0")
	settlement := aiapp.Settlement{Result: ai.Result{State: ai.KnownRejection, Code: code}, Reservation: reservation, Actual: &zero}
	return s.writeAISettlement(ctx, tx, family, attemptID, current, settlement, now)
}

// ResumeAIBudgetWaiting is the trusted queue owner for budget and concurrency waits.
// ReserveAIAttempt still serializes the final decision under the household lock.
func (s *Store) ResumeAIBudgetWaiting(ctx context.Context, now time.Time) (int64, error) {
	if now.IsZero() {
		return 0, aiapp.ErrInvalidBudgetQueue
	}
	tag, err := s.pool.Exec(ctx, `WITH latest AS (
	 SELECT DISTINCT ON(s.household_id,s.attempt_id) s.*
	 FROM want_keep.ai_attempt_states s
	 ORDER BY s.household_id,s.attempt_id,s.revision DESC
	), family_gate AS (
	 SELECT a.household_id,
	  COALESCE(bool_or(l.reconciliation_state='pending' OR l.external_started AND l.state='reserved' AND (j.state!='running' OR j.lease_until IS NULL OR j.lease_until<=clock_timestamp())),false) blocked,
	  count(*) FILTER(WHERE l.state IN ('counting','reserved')) active,
		  COALESCE(sum(CASE WHEN l.budget_month=date_trunc('month',$1::timestamptz)::date THEN CASE WHEN l.actual_usd IS NOT NULL THEN l.actual_usd WHEN l.state IN ('reserved','unknown') THEN l.reservation_usd ELSE 0 END ELSE 0 END),0) used
	 FROM want_keep.ai_attempts a JOIN latest l ON (l.household_id,l.attempt_id)=(a.household_id,a.id) JOIN want_keep.jobs j ON (j.household_id,j.id)=(a.household_id,a.job_id)
	 GROUP BY a.household_id
	), pending AS (
	 SELECT j.household_id,j.id
	 FROM want_keep.jobs j
	 LEFT JOIN family_gate g ON g.household_id=j.household_id
	 LEFT JOIN LATERAL (
		  SELECT l.budget_month,l.code,l.reservation_usd
	  FROM want_keep.ai_attempts a JOIN latest l ON (l.household_id,l.attempt_id)=(a.household_id,a.id)
	  WHERE (a.household_id,a.job_id)=(j.household_id,j.id)
	  ORDER BY a.created_at DESC,a.created_ns DESC,a.id DESC LIMIT 1
	 ) last_attempt ON true
	 WHERE j.kind IN ('ai','ai_answer') AND j.state='waiting' AND j.reason='budget_wait'
	  AND NOT j.cancel_requested AND NOT j.external_started
	  AND NOT COALESCE(g.blocked,false) AND COALESCE(g.active,0)<2 AND COALESCE(g.used,0)<50
	  AND (last_attempt.code IS DISTINCT FROM 'budget_exhausted'
	   OR last_attempt.budget_month<date_trunc('month',$1::timestamptz)::date
	   OR COALESCE(g.used,0)+COALESCE(last_attempt.reservation_usd,0)<=50)
	 ORDER BY j.available_at,j.id LIMIT 100 FOR UPDATE OF j SKIP LOCKED
	)
	UPDATE want_keep.jobs j SET state='ready',reason='',run_deadline=clock_timestamp()+INTERVAL '24 hours',available_at=clock_timestamp()
	FROM pending p WHERE (j.household_id,j.id)=(p.household_id,p.id)`, now.UTC())
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

// ResumeAIGatewayWaiting preserves the job's original deadline and only
// releases retries whose provider backoff has elapsed.
func (s *Store) ResumeAIGatewayWaiting(ctx context.Context) error {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	var now time.Time
	if err = tx.QueryRow(ctx, `SELECT clock_timestamp()`).Scan(&now); err != nil {
		return err
	}
	rows, err := tx.Query(ctx, `SELECT household_id,id FROM want_keep.jobs
	 WHERE kind IN ('ai','ai_answer') AND state='waiting' AND reason='gateway_unavailable'
	  AND COALESCE(run_deadline,deadline)<=$1
	 ORDER BY available_at,id LIMIT 100 FOR UPDATE SKIP LOCKED`, now)
	if err != nil {
		return err
	}
	type expiredJob struct {
		householdID household.HouseholdID
		jobID       string
	}
	expired := make([]expiredJob, 0, 100)
	for rows.Next() {
		var item expiredJob
		if err = rows.Scan(&item.householdID, &item.jobID); err != nil {
			rows.Close()
			return err
		}
		expired = append(expired, item)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range expired {
		if err = s.releaseSafeAIJobAttempts(ctx, tx, item.householdID, item.jobID, "gateway_wait_expired", now); err != nil {
			return err
		}
		tag, updateErr := tx.Exec(ctx, `UPDATE want_keep.jobs SET state='failed',reason='deadline_exceeded' WHERE household_id=$1 AND id=$2 AND state='waiting' AND reason='gateway_unavailable' AND COALESCE(run_deadline,deadline)<=$3`, item.householdID, item.jobID, now)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
	}
	if _, err = tx.Exec(ctx, `WITH pending AS (
	 SELECT household_id,id FROM want_keep.jobs
	 WHERE kind IN ('ai','ai_answer') AND state='waiting' AND reason='gateway_unavailable'
	  AND NOT cancel_requested AND NOT external_started
	  AND available_at<=clock_timestamp() AND COALESCE(run_deadline,deadline)>clock_timestamp()
	 ORDER BY available_at,id LIMIT 100 FOR UPDATE SKIP LOCKED
	)
	UPDATE want_keep.jobs j SET state='ready',reason='',available_at=clock_timestamp()
	FROM pending p WHERE (j.household_id,j.id)=(p.household_id,p.id)`); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) releaseSafeInterruptedAttempts(ctx context.Context, tx *transactionScope, p household.Principal, job jobs.Job, now time.Time) error {
	return s.releaseSafeAIJobAttempts(ctx, tx.tx, p.HouseholdID(), job.ID, "recovered_before_send", now)
}

func (s *Store) releaseSafeAIJobAttempts(ctx context.Context, tx pgx.Tx, family household.HouseholdID, jobID, code string, now time.Time) error {
	rows, err := tx.Query(ctx, `SELECT a.id FROM want_keep.ai_attempts a JOIN LATERAL (SELECT state,external_started FROM want_keep.ai_attempt_states s WHERE s.household_id=a.household_id AND s.attempt_id=a.id ORDER BY revision DESC LIMIT 1) l ON true WHERE a.household_id=$1 AND a.job_id=$2 AND l.state IN ('counting','reserved') AND NOT l.external_started ORDER BY a.created_at,a.id`, family, jobID)
	if err != nil {
		return err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, id := range ids {
		current, stateErr := s.lockAIState(ctx, tx, family, id)
		if stateErr != nil {
			return stateErr
		}
		reservation := ai.MustCost("0")
		if current.Reservation != nil {
			reservation = *current.Reservation
		}
		settlement := aiapp.Settlement{Result: ai.Result{State: ai.KnownRejection, Code: code}, Reservation: reservation}
		if err = s.writeAISettlement(ctx, tx, family, id, current, settlement, now); err != nil {
			return err
		}
	}
	return nil
}

// ReconcileAI is an operator-only path. The maintenance role can execute only
// the hardened database transition; it has no direct table privileges.
func (s *Store) ReconcileAI(ctx context.Context, attemptID string, outcome aiapp.ReconciliationOutcome, actual ai.Cost, evidenceRef string) error {
	if attemptID == "" || len(evidenceRef) == 0 || len(evidenceRef) > 2000 || actual.Validate() != nil || (outcome != aiapp.Charged && outcome != aiapp.NotCharged) {
		return aiapp.ErrInvalidReconciliation
	}
	if outcome == aiapp.NotCharged {
		comparison, compareErr := actual.Compare(ai.MustCost("0"))
		if compareErr != nil || comparison != 0 {
			return aiapp.ErrInvalidReconciliation
		}
	} else if comparison, compareErr := actual.Compare(ai.MustCost("0")); compareErr != nil || comparison <= 0 {
		return aiapp.ErrInvalidReconciliation
	}
	_, err := s.pool.Exec(ctx, `SELECT want_keep.reconcile_ai_attempt($1::uuid,$2,$3::numeric,$4)`, attemptID, outcome, actual.String(), evidenceRef)
	if err == nil {
		return nil
	}
	var providerError *pgconn.PgError
	if errors.As(err, &providerError) {
		switch providerError.Code {
		case "P0002":
			return ErrNotFound
		case "22023", "55006":
			return aiapp.ErrInvalidReconciliation
		}
	}
	return err
}
