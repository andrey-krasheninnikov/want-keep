package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pchkauu/want-keep/backend/internal/valuation"
)

// ReserveRateCall is global because one private Demo key is shared by the household.
func (s *Store) ReserveRateCall(ctx context.Context, at time.Time) error {
	if at.IsZero() {
		return valuation.ErrQuotaExceeded
	}
	tx, err := s.pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	if _, err = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(hashtextextended('coingecko_demo_quota',0))"); err != nil {
		return err
	}
	at = at.UTC()
	periods := []struct {
		name  string
		start time.Time
		limit int
	}{
		{"minute", at.Truncate(time.Minute), 25},
		{"month", time.Date(at.Year(), at.Month(), 1, 0, 0, 0, 0, time.UTC), 9000},
	}
	for _, period := range periods {
		var used int
		err = tx.QueryRow(ctx, `SELECT used FROM want_keep.rate_source_usage WHERE source='coingecko' AND period=$1 AND window_start=$2`, period.name, period.start).Scan(&used)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if used >= period.limit {
			return valuation.ErrQuotaExceeded
		}
	}
	for _, period := range periods {
		if _, err = tx.Exec(ctx, `INSERT INTO want_keep.rate_source_usage(source,period,window_start,used)
 VALUES('coingecko',$1,$2,1) ON CONFLICT(source,period,window_start)
 DO UPDATE SET used=want_keep.rate_source_usage.used+1`, period.name, period.start); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}
