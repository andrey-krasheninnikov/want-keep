package application

import (
	"context"
	"errors"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type Projector struct{ repository Repository }

func NewProjector(repository Repository) *Projector { return &Projector{repository: repository} }

func (p *Projector) ProjectRefunds(ctx context.Context, principal household.Principal, current ledger.Revision, _ *ledger.Revision) error {
	links, err := p.repository.RefundsForOperation(ctx, principal, current.OperationID)
	if err != nil {
		return err
	}
	if len(links) == 0 {
		return nil
	}
	purchase := current
	if current.OperationID != links[0].PurchaseID {
		var found bool
		purchase, found, err = p.repository.CurrentLedgerRevision(ctx, principal, links[0].PurchaseID)
		if err != nil || !found {
			return errOr(err, ledger.ErrNotFound)
		}
	}
	ids := make([]string, 0, len(links))
	for _, link := range links {
		ids = append(ids, link.OperationID)
	}
	refunds, err := p.repository.CurrentRefundRevisions(ctx, principal, ids)
	if err != nil {
		return err
	}
	if current.Type == ledger.Refund {
		refunds[current.OperationID] = current
	}
	basis, err := p.repository.PurchaseValuation(ctx, principal, purchase.OperationID, purchase.Revision)
	if err != nil {
		if errors.Is(err, expenses.ErrHistoricalBasisConflict) {
			return commands.Rejection{Code: "decision_conflict"}
		}
		return err
	}
	_, err = recalculate(ctx, p.repository, principal, purchase, links, refunds, basis, nil, current.ActorID, current.RecordedAt)
	return err
}
