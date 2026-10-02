package collector

import (
	"context"
	"encoding/json"
	"errors"

	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

type EvidenceRepository interface {
	SaveCollectorEvidence(context.Context, ingestion.EncryptedEvidenceBatch) error
	SetCollectorEvidenceDisposition(context.Context, ingestion.EvidenceDisposition) error
	StagedCollectorEvidence(context.Context, string, int) ([]ingestion.StagedEvidence, error)
}

type EvidenceStore struct {
	repository EvidenceRepository
	keys       *cryptobox.Keyring
}

func NewEvidenceStore(repository EvidenceRepository, keys *cryptobox.Keyring) (*EvidenceStore, error) {
	if repository == nil || keys == nil || !keys.Available() {
		return nil, cryptobox.ErrUnavailable
	}
	return &EvidenceStore{repository: repository, keys: keys}, nil
}

func (s *EvidenceStore) Save(ctx context.Context, batch ingestion.EvidenceBatch) error {
	if batch.Validate() != nil {
		return ingestion.ErrEvidence
	}
	encrypted := ingestion.EncryptedEvidenceBatch{HouseholdID: batch.HouseholdID, JobID: batch.JobID, PageReference: batch.PageReference, FetchedAt: batch.FetchedAt}
	for _, item := range batch.Items {
		plain, err := json.Marshal(item.Raw)
		if err != nil {
			return ingestion.ErrEvidence
		}
		aad, err := evidenceAAD(batch.HouseholdID, batch.JobID, batch.PageReference, item.Reference)
		if err != nil {
			clear(plain)
			return ingestion.ErrEvidence
		}
		ciphertext, err := s.keys.Seal(plain, aad)
		clear(plain)
		if err != nil {
			return errors.Join(ingestion.ErrEvidence, err)
		}
		encrypted.Items = append(encrypted.Items, ingestion.EncryptedEvidenceItem{Reference: item.Reference, Ciphertext: ciphertext})
	}
	if encrypted.Validate() != nil {
		return ingestion.ErrEvidence
	}
	return s.repository.SaveCollectorEvidence(ctx, encrypted)
}

func (s *EvidenceStore) SetDisposition(ctx context.Context, disposition ingestion.EvidenceDisposition) error {
	if disposition.Validate() != nil {
		return ingestion.ErrEvidence
	}
	return s.repository.SetCollectorEvidenceDisposition(ctx, disposition)
}

func (s *EvidenceStore) Staged(ctx context.Context, householdID string, limit int) ([]ingestion.StagedEvidence, error) {
	if householdID == "" || limit < 1 || limit > 1000 {
		return nil, ingestion.ErrEvidence
	}
	return s.repository.StagedCollectorEvidence(ctx, householdID, limit)
}

func evidenceAAD(householdID, jobID, pageReference, itemReference string) ([]byte, error) {
	return json.Marshal(struct {
		Version       int    `json:"version"`
		Purpose       string `json:"purpose"`
		HouseholdID   string `json:"householdId"`
		JobID         string `json:"jobId"`
		PageReference string `json:"pageReference"`
		ItemReference string `json:"itemReference"`
	}{1, "collector-evidence", householdID, jobID, pageReference, itemReference})
}
