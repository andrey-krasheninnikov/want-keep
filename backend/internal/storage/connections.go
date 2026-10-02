package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

const connectionRecordColumns = `c.id,c.household_id,c.external_owner_id,c.provider,c.connection_state,c.revision,c.generation,c.history_from,p.last_success_at,COALESCE(p.coverage,'unavailable'),COALESCE(p.gaps,ARRAY['history_not_loaded']::text[])`

func scanConnectionRecord(row pgx.Row) (connections.ConnectionRecord, error) {
	var c connections.ConnectionRecord
	var date *time.Time
	var success *time.Time
	err := row.Scan(&c.ID, &c.HouseholdID, &c.OwnerID, &c.Provider, &c.State, &c.Revision, &c.Generation, &date, &success, &c.Coverage, &c.Gaps)
	if errors.Is(err, pgx.ErrNoRows) {
		return c, connections.ErrConnectionNotFound
	}
	if err != nil {
		return c, err
	}
	if date != nil {
		c.HistoryFrom, err = calendar.ParseDate(date.Format(time.DateOnly))
	}
	if success != nil {
		c.LastSuccess = *success
	}
	return c, err
}
func (s *Store) ConnectionRecord(ctx context.Context, p household.Principal, id string) (connections.ConnectionRecord, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return connections.ConnectionRecord{}, err
	}
	c, err := scanConnectionRecord(q.QueryRow(ctx, `SELECT `+connectionRecordColumns+` FROM want_keep.connections c LEFT JOIN want_keep.sync_progress p ON (p.household_id,p.connection_id,p.generation)=(c.household_id,c.id,c.generation) WHERE c.household_id=$1 AND c.id=$2`, p.HouseholdID(), id))
	return c, err
}
func (s *Store) ConnectionRecords(ctx context.Context, p household.Principal, after string, limit int) ([]connections.ConnectionRecord, error) {
	if limit < 1 || limit > 101 {
		return nil, connections.ErrInvalidConnection
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT `+connectionRecordColumns+` FROM want_keep.connections c LEFT JOIN want_keep.sync_progress p ON (p.household_id,p.connection_id,p.generation)=(c.household_id,c.id,c.generation) WHERE c.household_id=$1 AND ($2='' OR c.id>NULLIF($2,'')::uuid) ORDER BY c.id LIMIT $3`, p.HouseholdID(), after, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []connections.ConnectionRecord{}
	for rows.Next() {
		c, err := scanConnectionRecord(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
func (s *Store) CreateConnectionRecord(ctx context.Context, p household.Principal, c connections.ConnectionRecord) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal || c.Provider != "raiffeisen" || c.Revision != 1 || c.Generation != 1 || c.State != "pending" || c.HistoryFrom.String() == "" {
		return connections.ErrInvalidConnection
	}
	var active bool
	err = scope.tx.QueryRow(ctx, `SELECT active FROM want_keep.memberships WHERE household_id=$1 AND user_id=$2`, p.HouseholdID(), c.OwnerID).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return household.ErrForbidden
	}
	if err != nil {
		return err
	}
	if !active {
		return household.ErrForbidden
	}
	if err = s.CreateConnection(ctx, admission.Connection{HouseholdID: p.HouseholdID(), ID: c.ID, Provider: c.Provider, Owner: c.OwnerID, Generation: 1, SecretPurpose: connections.OAuthTokens}); err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `UPDATE want_keep.connections SET history_from=$3 WHERE household_id=$1 AND id=$2`, p.HouseholdID(), c.ID, c.HistoryFrom.String())
	return err
}
func (s *Store) ChangeConnection(ctx context.Context, p household.Principal, c connections.ConnectionRecord, state string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal || c.HouseholdID != p.HouseholdID() || c.Revision >= 9007199254740991 || (state != "reauth_required" && state != "disconnected") {
		return connections.ErrInvalidConnection
	}
	if err = s.Disconnect(ctx, c.ID); err != nil {
		return err
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.connections SET revision=revision+1,connection_state=$4 WHERE household_id=$1 AND id=$2 AND revision=$3`, p.HouseholdID(), c.ID, c.Revision, state)
	if err == nil && tag.RowsAffected() != 1 {
		return connections.ErrConnectionVersion
	}
	return err
}
func (s *Store) AuthorizeConnection(ctx context.Context, p household.Principal, c connections.ConnectionRecord) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal || p.UserID() != c.OwnerID || c.Revision >= 9007199254740991 {
		return household.ErrForbidden
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.connections SET authorized=true,connection_state='connected',revision=revision+1 WHERE household_id=$1 AND id=$2 AND revision=$3 AND generation=$4 AND external_owner_id=$5 AND connection_state!='disconnected'`, p.HouseholdID(), c.ID, c.Revision, c.Generation, p.UserID())
	if err == nil && tag.RowsAffected() != 1 {
		return connections.ErrConnectionVersion
	}
	return err
}
func (s *Store) ConnectionEvent(ctx context.Context, p household.Principal, id string, revision uint64, event string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if p != scope.principal {
		return household.ErrForbidden
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.connection_events(household_id,id,connection_id,actor_id,revision,event_type,command_id) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid)`, p.HouseholdID(), newID(), id, p.UserID(), revision, event, commands.CurrentCommandID(ctx))
	if err != nil {
		return err
	}
	return s.EmitEvent(ctx, "connection", id, revision, "connection."+event)
}
func (s *Store) SaveOAuthAttempt(ctx context.Context, a connections.OAuthAttempt) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if a.OwnerID != scope.principal.UserID() || a.HouseholdID != scope.principal.HouseholdID() || a.Phase != "ready" || len(a.Ciphertext) == 0 {
		return connections.ErrOAuthAttempt
	}
	var count int
	err = scope.tx.QueryRow(ctx, `SELECT count(*) FROM want_keep.oauth_attempts WHERE household_id=$1 AND owner_id=$2 AND expires_at>clock_timestamp()-interval '10 minutes'`, a.HouseholdID, a.OwnerID).Scan(&count)
	if err != nil {
		return err
	}
	if count >= 10 {
		return connections.ErrOAuthAttempt
	}
	var claimed bool
	err = scope.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.oauth_attempts WHERE household_id=$1 AND connection_id=$2 AND generation=$3 AND phase IN ('claimed','unknown'))`, a.HouseholdID, a.ConnectionID, a.Generation).Scan(&claimed)
	if err != nil {
		return err
	}
	if claimed {
		return connections.ErrOAuthUnknown
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.oauth_attempts(household_id,id,connection_id,owner_id,session_hash,state_hash,generation,connection_revision,expires_at,phase,ciphertext) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, a.HouseholdID, a.ID, a.ConnectionID, a.OwnerID, a.SessionHash, a.StateHash, a.Generation, a.Revision, a.ExpiresAt, a.Phase, a.Ciphertext)
	return err
}
func (s *Store) OAuthAttempt(ctx context.Context, p household.Principal, stateHash string) (connections.OAuthAttempt, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return connections.OAuthAttempt{}, err
	}
	if p != scope.principal {
		return connections.OAuthAttempt{}, household.ErrForbidden
	}
	var a connections.OAuthAttempt
	err = scope.tx.QueryRow(ctx, `SELECT household_id,id,connection_id,owner_id,session_hash,state_hash,generation,connection_revision,expires_at,phase,ciphertext FROM want_keep.oauth_attempts WHERE household_id=$1 AND state_hash=$2 AND owner_id=$3 FOR UPDATE`, p.HouseholdID(), stateHash, p.UserID()).Scan(&a.HouseholdID, &a.ID, &a.ConnectionID, &a.OwnerID, &a.SessionHash, &a.StateHash, &a.Generation, &a.Revision, &a.ExpiresAt, &a.Phase, &a.Ciphertext)
	if errors.Is(err, pgx.ErrNoRows) {
		return a, connections.ErrOAuthAttempt
	}
	return a, err
}
func (s *Store) OAuthPhase(ctx context.Context, a connections.OAuthAttempt, from, to string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if a.HouseholdID != scope.principal.HouseholdID() || a.OwnerID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.oauth_attempts SET phase=$4 WHERE household_id=$1 AND id=$2 AND phase=$3`, a.HouseholdID, a.ID, from, to)
	if err == nil && tag.RowsAffected() != 1 {
		return connections.ErrOAuthUnknown
	}
	return err
}
