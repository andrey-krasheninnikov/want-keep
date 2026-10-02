package storage

import (
	"context"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

func (s *Store) ClaimRBOReport(ctx context.Context, p household.Principal, j jobs.Job, key string) (connections.RBOReport, bool, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return connections.RBOReport{}, false, err
	}
	if p != scope.principal || len(key) == 0 || len(key) > 2000 {
		return connections.RBOReport{}, false, connections.ErrInvalidConnection
	}
	if err = s.FenceSyncResult(ctx, p, j); err != nil {
		return connections.RBOReport{}, false, err
	}
	var unresolved bool
	if err = scope.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.raiffeisen_reports WHERE household_id=$1 AND connection_id=$2 AND generation=$3 AND request_key!=$4 AND phase IN ('claimed','unknown'))`, p.HouseholdID(), j.ConnectionID, j.ConnectionGeneration, key).Scan(&unresolved); err != nil {
		return connections.RBOReport{}, false, err
	}
	if unresolved {
		return connections.RBOReport{}, false, connections.ErrOAuthUnknown
	}
	r := connections.RBOReport{Key: key, AttemptID: newID(), Phase: "claimed"}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.raiffeisen_reports(household_id,connection_id,generation,request_key,attempt_id,phase) VALUES($1,$2,$3,$4,$5,'claimed') ON CONFLICT DO NOTHING`, p.HouseholdID(), j.ConnectionID, j.ConnectionGeneration, key, r.AttemptID)
	if err != nil {
		return r, false, err
	}
	claimed := tag.RowsAffected() == 1
	err = scope.tx.QueryRow(ctx, `SELECT attempt_id,COALESCE(report_id::text,''),phase FROM want_keep.raiffeisen_reports WHERE household_id=$1 AND connection_id=$2 AND generation=$3 AND request_key=$4`, p.HouseholdID(), j.ConnectionID, j.ConnectionGeneration, key).Scan(&r.AttemptID, &r.ReportID, &r.Phase)
	return r, claimed, err
}
func (s *Store) SaveRBOReport(ctx context.Context, p household.Principal, j jobs.Job, r connections.RBOReport, phase, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal {
		return household.ErrForbidden
	}
	if err = s.FenceSyncResult(ctx, p, j); err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.raiffeisen_reports SET phase=$6,report_id=COALESCE(NULLIF($7,'')::uuid,report_id) WHERE household_id=$1 AND connection_id=$2 AND generation=$3 AND request_key=$4 AND attempt_id=$5 AND phase=$8`, p.HouseholdID(), j.ConnectionID, j.ConnectionGeneration, r.Key, r.AttemptID, phase, id, r.Phase)
	if err == nil && tag.RowsAffected() != 1 {
		return connections.ErrOAuthUnknown
	}
	return err
}
func (s *Store) ClaimTokenRotation(ctx context.Context, p household.Principal, j jobs.Job, ref connections.SecretReference) (string, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return "", err
	}
	if p != scope.principal || ref.ConnectionID != j.ConnectionID || ref.Generation != j.ConnectionGeneration || ref.Purpose != connections.OAuthTokens {
		return "", connections.ErrSecretAccess
	}
	if err = s.FenceSyncResult(ctx, p, j); err != nil {
		return "", err
	}
	current, found, err := s.SecretReference(ctx, p, ref.ConnectionID, ref.Purpose)
	if err != nil {
		return "", err
	}
	if !found || current != ref {
		return "", connections.ErrSecretAccess
	}
	id := newID()
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.token_rotations(household_id,connection_id,secret_revision,generation,attempt_id,phase) VALUES($1,$2,$3,$4,$5,'claimed') ON CONFLICT DO NOTHING`, p.HouseholdID(), ref.ConnectionID, ref.Revision, ref.Generation, id)
	if err != nil {
		return "", err
	}
	if tag.RowsAffected() != 1 {
		return "", connections.ErrOAuthUnknown
	}
	return id, nil
}
func (s *Store) CompleteTokenRotation(ctx context.Context, p household.Principal, j jobs.Job, old connections.SecretReference, id string, ciphertext []byte) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal {
		return household.ErrForbidden
	}
	if err = s.FenceSyncResult(ctx, p, j); err != nil {
		return err
	}
	next := old
	next.Revision++
	if next.Validate() != nil {
		return connections.ErrSecretAccess
	}
	// Worker intent may be initiated by either member; secret ownership still comes from the connection.
	c, err := s.Connection(ctx, p, j.ConnectionID)
	if err != nil {
		return err
	}
	m, err := s.Membership(ctx, p.HouseholdID(), c.Owner)
	if err != nil {
		return err
	}
	_, err = m.Principal()
	if err != nil {
		return err
	}
	updated, err := scope.tx.Exec(ctx, `UPDATE want_keep.connection_secrets SET revision=$6,ciphertext=$7 WHERE household_id=$1 AND connection_id=$2 AND purpose='oauth_tokens' AND generation=$3 AND revision=$4 AND NOT revoked AND EXISTS(SELECT 1 FROM want_keep.token_rotations WHERE household_id=$1 AND connection_id=$2 AND generation=$3 AND secret_revision=$4 AND attempt_id=$5 AND phase='claimed')`, p.HouseholdID(), old.ConnectionID, old.Generation, old.Revision, id, next.Revision, ciphertext)
	if err != nil {
		return err
	}
	if updated.RowsAffected() != 1 {
		return connections.ErrSecretAccess
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.token_rotations SET phase='completed' WHERE household_id=$1 AND connection_id=$2 AND secret_revision=$3 AND attempt_id=$4 AND phase='claimed'`, p.HouseholdID(), old.ConnectionID, old.Revision, id)
	if err == nil && tag.RowsAffected() != 1 {
		return connections.ErrOAuthUnknown
	}
	if err != nil {
		return err
	}
	return s.ConnectionEvent(ctx, p, c.ID, 1, "rotation_completed")
}
func (s *Store) FailTokenRotation(ctx context.Context, p household.Principal, j jobs.Job, ref connections.SecretReference, id string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal {
		return household.ErrForbidden
	}
	if err = s.FenceSyncResult(ctx, p, j); err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.token_rotations SET phase='unknown' WHERE household_id=$1 AND connection_id=$2 AND secret_revision=$3 AND attempt_id=$4 AND phase='claimed'`, p.HouseholdID(), ref.ConnectionID, ref.Revision, id)
	return err
}
