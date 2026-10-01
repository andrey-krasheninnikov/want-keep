package reimbursements

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/http"

	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Server) execute(w http.ResponseWriter, r *http.Request, access identity.Access, kind, resourceID string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload, err := json.Marshal(struct {
		ResourceID string `json:"resourceId"`
		Input      any    `json:"input"`
	}{resourceID, input})
	if err != nil {
		s.problem(w, err)
		return
	}
	digest := sha256.Sum256(payload)
	result, err := s.mutations.Execute(r.Context(), access, commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: kind, PayloadHash: hex.EncodeToString(digest[:])}, apply)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var readErr error
		result, readErr = s.commands.Read(ctx, access.Principal, result.ID(), s.now())
		if errors.Is(readErr, command.ErrCommandExpired) {
			return nil
		}
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.commandResponse(w, access.Principal, result)
}

func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.ReimbursementCreate
	if err = s.decode(r, "ReimbursementCreate", &input); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	amount, err := positiveMoney(input.Amount)
	if err != nil {
		s.problem(w, err)
		return
	}
	request := application.ReimbursementCreateInput{CreditorMemberID: household.MembershipID(input.CreditorMemberId), DebtorMemberID: household.MembershipID(input.DebtorMemberId), Amount: amount, Reason: input.Reason}
	if input.ExpenseId != nil {
		request.ExpenseID = *input.ExpenseId
	}
	s.execute(w, r, access, "reimbursements.create", "", input, func(ctx context.Context) (command.Result, error) {
		return s.service.Create(ctx, access.Principal, request)
	})
}

func (s *Server) correct(w http.ResponseWriter, r *http.Request) {
	access, id, input, ok := s.correctionRequest(w, r)
	if !ok {
		return
	}
	request := application.ReimbursementCorrectionInput{ExpectedRevision: uint64(input.ExpectedRevision), Reason: input.Reason, DebtReason: input.DebtReason, Voided: input.Voided}
	if input.CreditorMemberId != nil {
		value := household.MembershipID(*input.CreditorMemberId)
		request.CreditorMemberID = &value
	}
	if input.DebtorMemberId != nil {
		value := household.MembershipID(*input.DebtorMemberId)
		request.DebtorMemberID = &value
	}
	if input.Amount != nil {
		value, err := positiveMoney(*input.Amount)
		if err != nil {
			s.problem(w, err)
			return
		}
		request.Amount = &value
	}
	clear := input.ClearExpense != nil && *input.ClearExpense
	if clear && input.ExpenseId != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	if input.ExpenseId != nil {
		value := string(*input.ExpenseId)
		request.ExpenseID = &value
	} else if clear {
		value := ""
		request.ExpenseID = &value
	}
	s.execute(w, r, access, "reimbursements.correct", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.Correct(ctx, access.Principal, id, request)
	})
}

func (s *Server) correctionRequest(w http.ResponseWriter, r *http.Request) (identity.Access, string, generated.ReimbursementCorrection, bool) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return access, "", generated.ReimbursementCorrection{}, false
	}
	id, err := reimbursementID(r)
	if err != nil {
		s.problem(w, err)
		return access, "", generated.ReimbursementCorrection{}, false
	}
	var input generated.ReimbursementCorrection
	if err = s.decode(r, "ReimbursementCorrection", &input); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return access, "", input, false
	}
	return access, id, input, true
}

func (s *Server) settle(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := reimbursementID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.SettlementCreate
	if err = s.decode(r, "SettlementCreate", &input); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	transferAmount, err := positiveMoney(input.TransferAmount)
	if err != nil {
		s.problem(w, err)
		return
	}
	settledAmount, err := positiveMoney(input.SettledAmount)
	if err != nil {
		s.problem(w, err)
		return
	}
	request := application.ReimbursementSettlementInput{ExpectedRevision: uint64(input.ExpectedRevision), TransferExpectedRevision: uint64(input.TransferExpectedRevision), TransferID: input.TransferId, TransferAmount: transferAmount, SettledAmount: settledAmount}
	s.execute(w, r, access, "reimbursements.settle", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.Settle(ctx, access.Principal, id, request)
	})
}

func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := reimbursementID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.ReimbursementUndo
	if err = s.decode(r, "ReimbursementUndo", &input); err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	request := application.ReimbursementUndoInput{ExpectedRevision: uint64(input.ExpectedRevision), DecisionID: input.DecisionId, Reason: input.Reason}
	s.execute(w, r, access, "reimbursements.undo", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.Undo(ctx, access.Principal, id, request)
	})
}

func positiveMoney(value generated.PositiveMoney) (money.Money, error) {
	result, err := money.NewMoney(value.Amount, money.Asset(value.Asset))
	if err != nil || result.Sign() <= 0 {
		return money.Money{}, money.ErrInvalidMoney
	}
	return result, nil
}
