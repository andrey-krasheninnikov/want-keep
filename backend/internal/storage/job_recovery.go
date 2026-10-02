package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) RecoverJobs(ctx context.Context, kind jobs.Kind) error {
	if !kind.Valid() {
		return jobs.ErrInvalidJob
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if err = s.recoverJobs(ctx, tx, string(kind), ""); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (s *Store) recoverJobs(ctx context.Context, tx pgx.Tx, kind string, dependency jobs.Reason) error {
	type transition struct {
		job     jobs.Job
		outcome jobs.Outcome
		now     time.Time
	}
	rows, err := tx.Query(ctx, `SELECT `+jobColumns+`,EXISTS(SELECT 1 FROM want_keep.memberships m WHERE m.household_id=j.household_id AND m.user_id=j.actor_id AND m.active),clock_timestamp() FROM want_keep.jobs j WHERE kind=$1 AND ((state='running' AND (lease_until<=clock_timestamp() OR COALESCE(run_deadline,deadline)<=clock_timestamp() OR cancel_requested)) OR (state='ready' AND ($2::text!='' OR cancel_requested OR COALESCE(run_deadline,deadline)<=clock_timestamp() OR attempt>=max_attempts OR NOT EXISTS(SELECT 1 FROM want_keep.memberships m WHERE m.household_id=j.household_id AND m.user_id=j.actor_id AND m.active)))) ORDER BY available_at,id LIMIT 100 FOR UPDATE OF j SKIP LOCKED`, kind, dependency)
	if err != nil {
		return err
	}
	var transitions []transition
	for rows.Next() {
		var active bool
		var now time.Time
		job, scanErr := scanJob(rows, &active, &now)
		if scanErr != nil {
			rows.Close()
			return scanErr
		}
		outcome, changed := job.Recover(now, active, dependency)
		if changed {
			transitions = append(transitions, transition{job: job, outcome: outcome, now: now})
		}
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	for _, item := range transitions {
		if item.job.Kind.ProviderCall() && (item.outcome.State == jobs.Canceled || item.outcome.State == jobs.Failed) {
			if err = s.releaseSafeAIJobAttempts(ctx, tx, item.job.HouseholdID, item.job.ID, "job_terminated_before_send", item.now); err != nil {
				return err
			}
		} else if item.job.Kind.ProviderCall() && !item.job.ExternalStarted && item.job.State == jobs.Running {
			if err = s.releaseSafeAIJobAttempts(ctx, tx, item.job.HouseholdID, item.job.ID, "recovered_before_send", item.now); err != nil {
				return err
			}
		}
		tag, updateErr := tx.Exec(ctx, `UPDATE want_keep.jobs SET state=$5,reason=$6,attempt=$7,external_started=CASE WHEN $5='unresolved' THEN external_started ELSE false END WHERE household_id=$1 AND id=$2 AND state=$3 AND attempt=$4`, item.job.HouseholdID, item.job.ID, item.job.State, item.job.Attempt, item.outcome.State, item.outcome.Reason, item.outcome.Attempt)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
	}
	return nil
}
