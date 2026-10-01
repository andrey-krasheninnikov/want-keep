package storage

import (
	"context"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) Accounts(ctx context.Context, p household.Principal, after string, limit int) ([]account.Account, string, error) {
	if limit < 1 || limit > 100 {
		return nil, "", account.ErrInvalidAccount
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Query(ctx, `SELECT id FROM want_keep.accounts WHERE household_id=$1 AND ($2='' OR id>NULLIF($2,'')::uuid) ORDER BY id LIMIT $3`, p.HouseholdID(), after, limit+1)
	if err != nil {
		return nil, "", err
	}
	var ids []string
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, "", err
		}
		ids = append(ids, id)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, "", err
	}
	next := ""
	if len(ids) > limit {
		ids = ids[:limit]
		next = ids[limit-1]
	}
	out := make([]account.Account, 0, len(ids))
	for _, id := range ids {
		a, err := s.Account(ctx, p, id)
		if err != nil {
			return nil, "", err
		}
		out = append(out, a)
	}
	return out, next, nil
}
func (s *Store) AccountTimezone(ctx context.Context, p household.Principal) (calendar.Timezone, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return calendar.Timezone{}, err
	}
	var zone string
	err = q.QueryRow(ctx, `SELECT timezone FROM want_keep.households WHERE id=$1`, p.HouseholdID()).Scan(&zone)
	if err != nil {
		return calendar.Timezone{}, err
	}
	return calendar.ParseTimezone(zone)
}
func (s *Store) ChangeAccountOwnership(ctx context.Context, p household.Principal, id string, expected uint64, next household.Ownership) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	a, err := s.Account(ctx, p, id)
	if err != nil {
		return err
	}
	if err = a.Ownership.RequireChange(p, next); err != nil {
		return err
	}
	if a.Revision != expected || expected >= command.MaxRevision {
		return command.ErrVersionConflict
	}
	if next.Scope() == household.Personal {
		var active bool
		if err = scope.tx.QueryRow(ctx, `SELECT active FROM want_keep.memberships WHERE household_id=$1 AND user_id=$2`, p.HouseholdID(), next.PersonalOwnerID()).Scan(&active); err != nil || !active {
			return household.ErrForbidden
		}
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.accounts SET scope=$3,owner_id=NULLIF($4,'')::uuid,revision=revision+1 WHERE household_id=$1 AND id=$2 AND revision=$5`, p.HouseholdID(), id, next.Scope(), string(next.PersonalOwnerID()), expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return command.ErrVersionConflict
	}
	return nil
}
func (s *Store) AccountEvent(ctx context.Context, p household.Principal, event account.Event) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if scope.principal != p {
		return household.ErrForbidden
	}
	if err = event.Validate(); err != nil {
		return err
	}
	stamp, ns := splitInstant(event.At)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.account_events(household_id,id,account_id,revision,actor_id,command_id,kind,reason,origin,eligible,at,at_ns) VALUES($1,$2,$3,$4,$5,NULLIF($6,'')::uuid,$7,$8,$9,$10,$11,$12)`, p.HouseholdID(), newID(), event.AccountID, event.Revision, p.UserID(), commands.CurrentCommandID(ctx), event.Kind, event.Reason, event.Origin, event.CelebrationEligible(), stamp, ns)
	if err != nil {
		return err
	}
	return s.EmitEvent(ctx, "account", event.AccountID, event.Revision, "account."+string(event.Kind))
}
func (s *Store) CardAliases(ctx context.Context, p household.Principal, id string) ([]account.CardAlias, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT id,label,last_four FROM want_keep.card_aliases WHERE household_id=$1 AND account_id=$2 ORDER BY id`, p.HouseholdID(), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	result := []account.CardAlias{}
	for rows.Next() {
		c := account.CardAlias{AccountID: id}
		if err = rows.Scan(&c.ID, &c.Label, &c.LastFour); err != nil {
			return nil, err
		}
		if err = c.Validate(); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}

func (s *Store) AuthorizeCommandResult(ctx context.Context, p household.Principal, r command.Result) error {
	switch r.ResourceType {
	case "account":
		_, err := s.Account(ctx, p, r.ResourceID)
		return err
	case "transaction":
		_, err := s.LedgerRevision(ctx, p, r.ResourceID, r.Revision)
		return err
	case "goal":
		_, err := s.Goal(ctx, p, r.ResourceID)
		return err
	case "connection":
		_, err := s.Connection(ctx, p, r.ResourceID)
		return err
	case "category":
		_, err := s.Category(ctx, p, r.ResourceID)
		return err
	case "merchant":
		_, err := s.Merchant(ctx, p, r.ResourceID)
		return err
	case "reconciliation":
		_, err := s.Reconciliation(ctx, p, r.ResourceID)
		return err
	case "allocation_rule":
		_, err := s.AllocationRule(ctx, p, r.ResourceID)
		return err
	case "reimbursement":
		_, err := s.Reimbursement(ctx, p, r.ResourceID)
		return err
	default:
		return household.ErrForbidden
	}
}
