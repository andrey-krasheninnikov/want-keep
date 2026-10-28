package access

import (
	"context"
	"time"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	domain "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identityapp "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Sessions interface {
	WithinSession(context.Context, identity.Token, func(context.Context, identityapp.Access) error) error
}
type Repository interface {
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	Connection(context.Context, household.Principal, string) (admission.Connection, error)
	CreateSecretGrant(context.Context, domain.SecretGrant) (domain.SecretGrant, error)
	SecretGrant(context.Context, household.Principal, string) (domain.SecretGrant, error)
	ConsumeSecretGrant(context.Context, domain.SecretGrant) error
	AuthorizeBrowserSession(context.Context, household.Principal, admission.Connection) error
	SecretReference(context.Context, household.Principal, string, domain.SecretPurpose) (domain.SecretReference, bool, error)
	Disconnect(context.Context, string) error
	PrivacyAudit(context.Context, household.Principal, string, string) error
	DatabaseTime(context.Context) (time.Time, error)
}
type Admission interface {
	WithReadPermit(context.Context, household.Principal, jobs.Job, func(context.Context) error) error
}
type Service struct {
	sessions  Sessions
	repo      Repository
	admission Admission
}

func NewService(s Sessions, r Repository, a Admission) *Service {
	return &Service{sessions: s, repo: r, admission: a}
}
func (s *Service) Begin(ctx context.Context, token identity.Token, id string, purpose domain.SecretPurpose) (domain.SecretGrant, error) {
	var grant domain.SecretGrant
	if !purpose.Valid() {
		return grant, domain.ErrSecretAccess
	}
	err := s.sessions.WithinSession(ctx, token, func(ctx context.Context, a identityapp.Access) error {
		return s.repo.WithinHousehold(ctx, a.Principal, func(ctx context.Context) error {
			c, err := s.repo.Connection(ctx, a.Principal, id)
			if err != nil || c.SecretPurpose != purpose {
				return domain.ErrSecretAccess
			}
			if err = (domain.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}).RequireAuthentication(a.Principal); err != nil {
				return domain.ErrSecretAccess
			}
			now, err := s.repo.DatabaseTime(ctx)
			if err != nil {
				return err
			}
			grant, err = s.repo.CreateSecretGrant(ctx, domain.SecretGrant{HouseholdID: a.Principal.HouseholdID(), OwnerID: a.Principal.UserID(), ConnectionID: id, Generation: c.Generation, Purpose: purpose, SessionHash: a.Session.TokenHash, ExpiresAt: now.Add(5 * time.Minute)})
			return err
		})
	})
	return grant, err
}
func (s *Service) Complete(ctx context.Context, token identity.Token, grantID string, purpose domain.SecretPurpose, save func(context.Context, domain.SecretReference) error) error {
	return s.sessions.WithinSession(ctx, token, func(ctx context.Context, a identityapp.Access) error {
		return s.repo.WithinHousehold(ctx, a.Principal, func(ctx context.Context) error {
			g, err := s.repo.SecretGrant(ctx, a.Principal, grantID)
			if err != nil {
				return domain.ErrSecretAccess
			}
			c, err := s.repo.Connection(ctx, a.Principal, g.ConnectionID)
			if err != nil || c.SecretPurpose != purpose {
				return domain.ErrSecretAccess
			}
			if err = (domain.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}).RequireAuthentication(a.Principal); err != nil {
				return domain.ErrSecretAccess
			}
			now, err := s.repo.DatabaseTime(ctx)
			if err != nil {
				return err
			}
			if err = g.Require(a.Principal, a.Session.TokenHash, purpose, c.Generation, now); err != nil {
				return err
			}
			browser := c.Provider == "alfa" && purpose == domain.BrowserSession
			if browser {
				if err = s.repo.ConsumeSecretGrant(ctx, g); err != nil {
					return err
				}
				if err = s.repo.Disconnect(ctx, c.ID); err != nil {
					return err
				}
				c, err = s.repo.Connection(ctx, a.Principal, c.ID)
				if err != nil {
					return err
				}
			}
			previous, exists, err := s.repo.SecretReference(ctx, a.Principal, c.ID, purpose)
			if err != nil {
				return err
			}
			revision := uint64(1)
			if exists {
				if previous.Revision == commands.MaxRevision {
					return commands.ErrVersionConflict
				}
				revision = previous.Revision + 1
			}
			ref := domain.SecretReference{HouseholdID: a.Principal.HouseholdID(), ConnectionID: c.ID, Purpose: purpose, Generation: c.Generation, Revision: revision}
			if err = save(ctx, ref); err != nil {
				return err
			}
			if !browser {
				if err = s.repo.ConsumeSecretGrant(ctx, g); err != nil {
					return err
				}
			} else if err = s.repo.AuthorizeBrowserSession(ctx, a.Principal, c); err != nil {
				return err
			}
			return s.repo.PrivacyAudit(ctx, a.Principal, c.ID, "secret_saved")
		})
	})
}
func (s *Service) ForJob(ctx context.Context, p household.Principal, job jobs.Job, purpose domain.SecretPurpose, read func(context.Context, domain.SecretReference) error) error {
	if !purpose.Valid() || job.Kind != "sync" || job.SecretPurpose != purpose || job.HouseholdID != p.HouseholdID() || job.ActorID != p.UserID() {
		return domain.ErrSecretAccess
	}
	return s.admission.WithReadPermit(ctx, p, job, func(ctx context.Context) error {
		ref, exists, err := s.repo.SecretReference(ctx, p, job.ConnectionID, purpose)
		if err != nil {
			return err
		}
		if !exists || ref.Generation != job.ConnectionGeneration {
			return domain.ErrSecretAccess
		}
		return read(ctx, ref)
	})
}
func (s *Service) Disconnect(ctx context.Context, token identity.Token, id string, generation uint64) error {
	return s.sessions.WithinSession(ctx, token, func(ctx context.Context, a identityapp.Access) error {
		return s.repo.WithinHousehold(ctx, a.Principal, func(ctx context.Context) error {
			c, err := s.repo.Connection(ctx, a.Principal, id)
			if err != nil {
				return domain.ErrSecretAccess
			}
			if err = (domain.ExternalOwnership{HouseholdID: c.HouseholdID, OwnerID: c.Owner}).RequireManage(a.Principal); err != nil {
				return domain.ErrSecretAccess
			}
			if c.Generation != generation {
				return commands.ErrVersionConflict
			}
			return s.repo.Disconnect(ctx, id)
		})
	})
}
