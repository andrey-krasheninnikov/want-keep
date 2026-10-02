package storage

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

// SaveRateObservation appends a provider fetch; the exact observation and its predecessor remain immutable.
func (s *Store) SaveRateObservation(ctx context.Context, value valuation.Observation) (valuation.Observation, error) {
	if err := value.Validate(); err != nil {
		return valuation.Observation{}, err
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return valuation.Observation{}, err
	}
	defer tx.Rollback(context.Background())
	key := value.Source + "/" + value.Transport + "/" + value.ProviderAssetID + "/" + value.RequestedDate.String() + "/" + value.Granularity
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended($1,0))", key); err != nil {
		return valuation.Observation{}, err
	}
	var previous *string
	var revision uint64
	err = tx.QueryRow(ctx, `SELECT id::text,revision FROM want_keep.rate_observations
 WHERE source=$1 AND transport=$2 AND provider_asset_id=$3 AND requested_date=$4 AND granularity=$5
 ORDER BY revision DESC LIMIT 1`, value.Source, value.Transport, value.ProviderAssetID, value.RequestedDate.String(), value.Granularity).Scan(&previous, &revision)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return valuation.Observation{}, err
	}
	if revision >= command.MaxRevision {
		return valuation.Observation{}, command.ErrVersionConflict
	}
	value.Revision = revision + 1
	value.ID = uuid.NewString()
	effective, effectiveNS := splitInstant(value.EffectiveAt)
	fetched, fetchedNS := splitInstant(value.FetchedAt)
	_, err = tx.Exec(ctx, `INSERT INTO want_keep.rate_observations
 (id,source,transport,provider_asset_id,rate_base,rate_quote,rate_value,requested_date,effective_at,effective_ns,fetched_at,fetched_ns,granularity,revision,previous_id)
 VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8,$9,$10,$11,$12,$13,$14,$15)`,
		value.ID, value.Source, value.Transport, value.ProviderAssetID, value.Rate.Base(), value.Rate.Quote(), value.Rate.Value(), value.RequestedDate.String(), effective, effectiveNS, fetched, fetchedNS, value.Granularity, value.Revision, previous)
	if err != nil {
		return valuation.Observation{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return valuation.Observation{}, err
	}
	return value, nil
}

func (s *Store) RateObservations(ctx context.Context, base, quote money.Asset, date calendar.Date, current bool) ([]valuation.Observation, error) {
	granularity := "daily"
	if base != money.USD && base != money.RUB {
		granularity = "instant"
		if !current {
			granularity = "daily"
		}
	}
	rows, err := s.pool.Query(ctx, `SELECT DISTINCT ON (source,transport) id::text,provider_asset_id,source,transport,revision,requested_date,
 effective_at,effective_ns,fetched_at,fetched_ns,granularity,rate_value::text
 FROM want_keep.rate_observations
 WHERE rate_base=$1 AND rate_quote=$2 AND granularity=$3 AND requested_date<=$4 AND ($5 OR requested_date=$4)
 ORDER BY source,transport,requested_date DESC,revision DESC`, base, quote, granularity, date.String(), current)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []valuation.Observation{}
	for rows.Next() {
		var value valuation.Observation
		var requested time.Time
		var effective, fetched time.Time
		var effectiveNS, fetchedNS int16
		var rateValue string
		if err = rows.Scan(&value.ID, &value.ProviderAssetID, &value.Source, &value.Transport, &value.Revision, &requested, &effective, &effectiveNS, &fetched, &fetchedNS, &value.Granularity, &rateValue); err != nil {
			return nil, err
		}
		value.RequestedDate, err = calendar.ParseDate(requested.Format(time.DateOnly))
		if err != nil {
			return nil, err
		}
		value.EffectiveAt, err = restoreInstant(effective, effectiveNS)
		if err != nil {
			return nil, err
		}
		value.FetchedAt, err = restoreInstant(fetched, fetchedNS)
		if err != nil {
			return nil, err
		}
		value.Rate, err = money.NewRate(base, quote, rateValue)
		if err != nil {
			return nil, err
		}
		if err = value.Validate(); err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	return result, rows.Err()
}
