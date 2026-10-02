package application

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	domain "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
)

type Repository interface {
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
	WithinAdmission(context.Context, string, string, func(context.Context) error) error
	ConnectionRecord(context.Context, household.Principal, string) (domain.ConnectionRecord, error)
	ConnectionRecords(context.Context, household.Principal, string, int) ([]domain.ConnectionRecord, error)
	CreateConnectionRecord(context.Context, household.Principal, domain.ConnectionRecord) error
	ChangeConnection(context.Context, household.Principal, domain.ConnectionRecord, string) error
	ConnectionEvent(context.Context, household.Principal, string, uint64, string) error
	SaveOAuthAttempt(context.Context, domain.OAuthAttempt) error
	OAuthAttempt(context.Context, household.Principal, string) (domain.OAuthAttempt, error)
	OAuthPhase(context.Context, domain.OAuthAttempt, string, string) error
	SecretReference(context.Context, household.Principal, string, domain.SecretPurpose) (domain.SecretReference, bool, error)
	SaveEncryptedSecret(context.Context, domain.SecretReference, []byte) error
	AuthorizeConnection(context.Context, household.Principal, domain.ConnectionRecord) error
	DatabaseTime(context.Context) (time.Time, error)
	Admission(context.Context, string, string) (domain.Admission, bool, error)
}
type Authorizer interface {
	Available() bool
	URL(domain.OAuthSecrets) (string, error)
	Exchange(context.Context, string, domain.OAuthSecrets) (domain.TokenSet, error)
}
type SecretBox interface {
	Available() bool
	Seal(domain.SecretReference, []byte) ([]byte, error)
	Open(domain.SecretReference, []byte) ([]byte, error)
}

type Service struct {
	Repository Repository
	Sessions   commands.SessionTransactions
	Executor   *commands.Executor
	Admission  *admission.Service
	Vault      SecretBox
	Authorizer Authorizer
	Bindings   map[string]domain.Binding
	NewID      func() string
}
type CreateInput struct {
	Provider    string
	OwnerID     household.UserID
	HistoryFrom calendar.Date
	Products    []string
}
type Authorization struct {
	ID, ConnectionID, URL string
	ExpiresAt             time.Time
}

func (s *Service) Read(ctx context.Context, p household.Principal, id string) (domain.ConnectionRecord, error) {
	c, err := s.Repository.ConnectionRecord(ctx, p, id)
	if err != nil {
		return c, err
	}
	if binding, ok := s.Bindings[c.Provider]; ok {
		c.Admission, c.HasAdmission, err = s.Repository.Admission(ctx, c.Provider, binding.Environment)
	}
	return c, err
}
func (s *Service) List(ctx context.Context, p household.Principal, after string, limit int) ([]domain.ConnectionRecord, error) {
	values, err := s.Repository.ConnectionRecords(ctx, p, after, limit)
	if err != nil {
		return nil, err
	}
	for i := range values {
		if binding, ok := s.Bindings[values[i].Provider]; ok {
			values[i].Admission, values[i].HasAdmission, err = s.Repository.Admission(ctx, values[i].Provider, binding.Environment)
			if err != nil {
				return nil, err
			}
		}
	}
	return values, nil
}

// Register remains independent; identity and admission locks precede the household mutation.
func (s *Service) Execute(ctx context.Context, a identity.Access, r commands.Request, action, id string, revision uint64, in CreateInput) (command.Command, error) {
	if _, err := s.Executor.Register(ctx, a.Principal, r); err != nil {
		return command.Command{}, err
	}
	var result command.Command
	err := s.Sessions.WithinSession(ctx, a.Token, func(ctx context.Context, current identity.Access) error {
		if current.Principal != a.Principal {
			return household.ErrForbidden
		}
		run := func(ctx context.Context) error {
			var err error
			result, err = s.Executor.ExecuteRegistered(ctx, current.Principal, r, func(ctx context.Context) (command.Result, error) {
				return s.mutate(ctx, current.Principal, action, id, revision, in)
			})
			return err
		}
		if action == "sync" {
			binding, ok := s.Bindings["raiffeisen"]
			if !ok {
				return run(ctx)
			}
			return s.Repository.WithinAdmission(ctx, binding.Provider, binding.Environment, run)
		}
		return run(ctx)
	})
	return result, err
}
func (s *Service) mutate(ctx context.Context, p household.Principal, action, id string, revision uint64, in CreateInput) (command.Result, error) {
	if action == "create" {
		if in.Provider != "raiffeisen" || in.HistoryFrom.String() == "" || len(in.Products) != 1 || in.Products[0] != "current" {
			return command.Result{}, commands.Rejection{Code: "feature_unavailable"}
		}
		c := domain.ConnectionRecord{ID: s.NewID(), HouseholdID: p.HouseholdID(), OwnerID: in.OwnerID, Provider: in.Provider, State: "pending", Revision: 1, Generation: 1, HistoryFrom: in.HistoryFrom}
		now, err := s.Repository.DatabaseTime(ctx)
		if err != nil {
			return command.Result{}, err
		}
		zone, _ := time.LoadLocation("Europe/Moscow")
		if in.HistoryFrom.String() > now.In(zone).Format(time.DateOnly) {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		if err := s.Repository.CreateConnectionRecord(ctx, p, c); err != nil {
			return command.Result{}, err
		}
		if err := s.Repository.ConnectionEvent(ctx, p, c.ID, c.Revision, "created"); err != nil {
			return command.Result{}, err
		}
		return command.Result{ResourceType: "connection", ResourceID: c.ID, Revision: c.Revision}, nil
	}
	c, err := s.Repository.ConnectionRecord(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if err = c.Require(p, revision, false); err != nil {
		return command.Result{}, commands.Rejection{Code: "version_conflict", CurrentRevision: c.Revision}
	}
	if action == "sync" {
		binding, ok := s.Bindings[c.Provider]
		if !ok {
			return command.Result{}, commands.Rejection{Code: "provider_not_admitted"}
		}
		now, err := s.Repository.DatabaseTime(ctx)
		if err != nil {
			return command.Result{}, err
		}
		if _, err = s.Admission.EnqueueLocked(ctx, p, id, binding, now.Add(24*time.Hour)); err != nil {
			if errors.Is(err, domain.ErrProviderNotAdmitted) {
				return command.Result{}, commands.Rejection{Code: "provider_not_admitted"}
			}
			return command.Result{}, err
		}
		if err = s.Repository.ConnectionEvent(ctx, p, id, c.Revision, "sync_requested"); err != nil {
			return command.Result{}, err
		}
		return command.Result{ResourceType: "connection", ResourceID: id, Revision: c.Revision}, nil
	}
	state, event := "disconnected", "disconnected"
	if action == "reauth" {
		state, event = "reauth_required", "reauth_requested"
	} else if action != "disconnect" {
		return command.Result{}, domain.ErrInvalidConnection
	}
	if err = s.Repository.ChangeConnection(ctx, p, c, state); err != nil {
		return command.Result{}, err
	}
	if err = s.Repository.ConnectionEvent(ctx, p, id, c.Revision+1, event); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "connection", ResourceID: id, Revision: c.Revision + 1}, nil
}
func randomSecret() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}
func hashSecret(value string) string {
	h := sha256.Sum256([]byte(value))
	return hex.EncodeToString(h[:])
}

func (s *Service) Begin(ctx context.Context, a identity.Access, id string, revision uint64) (Authorization, error) {
	if s.Authorizer == nil || !s.Authorizer.Available() || s.Vault == nil || !s.Vault.Available() {
		return Authorization{}, domain.ErrOAuthUnavailable
	}
	var out Authorization
	err := s.Sessions.WithinSession(ctx, a.Token, func(ctx context.Context, current identity.Access) error {
		if current.Principal != a.Principal {
			return household.ErrForbidden
		}
		return s.Repository.WithinHousehold(ctx, current.Principal, func(ctx context.Context) error {
			c, err := s.Repository.ConnectionRecord(ctx, current.Principal, id)
			if err != nil {
				return err
			}
			if err = c.Require(current.Principal, revision, true); err != nil {
				return err
			}
			if c.Provider != "raiffeisen" || c.State == "disconnected" {
				return domain.ErrOAuthAttempt
			}
			now, err := s.Repository.DatabaseTime(ctx)
			if err != nil {
				return err
			}
			secrets := domain.OAuthSecrets{}
			for _, v := range []*string{&secrets.State, &secrets.Nonce, &secrets.Verifier} {
				*v, err = randomSecret()
				if err != nil {
					return err
				}
			}
			attempt := domain.OAuthAttempt{ID: s.NewID(), ConnectionID: id, HouseholdID: c.HouseholdID, OwnerID: c.OwnerID, SessionHash: current.Session.TokenHash, StateHash: hashSecret(secrets.State), Generation: c.Generation, Revision: c.Revision, ExpiresAt: now.Add(5 * time.Minute), Phase: "ready"}
			plain, err := json.Marshal(secrets)
			if err != nil {
				return err
			}
			defer clear(plain)
			attempt.Ciphertext, err = s.Vault.Seal(attempt.Reference(), plain)
			if err != nil {
				return err
			}
			if err = s.Repository.SaveOAuthAttempt(ctx, attempt); err != nil {
				return err
			}
			url, err := s.Authorizer.URL(secrets)
			out = Authorization{ID: attempt.ID, ConnectionID: id, URL: url, ExpiresAt: attempt.ExpiresAt}
			return err
		})
	})
	return out, err
}

func (s *Service) Callback(ctx context.Context, a identity.Access, state, code string, denied bool) (string, error) {
	if len(state) > 2048 || len(state) < 32 || len(code) > 4096 || denied == (code != "") {
		return "", domain.ErrOAuthAttempt
	}
	var attempt domain.OAuthAttempt
	var now time.Time
	completed := false
	err := s.Sessions.WithinSession(ctx, a.Token, func(ctx context.Context, current identity.Access) error {
		if current.Principal != a.Principal {
			return household.ErrForbidden
		}
		return s.Repository.WithinHousehold(ctx, current.Principal, func(ctx context.Context) error {
			var err error
			attempt, err = s.Repository.OAuthAttempt(ctx, current.Principal, hashSecret(state))
			if err != nil {
				return err
			}
			c, err := s.Repository.ConnectionRecord(ctx, current.Principal, attempt.ConnectionID)
			if err != nil {
				return err
			}
			now, err = s.Repository.DatabaseTime(ctx)
			if err != nil {
				return err
			}
			if attempt.Phase == "completed" && attempt.OwnerID == current.Principal.UserID() && attempt.SessionHash == current.Session.TokenHash && c.Generation == attempt.Generation && c.Revision == attempt.Revision+1 && c.State == "connected" {
				completed = true
				return nil
			}
			if err = attempt.Require(current.Principal, current.Session.TokenHash, c, now); err != nil {
				return err
			}
			phase := "claimed"
			if denied {
				phase = "failed"
			}
			return s.Repository.OAuthPhase(ctx, attempt, "ready", phase)
		})
	})
	if err != nil {
		return "", err
	}
	if denied || completed {
		return attempt.ConnectionID, nil
	}
	plain, err := s.Vault.Open(attempt.Reference(), attempt.Ciphertext)
	if err != nil {
		return "", err
	}
	defer clear(plain)
	var secrets domain.OAuthSecrets
	if json.Unmarshal(plain, &secrets) != nil || hashSecret(secrets.State) != attempt.StateHash {
		return "", domain.ErrOAuthAttempt
	}
	if s.Authorizer == nil || !s.Authorizer.Available() {
		return "", domain.ErrOAuthUnavailable
	}
	tokens, exchangeErr := s.Authorizer.Exchange(ctx, code, secrets)
	err = s.Sessions.WithinSession(ctx, a.Token, func(ctx context.Context, current identity.Access) error {
		if current.Principal != a.Principal {
			return household.ErrForbidden
		}
		return s.Repository.WithinHousehold(ctx, current.Principal, func(ctx context.Context) error {
			c, err := s.Repository.ConnectionRecord(ctx, current.Principal, attempt.ConnectionID)
			if err != nil {
				return err
			}
			now, err = s.Repository.DatabaseTime(ctx)
			if err != nil {
				return err
			}
			if err = attempt.Require(current.Principal, current.Session.TokenHash, c, now); err != nil {
				return err
			}
			if exchangeErr != nil {
				return s.Repository.OAuthPhase(ctx, attempt, "claimed", "unknown")
			}
			ref, exists, err := s.Repository.SecretReference(ctx, current.Principal, c.ID, domain.OAuthTokens)
			if err != nil {
				return err
			}
			if !exists {
				ref = domain.SecretReference{HouseholdID: c.HouseholdID, ConnectionID: c.ID, Purpose: domain.OAuthTokens, Generation: c.Generation}
			}
			ref.Generation = c.Generation
			ref.Revision++
			if ref.Validate() != nil {
				return domain.ErrSecretAccess
			}
			plain, err := json.Marshal(tokens)
			if err != nil {
				return err
			}
			defer clear(plain)
			ciphertext, err := s.Vault.Seal(ref, plain)
			if err != nil {
				return err
			}
			if err = s.Repository.SaveEncryptedSecret(ctx, ref, ciphertext); err != nil {
				return err
			}
			if err = s.Repository.AuthorizeConnection(ctx, current.Principal, c); err != nil {
				return err
			}
			if err = s.Repository.OAuthPhase(ctx, attempt, "claimed", "completed"); err != nil {
				return err
			}
			return s.Repository.ConnectionEvent(ctx, current.Principal, c.ID, c.Revision+1, "authorized")
		})
	})
	if err != nil {
		return "", err
	}
	return attempt.ConnectionID, exchangeErr
}
