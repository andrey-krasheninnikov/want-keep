package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

const commandColumns = `id,kind,payload_hash,status,registered_at,registered_ns,completed_at,completed_ns,result_type,result_id::text,result_revision,failure_code,failure_revision`

func (s *Store) RegisterCommand(ctx context.Context, c command.Command) (command.Command, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return command.Command{}, err
	}
	if err = c.RequireVisible(scope.principal); err != nil {
		return command.Command{}, err
	}
	state := c.Snapshot()
	at, ns := splitInstant(state.RegisteredAt)
	// Financial audit survives D-41 cleanup. Its command reference must not execute again.
	var used bool
	err = scope.tx.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.operation_revisions WHERE household_id=$1 AND actor_id=$2 AND command_id=$3 UNION ALL SELECT 1 FROM want_keep.reservation_revisions WHERE household_id=$1 AND actor_id=$2 AND command_id=$3 UNION ALL SELECT 1 FROM want_keep.reimbursement_audit WHERE household_id=$1 AND actor_id=$2 AND command_id=$3 UNION ALL SELECT 1 FROM want_keep.connection_events WHERE household_id=$1 AND actor_id=$2 AND command_id=$3)`, state.HouseholdID, state.ActorID, state.ID).Scan(&used)
	if err != nil {
		return c, err
	}
	if used {
		existing, e := s.LoadCommand(ctx, scope.principal, c.ID())
		if errors.Is(e, command.ErrCommandNotFound) {
			return c, command.ErrCommandExpired
		}
		return existing, e
	}
	tag, err := scope.tx.Exec(ctx, `INSERT INTO want_keep.command_tombstones(household_id,actor_id,id,kind,payload_hash,status,registered_at,registered_ns) VALUES($1,$2,$3,$4,$5,'pending',$6,$7) ON CONFLICT(household_id,actor_id,id) DO NOTHING`, state.HouseholdID, state.ActorID, state.ID, state.Kind, state.PayloadHash, at, ns)
	if err != nil {
		return c, err
	}
	if tag.RowsAffected() == 1 {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.command_details(household_id,actor_id,command_id) VALUES($1,$2,$3)`, state.HouseholdID, state.ActorID, state.ID)
		if err != nil {
			return c, err
		}
	}
	return s.LoadCommand(ctx, scope.principal, c.ID())
}
func (s *Store) LoadCommand(ctx context.Context, p household.Principal, id string) (command.Command, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return command.Command{}, err
	}
	return scanCommand(q.QueryRow(ctx, `SELECT `+commandColumns+` FROM want_keep.command_tombstones WHERE household_id=$1 AND actor_id=$2 AND id=$3`, p.HouseholdID(), p.UserID(), id), p)
}
func scanCommand(row pgx.Row, p household.Principal) (command.Command, error) {
	state := command.Snapshot{HouseholdID: p.HouseholdID(), ActorID: p.UserID()}
	var at time.Time
	var ns int16
	var completed *time.Time
	var completedNS *int16
	var resultType, resultID, failure *string
	var revision, failureRevision *uint64
	err := row.Scan(&state.ID, &state.Kind, &state.PayloadHash, &state.Status, &at, &ns, &completed, &completedNS, &resultType, &resultID, &revision, &failure, &failureRevision)
	if errors.Is(err, pgx.ErrNoRows) {
		return command.Command{}, command.ErrCommandNotFound
	}
	if err != nil {
		return command.Command{}, err
	}
	state.RegisteredAt, err = restoreInstant(at, ns)
	if err != nil {
		return command.Command{}, err
	}
	if completed != nil && completedNS != nil {
		state.CompletedAt, err = restoreInstant(*completed, *completedNS)
		if err != nil {
			return command.Command{}, err
		}
	}
	if resultType != nil {
		state.Result.ResourceType = *resultType
	}
	if resultID != nil {
		state.Result.ResourceID = *resultID
	}
	if revision != nil {
		state.Result.Revision = *revision
	}
	if failure != nil {
		state.ErrorCode = *failure
	}
	if failureRevision != nil {
		state.CurrentRevision = *failureRevision
	}
	return command.Restore(state)
}
func (s *Store) SaveCommand(ctx context.Context, c command.Command) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = c.RequireVisible(scope.principal); err != nil {
		return err
	}
	state := c.Snapshot()
	if _, err = command.Restore(state); err != nil {
		return err
	}
	if c.Status() == command.Pending {
		return command.ErrInvalidCommand
	}
	at, ns := splitInstant(state.CompletedAt)
	var resultType, resultID, resultRevision, failure, failureRevision any
	if result, ok := c.Result(); ok {
		resultType, resultID, resultRevision = result.ResourceType, result.ResourceID, result.Revision
	} else {
		failure = c.ErrorCode()
		if revision, ok := c.CurrentRevision(); ok {
			failureRevision = revision
		}
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.command_tombstones SET status=$4,completed_at=$5,completed_ns=$6,result_type=$7,result_id=$8,result_revision=$9,failure_code=$10,failure_revision=$11 WHERE household_id=$1 AND actor_id=$2 AND id=$3 AND status='pending' AND kind=$12 AND payload_hash=$13`, state.HouseholdID, state.ActorID, state.ID, state.Status, at, ns, resultType, resultID, resultRevision, failure, failureRevision, state.Kind, state.PayloadHash)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return command.ErrFinalCommand
	}
	return nil
}

// RecentCommands uses an opaque registration-order cursor. It never scans another actor's commands.
func (s *Store) RecentCommands(ctx context.Context, p household.Principal, now calendar.Instant, after string, limit int) ([]command.Command, string, error) {
	if now.String() == "" || limit < 1 || limit > 100 {
		return nil, "", command.ErrInvalidCommand
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, "", err
	}
	var before *command.Snapshot
	if after != "" {
		c, e := s.LoadCommand(ctx, p, after)
		if e != nil {
			return nil, "", e
		}
		state := c.Snapshot()
		before = &state
	}
	var at any
	var ns any
	var id any
	if before != nil {
		atValue, nsValue := splitInstant(before.RegisteredAt)
		at, ns, id = atValue, nsValue, before.ID
	}
	current, sub := splitInstant(now)
	rows, err := q.Query(ctx, `SELECT `+commandColumns+` FROM want_keep.command_tombstones WHERE household_id=$1 AND actor_id=$2 AND (status='pending' OR (completed_at,completed_ns)>($3::timestamptz-INTERVAL '720 hours',$4::smallint)) AND ($5::timestamptz IS NULL OR (registered_at,registered_ns,id)<($5,$6::smallint,$7::uuid)) ORDER BY registered_at DESC,registered_ns DESC,id DESC LIMIT $8`, p.HouseholdID(), p.UserID(), current, sub, at, ns, id, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	result := []command.Command{}
	for rows.Next() {
		c, e := scanCommand(rows, p)
		if e != nil {
			return nil, "", e
		}
		result = append(result, c)
	}
	if err = rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(result) > limit {
		result = result[:limit]
		next = result[len(result)-1].ID()
	}
	return result, next, nil
}
