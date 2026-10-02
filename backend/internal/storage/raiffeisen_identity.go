package storage

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

// CheckSourceIdentity runs in the fenced page transaction before any journal effect.
func (s *Store) CheckSourceIdentity(ctx context.Context, p household.Principal, in ledger.SourceInput) (string, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return "", err
	}
	if p != scope.principal || in.Key.Provider != "raiffeisen" || !strings.HasPrefix(in.Key.RecordID, "camt-v1:") || in.SourceAsOf.String() == "" || len(in.Aliases) > 16 || scope.syncJobID != in.JobID || scope.syncConnectionID != in.ConnectionID {
		return "", ledger.ErrInvalidSource
	}
	classification := in.Classification
	priorBankAlias := false
	for _, alias := range in.Aliases {
		if alias.Kind == "" || alias.Value == "" || len(alias.Kind) > 128 || len(alias.Value) > 2000 {
			return "", ledger.ErrInvalidSource
		}
		encoded, _ := json.Marshal([]string{string(p.HouseholdID()), in.Key.Provider, in.Key.ExternalAccountID, in.Key.Log, alias.Kind, alias.Value})
		digest := sha256.Sum256(encoded)
		tag, insertErr := scope.tx.Exec(ctx, `INSERT INTO want_keep.source_aliases(household_id,provider,external_account_id,log,alias_digest,kind,value,canonical_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`, p.HouseholdID(), in.Key.Provider, in.Key.ExternalAccountID, in.Key.Log, digest[:], alias.Kind, alias.Value, in.Key.RecordID)
		if insertErr != nil {
			return "", insertErr
		}
		var external, log, kind, value, canonical string
		var ambiguous bool
		err = scope.tx.QueryRow(ctx, `SELECT external_account_id,log,kind,value,canonical_id,ambiguous FROM want_keep.source_aliases WHERE household_id=$1 AND provider=$2 AND alias_digest=$3`, p.HouseholdID(), in.Key.Provider, digest[:]).Scan(&external, &log, &kind, &value, &canonical, &ambiguous)
		if err != nil {
			return "", err
		}
		if tag.RowsAffected() == 0 && !ambiguous && canonical == in.Key.RecordID && (alias.Kind == "NtryRef" || alias.Kind == "AcctSvcrRef" || alias.Kind == "Refs/AcctSvcrRef" || alias.Kind == "Refs/InstrId" || alias.Kind == "Refs/TxId" || alias.Kind == "Refs/Prtry/Ref") {
			priorBankAlias = true
		}
		if external != in.Key.ExternalAccountID || log != in.Key.Log || kind != alias.Kind || value != alias.Value || canonical != in.Key.RecordID || ambiguous {
			classification = "ambiguous"
			_, err = scope.tx.Exec(ctx, `UPDATE want_keep.source_aliases SET ambiguous=true WHERE household_id=$1 AND provider=$2 AND alias_digest=$3`, p.HouseholdID(), in.Key.Provider, digest[:])
			if err != nil {
				return "", err
			}
		}
	}
	identity, _ := json.Marshal(in.Key)
	digest := in.Key.Digest()
	var full, hash string
	var at time.Time
	var ns int16
	err = scope.tx.QueryRow(ctx, `SELECT full_identity,payload_hash,source_at,source_ns FROM want_keep.raiffeisen_fact_versions WHERE household_id=$1 AND identity_digest=$2`, p.HouseholdID(), digest[:]).Scan(&full, &hash, &at, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		at, ns = splitInstant(in.SourceAsOf)
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.raiffeisen_fact_versions(household_id,identity_digest,full_identity,payload_hash,source_at,source_ns) VALUES($1,$2,$3,$4,$5,$6)`, p.HouseholdID(), digest[:], string(identity), in.PayloadHash, at, ns)
		return classification, err
	}
	if err != nil {
		return "", err
	}
	if full != string(identity) {
		return "ambiguous", nil
	}
	previous, err := restoreInstant(at, ns)
	if err != nil {
		return "", err
	}
	if classification == "ambiguous" {
		return classification, nil
	}
	if hash != in.PayloadHash {
		if !priorBankAlias || !in.SourceAsOf.Time().After(previous.Time()) {
			return "ambiguous", nil
		}
		classification = "correction"
	}
	if in.SourceAsOf.Time().After(previous.Time()) {
		at, ns = splitInstant(in.SourceAsOf)
		_, err = scope.tx.Exec(ctx, `UPDATE want_keep.raiffeisen_fact_versions SET payload_hash=$3,source_at=$4,source_ns=$5 WHERE household_id=$1 AND identity_digest=$2`, p.HouseholdID(), digest[:], in.PayloadHash, at, ns)
	}
	return classification, err
}
