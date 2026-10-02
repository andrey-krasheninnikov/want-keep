package application

import (
	"context"
	"errors"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Journal interface {
	AppendRevision(context.Context, ledger.Revision, uint64) error
}
type ReconciliationTrigger interface {
	ReconcileAccount(context.Context, household.Principal, string) error
}
type OwnershipChangeTrigger interface {
	ReconcileAccountOwnership(context.Context, household.Principal, string) error
}
type Service struct {
	repository   CatalogRepository
	journal      Journal
	reconciler   ReconciliationTrigger
	ownerChanges OwnershipChangeTrigger
	now          func() calendar.Instant
	newID        func() string
}

func NewService(r CatalogRepository, j Journal, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: r, journal: j, now: now, newID: newID}
}

func NewServiceWithReconciliation(r CatalogRepository, j Journal, reconciler ReconciliationTrigger, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: r, journal: j, reconciler: reconciler, now: now, newID: newID}
}
func NewServiceWithOwnershipReconciliation(r CatalogRepository, j Journal, reconciler ReconciliationTrigger, ownership OwnershipChangeTrigger, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: r, journal: j, reconciler: reconciler, ownerChanges: ownership, now: now, newID: newID}
}

type CreateInput struct {
	Name      string
	Ownership household.Ownership
	Asset     money.Asset
	Date      calendar.Date
	Balance   money.Money
}
type Correction struct {
	ExpectedRevision uint64
	Date             calendar.Date
	Amounts          account.Amounts
	Reason           string
}

func (s *Service) Create(ctx context.Context, p household.Principal, input CreateInput) (command.Result, error) {
	if err := input.Ownership.RequireCreate(p); err != nil {
		return command.Result{}, commands.Rejection{Code: "forbidden"}
	}
	if input.Asset != input.Balance.Asset() {
		return command.Result{}, commands.Rejection{Code: "asset_mismatch"}
	}
	values, err := account.CashAmounts(input.Balance)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	a := account.Account{ID: s.newID(), Name: input.Name, Ownership: input.Ownership, Asset: input.Asset, Product: "cash", Revision: 1, OpeningDate: input.Date}
	if err = a.Validate(); err != nil {
		return command.Result{}, s.reject(err)
	}
	zone, err := s.repository.AccountTimezone(ctx, p)
	if err != nil {
		return command.Result{}, err
	}
	o := account.Opening{AccountID: a.ID, OperationID: s.newID(), Revision: 1, Date: input.Date, Timezone: zone, Confirmed: true, Amounts: values, ActorID: p.UserID(), Reason: "manual_opening", At: s.now()}
	if err = o.Validate(a.Asset); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.CreateAccount(ctx, a); err != nil {
		return command.Result{}, err
	}
	if err = s.writeOpening(ctx, p, a, o, 0); err != nil {
		return command.Result{}, err
	}
	if s.reconciler != nil {
		if err = s.reconciler.ReconcileAccount(ctx, p, a.ID); err != nil {
			return command.Result{}, err
		}
	}
	return s.result(ctx, p, a.ID, "created", o.Reason, "interactive")
}
func (s *Service) CorrectOpening(ctx context.Context, p household.Principal, id string, input Correction) (command.Result, error) {
	a, err := s.repository.Account(ctx, p, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if a.Revision != input.ExpectedRevision || a.Revision >= command.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	if err = input.Amounts.Validate(a.Asset); err != nil {
		return command.Result{}, s.reject(err)
	}
	if a.Product == "cash" {
		m, known := input.Amounts.Owned.Value()
		if !known {
			return command.Result{}, commands.Rejection{Code: "invalid_money"}
		}
		cash, err := account.CashAmounts(m)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
		for i, v := range cash.Fields() {
			x, _ := v.Value()
			y, ok := input.Amounts.Fields()[i].Value()
			if !ok {
				return command.Result{}, commands.Rejection{Code: "invalid_availability"}
			}
			cmp, e := x.Compare(y)
			if e != nil || cmp != 0 {
				return command.Result{}, commands.Rejection{Code: "invalid_availability"}
			}
		}
	}
	old, exists, err := s.repository.Opening(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if !exists {
		return command.Result{}, commands.Rejection{Code: "clarification_required"}
	}
	o := old
	o.Revision++
	o.Date = input.Date
	o.Amounts = input.Amounts
	o.Confirmed = true
	o.ActorID = p.UserID()
	o.Reason = input.Reason
	o.At = s.now()
	if err = o.Validate(a.Asset); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.writeOpening(ctx, p, a, o, old.Revision); err != nil {
		return command.Result{}, err
	}
	if s.reconciler != nil {
		if err = s.reconciler.ReconcileAccount(ctx, p, id); err != nil {
			return command.Result{}, err
		}
	}
	return s.result(ctx, p, id, "opening_corrected", o.Reason, "interactive")
}
func (s *Service) ChangeOwnership(ctx context.Context, p household.Principal, id string, expected uint64, next household.Ownership, reason string) (command.Result, error) {
	if len(reason) < 1 || len(reason) > 2000 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if err := s.repository.ChangeAccountOwnership(ctx, p, id, expected, next); err != nil {
		return command.Result{}, s.reject(err)
	}
	if s.ownerChanges != nil {
		if err := s.ownerChanges.ReconcileAccountOwnership(ctx, p, id); err != nil {
			return command.Result{}, err
		}
	}
	return s.result(ctx, p, id, "ownership_changed", reason, "interactive")
}
func (s *Service) writeOpening(ctx context.Context, p household.Principal, a account.Account, o account.Opening, previous uint64) error {
	at, err := o.Instant()
	if err != nil {
		return s.reject(err)
	}
	r := ledger.Revision{OperationID: o.OperationID, Revision: o.Revision, ActorID: p.UserID(), Reason: o.Reason, Type: "opening", State: "draft", OccurredAt: at, CashDate: o.Date, HumanOverride: o.Confirmed, PayerState: "not_applicable", Allocation: ledger.NotApplicableAllocation()}
	if o.Confirmed {
		value, _ := o.Amounts.Owned.Value()
		r.State = "posted"
		r.Postings = []ledger.Posting{{AccountID: a.ID, Money: value, Role: "principal"}}
	}
	if err = s.journal.AppendRevision(ctx, r, previous); err != nil {
		return err
	}
	if err = s.repository.SaveOpening(ctx, o); err != nil {
		return err
	}
	return NewProjector(s.repository).Rebuild(ctx, p, a.ID)
}
func (s *Service) result(ctx context.Context, p household.Principal, id, kind, reason, origin string) (command.Result, error) {
	a, err := s.repository.Account(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	event := account.Event{AccountID: id, Revision: a.Revision, Kind: account.EventKind(kind), Origin: account.EventOrigin(origin), Reason: reason, At: s.now()}
	if err = event.Validate(); err != nil {
		return command.Result{}, err
	}
	if err = s.repository.AccountEvent(ctx, p, event); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "account", ResourceID: id, Revision: a.Revision}, nil
}
func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, household.ErrForbidden):
		return commands.Rejection{Code: "forbidden"}
	case errors.Is(err, account.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, command.ErrVersionConflict):
		return commands.Rejection{Code: "version_conflict"}
	case errors.Is(err, money.ErrAssetMismatch):
		return commands.Rejection{Code: "asset_mismatch"}
	case errors.Is(err, money.ErrInvalidMoney):
		return commands.Rejection{Code: "invalid_money"}
	case errors.Is(err, calendar.ErrInvalidTime):
		return commands.Rejection{Code: "invalid_time"}
	case errors.Is(err, account.ErrInvalidAccount), errors.Is(err, reporting.ErrInvalidAvailability), errors.Is(err, household.ErrInvalidOwnership):
		return commands.Rejection{Code: "invalid_request"}
	}
	return err
}
