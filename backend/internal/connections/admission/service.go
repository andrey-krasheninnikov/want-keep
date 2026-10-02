package admission

import (
	"context"
	"errors"
	"time"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type ResultKind string

const (
	PageResult            ResultKind = "page"
	ProviderOutcomeResult ResultKind = "provider_outcome"
	RejectedResult        ResultKind = "rejected_result"
	StaleResult           ResultKind = "stale_result"
)

func (k ResultKind) Valid() bool {
	return k == PageResult || k == ProviderOutcomeResult || k == RejectedResult || k == StaleResult
}

type ResultReceipt struct {
	HouseholdID household.HouseholdID
	JobID       string
	LeaseToken  string
	Attempt     int
	InputCursor string
	EvidenceRef string
	Kind        ResultKind
}

func (r ResultReceipt) Matches(p household.Principal, issued jobs.Job, evidence string, kind ResultKind) bool {
	return kind.Valid() && r.Kind == kind && r.HouseholdID == p.HouseholdID() && r.JobID == issued.ID && r.LeaseToken == issued.LeaseToken && r.Attempt == issued.Attempt && r.InputCursor == issued.Cursor && r.EvidenceRef == evidence
}

type Connection struct {
	HouseholdID   household.HouseholdID
	ID, Provider  string
	Owner         household.UserID
	Generation    uint64
	Authorized    bool
	SecretPurpose connections.SecretPurpose
}
type Transactions interface {
	WithinAdmission(context.Context, string, string, func(context.Context) error) error
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
}
type Repository interface {
	Admission(context.Context, string, string) (connections.Admission, bool, error)
	SaveAdmission(context.Context, connections.Admission) error
	InvalidateJobs(context.Context, connections.Admission) error
	Connection(context.Context, household.Principal, string) (Connection, error)
	CreateSyncJob(context.Context, Connection, connections.Admission, time.Time) (jobs.Job, error)
	CreateReplayJob(context.Context, Connection, connections.Admission, time.Time, string, time.Time, time.Time) (jobs.Job, error)
	Job(context.Context, household.Principal, string) (jobs.Job, error)
	DatabaseTime(context.Context) (time.Time, error)
	FenceSyncResult(context.Context, household.Principal, jobs.Job) error
	AcknowledgeExternalResult(context.Context, household.Principal, jobs.Job) error
	Quarantine(context.Context, jobs.Job, string, string) error
	SaveCheckpoint(context.Context, jobs.Job, string, string, []string) error
	ImportOmissions(context.Context, household.Principal, string) ([]string, error)
	FinishJob(context.Context, household.Principal, jobs.Job) error
	FailJob(context.Context, household.Principal, jobs.Job) error
	SetJobOutcome(context.Context, household.Principal, jobs.Job, jobs.State, jobs.Reason, time.Duration) error
	SaveResultReceipt(context.Context, household.Principal, jobs.Job, string, ResultKind) error
	ResultReceipt(context.Context, household.Principal, jobs.Job, string) (ResultReceipt, bool, error)
	EvidenceResult(context.Context, household.Principal, string, string) (ResultKind, bool, error)
	SyncDue(context.Context, string) (bool, error)
	AdvanceSyncSchedule(context.Context, string) error
}

func (s *Service) RequestReplay(ctx context.Context, p household.Principal, id string, b connections.Binding, admissionRevision int64, connectionGeneration uint64, deadline time.Time, requestID string, from, to time.Time, attach func(context.Context, jobs.Job) error) (jobs.Job, error) {
	request := jobs.Job{ReplayRequestID: requestID, RangeFrom: from, RangeTo: to}
	if err := request.ValidateReplay(); err != nil || b.Validate() != nil || admissionRevision < 1 || connectionGeneration < 1 || attach == nil {
		return jobs.Job{}, jobs.ErrInvalidJob
	}
	var result jobs.Job
	err := s.transactions.WithinAdmission(ctx, b.Provider, b.Environment, func(ctx context.Context) error {
		a, _, err := s.repository.Admission(ctx, b.Provider, b.Environment)
		if err != nil {
			return err
		}
		if err = a.RequireResult(b, admissionRevision); err != nil {
			return err
		}
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			connection, err := s.repository.Connection(ctx, p, id)
			if err != nil {
				return err
			}
			if err = (connections.ExternalOwnership{HouseholdID: connection.HouseholdID, OwnerID: connection.Owner}).RequireManage(p); err != nil {
				return err
			}
			if !connection.Authorized {
				return connections.ErrSecretAccess
			}
			if connection.Provider != b.Provider || connection.Generation != connectionGeneration {
				return connections.ErrProviderNotAdmitted
			}
			result, err = s.repository.CreateReplayJob(ctx, connection, a, deadline, requestID, from, to)
			if err != nil {
				return err
			}
			return attach(ctx, result)
		})
	})
	return result, err
}

// Service is the trusted server application boundary. User/AI commands cannot supply gate evidence.
type Service struct {
	transactions Transactions
	repository   Repository
}

func NewService(t Transactions, r Repository) *Service { return &Service{t, r} }
func (s *Service) RecordCheck(ctx context.Context, check connections.Check) (connections.Admission, error) {
	var a connections.Admission
	if err := check.Binding.Validate(); err != nil {
		return a, err
	}
	err := s.transactions.WithinAdmission(ctx, check.Binding.Provider, check.Binding.Environment, func(ctx context.Context) error {
		var exists bool
		var err error
		a, exists, err = s.repository.Admission(ctx, check.Binding.Provider, check.Binding.Environment)
		if err != nil {
			return err
		}
		if !exists {
			a, err = connections.NewAdmission(check.Binding)
			if err != nil {
				return err
			}
		}
		previous := a.Revision()
		a, err = a.RecordCheck(check)
		if err != nil {
			return err
		}
		if !exists || a.Revision() != previous {
			if err = s.repository.SaveAdmission(ctx, a); err != nil {
				return err
			}
			return s.repository.InvalidateJobs(ctx, a)
		}
		return nil
	})
	return a, err
}
func (s *Service) Rebind(ctx context.Context, b connections.Binding) (connections.Admission, error) {
	var a connections.Admission
	if err := b.Validate(); err != nil {
		return a, err
	}
	err := s.transactions.WithinAdmission(ctx, b.Provider, b.Environment, func(ctx context.Context) error {
		var exists bool
		var err error
		a, exists, err = s.repository.Admission(ctx, b.Provider, b.Environment)
		if err != nil {
			return err
		}
		previous := a.Revision()
		a, err = a.Rebind(b)
		if err != nil {
			return err
		}
		if !exists || a.Revision() != previous {
			if err = s.repository.SaveAdmission(ctx, a); err != nil {
				return err
			}
			return s.repository.InvalidateJobs(ctx, a)
		}
		return nil
	})
	return a, err
}
func (s *Service) RequestSync(ctx context.Context, p household.Principal, id string, b connections.Binding, deadline time.Time) (jobs.Job, error) {
	return s.requestSync(ctx, p, id, b, deadline, false)
}

// EnqueueLocked is for an authenticated command already holding admission before household.
// The repository enforces that lock scope again when creating the job.
func (s *Service) EnqueueLocked(ctx context.Context, p household.Principal, id string, b connections.Binding, deadline time.Time) (jobs.Job, error) {
	a, _, err := s.repository.Admission(ctx, b.Provider, b.Environment)
	if err != nil {
		return jobs.Job{}, err
	}
	if err = a.RequireSync(b); err != nil {
		return jobs.Job{}, err
	}
	c, err := s.repository.Connection(ctx, p, id)
	if err != nil {
		return jobs.Job{}, err
	}
	return s.repository.CreateSyncJob(ctx, c, a, deadline)
}
func (s *Service) ScheduleSync(ctx context.Context, p household.Principal, id string, b connections.Binding, deadline time.Time) (jobs.Job, error) {
	return s.requestSync(ctx, p, id, b, deadline, true)
}
func (s *Service) requestSync(ctx context.Context, p household.Principal, id string, b connections.Binding, deadline time.Time, scheduled bool) (jobs.Job, error) {
	var result jobs.Job
	if err := b.Validate(); err != nil {
		return result, connections.ErrProviderNotAdmitted
	}
	err := s.transactions.WithinAdmission(ctx, b.Provider, b.Environment, func(ctx context.Context) error {
		a, _, err := s.repository.Admission(ctx, b.Provider, b.Environment)
		if err != nil {
			return err
		}
		if err = a.RequireSync(b); err != nil {
			return err
		}
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			c, err := s.repository.Connection(ctx, p, id)
			if err != nil {
				return err
			}
			if err = (connections.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}).RequireManage(p); err != nil {
				return err
			}
			if !c.Authorized || c.Provider != b.Provider {
				return connections.ErrProviderNotAdmitted
			}
			if scheduled {
				due, e := s.repository.SyncDue(ctx, id)
				if e != nil || !due {
					return e
				}
			}
			result, err = s.repository.CreateSyncJob(ctx, c, a, deadline)
			if err != nil {
				return err
			}
			return s.repository.AdvanceSyncSchedule(ctx, id)
		})
	})
	return result, err
}
func (s *Service) BeforeRead(ctx context.Context, p household.Principal, issued jobs.Job) error {
	return s.WithReadPermit(ctx, p, issued, func(context.Context) error { return nil })
}

// WithReadPermit loads private inputs while the deployment and household fences are held.
// External provider IO belongs after this transaction and requires BeforeRead again.
func (s *Service) WithReadPermit(ctx context.Context, p household.Principal, issued jobs.Job, read func(context.Context) error) error {
	if err := issued.Binding.Validate(); err != nil {
		return connections.ErrProviderNotAdmitted
	}
	return s.transactions.WithinAdmission(ctx, issued.Binding.Provider, issued.Binding.Environment, func(ctx context.Context) error {
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			if err := s.requireCurrent(ctx, p, issued); err != nil {
				return err
			}
			return read(ctx)
		})
	})
}
func (s *Service) requireCurrent(ctx context.Context, p household.Principal, issued jobs.Job) error {
	if issued.HouseholdID != p.HouseholdID() {
		return household.ErrForbidden
	}
	a, _, err := s.repository.Admission(ctx, issued.Binding.Provider, issued.Binding.Environment)
	if err != nil {
		return err
	}
	if err = a.RequireResult(issued.Binding, issued.AdmissionRevision); err != nil {
		return err
	}
	current, err := s.repository.Job(ctx, p, issued.ID)
	if err != nil {
		return err
	}
	now, err := s.repository.DatabaseTime(ctx)
	if err != nil {
		return err
	}
	if err = current.RequireAttempt(issued, now); err != nil {
		return err
	}
	c, err := s.repository.Connection(ctx, p, current.ConnectionID)
	if err != nil {
		return err
	}
	if !c.Authorized || c.Generation != issued.ConnectionGeneration || c.Provider != issued.Binding.Provider || c.SecretPurpose != issued.SecretPurpose {
		return connections.ErrProviderNotAdmitted
	}
	return nil
}

type Page struct {
	EvidenceRef, Cursor, NextCursor, Coverage string
	Gaps                                      []string
	Complete                                  bool
}

func (s *Service) CommitPage(ctx context.Context, p household.Principal, issued jobs.Job, page Page, apply func(context.Context) error) (bool, error) {
	if issued.HouseholdID != p.HouseholdID() {
		return false, household.ErrForbidden
	}
	if err := page.Validate(); err != nil {
		return false, err
	}
	if err := issued.Binding.Validate(); err != nil {
		return false, connections.ErrProviderNotAdmitted
	}
	applied := false
	err := s.transactions.WithinAdmission(ctx, issued.Binding.Provider, issued.Binding.Environment, func(ctx context.Context) error {
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			current, err := s.repository.Job(ctx, p, issued.ID)
			if err != nil {
				return err
			}
			if err = s.requireCurrent(ctx, p, issued); err != nil {
				return err
			}
			if current.Cursor != page.Cursor {
				return jobs.ErrStaleAttempt
			}
			if err = s.repository.FenceSyncResult(ctx, p, issued); err != nil {
				return err
			}
			if err = apply(ctx); err != nil {
				return err
			}
			omissions, err := s.repository.ImportOmissions(ctx, p, issued.ID)
			if err != nil {
				return err
			}
			page = page.WithOmissions(append(append([]string{}, current.Gaps...), omissions...))
			if err = page.Validate(); err != nil {
				return err
			}
			if err = s.repository.SaveCheckpoint(ctx, issued, page.NextCursor, page.Coverage, page.Gaps); err != nil {
				return err
			}
			if page.Complete {
				if err = s.repository.FinishJob(ctx, p, issued); err != nil {
					return err
				}
			}
			if err = s.repository.SaveResultReceipt(ctx, p, issued, page.EvidenceRef, PageResult); err != nil {
				return err
			}
			applied = true
			return nil
		})
	})
	if errors.Is(err, jobs.ErrStaleAttempt) || errors.Is(err, connections.ErrProviderNotAdmitted) {
		// The failed transaction has rolled back before retaining the stale evidence.
		if quarantineErr := s.quarantineResult(ctx, p, issued, page.EvidenceRef); quarantineErr != nil {
			return false, errors.Join(err, quarantineErr)
		}
		return false, nil
	}
	return applied && err == nil, err
}

// CommitProviderOutcome records a provider outcome while the exact admission,
// connection generation, job attempt and lease remain current.
func (s *Service) CommitProviderOutcome(ctx context.Context, p household.Principal, issued jobs.Job, evidence string, state jobs.State, reason jobs.Reason, delay time.Duration, apply func(context.Context) error) (bool, error) {
	if issued.HouseholdID != p.HouseholdID() {
		return false, household.ErrForbidden
	}
	if evidence == "" || len(evidence) > 2000 || issued.Binding.Validate() != nil || delay < 0 || delay > jobs.MaxRetryDelay || apply == nil {
		return false, jobs.ErrInvalidJob
	}
	applied := false
	err := s.transactions.WithinAdmission(ctx, issued.Binding.Provider, issued.Binding.Environment, func(ctx context.Context) error {
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			if err := s.repository.FenceSyncResult(ctx, p, issued); err != nil {
				return err
			}
			if err := apply(ctx); err != nil {
				return err
			}
			if err := s.repository.Quarantine(ctx, issued, evidence, "provider_outcome"); err != nil {
				return err
			}
			if err := s.repository.AcknowledgeExternalResult(ctx, p, issued); err != nil {
				return err
			}
			if err := s.repository.SetJobOutcome(ctx, p, issued, state, reason, delay); err != nil {
				return err
			}
			if err := s.repository.SaveResultReceipt(ctx, p, issued, evidence, ProviderOutcomeResult); err != nil {
				return err
			}
			applied = true
			return nil
		})
	})
	if errors.Is(err, jobs.ErrStaleAttempt) || errors.Is(err, connections.ErrProviderNotAdmitted) {
		if quarantineErr := s.quarantineResult(ctx, p, issued, evidence); quarantineErr != nil {
			return false, errors.Join(err, quarantineErr)
		}
		return false, nil
	}
	return applied && err == nil, err
}

func (s *Service) ResultReceipt(ctx context.Context, p household.Principal, issued jobs.Job, evidence string, kind ResultKind) (bool, error) {
	if issued.HouseholdID != p.HouseholdID() || evidence == "" || len(evidence) > 2000 || !kind.Valid() {
		return false, jobs.ErrInvalidJob
	}
	receipt, found, err := s.repository.ResultReceipt(ctx, p, issued, evidence)
	if err != nil || !found {
		return false, err
	}
	return receipt.Matches(p, issued, evidence, kind), nil
}

func (s *Service) EvidenceResult(ctx context.Context, p household.Principal, jobID, evidence string) (ResultKind, bool, error) {
	if jobID == "" || evidence == "" || len(evidence) > 2000 {
		return "", false, jobs.ErrInvalidJob
	}
	return s.repository.EvidenceResult(ctx, p, jobID, evidence)
}

// RetainRejectedResult associates evidence that was staged before a page failed
// domain or persistence validation. It does not advance the checkpoint or job.
func (s *Service) RetainRejectedResult(ctx context.Context, p household.Principal, issued jobs.Job, evidence string) error {
	if issued.HouseholdID != p.HouseholdID() {
		return household.ErrForbidden
	}
	if evidence == "" || len(evidence) > 2000 {
		return jobs.ErrInvalidJob
	}
	return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if err := s.repository.Quarantine(ctx, issued, evidence, "rejected_result"); err != nil {
			return err
		}
		return s.repository.SaveResultReceipt(ctx, p, issued, evidence, RejectedResult)
	})
}

// CommitFailure retains the existing terminal boundary used by replay
// reconciliation. Provider adapters should use CommitProviderOutcome.
func (s *Service) CommitFailure(ctx context.Context, p household.Principal, issued jobs.Job, evidence string, apply func(context.Context) error) (bool, error) {
	if issued.HouseholdID != p.HouseholdID() {
		return false, household.ErrForbidden
	}
	if evidence == "" || len(evidence) > 2000 || issued.Binding.Validate() != nil || apply == nil {
		return false, jobs.ErrInvalidJob
	}
	applied := false
	err := s.transactions.WithinAdmission(ctx, issued.Binding.Provider, issued.Binding.Environment, func(ctx context.Context) error {
		return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
			if err := s.repository.FenceSyncResult(ctx, p, issued); err != nil {
				return err
			}
			if err := apply(ctx); err != nil {
				return err
			}
			if err := s.repository.FailJob(ctx, p, issued); err != nil {
				return err
			}
			applied = true
			return nil
		})
	})
	if errors.Is(err, jobs.ErrStaleAttempt) || errors.Is(err, connections.ErrProviderNotAdmitted) {
		if quarantineErr := s.quarantineResult(ctx, p, issued, evidence); quarantineErr != nil {
			return false, errors.Join(err, quarantineErr)
		}
		return false, nil
	}
	return applied && err == nil, err
}

func (s *Service) quarantineResult(ctx context.Context, p household.Principal, issued jobs.Job, evidence string) error {
	return s.transactions.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, err := s.repository.Job(ctx, p, issued.ID); err != nil {
			return err
		}
		if err := s.repository.Quarantine(ctx, issued, evidence, "stale_result"); err != nil {
			return err
		}
		return s.repository.SaveResultReceipt(ctx, p, issued, evidence, StaleResult)
	})
}
