package application

import (
	"context"
	"errors"
	"math/rand/v2"
	"sync"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

type WorkerConfig struct {
	Kind                   jobs.Kind
	Concurrency            int
	Lease, Heartbeat, Poll time.Duration
}

func DefaultWorkerConfig(kind jobs.Kind) WorkerConfig {
	return WorkerConfig{kind, 1, time.Minute, 20 * time.Second, time.Second}
}

type ReadAdmission interface {
	BeforeRead(context.Context, household.Principal, jobs.Job) error
}

type Worker struct {
	Admission  ReadAdmission
	Repository ExecutionRepository
	Handler    Handler
	Config     WorkerConfig
	Report     func(Diagnostic)
}

func (w Worker) Run(ctx context.Context) error {
	c := w.Config
	if !c.Kind.Valid() || c.Concurrency < 1 || c.Concurrency > 16 || c.Lease < time.Second || c.Lease > 5*time.Minute || c.Heartbeat <= 0 || c.Heartbeat >= c.Lease/2 || c.Poll <= 0 {
		return jobs.ErrInvalidJob
	}
	var group sync.WaitGroup
	for range c.Concurrency {
		group.Add(1)
		go func() {
			defer group.Done()
			ticker := time.NewTicker(c.Poll)
			defer ticker.Stop()
			for {
				if ctx.Err() != nil {
					return
				}
				_ = w.Step(ctx)
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
				}
			}
		}()
	}
	group.Wait()
	return ctx.Err()
}
func (w Worker) Step(ctx context.Context) (err error) {
	started := time.Now()
	diagnostic := Diagnostic{Kind: w.Config.Kind, Stage: "queue"}
	defer func() {
		if err != nil && ctx.Err() == nil && w.Report != nil {
			w.Report(diagnostic.Failure(err, time.Since(started)))
		}
	}()

	if w.Handler == nil {
		return w.Repository.PauseReady(ctx, w.Config.Kind, jobs.HandlerUnavailable)
	}
	if err := w.Repository.ResumeWaiting(ctx, w.Config.Kind, jobs.HandlerUnavailable); err != nil {
		return err
	}
	if err := w.Repository.ResumeWaiting(ctx, w.Config.Kind, jobs.GatewayUnavailable); err != nil {
		return err
	}
	claimed, err := w.Repository.ClaimJobs(ctx, string(w.Config.Kind), 1, w.Config.Lease)
	if err != nil || len(claimed) == 0 {
		return err
	}
	j := claimed[0]
	diagnostic.JobID, diagnostic.ConnectionID, diagnostic.TransactionID = j.ID, j.ConnectionID, j.ResourceID
	diagnostic.Stage = "principal"
	p, err := w.Repository.JobPrincipal(ctx, j)
	if err != nil {
		return err
	}
	if j.Kind == jobs.Sync {
		diagnostic.Stage = "admission"
		if w.Admission == nil {
			return w.Repository.SetJobOutcome(ctx, p, j, jobs.Waiting, jobs.ProviderNotAdmitted, 0)
		}
		if err := w.Admission.BeforeRead(ctx, p, j); err != nil {
			return err
		}
	}
	execution := Execution{Job: j, Principal: p, repository: w.Repository}
	run, cancel := context.WithCancel(ctx)
	defer cancel()
	heartbeatDone := make(chan error, 1)
	stopHeartbeat := make(chan struct{})
	go func() {
		tick := time.NewTicker(w.Config.Heartbeat)
		defer tick.Stop()
		for {
			select {
			case <-stopHeartbeat:
				heartbeatDone <- nil
				return
			case <-run.Done():
				heartbeatDone <- nil
				return
			case <-tick.C:
				if err := w.Repository.Heartbeat(run, p, j, w.Config.Lease); err != nil {
					cancel()
					heartbeatDone <- err
					return
				}
			}
		}
	}()
	diagnostic.Stage = "prepare"
	result, runErr := w.Handler.Prepare(run, execution)
	close(stopHeartbeat)
	heartbeatErr := <-heartbeatDone
	cancel()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if heartbeatErr != nil {
		diagnostic.Stage = "heartbeat"
		return heartbeatErr
	}
	if runErr != nil {
		result = Result{State: jobs.Ready, Reason: jobs.TemporaryFailure}
		if j.Kind == jobs.AI {
			result = Result{State: jobs.Waiting, Reason: jobs.GatewayUnavailable, MinimumDelay: jobs.DefaultRetryPolicy().Maximum}
		}
	}
	if result.MinimumDelay < 0 || result.MinimumDelay > jobs.DefaultRetryPolicy().Maximum || result.State != jobs.Ready && result.State != jobs.Waiting && result.MinimumDelay != 0 {
		return jobs.ErrInvalidJob
	}
	delay := time.Duration(0)
	if result.State == jobs.Ready {
		delay = jobs.DefaultRetryPolicy().Delay(j.Attempt, rand.Float64())
	}
	if result.MinimumDelay > delay {
		delay = result.MinimumDelay
	}
	if result.State == jobs.Succeeded || result.Apply != nil {
		diagnostic.Stage = "commit"
		err = (Executor{Repository: w.Repository}).CommitOutcome(ctx, execution, result, delay)
		// A lost commit acknowledgement leaves reconciliation to receipt readback or lease recovery.
		return err
	}
	if result.State != jobs.Ready && result.State != jobs.Failed && result.State != jobs.Waiting && result.State != jobs.Unresolved {
		return jobs.ErrInvalidJob
	}
	diagnostic.Stage = "outcome"
	err = w.Repository.SetJobOutcome(ctx, p, j, result.State, result.Reason, delay)
	if err == nil && runErr != nil {
		diagnostic.Stage = "prepare"
	}
	return errors.Join(runErr, err)
}
