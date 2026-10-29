package domain

import (
	"math/big"
	"slices"

	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (r Revision) rolePostings(role Role) []Posting {
	out := []Posting{}
	for _, p := range r.Postings {
		if p.Role == role {
			out = append(out, p)
		}
	}
	return out
}
func (r *Revision) replaceRole(role Role, entries []Posting) {
	out := []Posting{}
	inserted := false
	for _, p := range r.Postings {
		if p.Role == role {
			if !inserted {
				out = append(out, entries...)
				inserted = true
			}
			continue
		}
		out = append(out, p)
	}
	if !inserted {
		out = append(out, entries...)
	}
	r.Postings = out
}

func (r Revision) FieldEqual(other Revision, field Field) bool {
	switch field {
	case PrincipalField, FeesField:
		role := Principal
		if field == FeesField {
			role = Fee
			if r.FeeKnowledge != other.FeeKnowledge {
				return false
			}
		}
		a, b := r.rolePostings(role), other.rolePostings(role)
		if len(a) != len(b) {
			return false
		}
		for i, p := range a {
			q := b[i]
			if p.AccountID != q.AccountID || p.Role != q.Role || p.Funding != q.Funding || (p.Treatment != q.Treatment && !(p.MovesMoney() && q.MovesMoney())) || p.Money.Asset() != q.Money.Asset() || !p.SameMoney(q) {
				return false
			}
		}
		return true
	case DateField:
		return r.OccurredAt == other.OccurredAt && r.OccurredPrecision == other.OccurredPrecision
	case PayerField:
		return r.PayerState == other.PayerState && r.PayerMemberID == other.PayerMemberID
	case MerchantField:
		return r.Merchant == other.Merchant
	case NoteField:
		return r.Note == other.Note
	case MatchingField:
		a, b := r.Participation, other.Participation
		return a.GroupID == b.GroupID && a.Kind == b.Kind && a.State == b.State && slices.Equal(a.Parts, b.Parts)
	case ContributionField:
		return slices.Equal(r.Participation.Parts, other.Participation.Parts)
	case AccountingField:
		return r.Accounting() == other.Accounting()
	case CategoryField:
		return r.CategoryID == other.CategoryID
	case MerchantIDField:
		return r.MerchantID == other.MerchantID
	case ReceiptItemsField:
		return slices.EqualFunc(r.ReceiptItems, other.ReceiptItems, func(a, b ReceiptItem) bool {
			return a.ID == b.ID && a.Name == b.Name && a.Quantity == b.Quantity && a.CategoryID == b.CategoryID && a.Gross.Asset() == b.Gross.Asset() && a.Gross.Amount() == b.Gross.Amount() && a.Discount.Asset() == b.Discount.Asset() && a.Discount.Amount() == b.Discount.Amount()
		})
	case AllocationField:
		return allocationEqual(r.Allocation, other.Allocation) && slices.EqualFunc(r.ReceiptItems, other.ReceiptItems, func(a, b ReceiptItem) bool { return allocationEqual(a.Allocation, b.Allocation) })
	}
	return false
}

func (r *Revision) CopyField(from Revision, field Field) error {
	switch field {
	case PrincipalField:
		r.replaceRole(Principal, from.rolePostings(Principal))
	case FeesField:
		r.replaceRole(Fee, from.rolePostings(Fee))
		r.FeeKnowledge = from.FeeKnowledge
	case DateField:
		r.OccurredAt = from.OccurredAt
		r.OccurredPrecision = from.OccurredPrecision
	case PayerField:
		r.PayerState = from.PayerState
		r.PayerMemberID = from.PayerMemberID
	case MerchantField:
		r.Merchant = from.Merchant
	case NoteField:
		r.Note = from.Note
	case MatchingField:
		r.Participation = from.Clone().Participation
	case AccountingField:
		r.AccountingState = from.Accounting()
	case CategoryField:
		r.CategoryID = from.CategoryID
	case MerchantIDField:
		r.MerchantID = from.MerchantID
	case ReceiptItemsField:
		r.ReceiptItems = slices.Clone(from.ReceiptItems)
	case AllocationField:
		r.Allocation = from.Allocation.Clone()
		for i := range r.ReceiptItems {
			for _, item := range from.ReceiptItems {
				if r.ReceiptItems[i].ID == item.ID {
					r.ReceiptItems[i].Allocation = item.Allocation.Clone()
				}
			}
		}
	default:
		return ErrInvalidRevision
	}
	return nil
}

func allocationEqual(a, b AllocationSnapshot) bool {
	a, b = a.Clone(), b.Clone()
	if a.State != b.State || a.Purpose != b.Purpose || a.Mode != b.Mode || a.Origin != b.Origin || a.Reason != b.Reason || len(a.Inputs) != len(b.Inputs) || len(a.Members) != len(b.Members) || len(a.Unallocated) != len(b.Unallocated) || !slices.Equal(a.RuleRefs, b.RuleRefs) {
		return false
	}
	if (a.Basis == nil) != (b.Basis == nil) || a.Basis != nil && !allocationInputEqual(*a.Basis, *b.Basis) {
		return false
	}
	for i := range a.Inputs {
		left, right := a.Inputs[i], b.Inputs[i]
		if left.MemberID != right.MemberID || !sameDecimal(left.Share, right.Share) || (left.Amount == nil) != (right.Amount == nil) || left.Amount != nil && !sameMoney(*left.Amount, *right.Amount) {
			return false
		}
	}
	for i := range a.Members {
		if a.Members[i].MemberID != b.Members[i].MemberID || !sameMoney(a.Members[i].Money, b.Members[i].Money) {
			return false
		}
	}
	for i := range a.Unallocated {
		if !sameMoney(a.Unallocated[i], b.Unallocated[i]) {
			return false
		}
	}
	return true
}

func allocationInputEqual(a, b AllocationInput) bool {
	a, b = cloneAllocationInput(a), cloneAllocationInput(b)
	if a.Mode != b.Mode || a.Purpose != b.Purpose || a.Reason != b.Reason || a.Origin != b.Origin || len(a.Members) != len(b.Members) || !slices.Equal(a.RuleRefs, b.RuleRefs) {
		return false
	}
	for i := range a.Members {
		left, right := a.Members[i], b.Members[i]
		if left.MemberID != right.MemberID || !sameDecimal(left.Share, right.Share) || (left.Amount == nil) != (right.Amount == nil) || left.Amount != nil && !sameMoney(*left.Amount, *right.Amount) {
			return false
		}
	}
	return true
}

func sameMoney(left, right money.Money) bool {
	compared, err := left.Compare(right)
	return err == nil && compared == 0
}

func sameDecimal(left, right string) bool {
	if left == "" || right == "" {
		return left == right
	}
	leftValue, leftOK := new(big.Rat).SetString(left)
	rightValue, rightOK := new(big.Rat).SetString(right)
	return leftOK && rightOK && leftValue.Cmp(rightValue) == 0
}
