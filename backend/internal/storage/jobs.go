package storage

import (
	"context"
	"crypto/rand"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func newID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		panic(err)
	}
	b[6] = (b[6] & 15) | 64
	b[8] = (b[8] & 63) | 128
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

const jobColumns = `household_id,id,actor_id,kind,COALESCE(connection_id::text,''),COALESCE(connection_generation,0),binding,COALESCE(admission_revision,0),state,attempt,max_attempts,COALESCE(lease_token::text,''),lease_until,COALESCE(run_deadline,deadline),cancel_requested,cursor,coverage,gaps,secret_purpose,reason,external_started,COALESCE(resource_id::text,''),COALESCE(resource_revision,0)`

const replayJobColumns = jobColumns + `,COALESCE(replay_request_id::text,''),range_from,range_from_ns,range_to,range_to_ns`

func scanJob(row pgx.Row, additional ...any) (jobs.Job, error) {
	var j jobs.Job
	var binding []byte
	var until *time.Time
	fields := []any{&j.HouseholdID, &j.ID, &j.ActorID, &j.Kind, &j.ConnectionID, &j.ConnectionGeneration, &binding, &j.AdmissionRevision, &j.State, &j.Attempt, &j.MaxAttempts, &j.LeaseToken, &until, &j.Deadline, &j.CancelRequested, &j.Cursor, &j.Coverage, &j.Gaps, &j.SecretPurpose, &j.Reason, &j.ExternalStarted, &j.ResourceID, &j.ResourceRevision}
	err := row.Scan(append(fields, additional...)...)
	if err != nil {
		return j, err
	}
	if until != nil {
		j.LeaseUntil = *until
	}
	if len(binding) > 0 {
		j.Binding, err = decodeBinding(binding)
		if err != nil {
			return j, err
		}
		if err = j.Binding.Validate(); err != nil {
			return j, err
		}
	}
	return j, nil
}

func scanReplayJob(row pgx.Row, additional ...any) (jobs.Job, error) {
	var replayRequestID string
	var rangeFrom, rangeTo *time.Time
	var rangeFromNS, rangeToNS *int16
	job, err := scanJob(row, append([]any{&replayRequestID, &rangeFrom, &rangeFromNS, &rangeTo, &rangeToNS}, additional...)...)
	if err != nil {
		return job, err
	}
	job.ReplayRequestID = replayRequestID
	if rangeFrom != nil && rangeFromNS != nil && rangeTo != nil && rangeToNS != nil {
		job.RangeFrom = rangeFrom.UTC().Add(time.Duration(*rangeFromNS))
		job.RangeTo = rangeTo.UTC().Add(time.Duration(*rangeToNS))
	}
	return job, nil
}
func (s *Store) EmitEvent(ctx context.Context, resourceType, id string, revision uint64, event string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if id == "" || resourceType == "" || event == "" || revision < 1 {
		return jobs.ErrInvalidJob
	}
	var exists bool
	err = scope.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.outbox WHERE household_id=$1 AND resource_type=$2 AND resource_id=$3 AND revision=$4 AND event_type=$5)`, scope.principal.HouseholdID(), resourceType, id, revision, event).Scan(&exists)
	if err != nil || exists {
		return err
	}
	jobID := newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline) VALUES($1,$2,$3,'outbox','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours')`, scope.principal.HouseholdID(), jobID, scope.principal.UserID())
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.outbox(household_id,id,actor_id,resource_type,resource_id,revision,event_type) VALUES($1,$2,$3,$4,$5,$6,$7)`, scope.principal.HouseholdID(), jobID, scope.principal.UserID(), resourceType, id, revision, event)
	return err
}
func (s *Store) ClaimJobs(ctx context.Context, kind string, limit int, lease time.Duration) ([]jobs.Job, error) {
	if !jobs.Kind(kind).Valid() || limit < 1 || limit > 100 || lease < time.Second || lease > 5*time.Minute {
		return nil, jobs.ErrInvalidJob
	}
	if jobs.Kind(kind) == jobs.AIValidation {
		if err := s.recoverReviewAnswers(ctx); err != nil {
			return nil, err
		}
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(context.Background())
	if err = s.recoverJobs(ctx, tx, kind, ""); err != nil {
		return nil, err
	}
	rows, err := tx.Query(ctx, `SELECT `+replayJobColumns+` FROM want_keep.jobs WHERE kind=$1 AND NOT cancel_requested AND COALESCE(run_deadline,deadline)>clock_timestamp() AND attempt<max_attempts AND state='ready' AND available_at<=clock_timestamp() AND NOT external_started AND (kind!='sync' OR NOT EXISTS(SELECT 1 FROM want_keep.jobs blocked WHERE blocked.household_id=want_keep.jobs.household_id AND blocked.connection_id=want_keep.jobs.connection_id AND blocked.state='unresolved')) ORDER BY available_at,id LIMIT $2 FOR UPDATE SKIP LOCKED`, kind, limit)
	if err != nil {
		return nil, err
	}
	result := []jobs.Job{}
	for rows.Next() {
		j, e := scanReplayJob(rows)
		if e != nil {
			rows.Close()
			return nil, e
		}
		result = append(result, j)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, err
	}
	for i := range result {
		j := &result[i]
		j.LeaseToken = newID()
		err = tx.QueryRow(ctx, `UPDATE want_keep.jobs SET state='running',attempt=attempt+1,lease_token=$3,lease_until=LEAST(COALESCE(run_deadline,deadline),clock_timestamp()+$4::bigint*INTERVAL '1 microsecond') WHERE household_id=$1 AND id=$2 RETURNING attempt,lease_until`, j.HouseholdID, j.ID, j.LeaseToken, lease.Microseconds()).Scan(&j.Attempt, &j.LeaseUntil)
		if err != nil {
			return nil, err
		}
		j.State = "running"
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return result, nil
}
func (s *Store) Job(ctx context.Context, p household.Principal, id string) (jobs.Job, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return jobs.Job{}, err
	}
	return scanReplayJob(q.QueryRow(ctx, `SELECT `+replayJobColumns+` FROM want_keep.jobs WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id))
}
func (s *Store) Heartbeat(ctx context.Context, p household.Principal, j jobs.Job, lease time.Duration) error {
	if lease < time.Second || lease > 5*time.Minute {
		return jobs.ErrInvalidJob
	}
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		scope, err := s.familyScope(ctx)
		if err != nil {
			return err
		}
		if j.HouseholdID != p.HouseholdID() || j.ActorID != p.UserID() {
			return household.ErrForbidden
		}
		if _, err = s.FenceJob(ctx, p, j); err != nil {
			return err
		}
		tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET lease_until=LEAST(COALESCE(run_deadline,deadline),clock_timestamp()+$5::bigint*INTERVAL '1 microsecond') WHERE household_id=$1 AND id=$2 AND lease_token=$3 AND attempt=$4 AND state='running' AND NOT cancel_requested AND lease_until>clock_timestamp() AND COALESCE(run_deadline,deadline)>clock_timestamp()`, p.HouseholdID(), j.ID, j.LeaseToken, j.Attempt, lease.Microseconds())
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
		return nil
	})
}
func (s *Store) FinishJob(ctx context.Context, p household.Principal, j jobs.Job) error {
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		done, err := s.JobReceipt(ctx, p, j)
		if err != nil || done {
			return err
		}
		if err := s.RecordJobReceipt(ctx, p, j); err != nil {
			return err
		}
		if err := s.transitionJob(ctx, p, j, jobs.Succeeded, "", 0); err != nil {
			return err
		}
		if j.Kind != jobs.Sync {
			return nil
		}
		scope, _ := s.familyScope(ctx)
		_, err = scope.tx.Exec(ctx, `UPDATE want_keep.sync_progress SET completed=true,last_success_at=clock_timestamp() WHERE household_id=$1 AND connection_id=$2 AND generation=$3 AND last_job_id=$4`, p.HouseholdID(), j.ConnectionID, j.ConnectionGeneration, j.ID)
		return err
	})
}
func (s *Store) FailJob(ctx context.Context, p household.Principal, j jobs.Job) error {
	return s.transitionJob(ctx, p, j, jobs.Failed, jobs.PermanentFailure, 0)
}
func (s *Store) RetryJob(ctx context.Context, p household.Principal, j jobs.Job, delay time.Duration, ambiguous bool) error {
	if delay < 0 || delay > jobs.MaxRetryDelay {
		return jobs.ErrInvalidJob
	}
	state := jobs.Ready
	if ambiguous {
		state = jobs.Unresolved
	}
	return s.transitionJob(ctx, p, j, state, "", delay)
}
func (s *Store) transitionJob(ctx context.Context, p household.Principal, j jobs.Job, state jobs.State, reason jobs.Reason, delay time.Duration) error {
	return s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		scope, err := s.familyScope(ctx)
		if err != nil {
			return err
		}
		if j.HouseholdID != p.HouseholdID() || j.ActorID != p.UserID() {
			return household.ErrForbidden
		}
		current, err := s.FenceJob(ctx, p, j)
		if err != nil {
			return err
		}
		outcome, err := current.Complete(state, reason)
		if err != nil {
			return err
		}
		tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.jobs SET state=$5,reason=$7,attempt=$8,available_at=clock_timestamp()+$6::bigint*INTERVAL '1 microsecond' WHERE household_id=$1 AND id=$2 AND lease_token=$3 AND attempt=$4 AND state='running' AND NOT cancel_requested AND lease_until>clock_timestamp() AND COALESCE(run_deadline,deadline)>clock_timestamp()`, p.HouseholdID(), j.ID, j.LeaseToken, j.Attempt, outcome.State, delay.Microseconds(), outcome.Reason, outcome.Attempt)
		if err != nil {
			return err
		}
		if tag.RowsAffected() != 1 {
			return jobs.ErrStaleAttempt
		}
		return nil
	})
}
func (s *Store) DatabaseTime(ctx context.Context) (time.Time, error) {
	scope, err := s.scope(ctx)
	if err != nil {
		return time.Time{}, err
	}
	var now time.Time
	err = scope.tx.QueryRow(ctx, "SELECT clock_timestamp()").Scan(&now)
	return now, err
}
