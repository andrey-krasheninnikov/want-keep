package domain

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var ErrSourceAmbiguous = errors.New("source identity ambiguous")
var ErrInvalidSource = errors.New("invalid source record")
var sourceHashSyntax = regexp.MustCompile(`^[0-9a-f]{64}$`)

type SourceKey struct {
	HouseholdID                                         household.HouseholdID
	Provider, ExternalAccountID, Product, Log, RecordID string
}

func (k SourceKey) Validate() error {
	if k.HouseholdID == "" {
		return ErrInvalidSource
	}
	switch k.Provider {
	case "alfa", "raiffeisen", "ozon", "bybit", "aifory", "emcd":
	default:
		return ErrInvalidSource
	}
	for _, v := range []string{k.ExternalAccountID, k.Product, k.Log, k.RecordID} {
		if v == "" || strings.ContainsRune(v, 0) || !utf8.ValidString(v) || utf8.RuneCountInString(v) > 2000 {
			return ErrInvalidSource
		}
	}
	return nil
}
func (k SourceKey) Digest() [32]byte {
	data, _ := json.Marshal([]string{string(k.HouseholdID), k.Provider, k.ExternalAccountID, k.Product, k.Log, k.RecordID})
	return sha256.Sum256(data)
}

type SourceInput struct {
	Key                                           SourceKey
	PayloadHash, EvidenceRef, ConnectionID, JobID string
	FetchedAt                                     calendar.Instant
	// Correction is an explicit normalized provider decision, never inferred from hash alone.
	Classification   string
	UnresolvedReason string
	ExpectedRevision uint64
	Operation        *Revision
	Aliases          []SourceAlias
	SourceAsOf       calendar.Instant
}

type SourceAlias struct{ Kind, Value string }

func (i SourceInput) Validate() error {
	if err := i.Key.Validate(); err != nil {
		return err
	}
	if !sourceHashSyntax.MatchString(i.PayloadHash) || len(i.EvidenceRef) < 1 || len(i.EvidenceRef) > 2000 || i.ConnectionID == "" || i.JobID == "" || i.FetchedAt.String() == "" {
		return ErrInvalidSource
	}
	switch i.Classification {
	case "new", "correction", "ambiguous":
	default:
		return ErrInvalidSource
	}
	if len(i.UnresolvedReason) > 2000 {
		return ErrInvalidSource
	}
	return nil
}

type SourceRecord struct {
	ID                       string
	Key                      SourceKey
	Revision                 uint64
	PayloadHash, OperationID string
	Ambiguous                bool
}
type SourceOutcome struct {
	Record                       SourceRecord
	Duplicate, PreservedOverride bool
}

func (r SourceRecord) Next(i SourceInput) (SourceRecord, bool, error) {
	if err := i.Validate(); err != nil {
		return r, false, err
	}
	if r.Key != i.Key {
		return r, false, ErrSourceAmbiguous
	}
	confirmedCorrection := i.Classification == "correction" && i.ExpectedRevision == r.Revision
	if r.PayloadHash == i.PayloadHash && (!r.Ambiguous || !confirmedCorrection) && (i.Classification != "ambiguous" || r.Ambiguous) {
		return r, true, nil
	}
	if r.Revision >= 9007199254740991 {
		return r, false, ErrInvalidSource
	}
	r.Revision++
	r.PayloadHash = i.PayloadHash
	r.Ambiguous = !confirmedCorrection
	return r, false, nil
}
