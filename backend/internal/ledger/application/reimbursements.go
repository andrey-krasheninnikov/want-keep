package application

import (
	"context"
	"errors"
	"slices"
	"sort"
	"strings"
	"unicode/utf8"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type ReimbursementFilter struct {
	MemberID household.MembershipID
	Asset    money.Asset
	State    ledger.ReimbursementState
}

type ReimbursementCursor struct {
	At       calendar.Instant
	ID       string
	Revision uint64
}

type ReimbursementRepository interface {
	Reimbursement(context.Context, household.Principal, string) (ledger.Reimbursement, error)
	ReimbursementRevision(context.Context, household.Principal, string, uint64) (ledger.Reimbursement, error)
	Reimbursements(context.Context, household.Principal, ReimbursementFilter, ReimbursementCursor, int) ([]ledger.Reimbursement, *ReimbursementCursor, error)
	ReimbursementHistory(context.Context, household.Principal, string, ReimbursementCursor, int) ([]ledger.Reimbursement, *ReimbursementCursor, error)
	ReimbursementDecision(context.Context, household.Principal, string) (ledger.ReimbursementDecision, bool, error)
	ReimbursementDecisionUndone(context.Context, household.Principal, string) (bool, error)
	SaveReimbursement(context.Context, household.Principal, ledger.Reimbursement, uint64, ledger.ReimbursementDecision) error
	ReimbursementIDsForOperation(context.Context, household.Principal, string) ([]string, error)
	SettlementOperationIDsForAccount(context.Context, household.Principal, string) ([]string, error)
	ActiveTransferUsage(context.Context, household.Principal, string, money.Asset) (money.Money, error)
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	MatchingGroup(context.Context, household.Principal, string) (matching.Group, error)
	Account(context.Context, household.Principal, string) (account.Account, error)
	HouseholdMemberships(context.Context, household.Principal) ([]household.Membership, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}

type ReimbursementService struct {
	repository ReimbursementRepository
	now        func() calendar.Instant
	newID      func() string
}

func NewReimbursementService(repository ReimbursementRepository, now func() calendar.Instant, newID func() string) *ReimbursementService {
	return &ReimbursementService{repository: repository, now: now, newID: newID}
}

type ReimbursementCreateInput struct {
	CreditorMemberID, DebtorMemberID household.MembershipID
	Amount                           money.Money
	ExpenseID, Reason                string
}

func (s *ReimbursementService) Create(ctx context.Context, principal household.Principal, input ReimbursementCreateInput) (command.Result, error) {
	if err := s.requireMembers(ctx, principal, input.CreditorMemberID, input.DebtorMemberID); err != nil {
		return command.Result{}, err
	}
	if input.Amount.Validate() != nil || input.Amount.Sign() <= 0 || !validReimbursementReason(input.Reason) {
		return command.Result{}, s.reject(ledger.ErrInvalidReimbursement)
	}
	expenseRevision := uint64(0)
	if input.ExpenseID != "" {
		expense, found, err := s.repository.CurrentLedgerRevision(ctx, principal, input.ExpenseID)
		if err != nil || !found {
			if err == nil {
				err = ledger.ErrNotFound
			}
			return command.Result{}, s.reject(err)
		}
		if err = requireReimbursementExpense(expense); err != nil {
			return command.Result{}, s.reject(err)
		}
		expenseRevision = expense.Revision
	}
	id, decisionID := s.newID(), s.newID()
	value, err := ledger.NewReimbursement(id, input.CreditorMemberID, input.DebtorMemberID, input.Amount, input.ExpenseID, expenseRevision, input.Reason, principal.UserID(), s.now(), decisionID)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	decision := ledger.ReimbursementDecision{ID: decisionID, Kind: "create", ReimbursementID: id, ActorID: principal.UserID(), At: value.RecordedAt, Reason: input.Reason, Before: 0, After: 1, Fields: []ledger.ReimbursementField{ledger.ReimbursementPartiesField, ledger.ReimbursementPrincipalField, ledger.ReimbursementReasonField}}
	if input.ExpenseID != "" {
		decision.Fields = append(decision.Fields, ledger.ReimbursementExpenseField)
	}
	if err = s.repository.SaveReimbursement(ctx, principal, value, 0, decision); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "reimbursement", id, 1, "reimbursement.created"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "reimbursement", ResourceID: id, Revision: 1}, nil
}

type ReimbursementCorrectionInput struct {
	ExpectedRevision                 uint64
	CreditorMemberID, DebtorMemberID *household.MembershipID
	Amount                           *money.Money
	ExpenseID                        *string
	DebtReason                       *string
	Voided                           *bool
	Reason                           string
}

func (s *ReimbursementService) Correct(ctx context.Context, principal household.Principal, id string, input ReimbursementCorrectionInput) (command.Result, error) {
	if input.ExpectedRevision < 1 || !validReimbursementReason(input.Reason) {
		return command.Result{}, s.reject(ledger.ErrInvalidReimbursement)
	}
	current, err := s.repository.Reimbursement(ctx, principal, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != input.ExpectedRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict", CurrentRevision: current.Revision}
	}
	creditor, debtor := current.CreditorMemberID, current.DebtorMemberID
	if input.CreditorMemberID != nil {
		creditor = *input.CreditorMemberID
	}
	if input.DebtorMemberID != nil {
		debtor = *input.DebtorMemberID
	}
	if err = s.requireMembers(ctx, principal, creditor, debtor); err != nil {
		return command.Result{}, err
	}
	change := ledger.ReimbursementChange{CreditorMemberID: input.CreditorMemberID, DebtorMemberID: input.DebtorMemberID, Principal: input.Amount, ExpenseID: input.ExpenseID, Reason: input.DebtReason, Voided: input.Voided}
	if input.ExpenseID != nil && *input.ExpenseID != "" {
		expense, found, loadErr := s.repository.CurrentLedgerRevision(ctx, principal, *input.ExpenseID)
		if loadErr != nil || !found {
			if loadErr == nil {
				loadErr = ledger.ErrNotFound
			}
			return command.Result{}, s.reject(loadErr)
		}
		if loadErr = requireReimbursementExpense(expense); loadErr != nil {
			return command.Result{}, s.reject(loadErr)
		}
		change.ExpenseRevision = expense.Revision
	}
	decisionID := s.newID()
	next, fields, err := current.Correct(change, principal.UserID(), s.now(), decisionID)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	decision := ledger.ReimbursementDecision{ID: decisionID, Kind: "correction", ReimbursementID: id, ActorID: principal.UserID(), At: next.RecordedAt, Reason: input.Reason, Before: current.Revision, After: next.Revision, Fields: fields}
	if err = s.repository.SaveReimbursement(ctx, principal, next, current.Revision, decision); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "reimbursement", id, next.Revision, "reimbursement.changed"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "reimbursement", ResourceID: id, Revision: next.Revision}, nil
}

type ReimbursementSettlementInput struct {
	ExpectedRevision, TransferExpectedRevision uint64
	TransferID                                 string
	TransferAmount, SettledAmount              money.Money
}

func (s *ReimbursementService) Settle(ctx context.Context, principal household.Principal, id string, input ReimbursementSettlementInput) (command.Result, error) {
	current, err := s.repository.Reimbursement(ctx, principal, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != input.ExpectedRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict", CurrentRevision: current.Revision}
	}
	transfer, err := s.settlementTransfer(ctx, principal, input.TransferID, input.TransferExpectedRevision)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	members, err := s.members(ctx, principal)
	if err != nil {
		return command.Result{}, err
	}
	creditor, creditorOK := members[current.CreditorMemberID]
	debtor, debtorOK := members[current.DebtorMemberID]
	if !creditorOK || !debtorOK || transfer.FromOwnerID != debtor.UserID || transfer.ToOwnerID != creditor.UserID {
		return command.Result{}, s.reject(ledger.ErrInvalidReimbursement)
	}
	if input.TransferAmount.Validate() != nil || input.SettledAmount.Validate() != nil || input.TransferAmount.Sign() <= 0 || input.SettledAmount.Sign() <= 0 || input.TransferAmount.Asset() != transfer.Received.Asset() || input.SettledAmount.Asset() != current.Principal.Asset() {
		return command.Result{}, s.reject(ledger.ErrInvalidReimbursement)
	}
	if compared, compareErr := input.TransferAmount.Compare(transfer.Received); compareErr != nil || compared > 0 {
		return command.Result{}, s.reject(ledger.ErrReimbursementConflict)
	}
	used, err := s.repository.ActiveTransferUsage(ctx, principal, transfer.Key, transfer.Received.Asset())
	if err != nil {
		return command.Result{}, err
	}
	nextUsage, err := used.Add(input.TransferAmount)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if compared, compareErr := nextUsage.Compare(transfer.Received); compareErr != nil || compared > 0 {
		return command.Result{}, s.reject(ledger.ErrReimbursementConflict)
	}
	decisionID := s.newID()
	entry := ledger.ReimbursementSettlement{ID: s.newID(), DecisionID: decisionID, TransferID: input.TransferID, TransferKey: transfer.Key, TransferRevision: transfer.SelectedRevision, TransferAmount: input.TransferAmount, SettledAmount: input.SettledAmount, Fingerprint: transfer.Fingerprint(), OperationIDs: slices.Clone(transfer.OperationIDs), State: ledger.SettlementActive, ActorID: principal.UserID(), RecordedAt: s.now()}
	next, err := current.AddSettlement(entry, principal.UserID(), entry.RecordedAt, decisionID)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	decision := ledger.ReimbursementDecision{ID: decisionID, Kind: "settlement", ReimbursementID: id, SettlementID: entry.ID, ActorID: principal.UserID(), At: entry.RecordedAt, Reason: "settlement", Before: current.Revision, After: next.Revision, Fields: []ledger.ReimbursementField{ledger.ReimbursementSettlementField}}
	if err = s.repository.SaveReimbursement(ctx, principal, next, current.Revision, decision); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "reimbursement", id, next.Revision, "reimbursement.settled"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "reimbursement", ResourceID: id, Revision: next.Revision}, nil
}

type ReimbursementUndoInput struct {
	ExpectedRevision   uint64
	DecisionID, Reason string
}

func (s *ReimbursementService) Undo(ctx context.Context, principal household.Principal, id string, input ReimbursementUndoInput) (command.Result, error) {
	if input.ExpectedRevision < 1 || input.DecisionID == "" || !validReimbursementReason(input.Reason) {
		return command.Result{}, s.reject(ledger.ErrInvalidReimbursement)
	}
	current, err := s.repository.Reimbursement(ctx, principal, id)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if current.Revision != input.ExpectedRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict", CurrentRevision: current.Revision}
	}
	decision, found, err := s.repository.ReimbursementDecision(ctx, principal, input.DecisionID)
	if err != nil || !found || decision.ReimbursementID != id {
		if err == nil {
			err = ledger.ErrNotFound
		}
		return command.Result{}, s.reject(err)
	}
	undone, err := s.repository.ReimbursementDecisionUndone(ctx, principal, decision.ID)
	if err != nil {
		return command.Result{}, err
	}
	if undone {
		return command.Result{}, s.reject(ledger.ErrReimbursementConflict)
	}
	var before ledger.Reimbursement
	if decision.Kind == "correction" {
		before, err = s.repository.ReimbursementRevision(ctx, principal, id, decision.Before)
		if err != nil {
			return command.Result{}, s.reject(err)
		}
	}
	undoID := s.newID()
	next, fields, err := current.Undo(decision, before, principal.UserID(), s.now(), undoID)
	if err != nil {
		return command.Result{}, s.reject(err)
	}
	if slices.Contains(fields, ledger.ReimbursementExpenseField) && next.ExpenseID != "" {
		expense, found, loadErr := s.repository.CurrentLedgerRevision(ctx, principal, next.ExpenseID)
		if loadErr != nil {
			return command.Result{}, loadErr
		}
		if !found || expense.Revision != next.ExpenseRevision || requireReimbursementExpense(expense) != nil {
			return command.Result{}, s.reject(ledger.ErrReimbursementConflict)
		}
	}
	undo := ledger.ReimbursementDecision{ID: undoID, Kind: "undo", ReimbursementID: id, ActorID: principal.UserID(), At: next.RecordedAt, Reason: input.Reason, UndoOf: decision.ID, Before: current.Revision, After: next.Revision, Fields: fields}
	if err = s.repository.SaveReimbursement(ctx, principal, next, current.Revision, undo); err != nil {
		return command.Result{}, s.reject(err)
	}
	if err = s.repository.EmitEvent(ctx, "reimbursement", id, next.Revision, "reimbursement.changed"); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "reimbursement", ResourceID: id, Revision: next.Revision}, nil
}

func (s *ReimbursementService) Read(ctx context.Context, principal household.Principal, id string) (ledger.Reimbursement, error) {
	value, err := s.repository.Reimbursement(ctx, principal, id)
	return value, s.reject(err)
}

func (s *ReimbursementService) List(ctx context.Context, principal household.Principal, filter ReimbursementFilter, cursor ReimbursementCursor, limit int) ([]ledger.Reimbursement, *ReimbursementCursor, error) {
	if limit < 1 || limit > 100 || filter.Asset != "" && parseAssetError(filter.Asset) != nil || filter.State != "" && !filter.State.Valid() {
		return nil, nil, commands.Rejection{Code: "invalid_request"}
	}
	if filter.MemberID != "" {
		if err := s.requireMember(ctx, principal, filter.MemberID); err != nil {
			return nil, nil, err
		}
	}
	return s.repository.Reimbursements(ctx, principal, filter, cursor, limit)
}

func (s *ReimbursementService) History(ctx context.Context, principal household.Principal, id string, cursor ReimbursementCursor, limit int) ([]ledger.Reimbursement, *ReimbursementCursor, error) {
	if limit < 1 || limit > 100 {
		return nil, nil, commands.Rejection{Code: "invalid_request"}
	}
	if _, err := s.repository.Reimbursement(ctx, principal, id); err != nil {
		return nil, nil, s.reject(err)
	}
	return s.repository.ReimbursementHistory(ctx, principal, id, cursor, limit)
}

func (s *ReimbursementService) ReconcileLedgerRevision(ctx context.Context, principal household.Principal, current ledger.Revision, _ *ledger.Revision) error {
	ids, err := s.repository.ReimbursementIDsForOperation(ctx, principal, current.OperationID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		value, loadErr := s.repository.Reimbursement(ctx, principal, id)
		if loadErr != nil {
			return loadErr
		}
		if value.ExpenseID == current.OperationID && value.ExpenseRevision != current.Revision && value.AttentionReason == "" {
			if err = s.saveReferenceChange(ctx, principal, value, "linked_expense_changed", "reimbursement.expense_attention", ""); err != nil {
				return err
			}
			value, err = s.repository.Reimbursement(ctx, principal, id)
			if err != nil {
				return err
			}
		}
		for _, settlement := range value.Settlements {
			if settlement.State != ledger.SettlementActive || !slices.Contains(settlement.OperationIDs, current.OperationID) {
				continue
			}
			transfer, resolveErr := s.settlementTransfer(ctx, principal, settlement.TransferID, 0)
			if resolveErr == nil && transfer.Key == settlement.TransferKey && transfer.Fingerprint() == settlement.Fingerprint {
				continue
			}
			if err = s.saveReferenceChange(ctx, principal, value, "settlement_transfer_changed", "reimbursement.settlement_stale", settlement.ID); err != nil {
				return err
			}
			value, err = s.repository.Reimbursement(ctx, principal, id)
			if err != nil {
				return err
			}
		}
	}
	return nil
}

func (s *ReimbursementService) ReconcileAccountOwnership(ctx context.Context, principal household.Principal, accountID string) error {
	ids, err := s.repository.SettlementOperationIDsForAccount(ctx, principal, accountID)
	if err != nil {
		return err
	}
	for _, id := range ids {
		current, found, loadErr := s.repository.CurrentLedgerRevision(ctx, principal, id)
		if loadErr != nil {
			return loadErr
		}
		if !found {
			return ledger.ErrNotFound
		}
		if err := s.ReconcileLedgerRevision(ctx, principal, current, nil); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReimbursementService) saveReferenceChange(ctx context.Context, principal household.Principal, current ledger.Reimbursement, reason, event, settlementID string) error {
	decisionID := s.newID()
	var next ledger.Reimbursement
	var err error
	fields := []ledger.ReimbursementField{ledger.ReimbursementExpenseField}
	if settlementID == "" {
		next, err = current.RequireAttention(reason, principal.UserID(), s.now(), decisionID)
	} else {
		fields = []ledger.ReimbursementField{ledger.ReimbursementSettlementField}
		next, err = current.MarkSettlement(settlementID, ledger.SettlementStale, principal.UserID(), s.now(), decisionID, current.AttentionReason)
	}
	if errors.Is(err, ledger.ErrReimbursementNoChange) {
		return nil
	}
	if err != nil {
		return err
	}
	decision := ledger.ReimbursementDecision{ID: decisionID, Kind: "reference_change", ReimbursementID: current.ID, ActorID: principal.UserID(), At: next.RecordedAt, Reason: reason, Before: current.Revision, After: next.Revision, Fields: fields}
	if err = s.repository.SaveReimbursement(ctx, principal, next, current.Revision, decision); err != nil {
		return err
	}
	return s.repository.EmitEvent(ctx, "reimbursement", current.ID, next.Revision, event)
}

func (s *ReimbursementService) settlementTransfer(ctx context.Context, principal household.Principal, id string, expected uint64) (ledger.SettlementTransfer, error) {
	selected, found, err := s.repository.CurrentLedgerRevision(ctx, principal, id)
	if err != nil || !found {
		if err == nil {
			err = ledger.ErrNotFound
		}
		return ledger.SettlementTransfer{}, err
	}
	if expected != 0 && selected.Revision != expected {
		return ledger.SettlementTransfer{}, commands.Rejection{Code: "version_conflict", CurrentRevision: selected.Revision}
	}
	facts := []ledger.Revision{selected}
	key := "transaction:" + selected.OperationID
	if selected.Participation.State == "linked" {
		group, groupErr := s.repository.MatchingGroup(ctx, principal, selected.Participation.GroupID)
		if groupErr != nil || group.State != matching.Linked || group.Kind != matching.Transfer && group.Kind != matching.Exchange {
			if groupErr != nil {
				return ledger.SettlementTransfer{}, groupErr
			}
			return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
		}
		facts = facts[:0]
		for _, member := range group.Members {
			fact, exists, loadErr := s.repository.CurrentLedgerRevision(ctx, principal, member.OperationID)
			if loadErr != nil || !exists {
				if loadErr == nil {
					loadErr = ledger.ErrNotFound
				}
				return ledger.SettlementTransfer{}, loadErr
			}
			facts = append(facts, fact)
		}
		key = "matching:" + group.ID
	} else if selected.Participation.GroupID != "" || selected.Type != ledger.Transfer && selected.Type != ledger.Exchange {
		return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
	}
	var negative, positive *ledger.Posting
	operations := []string{}
	for _, fact := range facts {
		if fact.Accounting() != ledger.IncludedInAccounting {
			return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
		}
		operations = append(operations, fact.OperationID)
		for index := range fact.Postings {
			posting := fact.Postings[index]
			if posting.Role != ledger.Principal || !posting.MovesMoney() || !fact.Contributes(index) || fact.ContributionState(index) != ledger.Posted {
				continue
			}
			copy := posting
			if posting.Money.Sign() < 0 {
				if negative != nil {
					return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
				}
				negative = &copy
			} else if posting.Money.Sign() > 0 {
				if positive != nil {
					return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
				}
				positive = &copy
			}
		}
	}
	if negative == nil || positive == nil {
		return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
	}
	from, err := s.repository.Account(ctx, principal, negative.AccountID)
	if err != nil {
		return ledger.SettlementTransfer{}, err
	}
	to, err := s.repository.Account(ctx, principal, positive.AccountID)
	if err != nil {
		return ledger.SettlementTransfer{}, err
	}
	if from.Ownership.Scope() != household.Personal || to.Ownership.Scope() != household.Personal {
		return ledger.SettlementTransfer{}, ledger.ErrInvalidReimbursement
	}
	zero, _ := money.NewMoney("0", negative.Money.Asset())
	sent, err := zero.Subtract(negative.Money)
	if err != nil {
		return ledger.SettlementTransfer{}, err
	}
	sort.Strings(operations)
	result := ledger.SettlementTransfer{Key: key, SelectedID: id, SelectedRevision: selected.Revision, FromAccountID: negative.AccountID, ToAccountID: positive.AccountID, FromOwnerID: from.Ownership.PersonalOwnerID(), ToOwnerID: to.Ownership.PersonalOwnerID(), Sent: sent, Received: positive.Money, OperationIDs: operations}
	return result, result.Validate()
}

func (s *ReimbursementService) members(ctx context.Context, principal household.Principal) (map[household.MembershipID]household.Membership, error) {
	values, err := s.repository.HouseholdMemberships(ctx, principal)
	if err != nil {
		return nil, err
	}
	result := make(map[household.MembershipID]household.Membership, len(values))
	for _, member := range values {
		if member.Active && member.HouseholdID == principal.HouseholdID() {
			result[member.ID] = member
		}
	}
	return result, nil
}

func (s *ReimbursementService) requireMember(ctx context.Context, principal household.Principal, id household.MembershipID) error {
	members, err := s.members(ctx, principal)
	if err != nil {
		return err
	}
	if _, found := members[id]; !found {
		return commands.Rejection{Code: "invalid_request"}
	}
	return nil
}

func (s *ReimbursementService) requireMembers(ctx context.Context, principal household.Principal, creditor, debtor household.MembershipID) error {
	if creditor == "" || debtor == "" || creditor == debtor {
		return s.reject(ledger.ErrInvalidReimbursement)
	}
	members, err := s.members(ctx, principal)
	if err != nil {
		return err
	}
	if _, found := members[creditor]; !found {
		return commands.Rejection{Code: "invalid_request"}
	}
	if _, found := members[debtor]; !found {
		return commands.Rejection{Code: "invalid_request"}
	}
	return nil
}

func requireReimbursementExpense(value ledger.Revision) error {
	if value.Type != ledger.Expense || value.State != ledger.Posted || value.Accounting() != ledger.IncludedInAccounting {
		return ledger.ErrInvalidReimbursement
	}
	components, err := value.Components()
	if err != nil {
		return err
	}
	for _, component := range components {
		if component.Kind == "expense" && component.Money.Sign() < 0 {
			return nil
		}
	}
	return ledger.ErrInvalidReimbursement
}

func validReimbursementReason(value string) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= 2000
}

func parseAssetError(asset money.Asset) error {
	_, err := money.ParseAsset(string(asset))
	return err
}

func (s *ReimbursementService) reject(err error) error {
	switch {
	case err == nil:
		return nil
	case errors.Is(err, ledger.ErrNotFound):
		return commands.Rejection{Code: "not_found"}
	case errors.Is(err, ledger.ErrReimbursementNoChange):
		return commands.Rejection{Code: "no_change"}
	case errors.Is(err, ledger.ErrReimbursementConflict):
		return commands.Rejection{Code: "decision_conflict"}
	case errors.Is(err, money.ErrAssetMismatch):
		return commands.Rejection{Code: "asset_mismatch"}
	case errors.Is(err, money.ErrInvalidMoney):
		return commands.Rejection{Code: "invalid_money"}
	case errors.Is(err, ledger.ErrInvalidReimbursement), errors.Is(err, ledger.ErrInvalidRevision), errors.Is(err, ledger.ErrInvalidTransition):
		return commands.Rejection{Code: "invalid_transaction"}
	case errors.Is(err, household.ErrForbidden):
		return commands.Rejection{Code: "not_found"}
	default:
		return err
	}
}
