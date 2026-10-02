package storage

import (
	"context"
	"errors"
	"slices"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

func (s *Store) SaveValuationSnapshot(ctx context.Context, p household.Principal, value valuation.Snapshot) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p || value.Validate() != nil {
		return valuation.ErrInvalidSnapshot
	}
	var stored uint64
	err = scope.tx.QueryRow(ctx, `SELECT valuation_revision FROM want_keep.valuation_snapshots
 WHERE household_id=$1 AND operation_id=$2 AND operation_revision=$3 AND component_index=$4 AND reporting_asset=$5
 ORDER BY valuation_revision DESC LIMIT 1`, p.HouseholdID(), value.OperationID, value.OperationRevision, value.ComponentIndex, value.Target).Scan(&stored)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return err
	}
	if stored >= command.MaxRevision {
		return command.ErrVersionConflict
	}
	if stored != 0 {
		previous, found, loadErr := s.ValuationSnapshot(ctx, p, value.OperationID, value.OperationRevision, value.ComponentIndex, value.Target)
		if loadErr != nil {
			return loadErr
		}
		if found && sameSnapshot(previous, value) {
			return nil
		}
	}
	value.ValuationRevision = stored + 1
	status := "known"
	var converted any
	if value.Reporting != nil {
		converted = value.Reporting.Amount()
		if len(value.CoverageReasons) != 0 {
			status = "partial"
		}
	} else {
		status = "unavailable"
	}
	at, ns := splitInstant(value.RecordedAt)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.valuation_snapshots
 (household_id,operation_id,operation_revision,component_index,component_kind,reporting_asset,valuation_revision,native_asset,native_amount,reporting_amount,requested_date,status,reason,coverage_reasons,freshness,recorded_at,recorded_ns)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9::numeric,$10::numeric,$11,$12,$13,$14,$15,$16,$17)`,
		p.HouseholdID(), value.OperationID, value.OperationRevision, value.ComponentIndex, value.ComponentKind, value.Target, value.ValuationRevision, value.Native.Asset(), value.Native.Amount(), converted, value.RequestedDate.String(), status, value.Reason, value.CoverageReasons, value.Freshness, at, ns)
	if err != nil {
		return err
	}
	for i, leg := range value.Legs {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.valuation_snapshot_legs
 (household_id,operation_id,operation_revision,component_index,reporting_asset,valuation_revision,position,observation_id)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, p.HouseholdID(), value.OperationID, value.OperationRevision, value.ComponentIndex, value.Target, value.ValuationRevision, i+1, leg.ID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) ValuationSnapshot(ctx context.Context, p household.Principal, operationID string, operationRevision uint64, index int, target money.Asset) (valuation.Snapshot, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return valuation.Snapshot{}, false, err
	}
	value := valuation.Snapshot{OperationID: operationID, OperationRevision: operationRevision, ComponentIndex: index, Target: target}
	var nativeAsset, nativeAmount, reason, freshness string
	var report *string
	var date, recorded time.Time
	var recordedNS int16
	err = q.QueryRow(ctx, `SELECT component_kind,valuation_revision,native_asset,native_amount::text,reporting_amount::text,requested_date,reason,coverage_reasons,freshness,recorded_at,recorded_ns
 FROM want_keep.valuation_snapshots WHERE household_id=$1 AND operation_id=$2 AND operation_revision=$3 AND component_index=$4 AND reporting_asset=$5
	 ORDER BY valuation_revision DESC LIMIT 1`, p.HouseholdID(), operationID, operationRevision, index, target).Scan(&value.ComponentKind, &value.ValuationRevision, &nativeAsset, &nativeAmount, &report, &date, &reason, &value.CoverageReasons, &freshness, &recorded, &recordedNS)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	value.Native, err = money.NewMoney(nativeAmount, money.Asset(nativeAsset))
	if err != nil {
		return value, false, err
	}
	if report != nil {
		converted, convertErr := money.NewMoney(*report, target)
		if convertErr != nil {
			return value, false, convertErr
		}
		value.Reporting = &converted
	}
	value.Reason = reason
	value.Freshness, err = reporting.ParseFreshness(freshness)
	if err != nil {
		return value, false, err
	}
	value.RequestedDate, err = calendar.ParseDate(date.Format(time.DateOnly))
	if err != nil {
		return value, false, err
	}
	value.RecordedAt, err = restoreInstant(recorded, recordedNS)
	if err != nil {
		return value, false, err
	}
	rows, err := q.Query(ctx, `SELECT o.id::text,o.provider_asset_id,o.source,o.transport,o.revision,o.requested_date,o.effective_at,o.effective_ns,o.fetched_at,o.fetched_ns,o.granularity,o.rate_base,o.rate_quote,o.rate_value::text
 FROM want_keep.valuation_snapshot_legs l JOIN want_keep.rate_observations o ON o.id=l.observation_id
 WHERE l.household_id=$1 AND l.operation_id=$2 AND l.operation_revision=$3 AND l.component_index=$4 AND l.reporting_asset=$5 AND l.valuation_revision=$6 ORDER BY l.position`,
		p.HouseholdID(), operationID, operationRevision, index, target, value.ValuationRevision)
	if err != nil {
		return value, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var leg valuation.Observation
		var requested, effective, fetched time.Time
		var effectiveNS, fetchedNS int16
		var base, quote, rate string
		if err = rows.Scan(&leg.ID, &leg.ProviderAssetID, &leg.Source, &leg.Transport, &leg.Revision, &requested, &effective, &effectiveNS, &fetched, &fetchedNS, &leg.Granularity, &base, &quote, &rate); err != nil {
			return value, false, err
		}
		leg.RequestedDate, err = calendar.ParseDate(requested.Format(time.DateOnly))
		if err != nil {
			return value, false, err
		}
		leg.EffectiveAt, err = restoreInstant(effective, effectiveNS)
		if err != nil {
			return value, false, err
		}
		leg.FetchedAt, err = restoreInstant(fetched, fetchedNS)
		if err != nil {
			return value, false, err
		}
		leg.Rate, err = money.NewRate(money.Asset(base), money.Asset(quote), rate)
		if err != nil {
			return value, false, err
		}
		value.Legs = append(value.Legs, leg)
	}
	if err = rows.Err(); err != nil {
		return value, false, err
	}
	return value, true, value.Validate()
}

func sameSnapshot(a, b valuation.Snapshot) bool {
	if a.Native.Asset() != b.Native.Asset() || a.Native.Amount() != b.Native.Amount() || a.ComponentKind != b.ComponentKind || a.RequestedDate != b.RequestedDate || a.Target != b.Target || a.Reason != b.Reason || a.Freshness != b.Freshness || !slices.Equal(a.CoverageReasons, b.CoverageReasons) || (a.Reporting == nil) != (b.Reporting == nil) || len(a.Legs) != len(b.Legs) {
		return false
	}
	if a.Reporting != nil && a.Reporting.Amount() != b.Reporting.Amount() {
		return false
	}
	for i := range a.Legs {
		if a.Legs[i].ID != b.Legs[i].ID {
			return false
		}
	}
	return true
}
