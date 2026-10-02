package domain

import (
	"errors"
	"sort"
	"strings"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var (
	ErrInvalidRefund           = errors.New("invalid refund")
	ErrRefundExceedsPurchase   = errors.New("refund exceeds purchase")
	ErrClarificationRequired   = errors.New("refund needs clarification")
	ErrHistoricalBasisConflict = errors.New("historical refund valuation conflicts with purchase revision")
)

type State string

const (
	Applied       State = "applied"
	Clarification State = "clarification"
	Inactive      State = "inactive"
)

type ItemPortion struct {
	ItemID string
	Amount money.Money
}

type MemberAmount struct {
	MemberID household.MembershipID
	Amount   money.Money
}

type CategoryAmount struct {
	CategoryID string
	Amount     money.Money
}

type ValuationBasis struct {
	Purchase money.Money
	Value    money.Money
	Ref      string
}

type ValuationShare struct {
	Basis ValuationBasis
	Value money.Money
}

type Valuation struct {
	Value       money.Money
	Ref         string
	Basis       *ValuationBasis
	Members     []MemberAmount
	Categories  []CategoryAmount
	Unallocated []money.Money
}

type Refund struct {
	OperationID, PurchaseID, Reason            string
	Revision, PurchaseRevision, RefundRevision uint64
	ActorID                                    household.UserID
	State                                      State
	ExpenseMonth                               calendar.Month
	CashDate                                   calendar.Date
	RecordedAt                                 calendar.Instant
	Amount, Remaining                          money.Money
	Items                                      []ItemPortion
	Members                                    []MemberAmount
	Categories                                 []CategoryAmount
	Unallocated                                []money.Money
	Valuation                                  *Valuation
}

func (r Refund) Validate() error {
	if r.OperationID == "" || r.PurchaseID == "" || r.OperationID == r.PurchaseID || r.Revision < 1 || r.Revision > 9007199254740991 || r.PurchaseRevision < 1 || r.RefundRevision < 1 || r.ActorID == "" || !validState(r.State) || r.ExpenseMonth.String() == "" || r.CashDate.String() == "" || r.RecordedAt.String() == "" || strings.TrimSpace(r.Reason) == "" || utf8.RuneCountInString(r.Reason) > 2000 || r.Amount.Validate() != nil || r.Amount.Sign() <= 0 || r.Remaining.Validate() != nil || r.Remaining.Sign() < 0 || r.Remaining.Asset() != r.Amount.Asset() {
		return ErrInvalidRefund
	}
	seen := map[string]bool{}
	for _, item := range r.Items {
		if item.ItemID == "" || seen[item.ItemID] || item.Amount.Validate() != nil || item.Amount.Sign() <= 0 || item.Amount.Asset() != r.Amount.Asset() {
			return ErrInvalidRefund
		}
		seen[item.ItemID] = true
	}
	if r.State == Clarification && (len(r.Members)+len(r.Categories)+len(r.Unallocated) != 0 || r.Valuation != nil) {
		return ErrInvalidRefund
	}
	if r.State == Inactive && (len(r.Members)+len(r.Categories)+len(r.Unallocated) != 0 || r.Valuation != nil) {
		return ErrInvalidRefund
	}
	if r.State == Applied {
		if err := validateEffects(r.Amount, r.Members, r.Unallocated); err != nil {
			return err
		}
		if err := validateCategories(r.Amount, r.Categories); err != nil {
			return err
		}
		if r.Valuation != nil {
			if r.Valuation.Value.Validate() != nil || r.Valuation.Value.Sign() < 0 || strings.TrimSpace(r.Valuation.Ref) == "" || r.Valuation.Basis == nil || r.Valuation.Basis.Ref != r.Valuation.Ref || r.Valuation.Basis.Purchase.Validate() != nil || r.Valuation.Basis.Purchase.Sign() <= 0 || r.Valuation.Basis.Purchase.Asset() != r.Amount.Asset() || r.Valuation.Basis.Value.Validate() != nil || r.Valuation.Basis.Value.Sign() <= 0 || r.Valuation.Basis.Value.Asset() != r.Valuation.Value.Asset() || validateEffects(r.Valuation.Value, r.Valuation.Members, r.Valuation.Unallocated) != nil || validateCategories(r.Valuation.Value, r.Valuation.Categories) != nil {
				return ErrInvalidRefund
			}
		}
	}
	return nil
}

func validState(state State) bool {
	return state == Applied || state == Clarification || state == Inactive
}

// SameCalculation reports whether recalculation changed persisted attribution.
func (r Refund) SameCalculation(other Refund) bool {
	if r.OperationID != other.OperationID || r.PurchaseID != other.PurchaseID || r.PurchaseRevision != other.PurchaseRevision || r.RefundRevision != other.RefundRevision || r.State != other.State || r.ExpenseMonth != other.ExpenseMonth || r.CashDate != other.CashDate || !sameMoney(r.Amount, other.Amount) || !sameMoney(r.Remaining, other.Remaining) || !sameItems(r.Items, other.Items) || !sameMembers(r.Members, other.Members) || !sameCategories(r.Categories, other.Categories) || !sameMoneySlice(r.Unallocated, other.Unallocated) {
		return false
	}
	if r.Valuation == nil || other.Valuation == nil {
		return r.Valuation == nil && other.Valuation == nil
	}
	return r.Valuation.Ref == other.Valuation.Ref && sameMoney(r.Valuation.Value, other.Valuation.Value) && sameMembers(r.Valuation.Members, other.Valuation.Members) && sameCategories(r.Valuation.Categories, other.Valuation.Categories) && sameMoneySlice(r.Valuation.Unallocated, other.Valuation.Unallocated)
}

func Calculate(purchase, refund ledger.Revision, requested []ItemPortion, refunded money.Money, refundedItems map[string]money.Money, valuation *ValuationShare, revision uint64, reason string, actor household.UserID, recordedAt calendar.Instant) (Refund, error) {
	purchaseAmount, err := principal(purchase, -1)
	if err != nil || purchase.Type != ledger.Expense {
		return Refund{}, ErrInvalidRefund
	}
	refundAmount, err := principal(refund, 1)
	if err != nil || refund.Type != ledger.Refund {
		return Refund{}, ErrInvalidRefund
	}
	if refunded.Validate() != nil || refunded.Asset() != purchaseAmount.Asset() || refundAmount.Asset() != purchaseAmount.Asset() {
		return Refund{}, ErrInvalidRefund
	}
	remaining, err := purchaseAmount.Subtract(refunded)
	if err != nil || remaining.Sign() < 0 {
		return Refund{}, ErrRefundExceedsPurchase
	}
	active := Active(purchase, refund)
	if active {
		if compared, _ := refundAmount.Compare(remaining); compared > 0 {
			return Refund{}, ErrRefundExceedsPurchase
		}
	}
	result := Refund{OperationID: refund.OperationID, PurchaseID: purchase.OperationID, Revision: revision, PurchaseRevision: purchase.Revision, RefundRevision: refund.Revision, ActorID: actor, State: Applied, ExpenseMonth: purchase.ExpenseMonth, CashDate: refund.CashDate, RecordedAt: recordedAt, Amount: refundAmount, Remaining: remaining, Reason: reason, Items: cloneItems(requested)}
	if active {
		remaining, _ = remaining.Subtract(refundAmount)
		result.Remaining = remaining
	}
	if len(purchase.ReceiptItems) == 0 {
		if len(requested) != 0 {
			return Refund{}, ErrInvalidRefund
		}
		if !active {
			result.State = Inactive
			return result, result.Validate()
		}
		result.Members, result.Unallocated, err = allocate(refundAmount, purchase.Allocation)
		if err != nil {
			return Refund{}, err
		}
		result.Categories = []CategoryAmount{{CategoryID: purchase.CategoryID, Amount: refundAmount}}
	} else {
		if len(requested) == 0 {
			if active {
				result.State = Clarification
			} else {
				result.State = Inactive
			}
			return result, result.Validate()
		}
		items := map[string]ledger.ReceiptItem{}
		for _, item := range purchase.ReceiptItems {
			items[item.ID] = item
		}
		zero, _ := money.NewMoney("0", refundAmount.Asset())
		total := zero
		members := map[household.MembershipID]money.Money{}
		categories := map[string]money.Money{}
		unallocated := zero
		for _, portion := range requested {
			item, ok := items[portion.ItemID]
			if !ok || portion.Amount.Validate() != nil || portion.Amount.Sign() <= 0 || portion.Amount.Asset() != refundAmount.Asset() {
				return Refund{}, ErrInvalidRefund
			}
			net, netErr := item.Net()
			if netErr != nil {
				return Refund{}, ErrInvalidRefund
			}
			if active {
				already := refundedItems[portion.ItemID]
				if already.Validate() != nil {
					already, _ = money.NewMoney("0", refundAmount.Asset())
				}
				left, subErr := net.Subtract(already)
				if subErr != nil || left.Sign() < 0 {
					return Refund{}, ErrRefundExceedsPurchase
				}
				if compared, _ := portion.Amount.Compare(left); compared > 0 {
					return Refund{}, ErrRefundExceedsPurchase
				}
			}
			total, err = total.Add(portion.Amount)
			if err != nil {
				return Refund{}, ErrInvalidRefund
			}
			if active {
				parts, unknown, allocErr := allocate(portion.Amount, item.Allocation)
				if allocErr != nil {
					return Refund{}, allocErr
				}
				for _, part := range parts {
					members[part.MemberID], err = add(members[part.MemberID], part.Amount)
					if err != nil {
						return Refund{}, ErrInvalidRefund
					}
				}
				for _, amount := range unknown {
					unallocated, err = unallocated.Add(amount)
					if err != nil {
						return Refund{}, ErrInvalidRefund
					}
				}
				categories[item.CategoryID], err = add(categories[item.CategoryID], portion.Amount)
				if err != nil {
					return Refund{}, ErrInvalidRefund
				}
			}
		}
		if compared, _ := total.Compare(refundAmount); compared != 0 {
			return Refund{}, ErrInvalidRefund
		}
		if !active {
			result.State = Inactive
			return result, result.Validate()
		}
		for id, amount := range members {
			result.Members = append(result.Members, MemberAmount{MemberID: id, Amount: amount})
		}
		for id, amount := range categories {
			result.Categories = append(result.Categories, CategoryAmount{CategoryID: id, Amount: amount})
		}
		if unallocated.Sign() > 0 {
			result.Unallocated = []money.Money{unallocated}
		}
		sortEffects(&result)
	}
	if valuation != nil {
		valuation, valuationErr := valueRefund(*valuation, result)
		if valuationErr != nil {
			return Refund{}, valuationErr
		}
		result.Valuation = &valuation
	}
	return result, result.Validate()
}

// Active reports whether both principal postings contribute to current accounting.
func Active(purchase, refund ledger.Revision) bool {
	return principalContributes(purchase) && principalContributes(refund)
}

func principalContributes(revision ledger.Revision) bool {
	if revision.State != ledger.Posted || revision.Accounting() != ledger.IncludedInAccounting {
		return false
	}
	for position, posting := range revision.Postings {
		if posting.Role == ledger.Principal && posting.MovesMoney() {
			return revision.Contributes(position) && revision.ContributionState(position) == ledger.Posted
		}
	}
	return false
}

func Clarify(purchase, refund ledger.Revision, requested []ItemPortion, refunded money.Money, revision uint64, reason string, actor household.UserID, recordedAt calendar.Instant) (Refund, error) {
	purchaseAmount, err := principal(purchase, -1)
	if err != nil {
		return Refund{}, ErrInvalidRefund
	}
	refundAmount, err := principal(refund, 1)
	if err != nil || refunded.Asset() != purchaseAmount.Asset() {
		return Refund{}, ErrInvalidRefund
	}
	remaining, err := purchaseAmount.Subtract(refunded)
	if err != nil || remaining.Sign() < 0 {
		return Refund{}, ErrRefundExceedsPurchase
	}
	if compared, _ := refundAmount.Compare(remaining); compared > 0 {
		return Refund{}, ErrRefundExceedsPurchase
	}
	remaining, _ = remaining.Subtract(refundAmount)
	r := Refund{OperationID: refund.OperationID, PurchaseID: purchase.OperationID, Revision: revision, PurchaseRevision: purchase.Revision, RefundRevision: refund.Revision, ActorID: actor, State: Clarification, ExpenseMonth: purchase.ExpenseMonth, CashDate: refund.CashDate, RecordedAt: recordedAt, Amount: refundAmount, Remaining: remaining, Items: cloneItems(requested), Reason: reason}
	return r, r.Validate()
}

func principal(revision ledger.Revision, sign int) (money.Money, error) {
	var found *money.Money
	for _, posting := range revision.Postings {
		if posting.Role != ledger.Principal || !posting.MovesMoney() {
			continue
		}
		if found != nil {
			return money.Money{}, ErrInvalidRefund
		}
		value := posting.Money
		found = &value
	}
	if found == nil || found.Sign() != sign {
		return money.Money{}, ErrInvalidRefund
	}
	if sign < 0 {
		zero, _ := money.NewMoney("0", found.Asset())
		return zero.Subtract(*found)
	}
	return *found, nil
}

func allocate(total money.Money, snapshot ledger.AllocationSnapshot) ([]MemberAmount, []money.Money, error) {
	weights := []money.Weight{}
	for _, member := range snapshot.Members {
		if member.Money.Asset() == total.Asset() {
			weights = append(weights, money.Weight{ID: "member:" + string(member.MemberID), Value: member.Money.Amount()})
		}
	}
	for _, amount := range snapshot.Unallocated {
		if amount.Asset() == total.Asset() {
			weights = append(weights, money.Weight{ID: "unallocated:" + string(amount.Asset()), Value: amount.Amount()})
		}
	}
	if len(weights) == 0 {
		return nil, nil, ErrClarificationRequired
	}
	parts, err := total.Allocate(weights, allocationScale(total, weights))
	if err != nil {
		return nil, nil, ErrInvalidRefund
	}
	members := []MemberAmount{}
	unknown := []money.Money{}
	for _, part := range parts {
		if part.Money.Sign() == 0 {
			continue
		}
		if strings.HasPrefix(part.ID, "member:") {
			members = append(members, MemberAmount{MemberID: household.MembershipID(strings.TrimPrefix(part.ID, "member:")), Amount: part.Money})
		} else {
			unknown = append(unknown, part.Money)
		}
	}
	return members, unknown, nil
}

func AllocateValuations(basis ValuationBasis, purchase money.Money, refunds map[string]money.Money) (map[string]ValuationShare, error) {
	if basis.Purchase.Validate() != nil || basis.Value.Validate() != nil || basis.Purchase.Asset() != purchase.Asset() || strings.TrimSpace(basis.Ref) == "" {
		return nil, ErrInvalidRefund
	}
	if compared, _ := basis.Purchase.Compare(purchase); compared != 0 {
		return nil, ErrInvalidRefund
	}
	total, _ := money.NewMoney("0", purchase.Asset())
	weights := make([]money.Weight, 0, len(refunds))
	for id, amount := range refunds {
		if id == "" || amount.Validate() != nil || amount.Asset() != purchase.Asset() || amount.Sign() <= 0 {
			return nil, ErrInvalidRefund
		}
		var err error
		total, err = total.Add(amount)
		if err != nil {
			return nil, ErrInvalidRefund
		}
		weights = append(weights, money.Weight{ID: id, Value: amount.Amount()})
	}
	remaining, err := purchase.Subtract(total)
	if err != nil || remaining.Sign() < 0 {
		return nil, ErrRefundExceedsPurchase
	}
	if len(refunds) == 0 {
		return map[string]ValuationShare{}, nil
	}
	refundedValue := basis.Value
	refundScale := allocationScale(refundedValue, weights)
	if remaining.Sign() > 0 {
		principalWeights := []money.Weight{{ID: "refunded", Value: total.Amount()}, {ID: "remaining", Value: remaining.Amount()}}
		principalParts, allocateErr := basis.Value.Allocate(principalWeights, allocationScale(basis.Value, principalWeights))
		if allocateErr != nil {
			return nil, ErrInvalidRefund
		}
		refundedValue = principalParts[0].Money
		refundScale = decimalScale(refundedValue.Amount())
	}
	parts, err := refundedValue.Allocate(weights, refundScale)
	if err != nil {
		return nil, ErrInvalidRefund
	}
	result := make(map[string]ValuationShare, len(refunds))
	for _, part := range parts {
		if _, ok := refunds[part.ID]; ok {
			copy := basis
			result[part.ID] = ValuationShare{Basis: copy, Value: part.Money}
		}
	}
	return result, nil
}

func valueRefund(share ValuationShare, result Refund) (Valuation, error) {
	if share.Value.Validate() != nil || share.Value.Sign() < 0 || strings.TrimSpace(share.Basis.Ref) == "" {
		return Valuation{}, ErrInvalidRefund
	}
	basis := share.Basis
	v := Valuation{Value: share.Value, Ref: basis.Ref, Basis: &basis}
	var err error
	v.Members, v.Unallocated, err = scaleMembers(share.Value, result.Members, result.Unallocated)
	if err != nil {
		return Valuation{}, err
	}
	v.Categories, err = scaleCategories(share.Value, result.Categories)
	return v, err
}

func scaleMembers(total money.Money, members []MemberAmount, unallocated []money.Money) ([]MemberAmount, []money.Money, error) {
	weights := []money.Weight{}
	for _, value := range members {
		weights = append(weights, money.Weight{ID: "member:" + string(value.MemberID), Value: value.Amount.Amount()})
	}
	for i, value := range unallocated {
		weights = append(weights, money.Weight{ID: "unallocated:" + string(rune('a'+i)), Value: value.Amount()})
	}
	parts, err := total.Allocate(weights, allocationScale(total, weights))
	if err != nil {
		return nil, nil, ErrInvalidRefund
	}
	out, unknown := []MemberAmount{}, []money.Money{}
	for _, part := range parts {
		if part.Money.Sign() == 0 {
			continue
		}
		if strings.HasPrefix(part.ID, "member:") {
			out = append(out, MemberAmount{MemberID: household.MembershipID(strings.TrimPrefix(part.ID, "member:")), Amount: part.Money})
		} else {
			unknown = append(unknown, part.Money)
		}
	}
	return out, unknown, nil
}

func scaleCategories(total money.Money, categories []CategoryAmount) ([]CategoryAmount, error) {
	weights := make([]money.Weight, 0, len(categories))
	for i, value := range categories {
		weights = append(weights, money.Weight{ID: "category:" + value.CategoryID + ":" + string(rune('a'+i)), Value: value.Amount.Amount()})
	}
	parts, err := total.Allocate(weights, allocationScale(total, weights))
	if err != nil {
		return nil, ErrInvalidRefund
	}
	out := make([]CategoryAmount, 0, len(parts))
	for i, part := range parts {
		if part.Money.Sign() > 0 {
			out = append(out, CategoryAmount{CategoryID: categories[i].CategoryID, Amount: part.Money})
		}
	}
	return out, nil
}

func validateEffects(total money.Money, members []MemberAmount, unallocated []money.Money) error {
	zero, _ := money.NewMoney("0", total.Asset())
	sum := zero
	seen := map[household.MembershipID]bool{}
	for _, v := range members {
		if v.MemberID == "" || seen[v.MemberID] || v.Amount.Validate() != nil || v.Amount.Sign() <= 0 || v.Amount.Asset() != total.Asset() {
			return ErrInvalidRefund
		}
		seen[v.MemberID] = true
		sum, _ = sum.Add(v.Amount)
	}
	for _, v := range unallocated {
		if v.Validate() != nil || v.Sign() <= 0 || v.Asset() != total.Asset() {
			return ErrInvalidRefund
		}
		sum, _ = sum.Add(v)
	}
	if compared, _ := sum.Compare(total); compared != 0 {
		return ErrInvalidRefund
	}
	return nil
}

func validateCategories(total money.Money, values []CategoryAmount) error {
	zero, _ := money.NewMoney("0", total.Asset())
	sum := zero
	seen := map[string]bool{}
	for _, v := range values {
		if seen[v.CategoryID] || v.Amount.Validate() != nil || v.Amount.Sign() <= 0 || v.Amount.Asset() != total.Asset() {
			return ErrInvalidRefund
		}
		seen[v.CategoryID] = true
		sum, _ = sum.Add(v.Amount)
	}
	if compared, _ := sum.Compare(total); compared != 0 {
		return ErrInvalidRefund
	}
	return nil
}

func add(current, value money.Money) (money.Money, error) {
	if current.Validate() != nil {
		zero, _ := money.NewMoney("0", value.Asset())
		current = zero
	}
	return current.Add(value)
}
func decimalScale(value string) int32 {
	if i := strings.IndexByte(value, '.'); i >= 0 {
		return int32(len(value) - i - 1)
	}
	return 0
}

func allocationScale(total money.Money, weights []money.Weight) int32 {
	scale := decimalScale(total.Amount())
	for _, weight := range weights {
		if candidate := decimalScale(weight.Value); candidate > scale {
			scale = candidate
		}
	}
	if scale <= money.MaxDecimalLength-8 {
		scale += 6
	}
	return scale
}

func sameMoney(left, right money.Money) bool {
	if left.Validate() != nil || right.Validate() != nil || left.Asset() != right.Asset() {
		return false
	}
	compared, err := left.Compare(right)
	return err == nil && compared == 0
}

func sameItems(left, right []ItemPortion) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].ItemID != right[index].ItemID || !sameMoney(left[index].Amount, right[index].Amount) {
			return false
		}
	}
	return true
}

func sameMembers(left, right []MemberAmount) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].MemberID != right[index].MemberID || !sameMoney(left[index].Amount, right[index].Amount) {
			return false
		}
	}
	return true
}

func sameCategories(left, right []CategoryAmount) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if left[index].CategoryID != right[index].CategoryID || !sameMoney(left[index].Amount, right[index].Amount) {
			return false
		}
	}
	return true
}

func sameMoneySlice(left, right []money.Money) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		if !sameMoney(left[index], right[index]) {
			return false
		}
	}
	return true
}
func cloneItems(items []ItemPortion) []ItemPortion { return append([]ItemPortion(nil), items...) }
func sortEffects(r *Refund) {
	sort.Slice(r.Members, func(i, j int) bool { return r.Members[i].MemberID < r.Members[j].MemberID })
	sort.Slice(r.Categories, func(i, j int) bool { return r.Categories[i].CategoryID < r.Categories[j].CategoryID })
}
