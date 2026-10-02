package storage

import (
	"context"

	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
)

func (s *Store) SaveCollectorEvidence(ctx context.Context, batch ingestion.EncryptedEvidenceBatch) error {
	if batch.Validate() != nil {
		return ingestion.ErrEvidence
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(context.Background())
	at, ns := splitInstant(batch.FetchedAt)
	tag, err := tx.Exec(ctx, `INSERT INTO want_keep.collector_evidence_batches(household_id,page_reference,job_id,fetched_at,fetched_ns,disposition) VALUES($1,$2,$3,$4,$5,'staged') ON CONFLICT(household_id,page_reference) DO NOTHING`, batch.HouseholdID, batch.PageReference, batch.JobID, at, ns)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ingestion.ErrEvidence
	}
	for _, item := range batch.Items {
		if _, err = tx.Exec(ctx, `INSERT INTO want_keep.collector_evidence_items(household_id,page_reference,reference,ciphertext) VALUES($1,$2,$3,$4)`, batch.HouseholdID, batch.PageReference, item.Reference, item.Ciphertext); err != nil {
			return err
		}
	}
	return tx.Commit(ctx)
}

func (s *Store) SetCollectorEvidenceDisposition(ctx context.Context, disposition ingestion.EvidenceDisposition) error {
	if disposition.Validate() != nil {
		return ingestion.ErrEvidence
	}
	tag, err := s.pool.Exec(ctx, `UPDATE want_keep.collector_evidence_batches SET disposition=$4 WHERE household_id=$1 AND job_id=$2 AND page_reference=$3 AND disposition IN ('staged',$4)`, disposition.HouseholdID, disposition.JobID, disposition.PageReference, disposition.State)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ingestion.ErrEvidence
	}
	return nil
}

func (s *Store) StagedCollectorEvidence(ctx context.Context, householdID string, limit int) ([]ingestion.StagedEvidence, error) {
	if householdID == "" || limit < 1 || limit > 1000 {
		return nil, ingestion.ErrEvidence
	}
	rows, err := s.pool.Query(ctx, `SELECT b.job_id::text,b.page_reference FROM want_keep.collector_evidence_batches b WHERE b.household_id=$1 AND b.disposition='staged' AND EXISTS(SELECT 1 FROM want_keep.ingestion_result_receipts r WHERE (r.household_id,r.job_id,r.evidence_ref)=(b.household_id,b.job_id,b.page_reference)) ORDER BY b.fetched_at,b.page_reference LIMIT $2`, householdID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []ingestion.StagedEvidence{}
	for rows.Next() {
		item := ingestion.StagedEvidence{HouseholdID: householdID}
		if err = rows.Scan(&item.JobID, &item.PageReference); err != nil {
			return nil, err
		}
		if item.Validate() != nil {
			return nil, ingestion.ErrEvidence
		}
		result = append(result, item)
	}
	return result, rows.Err()
}

// ReconcileStagedCollectorEvidence finalizes only batches backed by one terminal receipt.
// It is a system recovery operation and must not depend on the original job actor remaining active.
func (s *Store) ReconcileStagedCollectorEvidence(ctx context.Context, limit int) (int, error) {
	if limit < 1 || limit > 1000 {
		return 0, ingestion.ErrEvidence
	}
	var reconciled int
	err := s.pool.QueryRow(ctx, `WITH candidates AS (
 SELECT b.household_id,b.page_reference,
  CASE r.kind WHEN 'page' THEN 'applied' WHEN 'provider_outcome' THEN 'provider_outcome' WHEN 'rejected_result' THEN 'rejected_result' WHEN 'stale_result' THEN 'stale_result' END AS disposition
 FROM want_keep.collector_evidence_batches b
 JOIN LATERAL (
  SELECT min(receipt.kind) AS kind
  FROM want_keep.ingestion_result_receipts receipt
  WHERE (receipt.household_id,receipt.job_id,receipt.evidence_ref)=(b.household_id,b.job_id,b.page_reference)
  HAVING count(*)=1
 ) r ON true
 WHERE b.disposition='staged'
 ORDER BY b.fetched_at,b.page_reference
 LIMIT $1
 FOR UPDATE OF b SKIP LOCKED
), updated AS (
 UPDATE want_keep.collector_evidence_batches b SET disposition=c.disposition
 FROM candidates c
 WHERE (b.household_id,b.page_reference)=(c.household_id,c.page_reference)
 RETURNING 1
)
SELECT count(*) FROM updated`, limit).Scan(&reconciled)
	return reconciled, err
}
