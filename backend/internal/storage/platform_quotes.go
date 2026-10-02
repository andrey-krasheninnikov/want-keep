package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

func (s *Store) RecordPlatformQuote(ctx context.Context, p household.Principal, quote valuation.PlatformQuote) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	if err = quote.Validate(); err != nil {
		return err
	}
	existing, found, err := s.platformQuoteByEvidence(ctx, scope.tx, p, quote.Provider, quote.EvidenceRef)
	if err != nil {
		return err
	}
	if found {
		if samePlatformQuote(existing, quote) {
			return nil
		}
		return valuation.ErrInvalidQuote
	}
	observed, observedNS := splitInstant(quote.ObservedAt)
	fetched, fetchedNS := splitInstant(quote.FetchedAt)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.platform_quotes
 (household_id,id,provider,evidence_ref,direction,base_asset,quote_asset,applicable_amount,rate_value,fee_coverage,spread_coverage,observed_at,observed_ns,fetched_at,fetched_ns)
 VALUES($1,$2,$3,$4,$5,$6,$7,$8::numeric,$9::numeric,$10,$11,$12,$13,$14,$15)`,
		p.HouseholdID(), quote.ID, quote.Provider, quote.EvidenceRef, quote.Direction, quote.Rate.Base(), quote.Rate.Quote(), quote.Amount.Amount(), quote.Rate.Value(), quote.FeeCoverage, quote.SpreadCoverage, observed, observedNS, fetched, fetchedNS)
	if err != nil {
		return err
	}
	for i, fee := range quote.Fees {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.platform_quote_fees(household_id,quote_id,position,asset,amount) VALUES($1,$2,$3,$4,$5::numeric)`, p.HouseholdID(), quote.ID, i+1, fee.Asset(), fee.Amount()); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) LatestPlatformQuote(ctx context.Context, p household.Principal, provider string, base, target money.Asset, direction valuation.Direction, amount money.Money) (valuation.PlatformQuote, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return valuation.PlatformQuote{}, false, err
	}
	return s.platformQuote(ctx, q, p, `SELECT id::text,provider,evidence_ref,direction,base_asset,quote_asset,applicable_amount::text,rate_value::text,fee_coverage,spread_coverage,observed_at,observed_ns,fetched_at,fetched_ns
 FROM want_keep.platform_quotes WHERE household_id=$1 AND provider=$2 AND base_asset=$3 AND quote_asset=$4 AND direction=$5 AND applicable_amount=$6::numeric
 ORDER BY observed_at DESC,observed_ns DESC,id DESC LIMIT 1`, p.HouseholdID(), provider, base, target, direction, amount.Amount())
}

func (s *Store) platformQuoteByEvidence(ctx context.Context, q reader, p household.Principal, provider, evidence string) (valuation.PlatformQuote, bool, error) {
	return s.platformQuote(ctx, q, p, `SELECT id::text,provider,evidence_ref,direction,base_asset,quote_asset,applicable_amount::text,rate_value::text,fee_coverage,spread_coverage,observed_at,observed_ns,fetched_at,fetched_ns
 FROM want_keep.platform_quotes WHERE household_id=$1 AND provider=$2 AND evidence_ref=$3`, p.HouseholdID(), provider, evidence)
}

func (s *Store) platformQuote(ctx context.Context, q reader, p household.Principal, query string, args ...any) (valuation.PlatformQuote, bool, error) {
	var value valuation.PlatformQuote
	var base, target, amount, rate string
	var observed, fetched time.Time
	var observedNS, fetchedNS int16
	err := q.QueryRow(ctx, query, args...).Scan(&value.ID, &value.Provider, &value.EvidenceRef, &value.Direction, &base, &target, &amount, &rate, &value.FeeCoverage, &value.SpreadCoverage, &observed, &observedNS, &fetched, &fetchedNS)
	if errors.Is(err, pgx.ErrNoRows) {
		return value, false, nil
	}
	if err != nil {
		return value, false, err
	}
	value.Amount, err = money.NewMoney(amount, money.Asset(base))
	if err != nil {
		return value, false, err
	}
	value.Rate, err = money.NewRate(money.Asset(base), money.Asset(target), rate)
	if err != nil {
		return value, false, err
	}
	value.ObservedAt, err = restoreInstant(observed, observedNS)
	if err != nil {
		return value, false, err
	}
	value.FetchedAt, err = restoreInstant(fetched, fetchedNS)
	if err != nil {
		return value, false, err
	}
	rows, err := q.Query(ctx, `SELECT asset,amount::text FROM want_keep.platform_quote_fees WHERE household_id=$1 AND quote_id=$2 ORDER BY position`, p.HouseholdID(), value.ID)
	if err != nil {
		return value, false, err
	}
	defer rows.Close()
	for rows.Next() {
		var asset, amount string
		if err = rows.Scan(&asset, &amount); err != nil {
			return value, false, err
		}
		fee, parseErr := money.NewMoney(amount, money.Asset(asset))
		if parseErr != nil {
			return value, false, parseErr
		}
		value.Fees = append(value.Fees, fee)
	}
	if err = rows.Err(); err != nil {
		return value, false, err
	}
	return value, true, value.Validate()
}

func samePlatformQuote(a, b valuation.PlatformQuote) bool {
	if a.Provider != b.Provider || a.EvidenceRef != b.EvidenceRef || a.Direction != b.Direction || a.Amount.Asset() != b.Amount.Asset() || a.Amount.Amount() != b.Amount.Amount() || a.Rate.Base() != b.Rate.Base() || a.Rate.Quote() != b.Rate.Quote() || a.Rate.Value() != b.Rate.Value() || a.FeeCoverage != b.FeeCoverage || a.SpreadCoverage != b.SpreadCoverage || a.ObservedAt != b.ObservedAt || a.FetchedAt != b.FetchedAt || len(a.Fees) != len(b.Fees) {
		return false
	}
	for i := range a.Fees {
		if a.Fees[i].Asset() != b.Fees[i].Asset() || a.Fees[i].Amount() != b.Fees[i].Amount() {
			return false
		}
	}
	return true
}
