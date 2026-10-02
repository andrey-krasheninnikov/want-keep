package domain

import (
	"slices"
)

func (r Revision) Correct(c Correction) (Revision, []Field, error) {
	if r.Type == Opening {
		return r, nil, ErrFeatureUnavailable
	}
	next := r.Clone()
	fields := []Field{}
	if c.Principal != nil {
		if !slices.Contains([]Type{Income, Expense, Transfer, Exchange, Refund}, r.Type) {
			return r, nil, ErrFeatureUnavailable
		}
		old := r.rolePostings(Principal)
		proposed := slices.Clone(*c.Principal)
		if len(old) != len(*c.Principal) {
			return r, nil, ErrInvalidRevision
		}
		for i, p := range proposed {
			if p.Funding == "" {
				p.Funding = old[i].Funding
			}
			if p.Treatment == "" {
				p.Treatment = old[i].Treatment
			}
			proposed[i] = p
			if p.Role != Principal || p.AccountID != old[i].AccountID || p.Money.Asset() != old[i].Money.Asset() || p.Funding != old[i].Funding || p.Treatment != old[i].Treatment {
				return r, nil, ErrInvalidRevision
			}
		}
		next.replaceRole(Principal, proposed)
		if !r.FieldEqual(next, PrincipalField) {
			fields = append(fields, PrincipalField)
		}
	}
	if c.Fees != nil {
		if !slices.Contains([]Type{Income, Expense, Transfer, Exchange, Refund}, r.Type) {
			return r, nil, ErrFeatureUnavailable
		}
		old := r.rolePostings(Fee)
		proposed := slices.Clone(*c.Fees)
		for i, p := range proposed {
			if i < len(old) && p.AccountID == old[i].AccountID && p.Money.Asset() == old[i].Money.Asset() {
				p.FeeID = old[i].FeeID
				if p.Funding == "" {
					p.Funding = old[i].Funding
				}
				if p.Treatment == "" {
					p.Treatment = old[i].Treatment
				}
			}
			if p.Role != Fee || p.Treatment == Included || p.Treatment == Valuation {
				return r, nil, ErrInvalidRevision
			}
			proposed[i] = p
		}
		next.replaceRole(Fee, proposed)
		next.FeeKnowledge = KnownFees
		if !r.FieldEqual(next, FeesField) {
			fields = append(fields, FeesField)
		}
	}
	if c.OccurredAt != nil {
		next.OccurredAt = *c.OccurredAt
		if !r.FieldEqual(next, DateField) {
			fields = append(fields, DateField)
		}
	}
	if c.Payer != nil {
		if r.Type == Expense && c.Payer.State == "not_applicable" {
			return r, nil, ErrInvalidRevision
		}
		next.PayerState = c.Payer.State
		next.PayerMemberID = c.Payer.MemberID
		if !r.FieldEqual(next, PayerField) {
			fields = append(fields, PayerField)
		}
	}
	if c.Merchant != nil {
		next.Merchant = *c.Merchant
		if !r.FieldEqual(next, MerchantField) {
			fields = append(fields, MerchantField)
		}
	}
	if c.Note != nil {
		next.Note = *c.Note
		if !r.FieldEqual(next, NoteField) {
			fields = append(fields, NoteField)
		}
	}
	if c.CategoryID != nil {
		if r.Type != Expense {
			return r, nil, ErrFeatureUnavailable
		}
		next.CategoryID = *c.CategoryID
		if !r.FieldEqual(next, CategoryField) {
			fields = append(fields, CategoryField)
		}
	}
	if c.MerchantID != nil {
		if r.Type != Expense {
			return r, nil, ErrFeatureUnavailable
		}
		next.MerchantID = *c.MerchantID
		if !r.FieldEqual(next, MerchantIDField) {
			fields = append(fields, MerchantIDField)
		}
	}
	if c.ReceiptItems != nil {
		if r.Type != Expense {
			return r, nil, ErrFeatureUnavailable
		}
		items, err := c.ReceiptItems.Build(next)
		if err != nil {
			return r, nil, err
		}
		next.ReceiptItems = items
		if !r.FieldEqual(next, ReceiptItemsField) {
			fields = append(fields, ReceiptItemsField)
		}
	}
	if c.Allocation != nil {
		if r.Type == Income {
			return r, nil, ErrFeatureUnavailable
		}
		allocated, err := next.WithAllocation(c.Allocation.Allocation, c.Allocation.Items, c.Allocation.Members)
		if err != nil {
			return r, nil, err
		}
		next = allocated
		if !r.FieldEqual(next, AllocationField) {
			fields = append(fields, AllocationField)
		}
	} else if slices.Contains(fields, PrincipalField) || slices.Contains(fields, FeesField) || slices.Contains(fields, ReceiptItemsField) {
		allocated, err := next.RefreshAllocation()
		if err != nil {
			return r, nil, err
		}
		next = allocated
		if !r.FieldEqual(next, AllocationField) {
			fields = append(fields, AllocationField)
		}
	}
	if err := next.validateClassification(); err != nil {
		return r, nil, err
	}
	if len(fields) == 0 {
		return r, nil, ErrNoChange
	}
	return next, fields, nil
}

func allocationInput(snapshot AllocationSnapshot) AllocationInput {
	return AllocationInput{Mode: snapshot.Mode, Purpose: snapshot.Purpose, Members: cloneAllocationInputs(snapshot.Inputs), Reason: snapshot.Reason, Origin: snapshot.Origin, RuleRefs: slices.Clone(snapshot.RuleRefs)}
}
