package storage

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

// The caller selects and locks the affected jobs in its existing transaction.
func (s *Store) cancelJobs(ctx context.Context, tx pgx.Tx, rows pgx.Rows) error {
	type transition struct {
		job     jobs.Job
		outcome jobs.Outcome
	}
	var transitions []transition
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			rows.Close()
			return err
		}
		transitions = append(transitions, transition{job: job, outcome: job.Cancel()})
	}
	err := rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	now := time.Now().UTC()
	for _, item := range transitions {
		if item.job.Kind.ProviderCall() && (item.outcome.State == jobs.Canceled || item.outcome.State == jobs.Failed) {
			if err = s.releaseSafeAIJobAttempts(ctx, tx, item.job.HouseholdID, item.job.ID, "job_terminated_before_send", now); err != nil {
				return err
			}
		}
		tag, updateErr := tx.Exec(ctx, `UPDATE want_keep.jobs SET state=$5,reason=$6,external_started=CASE WHEN $5='unresolved' THEN external_started ELSE false END,cancel_requested=true WHERE household_id=$1 AND id=$2 AND state=$3 AND attempt=$4`, item.job.HouseholdID, item.job.ID, item.job.State, item.job.Attempt, item.outcome.State, item.outcome.Reason)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
	}
	return nil
}
