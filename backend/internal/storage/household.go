package storage

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func (s *Store) InitializeHousehold(ctx context.Context, h household.Household, users []household.User, members []household.Membership, zone calendar.Timezone, maximum int) error {
	if err := h.ValidateMemberLimit(members, maximum); err != nil {
		return err
	}
	if _, err := calendar.ParseTimezone(zone.String()); err != nil {
		return err
	}
	if len(users) != len(members) || len(users) == 0 {
		return household.ErrInvalidMembership
	}
	return s.transact(ctx, func(ctx context.Context, scope *transactionScope) error {
		if scope.householdLocked || scope.admissionKey != "" {
			return ErrTransactionRequired
		}
		if _, err := scope.tx.Exec(ctx, "INSERT INTO want_keep.households(id,name,timezone,max_members) VALUES($1,$2,$3,$4)", h.ID, h.Name, zone.String(), maximum); err != nil {
			return err
		}
		for i, user := range users {
			if user.ID != members[i].UserID || user.Name == "" {
				return household.ErrInvalidMembership
			}
			if _, err := scope.tx.Exec(ctx, "INSERT INTO want_keep.users(id,name) VALUES($1,$2) ON CONFLICT(id) DO NOTHING", user.ID, user.Name); err != nil {
				return err
			}
			if _, err := scope.tx.Exec(ctx, "INSERT INTO want_keep.memberships(household_id,id,user_id,active) VALUES($1,$2,$3,$4)", h.ID, members[i].ID, user.ID, members[i].Active); err != nil {
				return err
			}
		}
		return s.insertStarterCategories(ctx, scope, h.ID)
	})
}
func (s *Store) AddMember(ctx context.Context, user household.User, m household.Membership) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if m.HouseholdID != scope.principal.HouseholdID() || m.UserID != user.ID || m.ID == "" || !m.Active {
		return household.ErrInvalidMembership
	}
	var maximum int
	if err = scope.tx.QueryRow(ctx, "SELECT max_members FROM want_keep.households WHERE id=$1", m.HouseholdID).Scan(&maximum); err != nil {
		return err
	}
	rows, err := scope.tx.Query(ctx, "SELECT id,user_id,active FROM want_keep.memberships WHERE household_id=$1", m.HouseholdID)
	if err != nil {
		return err
	}
	members := []household.Membership{m}
	for rows.Next() {
		entry := household.Membership{HouseholdID: m.HouseholdID}
		if err = rows.Scan(&entry.ID, &entry.UserID, &entry.Active); err != nil {
			rows.Close()
			return err
		}
		members = append(members, entry)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return err
	}
	if err = (household.Household{ID: m.HouseholdID}).ValidateMemberLimit(members, maximum); err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.users(id,name) VALUES($1,$2) ON CONFLICT(id) DO NOTHING", user.ID, user.Name); err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, "INSERT INTO want_keep.memberships(household_id,id,user_id,active) VALUES($1,$2,$3,$4)", m.HouseholdID, m.ID, m.UserID, m.Active)
	return err
}
func (s *Store) Membership(ctx context.Context, family household.HouseholdID, user household.UserID) (household.Membership, error) {
	m := household.Membership{HouseholdID: family, UserID: user}
	var q reader = s.pool
	if ctx.Value(transactionKey{}) != nil {
		scope, err := s.scope(ctx)
		if err != nil {
			return m, err
		}
		q = scope.tx
	}
	err := q.QueryRow(ctx, "SELECT id,active FROM want_keep.memberships WHERE household_id=$1 AND user_id=$2", family, user).Scan(&m.ID, &m.Active)
	if errors.Is(err, pgx.ErrNoRows) {
		return m, household.ErrForbidden
	}
	if err != nil {
		return m, err
	}
	if _, err := m.Principal(); err != nil {
		return m, errors.Join(household.ErrForbidden, err)
	}
	return m, nil
}
