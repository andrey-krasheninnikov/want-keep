package collector

import (
	"context"
	"time"
)

const (
	reconciliationBatch    = 100
	reconciliationInterval = 30 * time.Second
)

type StagedReconciler struct {
	Reconcile func(context.Context, int) (int, error)
	Report    func(error)
}

func (r StagedReconciler) Step(ctx context.Context) error {
	_, err := r.Reconcile(ctx, reconciliationBatch)
	return err
}

func (r StagedReconciler) Run(ctx context.Context) error {
	if r.Reconcile == nil || r.Report == nil {
		return ErrUnavailable
	}
	ticker := time.NewTicker(reconciliationInterval)
	defer ticker.Stop()
	for {
		if err := r.Step(ctx); err != nil && ctx.Err() == nil {
			r.Report(err)
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}
