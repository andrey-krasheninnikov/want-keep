package application

import (
	"context"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
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
	if current.Participation.State == "linked" {
		groupLinks, err := p.repository.RefundsForMatchingGroup(ctx, principal, current.Participation.GroupID)
		if err != nil {
			return reject(err)
		}
		links = append(links, groupLinks...)
	}
	if len(links) == 0 {
		return nil
	}
	for _, link := range links {
		if link.PurchaseID != links[0].PurchaseID {
			return commands.Rejection{Code: "matching_conflict"}
		}
	}
	purchase := current
	if current.OperationID != links[0].PurchaseID {
		var found bool
		purchase, found, err = p.repository.CurrentLedgerRevision(ctx, principal, links[0].PurchaseID)
		if err != nil || !found {
			return errOr(err, ledger.ErrNotFound)
		}
	}
	links, err = p.repository.RefundsForOperation(ctx, principal, purchase.OperationID)
	if err != nil {
		return reject(err)
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
		return reject(err)
	}
	_, err = recalculate(ctx, p.repository, principal, purchase, links, refunds, basis, nil, current.ActorID, current.RecordedAt)
	return reject(err)
}
