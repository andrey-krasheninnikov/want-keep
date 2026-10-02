package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

// pinPurchaseValuation freezes a compatible historical snapshot in the initiating member's reporting asset.
func (s *Store) pinPurchaseValuation(ctx context.Context, p household.Principal, operationID string, revision uint64) (*expenses.ValuationBasis, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return nil, err
	}
	var snapshotRevision, valuationRevision uint64
	var index int
	var target string
	err = scope.tx.QueryRow(ctx, `SELECT v.operation_revision,v.component_index,v.reporting_asset,v.valuation_revision
		FROM want_keep.valuation_snapshots v
		JOIN want_keep.identity_profiles profile ON profile.household_id=v.household_id AND profile.user_id=$4 AND profile.reporting_asset=v.reporting_asset
		JOIN want_keep.operation_revisions original ON (original.household_id,original.operation_id,original.revision)=(v.household_id,v.operation_id,v.operation_revision)
		JOIN want_keep.operation_revisions current ON (current.household_id,current.operation_id,current.revision)=(v.household_id,v.operation_id,$3)
		WHERE v.household_id=$1 AND v.operation_id=$2 AND v.operation_revision<=$3
		AND v.component_kind='expense' AND v.status='known' AND cardinality(v.coverage_reasons)=0 AND v.reporting_amount<0
		AND v.requested_date=current.cash_date AND (original.occurred_at,original.occurred_ns)=(current.occurred_at,current.occurred_ns)
		AND EXISTS(SELECT 1 FROM want_keep.postings posting WHERE (posting.household_id,posting.operation_id,posting.revision)=(v.household_id,v.operation_id,$3)
			AND posting.role='principal' AND posting.treatment IN ('','movement') AND posting.amount=v.native_amount AND posting.asset=v.native_asset)
		AND NOT EXISTS(SELECT 1 FROM want_keep.valuation_snapshots newer WHERE (newer.household_id,newer.operation_id,newer.operation_revision,newer.component_index,newer.reporting_asset)=(v.household_id,v.operation_id,v.operation_revision,v.component_index,v.reporting_asset) AND newer.valuation_revision>v.valuation_revision)
		ORDER BY v.operation_revision DESC,v.component_index LIMIT 1`, p.HouseholdID(), operationID, revision, p.UserID()).Scan(&snapshotRevision, &index, &target, &valuationRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	snapshot, found, err := s.ValuationSnapshot(ctx, p, operationID, snapshotRevision, index, money.Asset(target))
	if err != nil || !found {
		return nil, err
	}
	if snapshot.ValuationRevision != valuationRevision || snapshot.Reporting == nil {
		return nil, expenses.ErrHistoricalBasisConflict
	}
	zeroNative, _ := money.NewMoney("0", snapshot.Native.Asset())
	native, err := zeroNative.Subtract(snapshot.Native)
	if err != nil {
		return nil, err
	}
	zeroReporting, _ := money.NewMoney("0", snapshot.Target)
	reporting, err := zeroReporting.Subtract(*snapshot.Reporting)
	if err != nil {
		return nil, err
	}
	ref := fmt.Sprintf("valuation:%s:%d:%d:%s:%d", operationID, snapshotRevision, index, target, valuationRevision)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.transaction_historical_values(household_id,operation_id,operation_revision,basis_ref,native_amount,native_asset,reporting_amount,reporting_asset)
		VALUES($1,$2,$3,$4,$5::numeric,$6,$7::numeric,$8)`, p.HouseholdID(), operationID, revision, ref, native.Amount(), native.Asset(), reporting.Amount(), reporting.Asset())
	if err != nil {
		return nil, err
	}
	return &expenses.ValuationBasis{Purchase: native, Value: reporting, Ref: ref}, nil
}
