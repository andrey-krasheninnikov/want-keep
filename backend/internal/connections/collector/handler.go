package collector

import (
	"context"
	"errors"

	"github.com/pchkauu/want-keep/backend/internal/connections/credentials"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	integrations "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	contract "github.com/pchkauu/want-keep/backend/internal/integrations/contract"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobdomain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type Handler struct {
	Socket  string
	Vault   *credentials.Vault
	Service *integrations.Service
}

func (h Handler) Prepare(ctx context.Context, execution jobs.Execution) (jobs.Result, error) {
	if execution.Job.Kind != jobdomain.Sync || execution.Job.SecretPurpose != connections.BrowserSession || h.Vault == nil || h.Service == nil {
		return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.HandlerUnavailable}, nil
	}
	var applied bool
	var providerFailure bool
	var client *Client
	err := h.Vault.WithJobSecret(ctx, execution.Principal, execution.Job, connections.BrowserSession, func(session []byte) error {
		var err error
		client, err = NewClient(h.Socket, execution.Job.Binding, execution.Job.AdmissionRevision, session, execution.BeginExternal)
		if err != nil {
			return err
		}
		defer client.Close()
		if !client.Ready(ctx) {
			return ErrUnavailable
		}
		gateway, err := contract.NewGateway(execution.Job.Binding, client)
		if err != nil {
			return err
		}
		currentApplied, typedFailure, err := h.Service.Ingest(ctx, execution.Principal, execution.Job, gateway)
		applied = currentApplied
		providerFailure = typedFailure != nil
		return err
	})
	if err != nil {
		if (errors.Is(err, ErrBusy) || errors.Is(err, ErrPreflightRejected) || errors.Is(err, ErrSessionInvalid) || errors.Is(err, ErrBeforeIOUnavailable)) && client != nil && client.ExternalStarted() {
			if err := execution.RejectBeforeProviderIO(ctx); err != nil {
				return jobs.Result{}, err
			}
			if errors.Is(err, ErrSessionInvalid) {
				return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.ReauthRequired}, nil
			}
			if errors.Is(err, ErrPreflightRejected) {
				return jobs.Result{State: jobdomain.Failed, Reason: jobdomain.PermanentFailure}, nil
			}
			return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.HandlerUnavailable}, nil
		}
		if errors.Is(err, ErrUnavailable) && (client == nil || !client.ExternalStarted()) {
			return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.HandlerUnavailable}, nil
		}
		if (errors.Is(err, connections.ErrSecretAccess) || errors.Is(err, ErrSessionInvalid)) && client == nil {
			return jobs.Result{State: jobdomain.Waiting, Reason: jobdomain.ReauthRequired}, nil
		}
		return jobs.Result{}, err
	}
	page, complete, failure := client.Outcome()
	if providerFailure != failure || !page && !failure {
		return jobs.Result{}, ErrUnavailable
	}
	if applied && (complete || failure) {
		return jobs.Result{Committed: true}, nil
	}
	if applied {
		return jobs.Result{State: jobdomain.Ready}, nil
	}
	return jobs.Result{State: jobdomain.Unresolved}, nil
}
