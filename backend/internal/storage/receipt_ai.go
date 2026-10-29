package storage

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) ensureAISubject(ctx context.Context, p household.Principal, job jobs.Job) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	query := `INSERT INTO want_keep.ai_request_subjects(household_id,resource_id,resource_revision,operation_id,operation_revision) VALUES($1,$2,$3,$2,$3) ON CONFLICT DO NOTHING`
	if job.Kind == jobs.ReceiptExtract {
		query = `INSERT INTO want_keep.ai_request_subjects(household_id,resource_id,resource_revision,page_id,page_revision) VALUES($1,$2,$3,$2,$3) ON CONFLICT DO NOTHING`
	}
	_, err = scope.tx.Exec(ctx, query, p.HouseholdID(), job.ResourceID, job.ResourceRevision)
	return err
}
func (s *Store) enqueueValidation(ctx context.Context, p household.Principal, job jobs.Job, attempt string) error {
	if job.Kind != jobs.ReceiptExtract {
		return s.enqueueAIValidation(ctx, p, job, attempt)
	}
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	id := newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline,resource_id,resource_revision) VALUES($1,$2,$3,'receipt_validation','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours',$4,$5)`, p.HouseholdID(), id, p.UserID(), job.ResourceID, job.ResourceRevision)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_validation_jobs(household_id,job_id,attempt_id) VALUES($1,$2,$3) ON CONFLICT DO NOTHING`, p.HouseholdID(), id, attempt)
	return err
}
