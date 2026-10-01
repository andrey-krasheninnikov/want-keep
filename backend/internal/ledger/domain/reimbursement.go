package domain

import (
	"encoding/hex"
	"errors"
	"slices"
	"strings"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

const (
	MaxReimbursementRevision    = uint64(9007199254740991)
	MaxReimbursementSettlements = 1000
)

var (
	ErrInvalidReimbursement  = errors.New("invalid reimbursement")
	ErrReimbursementConflict = errors.New("reimbursement decision conflicts with current state")
	ErrReimbursementNoChange = errors.New("reimbursement has no change")
)

type ReimbursementState string

const (
	ReimbursementOpen              ReimbursementState = "open"
	ReimbursementSettled           ReimbursementState = "settled"
	ReimbursementAttentionRequired ReimbursementState = "attention_required"
	ReimbursementVoided            ReimbursementState = "voided"
)

func (s ReimbursementState) Valid() bool {
	return slices.Contains([]ReimbursementState{ReimbursementOpen, ReimbursementSettled, ReimbursementAttentionRequired, ReimbursementVoided}, s)
}

type ReimbursementField string

const (
	ReimbursementPartiesField    ReimbursementField = "parties"
	ReimbursementPrincipalField  ReimbursementField = "principal"
	ReimbursementExpenseField    ReimbursementField = "expense"
	ReimbursementReasonField     ReimbursementField = "reason"
	ReimbursementVoidedField     ReimbursementField = "voided"
	ReimbursementSettlementField ReimbursementField = "settlement"
)

func (f ReimbursementField) Valid() bool {
	return slices.Contains([]ReimbursementField{ReimbursementPartiesField, ReimbursementPrincipalField, ReimbursementExpenseField, ReimbursementReasonField, ReimbursementVoidedField, ReimbursementSettlementField}, f)
}

type SettlementState string

const (
	SettlementActive SettlementState = "active"
	SettlementStale  SettlementState = "stale"
	SettlementUndone SettlementState = "undone"
)

type ReimbursementSettlement struct {
	ID, DecisionID, TransferID, TransferKey, Fingerprint string
	TransferRevision                                     uint64
	TransferAmount, SettledAmount                        money.Money
	OperationIDs                                         []string
	State                                                SettlementState
	ActorID                                              household.UserID
	RecordedAt                                           calendar.Instant
}

func (s ReimbursementSettlement) Validate() error {
	fingerprint, fingerprintErr := hex.DecodeString(s.Fingerprint)
	if s.ID == "" || s.DecisionID == "" || s.TransferID == "" || !validReimbursementText(s.TransferKey) || fingerprintErr != nil || len(fingerprint) != 32 || s.TransferRevision < 1 || s.TransferRevision > MaxReimbursementRevision || s.ActorID == "" || s.RecordedAt.String() == "" || !slices.Contains([]SettlementState{SettlementActive, SettlementStale, SettlementUndone}, s.State) || len(s.OperationIDs) < 1 || len(s.OperationIDs) > 100 {
		return ErrInvalidReimbursement
	}
	if s.TransferAmount.Validate() != nil || s.SettledAmount.Validate() != nil || s.TransferAmount.Sign() <= 0 || s.SettledAmount.Sign() <= 0 {
		return ErrInvalidReimbursement
	}
	if s.TransferAmount.Asset() == s.SettledAmount.Asset() {
		compared, err := s.TransferAmount.Compare(s.SettledAmount)
		if err != nil || compared != 0 {
			return ErrInvalidReimbursement
		}
	}
	seen := map[string]bool{}
	for _, id := range s.OperationIDs {
		if id == "" || seen[id] {
			return ErrInvalidReimbursement
		}
		seen[id] = true
	}
	return nil
}

type Reimbursement struct {
	ID, DecisionID, ExpenseID, Reason, AttentionReason string
	Revision, ExpenseRevision                          uint64
	CreditorMemberID, DebtorMemberID                   household.MembershipID
	Principal, Outstanding                             money.Money
	State                                              ReimbursementState
	Voided                                             bool
	ActorID                                            household.UserID
	RecordedAt                                         calendar.Instant
	FieldVersions                                      map[ReimbursementField]uint64
	Settlements                                        []ReimbursementSettlement
}

func NewReimbursement(id string, creditor, debtor household.MembershipID, principal money.Money, expenseID string, expenseRevision uint64, reason string, actor household.UserID, at calendar.Instant, decisionID string) (Reimbursement, error) {
	r := Reimbursement{ID: id, Revision: 1, CreditorMemberID: creditor, DebtorMemberID: debtor, Principal: principal, Outstanding: principal, ExpenseID: expenseID, ExpenseRevision: expenseRevision, Reason: reason, State: ReimbursementOpen, ActorID: actor, RecordedAt: at, DecisionID: decisionID, FieldVersions: map[ReimbursementField]uint64{ReimbursementPartiesField: 1, ReimbursementPrincipalField: 1, ReimbursementReasonField: 1}}
	if expenseID != "" {
		r.FieldVersions[ReimbursementExpenseField] = 1
	}
	return r, r.Validate()
}

func (r Reimbursement) Validate() error {
	if r.ID == "" || r.Revision < 1 || r.Revision > MaxReimbursementRevision || r.CreditorMemberID == "" || r.DebtorMemberID == "" || r.CreditorMemberID == r.DebtorMemberID || r.ActorID == "" || r.RecordedAt.String() == "" || r.DecisionID == "" || !r.State.Valid() || !validReimbursementText(r.Reason) || r.AttentionReason != "" && !validReimbursementText(r.AttentionReason) || (r.ExpenseID == "") != (r.ExpenseRevision == 0) {
		return ErrInvalidReimbursement
	}
	if r.Principal.Validate() != nil || r.Outstanding.Validate() != nil || r.Principal.Asset() != r.Outstanding.Asset() || r.Principal.Sign() <= 0 || r.Outstanding.Sign() < 0 {
		return ErrInvalidReimbursement
	}
	for field, revision := range r.FieldVersions {
		if !field.Valid() || revision < 1 || revision > r.Revision {
			return ErrInvalidReimbursement
		}
	}
	if len(r.Settlements) > MaxReimbursementSettlements {
		return ErrInvalidReimbursement
	}
	seen := map[string]bool{}
	for _, settlement := range r.Settlements {
		if seen[settlement.ID] || settlement.Validate() != nil {
			return ErrInvalidReimbursement
		}
		seen[settlement.ID] = true
	}
	expected, state, err := r.projection()
	if err != nil || !sameMoneyValue(expected, r.Outstanding) || state != r.State {
		return ErrInvalidReimbursement
	}
	return nil
}

type ReimbursementChange struct {
	CreditorMemberID, DebtorMemberID *household.MembershipID
	Principal                        *money.Money
	ExpenseID                        *string
	ExpenseRevision                  uint64
	Reason                           *string
	Voided                           *bool
}

func (r Reimbursement) Correct(change ReimbursementChange, actor household.UserID, at calendar.Instant, decisionID string) (Reimbursement, []ReimbursementField, error) {
	next := r.clone()
	fields := []ReimbursementField{}
	if change.CreditorMemberID != nil || change.DebtorMemberID != nil {
		if change.CreditorMemberID != nil {
			next.CreditorMemberID = *change.CreditorMemberID
		}
		if change.DebtorMemberID != nil {
			next.DebtorMemberID = *change.DebtorMemberID
		}
		if next.CreditorMemberID != r.CreditorMemberID || next.DebtorMemberID != r.DebtorMemberID {
			fields = append(fields, ReimbursementPartiesField)
		}
	}
	if change.Principal != nil && !sameMoneyValue(*change.Principal, r.Principal) {
		next.Principal = *change.Principal
		fields = append(fields, ReimbursementPrincipalField)
	}
	if change.ExpenseID != nil && (*change.ExpenseID != r.ExpenseID || change.ExpenseRevision != r.ExpenseRevision) {
		next.ExpenseID, next.ExpenseRevision = *change.ExpenseID, change.ExpenseRevision
		next.AttentionReason = ""
		fields = append(fields, ReimbursementExpenseField)
	}
	if change.Reason != nil && *change.Reason != r.Reason {
		next.Reason = *change.Reason
		fields = append(fields, ReimbursementReasonField)
	}
	if change.Voided != nil && *change.Voided != r.Voided {
		next.Voided = *change.Voided
		fields = append(fields, ReimbursementVoidedField)
	}
	if len(fields) == 0 {
		return r, nil, ErrReimbursementNoChange
	}
	if len(r.activeSettlements()) > 0 && (slices.Contains(fields, ReimbursementPartiesField) || r.Principal.Asset() != next.Principal.Asset() || next.Voided) {
		return r, nil, ErrReimbursementConflict
	}
	next.bump(actor, at, decisionID, fields)
	if err := next.recalculate(); err != nil {
		return r, nil, err
	}
	return next, fields, next.Validate()
}

func (r Reimbursement) AddSettlement(settlement ReimbursementSettlement, actor household.UserID, at calendar.Instant, decisionID string) (Reimbursement, error) {
	if r.Voided || r.AttentionReason != "" || len(r.Settlements) >= MaxReimbursementSettlements || settlement.State != SettlementActive || settlement.DecisionID != decisionID {
		return r, ErrReimbursementConflict
	}
	next := r.clone()
	next.Settlements = append(next.Settlements, settlement)
	next.bump(actor, at, decisionID, []ReimbursementField{ReimbursementSettlementField})
	if err := next.recalculate(); err != nil {
		return r, err
	}
	return next, next.Validate()
}

func (r Reimbursement) MarkSettlement(settlementID string, state SettlementState, actor household.UserID, at calendar.Instant, decisionID, attention string) (Reimbursement, error) {
	if state != SettlementStale && state != SettlementUndone {
		return r, ErrInvalidReimbursement
	}
	next := r.clone()
	changed := false
	for i := range next.Settlements {
		if next.Settlements[i].ID == settlementID && next.Settlements[i].State == SettlementActive {
			next.Settlements[i].State = state
			changed = true
		}
	}
	if !changed {
		return r, ErrReimbursementConflict
	}
	next.AttentionReason = attention
	next.bump(actor, at, decisionID, []ReimbursementField{ReimbursementSettlementField})
	if err := next.recalculate(); err != nil {
		return r, err
	}
	return next, next.Validate()
}

func (r Reimbursement) RequireAttention(reason string, actor household.UserID, at calendar.Instant, decisionID string) (Reimbursement, error) {
	if reason == "" || reason == r.AttentionReason {
		return r, ErrReimbursementNoChange
	}
	next := r.clone()
	next.AttentionReason = reason
	next.bump(actor, at, decisionID, []ReimbursementField{ReimbursementExpenseField})
	if err := next.recalculate(); err != nil {
		return r, err
	}
	return next, next.Validate()
}

type ReimbursementDecision struct {
	ID, Kind, Reason, UndoOf, ReimbursementID, SettlementID string
	ActorID                                                 household.UserID
	At                                                      calendar.Instant
	Before, After                                           uint64
	Fields                                                  []ReimbursementField
}

func (d ReimbursementDecision) Validate() error {
	if d.ID == "" || d.ReimbursementID == "" || d.ActorID == "" || d.At.String() == "" || !validReimbursementText(d.Reason) || d.Before >= MaxReimbursementRevision || d.After < 1 || d.Before+1 != d.After || d.After > MaxReimbursementRevision || !slices.Contains([]string{"create", "correction", "settlement", "undo", "reference_change"}, d.Kind) || (d.Kind == "undo") != (d.UndoOf != "") || len(d.Fields) < 1 {
		return ErrInvalidReimbursement
	}
	seen := map[ReimbursementField]bool{}
	for _, field := range d.Fields {
		if !field.Valid() || seen[field] {
			return ErrInvalidReimbursement
		}
		seen[field] = true
	}
	if (d.Kind == "settlement") != (d.SettlementID != "") {
		return ErrInvalidReimbursement
	}
	return nil
}

func (r Reimbursement) Undo(decision ReimbursementDecision, before Reimbursement, actor household.UserID, at calendar.Instant, undoID string) (Reimbursement, []ReimbursementField, error) {
	if decision.ReimbursementID != r.ID || decision.After > r.Revision || decision.Kind == "undo" || decision.Kind == "create" || decision.Kind == "reference_change" {
		return r, nil, ErrReimbursementConflict
	}
	if decision.Kind == "settlement" {
		next, err := r.MarkSettlement(decision.SettlementID, SettlementUndone, actor, at, undoID, r.AttentionReason)
		return next, []ReimbursementField{ReimbursementSettlementField}, err
	}
	if decision.Before != before.Revision {
		return r, nil, ErrReimbursementConflict
	}
	for _, field := range decision.Fields {
		if r.FieldVersions[field] != decision.After {
			return r, nil, ErrReimbursementConflict
		}
	}
	next := r.clone()
	for _, field := range decision.Fields {
		switch field {
		case ReimbursementPartiesField:
			next.CreditorMemberID, next.DebtorMemberID = before.CreditorMemberID, before.DebtorMemberID
		case ReimbursementPrincipalField:
			next.Principal = before.Principal
		case ReimbursementExpenseField:
			next.ExpenseID, next.ExpenseRevision, next.AttentionReason = before.ExpenseID, before.ExpenseRevision, before.AttentionReason
		case ReimbursementReasonField:
			next.Reason = before.Reason
		case ReimbursementVoidedField:
			next.Voided = before.Voided
		default:
			return r, nil, ErrReimbursementConflict
		}
	}
	if len(next.activeSettlements()) > 0 && (slices.Contains(decision.Fields, ReimbursementPartiesField) || next.Principal.Asset() != r.Principal.Asset() || next.Voided) {
		return r, nil, ErrReimbursementConflict
	}
	next.bump(actor, at, undoID, decision.Fields)
	if err := next.recalculate(); err != nil {
		return r, nil, err
	}
	return next, slices.Clone(decision.Fields), next.Validate()
}

func (r Reimbursement) activeSettlements() []ReimbursementSettlement {
	result := []ReimbursementSettlement{}
	for _, settlement := range r.Settlements {
		if settlement.State == SettlementActive {
			result = append(result, settlement)
		}
	}
	return result
}

func (r *Reimbursement) recalculate() error {
	outstanding, state, err := r.projection()
	if err != nil {
		return err
	}
	r.Outstanding, r.State = outstanding, state
	return nil
}

func (r Reimbursement) projection() (money.Money, ReimbursementState, error) {
	settled, _ := money.NewMoney("0", r.Principal.Asset())
	for _, settlement := range r.activeSettlements() {
		if settlement.SettledAmount.Asset() != r.Principal.Asset() {
			return money.Money{}, "", ErrReimbursementConflict
		}
		var err error
		settled, err = settled.Add(settlement.SettledAmount)
		if err != nil {
			return money.Money{}, "", err
		}
	}
	outstanding, err := r.Principal.Subtract(settled)
	if err != nil || outstanding.Sign() < 0 {
		return money.Money{}, "", ErrReimbursementConflict
	}
	switch {
	case r.Voided:
		if settled.Sign() != 0 {
			return money.Money{}, "", ErrReimbursementConflict
		}
		return outstanding, ReimbursementVoided, nil
	case r.AttentionReason != "":
		return outstanding, ReimbursementAttentionRequired, nil
	case outstanding.Sign() == 0:
		return outstanding, ReimbursementSettled, nil
	default:
		return outstanding, ReimbursementOpen, nil
	}
}

func (r *Reimbursement) bump(actor household.UserID, at calendar.Instant, decisionID string, fields []ReimbursementField) {
	r.Revision++
	r.ActorID, r.RecordedAt, r.DecisionID = actor, at, decisionID
	if r.FieldVersions == nil {
		r.FieldVersions = map[ReimbursementField]uint64{}
	}
	for _, field := range fields {
		r.FieldVersions[field] = r.Revision
	}
}

func (r Reimbursement) clone() Reimbursement {
	r.FieldVersions = cloneReimbursementVersions(r.FieldVersions)
	r.Settlements = slices.Clone(r.Settlements)
	for i := range r.Settlements {
		r.Settlements[i].OperationIDs = slices.Clone(r.Settlements[i].OperationIDs)
	}
	return r
}

func cloneReimbursementVersions(source map[ReimbursementField]uint64) map[ReimbursementField]uint64 {
	result := make(map[ReimbursementField]uint64, len(source))
	for field, revision := range source {
		result[field] = revision
	}
	return result
}

func sameMoneyValue(left, right money.Money) bool {
	compared, err := left.Compare(right)
	return err == nil && compared == 0
}

func validReimbursementText(value string) bool {
	return utf8.ValidString(value) && !strings.ContainsRune(value, 0) && strings.TrimSpace(value) != "" && utf8.RuneCountInString(value) <= 2000
}
