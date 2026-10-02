package ledger

import (
	"context"
	"net/http"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	expensedomain "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Server) refund(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	if s.refunds == nil {
		s.problem(w, ledger.ErrFeatureUnavailable)
		return
	}
	var in generated.RefundCreate
	if err = s.decode(r, "RefundCreate", &in); err != nil {
		s.problem(w, err)
		return
	}
	input, err := s.refundInput(in)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.execute(w, r, a, "transactions.refund", in, func(ctx context.Context) (command.Result, error) {
		return s.refunds.Create(ctx, a.Principal, input)
	})
}

func (s *Server) refundInput(in generated.RefundCreate) (expenses.CreateInput, error) {
	result := expenses.CreateInput{PurchaseID: in.PurchaseId, PurchaseExpectedRevision: uint64(in.PurchaseExpectedRevision), AccountID: in.ReceivingAccountId, Reason: in.Reason}
	var err error
	result.At, err = calendar.ParseInstant(in.OccurredAt)
	if err != nil {
		return result, err
	}
	result.Amount, err = money.NewMoney(in.Amount.Amount, money.Asset(in.Amount.Asset))
	if err != nil {
		return result, err
	}
	result.Items, err = s.refundItems(in.ReturnedItems)
	if err != nil {
		return result, err
	}
	for _, value := range in.Fees {
		amount, parseErr := money.NewMoney(value.Amount.Amount, money.Asset(value.Amount.Asset))
		if parseErr != nil {
			return result, parseErr
		}
		fee := expenses.FeeInput{AccountID: value.AccountId, Amount: amount}
		if value.Funding != nil {
			fee.Funding = ledger.FundingKind(*value.Funding)
		}
		result.Fees = append(result.Fees, fee)
	}
	return result, nil
}

func (s *Server) refundItems(values []generated.RefundItemInput) ([]expensedomain.ItemPortion, error) {
	result := make([]expensedomain.ItemPortion, 0, len(values))
	for _, value := range values {
		amount, err := money.NewMoney(value.Amount.Amount, money.Asset(value.Amount.Asset))
		if err != nil {
			return nil, err
		}
		result = append(result, expensedomain.ItemPortion{ItemID: value.ItemId, Amount: amount})
	}
	return result, nil
}

func (s *Server) refundDTO(value expensedomain.Refund) (generated.RefundAttribution, error) {
	amount, err := (contract.MoneyConverter{}).ToDTO(value.Amount)
	if err != nil {
		return generated.RefundAttribution{}, err
	}
	remaining, err := (contract.MoneyConverter{}).ToDTO(value.Remaining)
	if err != nil {
		return generated.RefundAttribution{}, err
	}
	result := generated.RefundAttribution{Id: value.OperationID, Revision: int64(value.Revision), PurchaseId: value.PurchaseID, PurchaseRevision: int64(value.PurchaseRevision), RefundRevision: int64(value.RefundRevision), State: generated.RefundAttributionState(value.State), ExpenseMonth: value.ExpenseMonth.String(), CashDate: value.CashDate.String(), Amount: amount, Remaining: remaining, ReturnedItems: []generated.RefundItemInput{}, Members: []generated.MemberAmount{}, Categories: []generated.CategoryAmount{}, Unallocated: []generated.Money{}, Reason: value.Reason}
	for _, item := range value.Items {
		positive, conversionErr := s.positiveDTO(item.Amount)
		if conversionErr != nil {
			return result, conversionErr
		}
		result.ReturnedItems = append(result.ReturnedItems, generated.RefundItemInput{ItemId: item.ItemID, Amount: positive})
	}
	result.Members, result.Categories, result.Unallocated, err = s.refundEffectsDTO(value.Members, value.Categories, value.Unallocated)
	if err != nil {
		return result, err
	}
	if value.Valuation == nil {
		err = result.Valuation.FromUnavailableRefundValuation(generated.UnavailableRefundValuation{State: "unavailable", Reason: "historical_basis_unavailable"})
		return result, err
	}
	valuationAmount, err := (contract.MoneyConverter{}).ToDTO(value.Valuation.Value)
	if err != nil {
		return result, err
	}
	members, categories, unallocated, err := s.refundEffectsDTO(value.Valuation.Members, value.Valuation.Categories, value.Valuation.Unallocated)
	if err != nil {
		return result, err
	}
	err = result.Valuation.FromKnownRefundValuation(generated.KnownRefundValuation{State: "known", Amount: valuationAmount, BasisRef: value.Valuation.Ref, Members: members, Categories: categories, Unallocated: unallocated})
	return result, err
}

func (s *Server) refundEffectsDTO(members []expensedomain.MemberAmount, categories []expensedomain.CategoryAmount, unallocated []money.Money) ([]generated.MemberAmount, []generated.CategoryAmount, []generated.Money, error) {
	memberDTOs := make([]generated.MemberAmount, 0, len(members))
	categoryDTOs := make([]generated.CategoryAmount, 0, len(categories))
	unallocatedDTOs := make([]generated.Money, 0, len(unallocated))
	converter := contract.MoneyConverter{}
	for _, value := range members {
		amount, err := converter.ToDTO(value.Amount)
		if err != nil {
			return nil, nil, nil, err
		}
		memberDTOs = append(memberDTOs, generated.MemberAmount{MemberId: string(value.MemberID), Amount: amount})
	}
	for _, value := range categories {
		amount, err := converter.ToDTO(value.Amount)
		if err != nil {
			return nil, nil, nil, err
		}
		dto := generated.CategoryAmount{Amount: amount}
		if value.CategoryID != "" {
			dto.CategoryId = &value.CategoryID
		}
		categoryDTOs = append(categoryDTOs, dto)
	}
	for _, value := range unallocated {
		amount, err := converter.ToDTO(value)
		if err != nil {
			return nil, nil, nil, err
		}
		unallocatedDTOs = append(unallocatedDTOs, amount)
	}
	return memberDTOs, categoryDTOs, unallocatedDTOs, nil
}
