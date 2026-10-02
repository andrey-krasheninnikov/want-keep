package raiffeisen

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	"github.com/pchkauu/want-keep/backend/internal/connections/credentials"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	integrations "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobdomain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Repository interface {
	Reports
	ConnectionRecord(context.Context, household.Principal, string) (connections.ConnectionRecord, error)
	ClaimTokenRotation(context.Context, household.Principal, jobdomain.Job, connections.SecretReference) (string, error)
	CompleteTokenRotation(context.Context, household.Principal, jobdomain.Job, connections.SecretReference, string, []byte) error
	FailTokenRotation(context.Context, household.Principal, jobdomain.Job, connections.SecretReference, string) error
}
type Handler struct {
	Vault           *credentials.Vault
	Service         *integrations.Service
	Gate            *admission.Service
	Repository      Repository
	Authorizer      *Authorizer
	TransportClient func() *Client
	Now             func() time.Time
	Binding         connections.Binding
}

func (h Handler) Prepare(ctx context.Context, execution jobs.Execution) (jobs.Result, error) {
	if execution.Job.Kind != jobdomain.Sync || execution.Job.Binding.Provider != "raiffeisen" || execution.Job.SecretPurpose != connections.OAuthTokens || h.Vault == nil || h.Service == nil || h.Gate == nil || h.Repository == nil || h.Now == nil || h.Binding.Validate() != nil || h.Binding != execution.Job.Binding {
		return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.HandlerUnavailable}, nil
	}
	for {
		var result jobs.Result
		pageCommitted := false
		err := h.Vault.WithJobReference(ctx, execution.Principal, execution.Job, connections.OAuthTokens, func(ref connections.SecretReference, plain []byte) error {
			var bundle connections.TokenSet
			if json.Unmarshal(plain, &bundle) != nil {
				return connections.ErrSecretAccess
			}
			tokens := Tokens{Access: bundle.Access, ID: bundle.ID, Refresh: bundle.Refresh, Type: bundle.Type}
			if tokens.Validate() != nil {
				return connections.ErrSecretAccess
			}
			var c connections.ConnectionRecord
			if err := h.Gate.WithReadPermit(ctx, execution.Principal, execution.Job, func(ctx context.Context) error {
				var err error
				c, err = h.Repository.ConnectionRecord(ctx, execution.Principal, execution.Job.ConnectionID)
				return err
			}); err != nil {
				return err
			}
			started := false
			beforeIO := func(ctx context.Context) error {
				if started {
					return nil
				}
				if err := execution.BeginExternal(ctx); err != nil {
					return err
				}
				started = true
				return nil
			}
			expires := bundle.ExpiresAt
			if expires.IsZero() {
				expires = bundle.IssuedAt.Add(24 * time.Hour)
			}
			if bundle.IssuedAt.IsZero() || !h.Now().Before(expires.Add(-time.Hour)) {
				if h.Authorizer == nil || !h.Authorizer.Available() {
					return connections.ErrSecretAccess
				}
				var attempt string
				if err := h.Gate.WithReadPermit(ctx, execution.Principal, execution.Job, func(ctx context.Context) error {
					var err error
					attempt, err = h.Repository.ClaimTokenRotation(ctx, execution.Principal, execution.Job, ref)
					return err
				}); err != nil {
					return err
				}
				if err := beforeIO(ctx); err != nil {
					return err
				}
				authorizer := *h.Authorizer
				authorizer.Now = h.Now
				authorizer.Client = authorizer.Client.withPermit(func(ctx context.Context) error {
					if err := h.Gate.BeforeRead(ctx, execution.Principal, execution.Job); err != nil {
						return err
					}
					return beforeIO(ctx)
				})
				fresh, err := authorizer.Refresh(ctx, bundle)
				if err != nil {
					_ = h.Gate.WithReadPermit(ctx, execution.Principal, execution.Job, func(ctx context.Context) error {
						return h.Repository.FailTokenRotation(ctx, execution.Principal, execution.Job, ref, attempt)
					})
					return err
				}
				data, err := json.Marshal(fresh)
				if err != nil {
					return err
				}
				defer clear(data)
				next := ref
				next.Revision++
				ciphertext, err := h.Vault.Seal(next, data)
				if err != nil {
					return err
				}
				if err = h.Gate.WithReadPermit(ctx, execution.Principal, execution.Job, func(ctx context.Context) error {
					return h.Repository.CompleteTokenRotation(ctx, execution.Principal, execution.Job, ref, attempt, ciphertext)
				}); err != nil {
					return err
				}
				tokens = Tokens{Access: fresh.Access, ID: fresh.ID, Refresh: fresh.Refresh, Type: fresh.Type}
			}
			client := NewClient(nil)
			if h.TransportClient != nil {
				client = h.TransportClient()
			}
			gateway := &Gateway{Client: client, Tokens: tokens, Gate: h.Gate, Reports: h.Repository, Principal: execution.Principal, Job: execution.Job, Connection: c, Now: h.Now, BeforeIO: beforeIO, DeploymentBinding: h.Binding}
			applied, failure, err := h.Service.Ingest(ctx, execution.Principal, execution.Job, gateway)
			if err != nil {
				return err
			}
			if !applied {
				result = jobs.Result{State: jobdomain.Unresolved}
				return nil
			}
			if gateway.Complete || failure != nil {
				result = jobs.Result{Committed: true}
				return nil
			}
			pageCommitted = true
			return nil
		})
		if err != nil {
			if errors.Is(err, connections.ErrOAuthUnknown) {
				return jobs.Result{State: jobdomain.Unresolved}, nil
			}
			if errors.Is(err, connections.ErrSecretAccess) {
				return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.ReauthRequired}, nil
			}
			return jobs.Result{}, err
		}
		if !pageCommitted {
			return result, nil
		}
		execution, err = execution.Refresh(ctx)
		if err != nil {
			return jobs.Result{}, err
		}
	}
}

// Dispatcher retains the existing collector for providers requiring browser sessions.
type Dispatcher struct{ Native, Collector jobs.Handler }

func (d Dispatcher) Prepare(ctx context.Context, e jobs.Execution) (jobs.Result, error) {
	h := d.Collector
	if e.Job.Binding.Provider == "raiffeisen" {
		h = d.Native
	}
	if h == nil {
		return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.HandlerUnavailable}, nil
	}
	return h.Prepare(ctx, e)
}
