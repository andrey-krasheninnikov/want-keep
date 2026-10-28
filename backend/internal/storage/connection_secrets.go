package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	domain "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) CreateSecretGrant(ctx context.Context, g domain.SecretGrant) (domain.SecretGrant, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return g, err
	}
	if !g.Purpose.Valid() || scope.principal.HouseholdID() != g.HouseholdID || scope.principal.UserID() != g.OwnerID {
		return g, domain.ErrSecretAccess
	}
	g.ID = newID()
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.connection_secret_grants(household_id,id,connection_id,owner_id,generation,purpose,session_hash,expires_at)
 SELECT $1::uuid,$2::uuid,$3::uuid,$4::uuid,$5::want_keep.revision,$6::text,$7::text,$8::timestamptz WHERE EXISTS(SELECT 1 FROM want_keep.connections WHERE household_id=$1 AND id=$3 AND external_owner_id=$4 AND generation=$5 AND secret_purpose=$6) AND EXISTS(SELECT 1 FROM want_keep.identity_sessions WHERE token_hash=$7 AND user_id=$4 AND NOT revoked AND created_at+INTERVAL '12 hours'>clock_timestamp() AND last_activity_at+INTERVAL '30 minutes'>clock_timestamp()) AND (SELECT count(*) FROM want_keep.connection_secret_grants WHERE household_id=$1 AND connection_id=$3 AND NOT consumed AND NOT revoked AND expires_at>clock_timestamp())<10`, g.HouseholdID, g.ID, g.ConnectionID, g.OwnerID, g.Generation, g.Purpose, g.SessionHash, g.ExpiresAt)
	if err != nil {
		return g, err
	}
	if tag.RowsAffected() != 1 {
		return g, domain.ErrSecretAccess
	}
	return g, s.PrivacyAudit(ctx, scope.principal, g.ConnectionID, "secret_grant_created")
}
func (s *Store) SecretGrant(ctx context.Context, p household.Principal, id string) (domain.SecretGrant, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return domain.SecretGrant{}, err
	}
	var g domain.SecretGrant
	err = q.QueryRow(ctx, `SELECT household_id,id,connection_id,owner_id,generation,purpose,session_hash,expires_at,consumed,revoked FROM want_keep.connection_secret_grants WHERE household_id=$1 AND id=$2 AND owner_id=$3`, p.HouseholdID(), id, p.UserID()).Scan(&g.HouseholdID, &g.ID, &g.ConnectionID, &g.OwnerID, &g.Generation, &g.Purpose, &g.SessionHash, &g.ExpiresAt, &g.Consumed, &g.Revoked)
	if errors.Is(err, pgx.ErrNoRows) {
		return g, domain.ErrSecretAccess
	}
	return g, err
}
func (s *Store) ConsumeSecretGrant(ctx context.Context, g domain.SecretGrant) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal.HouseholdID() != g.HouseholdID || scope.principal.UserID() != g.OwnerID {
		return domain.ErrSecretAccess
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.connection_secret_grants SET consumed=true WHERE household_id=$1 AND id=$2 AND owner_id=$3 AND NOT consumed AND NOT revoked AND expires_at>clock_timestamp()`, g.HouseholdID, g.ID, g.OwnerID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrSecretAccess
	}
	return nil
}
func (s *Store) SecretReference(ctx context.Context, p household.Principal, id string, purpose domain.SecretPurpose) (domain.SecretReference, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return domain.SecretReference{}, false, err
	}
	r := domain.SecretReference{HouseholdID: p.HouseholdID(), ConnectionID: id, Purpose: purpose}
	err = q.QueryRow(ctx, `SELECT generation,revision FROM want_keep.connection_secrets WHERE household_id=$1 AND connection_id=$2 AND purpose=$3`, p.HouseholdID(), id, purpose).Scan(&r.Generation, &r.Revision)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, false, nil
	}
	return r, err == nil, err
}
func (s *Store) SaveEncryptedSecret(ctx context.Context, r domain.SecretReference, ciphertext []byte) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if r.Validate() != nil || r.HouseholdID != scope.principal.HouseholdID() || len(ciphertext) == 0 {
		return domain.ErrSecretAccess
	}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.connection_secrets(household_id,connection_id,purpose,generation,revision,ciphertext)
 SELECT $1::uuid,$2::uuid,$3::text,$4::want_keep.revision,$5::want_keep.revision,$6::bytea WHERE EXISTS(SELECT 1 FROM want_keep.connections WHERE household_id=$1 AND id=$2 AND generation=$4 AND external_owner_id=$7 AND secret_purpose=$3)
 ON CONFLICT(household_id,connection_id,purpose) DO UPDATE SET generation=EXCLUDED.generation,revision=EXCLUDED.revision,ciphertext=EXCLUDED.ciphertext,revoked=false WHERE connection_secrets.revision=EXCLUDED.revision-1`, r.HouseholdID, r.ConnectionID, r.Purpose, r.Generation, r.Revision, ciphertext, scope.principal.UserID())
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return domain.ErrSecretAccess
	}
	return nil
}
func (s *Store) EncryptedSecret(ctx context.Context, r domain.SecretReference) ([]byte, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return nil, err
	}
	if r.Validate() != nil || r.HouseholdID != scope.principal.HouseholdID() {
		return nil, domain.ErrSecretAccess
	}
	var data []byte
	err = scope.tx.QueryRow(ctx, `SELECT s.ciphertext FROM want_keep.connection_secrets s JOIN want_keep.connections c ON (c.household_id,c.id)=(s.household_id,s.connection_id) WHERE s.household_id=$1 AND s.connection_id=$2 AND s.purpose=$3 AND s.generation=$4 AND s.revision=$5 AND NOT s.revoked AND c.authorized AND c.generation=s.generation`, r.HouseholdID, r.ConnectionID, r.Purpose, r.Generation, r.Revision).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, domain.ErrSecretAccess
	}
	return data, err
}
func (s *Store) RevokeConnectionSecrets(ctx context.Context, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, `UPDATE want_keep.connection_secrets SET revoked=true WHERE household_id=$1 AND connection_id=$2`, scope.principal.HouseholdID(), id); err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, `UPDATE want_keep.connection_secret_grants SET revoked=true WHERE household_id=$1 AND connection_id=$2`, scope.principal.HouseholdID(), id); err != nil {
		return err
	}
	return s.PrivacyAudit(ctx, scope.principal, id, "connection_access_revoked")
}

func (s *Store) AuthorizeBrowserSession(ctx context.Context, p household.Principal, c admission.Connection) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal || c.HouseholdID != p.HouseholdID() || c.Owner != p.UserID() || c.SecretPurpose != domain.BrowserSession {
		return domain.ErrSecretAccess
	}
	var revision uint64
	err = scope.tx.QueryRow(ctx, `UPDATE want_keep.connections SET authorized=true,connection_state='connected',revision=revision+1 WHERE household_id=$1 AND id=$2 AND external_owner_id=$3 AND generation=$4 AND secret_purpose='browser_session' AND revision<9007199254740991 RETURNING revision`, p.HouseholdID(), c.ID, p.UserID(), c.Generation).Scan(&revision)
	if err != nil {
		return err
	}
	return s.ConnectionEvent(ctx, p, c.ID, revision, "authorized")
}
