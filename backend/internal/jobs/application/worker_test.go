package application

import (
	"context"
	"testing"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func TestCommittedResultSkipsSecondJobTransition(t *testing.T) {
	repository := &workerRepository{job: jobs.Job{ID: "job", Kind: jobs.Sync, State: jobs.Running}}
	worker := Worker{Admission: allowAdmission{}, Repository: repository, Handler: committedHandler{}, Config: DefaultWorkerConfig(jobs.Sync)}
	if err := worker.Step(context.Background()); err != nil {
		t.Fatal(err)
	}
	if repository.outcomes != 0 {
		t.Fatal("committed sync result received a second transition")
	}
}

func TestKnownNoIORejectionAcknowledgesWithinHousehold(t *testing.T) {
	repository := &workerRepository{}
	execution := Execution{repository: repository}
	if err := execution.RejectBeforeProviderIO(context.Background()); err != nil || repository.acknowledged != 1 {
		t.Fatal("pre-provider rejection was not acknowledged", err, repository.acknowledged)
	}
}

type committedHandler struct{}

type allowAdmission struct{}

func (allowAdmission) BeforeRead(context.Context, household.Principal, jobs.Job) error { return nil }

func (committedHandler) Prepare(context.Context, Execution) (Result, error) {
	return Result{Committed: true}, nil
}

type workerRepository struct {
	job          jobs.Job
	outcomes     int
	acknowledged int
	inHousehold  bool
}

func (r *workerRepository) ClaimJobs(context.Context, string, int, time.Duration) ([]jobs.Job, error) {
	return []jobs.Job{r.job}, nil
}
func (*workerRepository) Heartbeat(context.Context, household.Principal, jobs.Job, time.Duration) error {
	return nil
}
func (*workerRepository) FinishJob(context.Context, household.Principal, jobs.Job) error { return nil }
func (*workerRepository) FailJob(context.Context, household.Principal, jobs.Job) error   { return nil }
func (*workerRepository) RetryJob(context.Context, household.Principal, jobs.Job, time.Duration, bool) error {
	return nil
}
func (*workerRepository) RecoverJobs(context.Context, jobs.Kind) error { return nil }
func (r *workerRepository) WithinHousehold(ctx context.Context, _ household.Principal, fn func(context.Context) error) error {
	r.inHousehold = true
	defer func() { r.inHousehold = false }()
	return fn(ctx)
}
func (*workerRepository) JobPrincipal(context.Context, jobs.Job) (household.Principal, error) {
	return household.Principal{}, nil
}
func (*workerRepository) FenceJob(context.Context, household.Principal, jobs.Job) (jobs.Job, error) {
	return jobs.Job{}, nil
}
func (*workerRepository) JobReceipt(context.Context, household.Principal, jobs.Job) (bool, error) {
	return false, nil
}
func (*workerRepository) RecordJobReceipt(context.Context, household.Principal, jobs.Job) error {
	return nil
}
func (*workerRepository) BeginExternal(context.Context, household.Principal, jobs.Job) error {
	return nil
}

func (r *workerRepository) AcknowledgeExternalResult(context.Context, household.Principal, jobs.Job) error {
	if !r.inHousehold {
		return jobs.ErrInvalidJob
	}
	r.acknowledged++
	return nil
}
func (r *workerRepository) SetJobOutcome(context.Context, household.Principal, jobs.Job, jobs.State, jobs.Reason, time.Duration) error {
	r.outcomes++
	return nil
}
func (*workerRepository) PauseReady(context.Context, jobs.Kind, jobs.Reason) error    { return nil }
func (*workerRepository) ResumeWaiting(context.Context, jobs.Kind, jobs.Reason) error { return nil }
