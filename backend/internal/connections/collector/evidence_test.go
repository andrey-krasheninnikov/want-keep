package collector

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

func TestEvidenceStoreAcceptsMaximumPage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys")
	if err := cryptobox.Generate(path, "connections"); err != nil {
		t.Fatal(err)
	}
	keys, err := cryptobox.Load(path, "connections")
	if err != nil {
		t.Fatal(err)
	}
	repository := &evidenceRepository{}
	store, err := NewEvidenceStore(repository, keys)
	if err != nil {
		t.Fatal(err)
	}
	data := make([]byte, ingestion.MaxEvidenceBytes)
	digest := sha256.Sum256(data)
	at, _ := calendar.ParseInstant("2026-09-14T09:00:00Z")
	batch := ingestion.EvidenceBatch{
		HouseholdID: "household", JobID: "job", PageReference: "page", FetchedAt: at, Disposition: ingestion.EvidenceStaged,
		Items: []ingestion.StoredEvidence{{Reference: "item", Raw: ingestion.Evidence{ID: "source", MediaType: "application/json", Digest: hex.EncodeToString(digest[:]), Locator: "synthetic:maximum", Data: data}}},
	}
	if err = store.Save(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	if got := len(repository.saved.Items[0].Ciphertext); got > ingestion.MaxEncryptedEvidenceBytes {
		t.Fatalf("encrypted page exceeds storage contract: %d", got)
	}
}
