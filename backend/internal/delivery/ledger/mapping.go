package ledger

import (
	"sort"
	"strconv"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Server) transactionDTO(p household.Principal, v application.View) (generated.Transaction, error) {
	r := v.Revision
	if err := r.Validate(); err != nil {
		return generated.Transaction{}, err
	}
	out := generated.Transaction{Id: r.OperationID, Revision: int64(r.Revision), ActorId: string(r.ActorID), HouseholdId: string(p.HouseholdID()), Type: generated.TransactionType(r.Type), State: generated.TransactionState(r.State), OccurredAt: r.OccurredAt.String(), CashDate: r.CashDate.String(), AiState: "waiting", Origin: "legacy", FeeKnowledge: "unknown", Postings: []generated.Posting{}, ReceiptItems: []generated.ReceiptItem{}, Refunds: []generated.RefundAttribution{}, Sources: []generated.SourceReference{}, BalanceEffects: []generated.TransactionBalanceEffect{}, EconomicComponents: []generated.EconomicComponent{}, Holds: []generated.TransactionHold{}}
	for _, refund := range v.Refunds {
		dto, refundErr := s.refundDTO(refund)
		if refundErr != nil {
			return out, refundErr
		}
		out.Refunds = append(out.Refunds, dto)
		if r.Type == ledger.Refund && refund.OperationID == r.OperationID {
			purchaseID := refund.PurchaseID
			out.OriginalTransactionId = &purchaseID
		}
	}
	if r.Participation.GroupID != "" {
		v := r.Participation
		dto := generated.EffectParticipation{GroupId: v.GroupID, Kind: generated.MatchingKind(v.Kind), State: generated.EffectParticipationState(v.State), Components: []generated.EffectContribution{}}
		for _, c := range v.Parts {
			dto.Components = append(dto.Components, generated.EffectContribution{Position: c.Position, CarrierId: c.CarrierID, CarrierPosition: c.CarrierPosition, Role: generated.EffectContributionRole(c.Role), State: generated.EffectContributionState(c.State), At: c.At.String()})
		}
		out.Participation = &dto
	}
	out.AccountingState = generated.TransactionAccountingState(r.Accounting())
	out.ProtectedFields = []generated.FieldProtection{}
	out.SourceConflict = r.SourceConflict
	if v.Review != nil {
		rv := v.Review
		out.Review = &generated.TransactionReview{Revision: int64(rv.Revision), ActorId: string(rv.ActorID), State: generated.TransactionReviewState(rv.State), Rationale: rv.Rationale, RecordedAt: rv.At.String(), Evidence: []generated.DecisionEvidence{}}
		for _, e := range rv.Evidence {
			out.Review.Evidence = append(out.Review.Evidence, generated.DecisionEvidence{Kind: generated.DecisionEvidenceKind(e.Kind), Id: e.ID, Revision: int64(e.Revision)})
		}
		if rv.Proposal != nil {
			proposal := &generated.ClassificationProposal{ReceiptItems: []generated.ClassificationProposalItem{}}
			if rv.Proposal.CategoryID != "" {
				proposal.CategoryId = &rv.Proposal.CategoryID
			}
			if rv.Proposal.MerchantID != "" {
				proposal.MerchantId = &rv.Proposal.MerchantID
			}
			if rv.Proposal.MerchantAlias != "" {
				proposal.MerchantAlias = &rv.Proposal.MerchantAlias
			}
			for _, item := range rv.Proposal.ReceiptItems {
				dto, e := classificationProposalItemDTO(item)
				if e != nil {
					return out, e
				}
				proposal.ReceiptItems = append(proposal.ReceiptItems, dto)
			}
			out.Review.ClassificationProposal = proposal
		}
	}
	out.SourceFacts = []generated.SourceTransactionFact{}
	for _, fact := range v.SourceFacts {
		dto, err := s.sourceFactDTO(fact)
		if err != nil {
			return out, err
		}
		out.SourceFacts = append(out.SourceFacts, dto)
	}
	if r.DecisionID != "" {
		out.DecisionId = &r.DecisionID
	}
	if r.ReviewState != "" {
		out.AiState = generated.TransactionAiState(r.ReviewState)
	}
	fields := []string{}
	for f := range r.Protections {
		fields = append(fields, string(f))
	}
	sort.Strings(fields)
	for _, f := range fields {
		p := r.Protections[ledger.Field(f)]
		v := generated.FieldProtection{Field: generated.LedgerField(f), Revision: int64(p.Revision)}
		if p.DecisionID != "" {
			v.DecisionId = &p.DecisionID
		}
		out.ProtectedFields = append(out.ProtectedFields, v)
	}
	if r.Origin != "" {
		out.Origin = generated.TransactionOrigin(r.Origin)
	}
	if r.FeeKnowledge != "" {
		out.FeeKnowledge = generated.TransactionFeeKnowledge(r.FeeKnowledge)
	}
	if r.PostedAt.String() != "" {
		value := r.PostedAt.String()
		out.PostedAt = &value
	}
	if r.Timezone.String() != "" {
		value := r.Timezone.String()
		out.Timezone = &value
	}
	if r.ExpenseMonth.String() != "" {
		value := r.ExpenseMonth.String()
		out.ExpenseMonth = &value
	}
	if r.Merchant != "" {
		out.Merchant = &r.Merchant
	}
	if r.CategoryID != "" {
		out.CategoryId = &r.CategoryID
	}
	if r.MerchantID != "" {
		out.MerchantId = &r.MerchantID
	}
	for _, item := range r.ReceiptItems {
		dto, err := receiptItemDTO(item)
		if err != nil {
			return out, err
		}
		out.ReceiptItems = append(out.ReceiptItems, dto)
	}
	if r.Note != "" {
		out.Note = &r.Note
	}
	if r.AttachmentID != "" {
		out.AttachmentId = &r.AttachmentID
	}
	if r.PnLBasis != "" {
		value := generated.TransactionPnlBasis(r.PnLBasis)
		out.PnlBasis = &value
	}
	allocation, err := s.allocationDTO(r.Allocation)
	if err != nil {
		return out, err
	}
	out.Allocation = allocation
	if r.PayerState == "known" {
		err = out.Payer.FromKnownPayer(generated.KnownPayer{State: "known", MemberId: string(r.PayerMemberID)})
	} else {
		err = out.Payer.FromUnspecifiedPayer(generated.UnspecifiedPayer{State: generated.UnspecifiedPayerState(r.PayerState)})
	}
	if err != nil {
		return out, err
	}
	out.Quality.Coverage, err = s.boundary.CoverageToDTO(v.Coverage)
	if err != nil {
		return out, err
	}
	out.Quality.Freshness = "unknown"
	for _, posting := range r.Postings {
		dto, err := s.postingDTO(posting)
		if err != nil {
			return out, err
		}
		out.Postings = append(out.Postings, dto)
	}
	for _, src := range v.Sources {
		out.Sources = append(out.Sources, generated.SourceReference{Provider: generated.Provider(src.Key.Provider), ExternalAccountId: src.Key.ExternalAccountID, Product: src.Key.Product, Log: src.Key.Log, SourceId: src.Key.RecordID, Revision: strconv.FormatUint(src.Revision, 10), ConnectionId: src.ConnectionID})
	}
	components, err := r.Components()
	if err != nil {
		return out, err
	}
	for _, c := range components {
		amount, e := (contract.MoneyConverter{}).ToDTO(c.Money)
		if e != nil {
			return out, e
		}
		out.EconomicComponents = append(out.EconomicComponents, generated.EconomicComponent{Kind: generated.EconomicComponentKind(c.Kind), Money: amount, Treatment: generated.PostingTreatment(c.Treatment)})
	}
	effects, err := r.BalanceEffects()
	if err != nil {
		return out, err
	}
	for _, e := range effects {
		dto, err := s.effectDTO(e)
		if err != nil {
			return out, err
		}
		out.BalanceEffects = append(out.BalanceEffects, dto)
	}
	holds, err := r.Holds()
	if err != nil {
		return out, err
	}
	for _, h := range holds {
		amount, e := s.positiveDTO(h.Amount)
		if e != nil {
			return out, e
		}
		out.Holds = append(out.Holds, generated.TransactionHold{AccountId: h.AccountID, Amount: amount, Funding: generated.PostingFunding(h.Funding)})
	}
	exchange, err := r.ExchangeAmounts()
	if err != nil {
		return out, err
	}
	if exchange != nil {
		x := &generated.ExecutedExchange{}
		x.Sent, err = s.positiveDTO(exchange.Sent)
		if err != nil {
			return out, err
		}
		x.Received, err = s.positiveDTO(exchange.Received)
		if err != nil {
			return out, err
		}
		out.Exchange = x
	}
	return out, nil
}

func receiptItemDTO(item ledger.ReceiptItem) (generated.ReceiptItem, error) {
	common, err := receiptItemValues(item)
	if err != nil {
		return generated.ReceiptItem{}, err
	}
	allocation, err := allocationDTO(item.Allocation)
	if err != nil {
		return generated.ReceiptItem{}, err
	}
	out := generated.ReceiptItem{Id: common.ID, Name: common.Name, Quantity: common.Quantity, Gross: common.Gross, Discount: common.Discount, Net: common.Net, Allocation: allocation, CategoryId: common.CategoryID}
	return out, nil
}

func classificationProposalItemDTO(item ledger.ReceiptItem) (generated.ClassificationProposalItem, error) {
	common, err := receiptItemValues(item)
	if err != nil {
		return generated.ClassificationProposalItem{}, err
	}
	return generated.ClassificationProposalItem{Id: common.ID, Name: common.Name, Quantity: common.Quantity, Gross: common.Gross, Discount: common.Discount, Net: common.Net, CategoryId: common.CategoryID}, nil
}

type receiptItemValuesDTO struct {
	ID, Name, Quantity   string
	Gross, Discount, Net generated.Money
	CategoryID           *string
}

func receiptItemValues(item ledger.ReceiptItem) (receiptItemValuesDTO, error) {
	gross, err := (contract.MoneyConverter{}).ToDTO(item.Gross)
	if err != nil {
		return receiptItemValuesDTO{}, err
	}
	discount, err := (contract.MoneyConverter{}).ToDTO(item.Discount)
	if err != nil {
		return receiptItemValuesDTO{}, err
	}
	net, err := item.Net()
	if err != nil {
		return receiptItemValuesDTO{}, err
	}
	netDTO, err := (contract.MoneyConverter{}).ToDTO(net)
	if err != nil {
		return receiptItemValuesDTO{}, err
	}
	out := receiptItemValuesDTO{ID: item.ID, Name: item.Name, Quantity: item.Quantity, Gross: gross, Discount: discount, Net: netDTO}
	if item.CategoryID != "" {
		out.CategoryID = &item.CategoryID
	}
	return out, nil
}

func (s *Server) allocationDTO(value ledger.AllocationSnapshot) (generated.AllocationSnapshot, error) {
	return allocationDTO(value)
}

func allocationDTO(value ledger.AllocationSnapshot) (generated.AllocationSnapshot, error) {
	if err := value.Validate(); err != nil {
		return generated.AllocationSnapshot{}, err
	}
	out := generated.AllocationSnapshot{State: generated.AllocationSnapshotState(value.State), Origin: generated.AllocationSnapshotOrigin(value.Origin), Reason: value.Reason, Members: []generated.MemberAmount{}, Unallocated: []generated.Money{}, Rules: []generated.AllocationRuleReference{}}
	if value.Mode != "" {
		mode := generated.AllocationSnapshotMode(value.Mode)
		out.Mode = &mode
	}
	if value.Purpose != "" {
		purpose := generated.AllocationSnapshotPurpose(value.Purpose)
		out.Purpose = &purpose
	}
	converter := contract.MoneyConverter{}
	for _, member := range value.Members {
		amount, err := converter.ToDTO(member.Money)
		if err != nil {
			return out, err
		}
		out.Members = append(out.Members, generated.MemberAmount{MemberId: string(member.MemberID), Amount: amount})
	}
	for _, amount := range value.Unallocated {
		dto, err := converter.ToDTO(amount)
		if err != nil {
			return out, err
		}
		out.Unallocated = append(out.Unallocated, dto)
	}
	for _, rule := range value.RuleRefs {
		out.Rules = append(out.Rules, generated.AllocationRuleReference{RuleId: rule.ID, Revision: int64(rule.Revision)})
	}
	return out, nil
}
func (s *Server) effectDTO(e ledger.BalanceEffect) (generated.TransactionBalanceEffect, error) {
	out := generated.TransactionBalanceEffect{AccountId: e.AccountID, At: e.At.String()}
	var err error
	out.Owned, err = s.boundary.AmountToDTO(e.Owned)
	if err != nil {
		return out, err
	}
	out.Available, err = s.boundary.AmountToDTO(e.Available)
	if err != nil {
		return out, err
	}
	out.Locked, err = s.boundary.AmountToDTO(e.Locked)
	if err != nil {
		return out, err
	}
	out.Debt, err = s.boundary.AmountToDTO(e.Debt)
	return out, err
}
