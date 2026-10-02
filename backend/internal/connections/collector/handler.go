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
	var result jobs.Result
	var client *Client
	err := h.Vault.WithJobSecret(ctx, execution.Principal, execution.Job, connections.BrowserSession, func(session []byte) error {
		var err error
		client, err = NewClient(h.Socket, execution.Job.Binding, execution.Job.AdmissionRevision, session, func(ctx context.Context) error {
			return execution.BeginExternal(ctx)
		})
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
		for {
			applied, typedFailure, err := h.Service.Ingest(ctx, execution.Principal, execution.Job, gateway)
			if err != nil {
				return err
			}
			page, complete, failure := client.Outcome()
			if (typedFailure != nil) != failure || !page && !failure {
				return ErrUnavailable
			}
			if !applied {
				result = jobs.Result{State: jobdomain.Unresolved}
				return nil
			}
			if complete || failure {
				result = jobs.Result{Committed: true}
				return nil
			}
			execution, err = execution.Refresh(ctx)
			if err != nil {
				return err
			}
		}
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
	return result, nil
}
