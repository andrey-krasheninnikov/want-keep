package ledger

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Server) transactionID(r *http.Request) (string, error) {
	id := r.PathValue("transactionId")
	v, err := uuid.Parse(id)
	if err != nil || v.Version() != 4 || v.String() != id {
		return "", contract.ErrInvalidRequest
	}
	return id, nil
}
func (s *Server) correct(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.TransactionCorrection
	if err = s.decode(r, "TransactionCorrection", &in); err != nil {
		s.problem(w, err)
		return
	}
	c, err := s.correctionInput(in)
	if err != nil {
		s.problem(w, err)
		return
	}
	payload := struct {
		TransactionID string
		Input         generated.TransactionCorrection
	}{id, in}
	s.execute(w, r, a, "transactions.corrections", payload, func(ctx context.Context) (command.Result, error) {
		changes := []application.Change{{OperationID: id, Expected: uint64(in.ExpectedRevision), Correction: c}}
		if in.RelatedChanges != nil {
			for _, related := range *in.RelatedChanges {
				value, err := s.correctionInput(generated.TransactionCorrection{Principal: related.Principal, Fees: related.Fees, ReceiptItems: related.ReceiptItems})
				if err != nil {
					return command.Result{}, err
				}
				changes = append(changes, application.Change{OperationID: related.TransactionId, Expected: uint64(related.ExpectedRevision), Correction: value})
			}
		}
		return s.service.ApplyChanges(ctx, a.Principal, changes, in.Reason)
	})
}
func (s *Server) exclude(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.ExcludeInput
	if err = s.decode(r, "ExcludeInput", &in); err != nil {
		s.problem(w, err)
		return
	}
	payload := struct {
		TransactionID string
		Input         generated.ExcludeInput
	}{id, in}
	s.execute(w, r, a, "transactions.exclude", payload, func(ctx context.Context) (command.Result, error) {
		changes := []application.Change{{OperationID: id, Expected: uint64(in.ExpectedRevision), Exclude: true}}
		if in.RelatedRevisions != nil {
			for _, v := range *in.RelatedRevisions {
				changes = append(changes, application.Change{OperationID: v.TransactionId, Expected: uint64(v.ExpectedRevision), Exclude: true})
			}
		}
		return s.service.ApplyChanges(ctx, a.Principal, changes, in.Reason)
	})
}
func (s *Server) undo(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.UndoInput
	if err = s.decode(r, "UndoInput", &in); err != nil {
		s.problem(w, err)
		return
	}
	versions := []application.ExpectedRevision{}
	hasTarget := false
	for _, v := range in.ExpectedRevisions {
		versions = append(versions, application.ExpectedRevision{OperationID: v.TransactionId, Revision: uint64(v.ExpectedRevision)})
		hasTarget = hasTarget || v.TransactionId == id
	}
	if !hasTarget {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload := struct {
		TransactionID string
		Input         generated.UndoInput
	}{id, in}
	s.execute(w, r, a, "transactions.undo", payload, func(ctx context.Context) (command.Result, error) {
		return s.service.Undo(ctx, a.Principal, in.DecisionId, versions, in.Reason)
	})
}
func (s *Server) correctionInput(in generated.TransactionCorrection) (ledger.Correction, error) {
	c := ledger.Correction{Merchant: in.Merchant, Note: in.Note}
	if in.OccurredAt != nil {
		at, err := calendar.ParseInstant(*in.OccurredAt)
		if err != nil {
			return c, err
		}
		c.OccurredAt = &at
	}
	if in.Payer != nil {
		known, err := in.Payer.AsKnownPayer()
		if err != nil {
			return c, err
		}
		p := ledger.PayerChange{State: "known", MemberID: household.MembershipID(known.MemberId)}
		if known.State != "known" {
			v, err := in.Payer.AsUnspecifiedPayer()
			if err != nil {
				return c, err
			}
			p.State = string(v.State)
		}
		c.Payer = &p
	}
	if in.Principal != nil {
		p, err := s.correctionPostings(*in.Principal)
		if err != nil {
			return c, err
		}
		c.Principal = &p
	}
	if in.Fees != nil {
		p, err := s.correctionPostings(*in.Fees)
		if err != nil {
			return c, err
		}
		c.Fees = &p
	}
	if in.Category != nil {
		value, err := classificationReference(*in.Category)
		if err != nil {
			return c, err
		}
		c.CategoryID = &value
	}
	if in.MerchantIdentity != nil {
		value, err := classificationReference(*in.MerchantIdentity)
		if err != nil {
			return c, err
		}
		c.MerchantID = &value
	}
	if in.ReceiptItems != nil {
		change, err := s.receiptItemsCorrection(*in.ReceiptItems)
		if err != nil {
			return c, err
		}
		c.ReceiptItems = &change
	}
	if in.Allocation != nil {
		allocation, err := AllocationFromDTO(*in.Allocation)
		if err != nil {
			return c, err
		}
		c.Allocation = &ledger.AllocationChange{Allocation: allocation}
	}
	return c, nil
}

func (s *Server) allocate(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.AllocationChange
	if err = s.decode(r, "AllocationChange", &input); err != nil {
		s.problem(w, err)
		return
	}
	allocation, err := AllocationFromDTO(input.Allocation)
	if err != nil {
		s.problem(w, err)
		return
	}
	items := []ledger.ItemAllocationInput{}
	if input.Items != nil {
		for _, item := range *input.Items {
			value, parseErr := AllocationFromDTO(item.Allocation)
			if parseErr != nil {
				s.problem(w, parseErr)
				return
			}
			items = append(items, ledger.ItemAllocationInput{ItemID: item.ItemId, Allocation: value})
		}
	}
	payload := struct {
		TransactionID string
		Input         generated.AllocationChange
	}{id, input}
	s.execute(w, r, a, "transactions.allocations", payload, func(ctx context.Context) (command.Result, error) {
		return s.service.Allocate(ctx, a.Principal, id, uint64(input.ExpectedRevision), allocation, items, input.Reason)
	})
}

func classificationReference(in generated.ClassificationReferenceChange) (string, error) {
	switch in.Action {
	case "set":
		if in.Id == nil {
			return "", contract.ErrInvalidRequest
		}
		return *in.Id, nil
	case "clear":
		if in.Id != nil {
			return "", contract.ErrInvalidRequest
		}
		return "", nil
	default:
		return "", contract.ErrInvalidRequest
	}
}

func (s *Server) receiptItemsCorrection(in generated.ReceiptItemsChange) (ledger.ReceiptItemsCorrection, error) {
	if in.Action == "clear" {
		if in.Items != nil || in.TotalDiscount != nil {
			return ledger.ReceiptItemsCorrection{}, contract.ErrInvalidRequest
		}
		return ledger.ReceiptItemsCorrection{Clear: true}, nil
	}
	if in.Action != "replace" || in.Items == nil || in.TotalDiscount == nil {
		return ledger.ReceiptItemsCorrection{}, contract.ErrInvalidRequest
	}
	total, err := money.NewMoney(in.TotalDiscount.Amount, money.Asset(in.TotalDiscount.Asset))
	if err != nil {
		return ledger.ReceiptItemsCorrection{}, err
	}
	change := ledger.ReceiptItemsCorrection{TotalDiscount: total}
	for _, item := range *in.Items {
		gross, err := money.NewMoney(item.Gross.Amount, money.Asset(item.Gross.Asset))
		if err != nil {
			return ledger.ReceiptItemsCorrection{}, err
		}
		entry := ledger.ReceiptItemInput{ID: item.Id, Name: item.Name, Quantity: item.Quantity, Gross: gross}
		if item.CategoryId != nil {
			entry.CategoryID = *item.CategoryId
		}
		if item.Discount != nil {
			discount, err := money.NewMoney(item.Discount.Amount, money.Asset(item.Discount.Asset))
			if err != nil {
				return ledger.ReceiptItemsCorrection{}, err
			}
			entry.Discount = &discount
		}
		change.Items = append(change.Items, entry)
	}
	return change, nil
}
func (s *Server) correctionPostings(in []generated.Posting) ([]ledger.Posting, error) {
	out := make([]ledger.Posting, 0, len(in))
	for _, v := range in {
		m, err := money.NewMoney(v.Money.Amount, money.Asset(v.Money.Asset))
		if err != nil {
			return nil, err
		}
		p := ledger.Posting{AccountID: v.AccountId, Money: m, Role: ledger.Role(v.Role)}
		if v.Funding != nil {
			p.Funding = ledger.FundingKind(*v.Funding)
		}
		if v.Treatment != nil {
			p.Treatment = ledger.Treatment(*v.Treatment)
		}
		out = append(out, p)
	}
	return out, nil
}
