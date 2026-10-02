package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

// FenceJob must be called inside the household transaction that applies the effect.
func (s *Store) FenceJob(ctx context.Context, p household.Principal, issued jobs.Job) (jobs.Job, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return jobs.Job{}, err
	}
	if p != scope.principal || issued.HouseholdID != p.HouseholdID() || issued.ActorID != p.UserID() {
		return jobs.Job{}, household.ErrForbidden
	}
	current, err := scanReplayJob(scope.tx.QueryRow(ctx, `SELECT `+replayJobColumns+` FROM want_keep.jobs WHERE household_id=$1 AND id=$2 FOR UPDATE`, p.HouseholdID(), issued.ID))
	if err != nil {
		return current, err
	}
	now, err := s.DatabaseTime(ctx)
	if err != nil {
		return current, err
	}
	return current, current.RequireAttempt(issued, now)
}
func (s *Store) JobPrincipal(ctx context.Context, issued jobs.Job) (household.Principal, error) {
	m, err := s.Membership(ctx, issued.HouseholdID, issued.ActorID)
	if err != nil {
		return household.Principal{}, err
	}
	p, err := m.Principal()
	if err != nil {
		return p, err
	}
	err = s.WithinHousehold(ctx, p, func(ctx context.Context) error { _, e := s.FenceJob(ctx, p, issued); return e })
	return p, err
}
func (s *Store) BeginExternal(ctx context.Context, p household.Principal, j jobs.Job) error {
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		current, err := s.FenceJob(ctx, p, j)
		if err != nil {
			return err
		}
		if current.ExternalStarted {
			return jobs.ErrStaleAttempt
		}
		scope, _ := s.familyScope(ctx)
		_, err = scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET external_started=true WHERE household_id=$1 AND id=$2`, p.HouseholdID(), j.ID)
		return err
	})
}

func (s *Store) AcknowledgeExternalResult(ctx context.Context, p household.Principal, j jobs.Job) error {
	current, err := s.FenceJob(ctx, p, j)
	if err != nil {
		return err
	}
	if !current.ExternalStarted {
		return jobs.ErrStaleAttempt
	}
	scope, _ := s.familyScope(ctx)
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET external_started=false WHERE household_id=$1 AND id=$2 AND lease_token=$3 AND attempt=$4 AND state='running' AND external_started`, p.HouseholdID(), j.ID, j.LeaseToken, j.Attempt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return jobs.ErrStaleAttempt
	}
	return nil
}

func (s *Store) JobReceipt(ctx context.Context, p household.Principal, j jobs.Job) (bool, error) {
	if j.HouseholdID != p.HouseholdID() || j.ActorID != p.UserID() {
		return false, household.ErrForbidden
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var token string
	var attempt int
	err = q.QueryRow(ctx, `SELECT lease_token,attempt FROM want_keep.job_receipts WHERE household_id=$1 AND job_id=$2`, p.HouseholdID(), j.ID).Scan(&token, &attempt)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	if j.ActorID != p.UserID() || token != j.LeaseToken || attempt != j.Attempt {
		return false, jobs.ErrStaleAttempt
	}
	return true, nil
}
func (s *Store) RecordJobReceipt(ctx context.Context, p household.Principal, j jobs.Job) error {
	if _, err := s.FenceJob(ctx, p, j); err != nil {
		return err
	}
	return s.insertJobReceipt(ctx, p, j)
}

func (s *Store) insertJobReceipt(ctx context.Context, p household.Principal, j jobs.Job) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.job_receipts(household_id,job_id,lease_token,attempt) VALUES($1,$2,$3,$4)`, p.HouseholdID(), j.ID, j.LeaseToken, j.Attempt)
	return err
}
func (s *Store) SetJobOutcome(ctx context.Context, p household.Principal, j jobs.Job, state jobs.State, reason jobs.Reason, delay time.Duration) error {
	if delay < 0 || delay > jobs.MaxRetryDelay || (state != jobs.Ready && state != jobs.Failed && state != jobs.Waiting && state != jobs.Unresolved) || (state == jobs.Waiting && !reason.Waiting()) {
		return jobs.ErrInvalidJob
	}
	return s.transitionJob(ctx, p, j, state, reason, delay)
}

// PauseReady never takes a lease or consumes an execution attempt.
func (s *Store) PauseReady(ctx context.Context, kind jobs.Kind, reason jobs.Reason) error {
	if !kind.Valid() || !reason.Waiting() {
		return jobs.ErrInvalidJob
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err = s.recoverJobs(ctx, tx, string(kind), reason); err != nil {
		return err
	}
	return tx.Commit(ctx)
}
func (s *Store) ResumeWaiting(ctx context.Context, kind jobs.Kind, reason jobs.Reason) error {
	if !kind.Valid() || !reason.Waiting() {
		return jobs.ErrInvalidJob
	}
	_, err := s.pool.Exec(ctx, `WITH pending AS (SELECT household_id,id FROM want_keep.jobs WHERE kind=$1 AND state='waiting' AND reason=$2 AND available_at<=clock_timestamp() AND NOT cancel_requested AND NOT external_started ORDER BY available_at,id LIMIT 100 FOR UPDATE SKIP LOCKED) UPDATE want_keep.jobs j SET state='ready',reason='',run_deadline=clock_timestamp()+INTERVAL '24 hours',available_at=clock_timestamp() FROM pending p WHERE (j.household_id,j.id)=(p.household_id,p.id)`, kind, reason)
	return err
}
