package application

import (
	"context"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

var reportingAssets = []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH}

type SnapshotRepository interface {
	LedgerRevision(context.Context, household.Principal, string, uint64) (ledger.Revision, error)
	SaveValuationSnapshot(context.Context, household.Principal, valuation.Snapshot) error
}

type SnapshotPreparer struct {
	Rates      Service
	Repository SnapshotRepository
	Now        func() time.Time
}

func (s SnapshotPreparer) PrepareRevision(ctx context.Context, p household.Principal, operationID string, revision uint64) ([]valuation.Snapshot, error) {
	fact, err := s.Repository.LedgerRevision(ctx, p, operationID, revision)
	if err != nil {
		return nil, err
	}
	components, err := fact.Components()
	if err != nil {
		return nil, err
	}
	at, err := calendar.ParseInstant(s.Now().UTC().Format(time.RFC3339Nano))
	if err != nil {
		return nil, err
	}
	result := make([]valuation.Snapshot, 0, len(components)*len(reportingAssets))
	for index, component := range components {
		for _, target := range reportingAssets {
			item := valuation.Snapshot{OperationID: operationID, OperationRevision: revision, ComponentIndex: index, ComponentKind: component.Kind, ValuationRevision: 1, Native: component.Money, Target: target, RequestedDate: fact.CashDate, RecordedAt: at, Freshness: reporting.UnknownFreshness}
			if component.Money.Asset() == target {
				value := component.Money
				item.Reporting = &value
			} else {
				reference, lookupErr := s.Rates.Reference(ctx, component.Money.Asset(), target, fact.CashDate, false)
				if lookupErr != nil {
					return nil, lookupErr
				}
				if reference.Reason != "" {
					if reference.Reason == "missing_observation" {
						return nil, valuation.ErrRateUnavailable
					}
					item.Reason = reference.Reason
				} else {
					converted, _, _, convertErr := valuation.Convert(component.Money, target, reference.Legs)
					if convertErr != nil {
						return nil, convertErr
					}
					item.Reporting = &converted
					item.Legs = reference.Legs
					item.CoverageReasons = reference.Coverage.Reasons()
					item.Freshness = reference.Freshness
				}
			}
			if err = item.Validate(); err != nil {
				return nil, err
			}
			result = append(result, item)
		}
	}
	return result, nil
}

func (s SnapshotPreparer) Save(ctx context.Context, p household.Principal, values []valuation.Snapshot) error {
	for _, value := range values {
		if err := s.Repository.SaveValuationSnapshot(ctx, p, value); err != nil {
			return err
		}
	}
	return nil
}
