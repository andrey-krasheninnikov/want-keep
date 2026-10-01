package application

import (
	"context"
	"errors"
	"strings"
	"time"
	"unicode/utf8"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type Repository interface {
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	Account(context.Context, household.Principal, string) (account.Account, error)
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	Refund(context.Context, household.Principal, string) (expenses.Refund, bool, error)
	ActiveRefundCountForPurchase(context.Context, household.Principal, string) (int, error)
	RefundsForOperation(context.Context, household.Principal, string) ([]expenses.Refund, error)
	CurrentRefundRevisions(context.Context, household.Principal, []string) (map[string]ledger.Revision, error)
	PurchaseValuation(context.Context, household.Principal, string, uint64) (*expenses.ValuationBasis, error)
	SaveRefunds(context.Context, household.Principal, []expenses.Refund, map[string]uint64) error
}

type JournalWriter interface {
	Append(context.Context, household.Principal, ledger.Revision, uint64) error
}

type FeeInput struct {
	AccountID string
	Amount    money.Money
	Funding   ledger.FundingKind
}

type CreateInput struct {
	PurchaseID               string
	PurchaseExpectedRevision uint64
	AccountID                string
	At                       calendar.Instant
	Amount                   money.Money
	Items                    []expenses.ItemPortion
	Fees                     []FeeInput
	Reason                   string
}

type LinkInput struct {
	RefundID, PurchaseID                             string
	RefundExpectedRevision, PurchaseExpectedRevision uint64
	ExpectedRevision                                 uint64
	Items                                            []expenses.ItemPortion
	Reason                                           string
}

type Service struct {
	repository Repository
	writer     JournalWriter
	now        func() calendar.Instant
	newID      func() string
}

func NewService(repository Repository, writer JournalWriter, now func() calendar.Instant, newID func() string) *Service {
	return &Service{repository: repository, writer: writer, now: now, newID: newID}
}

func (s *Service) Create(ctx context.Context, principal household.Principal, input CreateInput) (command.Result, error) {
	if strings.TrimSpace(input.Reason) == "" || utf8.RuneCountInString(input.Reason) > 2000 || input.At.String() == "" || input.At.Time().After(s.now().Time()) || input.Amount.Validate() != nil || input.Amount.Sign() <= 0 || len(input.Items) > 1000 || len(input.Fees) > 1000 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	purchase, err := s.current(ctx, principal, input.PurchaseID, input.PurchaseExpectedRevision)
	if err != nil {
		return command.Result{}, err
	}
	account, err := s.repository.Account(ctx, principal, input.AccountID)
	if err != nil || account.Asset != input.Amount.Asset() {
		return command.Result{}, s.reject(errOr(err, money.ErrAssetMismatch))
	}
	zone, err := s.repository.AccountTimezone(ctx, principal)
	if err != nil {
		return command.Result{}, err
	}
	date, err := input.At.DateIn(zone)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	month, err := calendar.ParseMonth(date.String()[:7])
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	operationID := s.newID()
	recordedAt := s.now()
	revision := ledger.Revision{OperationID: operationID, Revision: 1, ActorID: principal.UserID(), Reason: input.Reason, Type: ledger.Refund, State: ledger.Posted, OccurredAt: input.At, CashDate: date, ExpenseMonth: month, Timezone: zone, Origin: "manual", FeeKnowledge: ledger.KnownFees, PayerState: "not_applicable", AllocationReason: "refund_fee_allocation_unresolved", Postings: []ledger.Posting{{AccountID: input.AccountID, Money: input.Amount, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}, RecordedAt: recordedAt, HumanOverride: true, Protections: map[ledger.Field]ledger.Protection{ledger.PrincipalField: {Revision: 1}, ledger.FeesField: {Revision: 1}, ledger.DateField: {Revision: 1}}, FieldVersions: map[ledger.Field]uint64{}}
	for _, fee := range input.Fees {
		if fee.Amount.Validate() != nil || fee.Amount.Sign() <= 0 {
			return command.Result{}, commands.Rejection{Code: "invalid_money"}
		}
		feeAccount, accountErr := s.repository.Account(ctx, principal, fee.AccountID)
		if accountErr != nil || feeAccount.Asset != fee.Amount.Asset() {
			return command.Result{}, s.reject(errOr(accountErr, money.ErrAssetMismatch))
		}
		zero, _ := money.NewMoney("0", fee.Amount.Asset())
		value, _ := zero.Subtract(fee.Amount)
		revision.Postings = append(revision.Postings, ledger.Posting{AccountID: fee.AccountID, Money: value, Role: ledger.Fee, Funding: fee.Funding, Treatment: ledger.Movement})
	}
	if err = s.writer.Append(ctx, principal, revision, 0); err != nil {
		return command.Result{}, s.reject(err)
	}
	if _, err = s.save(ctx, principal, purchase, revision, input.Items, 0, input.Reason, principal.UserID(), recordedAt); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "transaction", ResourceID: operationID, Revision: 1}, nil
}

func (s *Service) Link(ctx context.Context, principal household.Principal, input LinkInput) (command.Result, error) {
	if strings.TrimSpace(input.Reason) == "" || utf8.RuneCountInString(input.Reason) > 2000 || input.ExpectedRevision > 9007199254740991 || len(input.Items) > 1000 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	refund, err := s.current(ctx, principal, input.RefundID, input.RefundExpectedRevision)
	if err != nil {
		return command.Result{}, err
	}
	purchase, err := s.current(ctx, principal, input.PurchaseID, input.PurchaseExpectedRevision)
	if err != nil {
		return command.Result{}, err
	}
	result, err := s.save(ctx, principal, purchase, refund, input.Items, input.ExpectedRevision, input.Reason, principal.UserID(), s.after(refund.RecordedAt))
	if err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "transaction", ResourceID: refund.OperationID, Revision: result.RefundRevision}, nil
}

func (s *Service) save(ctx context.Context, principal household.Principal, purchase, refund ledger.Revision, items []expenses.ItemPortion, expected uint64, reason string, actor household.UserID, recordedAt calendar.Instant) (expenses.Refund, error) {
	existing, found, err := s.repository.Refund(ctx, principal, refund.OperationID)
	if err != nil {
		return expenses.Refund{}, s.reject(err)
	}
	if found && (existing.PurchaseID != purchase.OperationID || existing.Revision != expected) || !found && expected != 0 {
		return expenses.Refund{}, commands.Rejection{Code: "version_conflict", CurrentRevision: existing.Revision}
	}
	if !found {
		count, err := s.repository.ActiveRefundCountForPurchase(ctx, principal, purchase.OperationID)
		if err != nil {
			return expenses.Refund{}, s.reject(err)
		}
		if count >= 1000 {
			return expenses.Refund{}, commands.Rejection{Code: "invalid_request"}
		}
	}
	links, err := s.repository.RefundsForOperation(ctx, principal, purchase.OperationID)
	if err != nil {
		return expenses.Refund{}, s.reject(err)
	}
	ids := make([]string, 0, len(links)+1)
	for _, link := range links {
		ids = append(ids, link.OperationID)
	}
	ids = append(ids, refund.OperationID)
	refunds, err := s.repository.CurrentRefundRevisions(ctx, principal, ids)
	if err != nil {
		return expenses.Refund{}, s.reject(err)
	}
	refunds[refund.OperationID] = refund
	basis, err := s.repository.PurchaseValuation(ctx, principal, purchase.OperationID, purchase.Revision)
	if err != nil {
		return expenses.Refund{}, s.reject(err)
	}
	result, err := recalculate(ctx, s.repository, principal, purchase, links, refunds, basis, &refundChange{refund: refund, items: items, expected: expected, reason: reason}, actor, recordedAt)
	return result, s.reject(err)
}

func (s *Service) after(boundary calendar.Instant) calendar.Instant {
	now := s.now()
	if now.Time().After(boundary.Time()) {
		return now
	}
	next, _ := calendar.ParseInstant(boundary.Time().Add(time.Nanosecond).Format(time.RFC3339Nano))
	return next
}

func (s *Service) current(ctx context.Context, principal household.Principal, id string, expected uint64) (ledger.Revision, error) {
	revision, found, err := s.repository.CurrentLedgerRevision(ctx, principal, id)
	if err != nil {
		return revision, s.reject(err)
	}
	if !found {
		return revision, commands.Rejection{Code: "not_found"}
	}
	if revision.Revision != expected || expected < 1 || expected > 9007199254740991 {
		return revision, commands.Rejection{Code: "version_conflict", CurrentRevision: revision.Revision}
	}
	return revision, nil
}

func principalAsset(revision ledger.Revision) money.Asset {
	for _, posting := range revision.Postings {
		if posting.Role == ledger.Principal && posting.MovesMoney() {
			return posting.Money.Asset()
		}
	}
	return ""
}

func errOr(err, fallback error) error {
	if err != nil {
		return err
	}
	return fallback
}

func (s *Service) reject(err error) error {
	switch {
	case errors.Is(err, expenses.ErrRefundExceedsPurchase):
		return commands.Rejection{Code: "refund_exceeds_purchase"}
	case errors.Is(err, expenses.ErrClarificationRequired):
		return commands.Rejection{Code: "clarification_required"}
	case errors.Is(err, expenses.ErrHistoricalBasisConflict):
		return commands.Rejection{Code: "decision_conflict"}
	case errors.Is(err, expenses.ErrInvalidRefund), errors.Is(err, money.ErrAssetMismatch), errors.Is(err, money.ErrInvalidMoney):
		return commands.Rejection{Code: "invalid_refund"}
	case errors.Is(err, command.ErrVersionConflict):
		return commands.Rejection{Code: "version_conflict"}
	case errors.Is(err, ledger.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	}
	return err
}
