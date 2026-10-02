package domain

import (
	"math/big"
	"regexp"
	"slices"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type AllocationState string
type AllocationPurpose string
type AllocationMode string
type AllocationOrigin string

const (
	AllocationResolved      AllocationState = "resolved"
	AllocationPartial       AllocationState = "partial"
	AllocationUnresolved    AllocationState = "unresolved"
	AllocationNotApplicable AllocationState = "not_applicable"

	AllocationPersonal AllocationPurpose = "personal"
	AllocationShared   AllocationPurpose = "shared"

	AllocationByAmounts AllocationMode = "amounts"
	AllocationByShares  AllocationMode = "shares"
	AllocationEqual     AllocationMode = "equal"
	AllocationUnknown   AllocationMode = "unresolved"
	AllocationComposite AllocationMode = "composite"

	AllocationExplicitPurchase AllocationOrigin = "explicit_purchase"
	AllocationExplicitItem     AllocationOrigin = "explicit_item"
	AllocationRule             AllocationOrigin = "rule"
	AllocationEqualDefault     AllocationOrigin = "equal_default"
	AllocationUnknownOrigin    AllocationOrigin = "unresolved"
	AllocationMixed            AllocationOrigin = "mixed"
	AllocationNone             AllocationOrigin = "not_applicable"
)

var allocationDecimalSyntax = regexp.MustCompile(`^(0|[1-9][0-9]*)(\.[0-9]+)?$`)

type AllocationMemberInput struct {
	MemberID household.MembershipID
	Amount   *money.Money
	Share    string
}

type AllocationInput struct {
	Mode     AllocationMode
	Purpose  AllocationPurpose
	Members  []AllocationMemberInput
	Reason   string
	Origin   AllocationOrigin
	RuleRefs []AllocationRuleRef
}

type AllocationRuleRef struct {
	ID       string
	Revision uint64
}

type MemberAmount struct {
	MemberID household.MembershipID
	Money    money.Money
}

type AllocationSnapshot struct {
	State       AllocationState
	Purpose     AllocationPurpose
	Mode        AllocationMode
	Origin      AllocationOrigin
	Reason      string
	Basis       *AllocationInput
	Inputs      []AllocationMemberInput
	Members     []MemberAmount
	Unallocated []money.Money
	RuleRefs    []AllocationRuleRef
}

type ItemAllocationInput struct {
	ItemID     string
	Allocation AllocationInput
}

func NotApplicableAllocation() AllocationSnapshot {
	return AllocationSnapshot{State: AllocationNotApplicable, Origin: AllocationNone}
}

func (a AllocationSnapshot) Clone() AllocationSnapshot {
	if a.Basis != nil {
		basis := cloneAllocationInput(*a.Basis)
		a.Basis = &basis
	}
	a.Inputs = cloneAllocationInputs(a.Inputs)
	a.Members = slices.Clone(a.Members)
	a.Unallocated = slices.Clone(a.Unallocated)
	a.RuleRefs = slices.Clone(a.RuleRefs)
	sort.Slice(a.Members, func(i, j int) bool {
		if a.Members[i].MemberID == a.Members[j].MemberID {
			return a.Members[i].Money.Asset() < a.Members[j].Money.Asset()
		}
		return a.Members[i].MemberID < a.Members[j].MemberID
	})
	sort.Slice(a.Unallocated, func(i, j int) bool { return a.Unallocated[i].Asset() < a.Unallocated[j].Asset() })
	sort.Slice(a.RuleRefs, func(i, j int) bool { return a.RuleRefs[i].ID < a.RuleRefs[j].ID })
	return a
}

func (a AllocationSnapshot) Validate() error {
	if !slices.Contains([]AllocationState{AllocationResolved, AllocationPartial, AllocationUnresolved, AllocationNotApplicable}, a.State) || utf8.RuneCountInString(a.Reason) > 2000 {
		return ErrInvalidAllocation
	}
	if a.State == AllocationNotApplicable {
		if a.Purpose != "" || a.Mode != "" || a.Origin != AllocationNone || a.Reason != "" || len(a.Inputs)+len(a.Members)+len(a.Unallocated)+len(a.RuleRefs) != 0 {
			return ErrInvalidAllocation
		}
		if a.Basis != nil && validateAllocationInput(*a.Basis) != nil {
			return ErrInvalidAllocation
		}
		return nil
	}
	if !slices.Contains([]AllocationMode{AllocationByAmounts, AllocationByShares, AllocationEqual, AllocationUnknown, AllocationComposite}, a.Mode) || !slices.Contains([]AllocationOrigin{AllocationExplicitPurchase, AllocationExplicitItem, AllocationRule, AllocationEqualDefault, AllocationUnknownOrigin, AllocationMixed}, a.Origin) {
		return ErrInvalidAllocation
	}
	if a.Mode == AllocationUnknown {
		if a.Purpose != "" || strings.TrimSpace(a.Reason) == "" || len(a.Inputs) != 0 || len(a.Members) != 0 || len(a.Unallocated) == 0 || a.State != AllocationUnresolved {
			return ErrInvalidAllocation
		}
	} else if !slices.Contains([]AllocationPurpose{AllocationPersonal, AllocationShared}, a.Purpose) || a.Mode != AllocationComposite && len(a.Inputs) == 0 {
		return ErrInvalidAllocation
	}
	if a.Basis != nil && validateAllocationInput(*a.Basis) != nil {
		return ErrInvalidAllocation
	}
	if a.Mode == AllocationComposite {
		if a.Basis == nil || len(a.Inputs) != 0 {
			return ErrInvalidAllocation
		}
	}
	seenInputs := map[string]bool{}
	inputMembers := map[household.MembershipID]bool{}
	for _, input := range a.Inputs {
		key := string(input.MemberID)
		if a.Mode == AllocationByAmounts && input.Amount != nil {
			key += "\x00" + string(input.Amount.Asset())
		}
		if input.MemberID == "" || seenInputs[key] {
			return ErrInvalidAllocation
		}
		seenInputs[key] = true
		inputMembers[input.MemberID] = true
		if a.Mode == AllocationEqual {
			if input.Amount != nil || input.Share != "" {
				return ErrInvalidAllocation
			}
			continue
		}
		if (input.Amount == nil) == (input.Share == "") || input.Amount != nil && (input.Amount.Validate() != nil || input.Amount.Sign() < 0) {
			return ErrInvalidAllocation
		}
		if input.Share != "" && !validPositiveDecimal(input.Share) {
			return ErrInvalidAllocation
		}
	}
	if a.Purpose == AllocationPersonal && a.Mode != AllocationComposite && len(inputMembers) != 1 {
		return ErrInvalidAllocation
	}
	seenAmounts := map[string]bool{}
	allocatedMembers := map[household.MembershipID]bool{}
	for _, member := range a.Members {
		key := string(member.MemberID) + "\x00" + string(member.Money.Asset())
		if member.MemberID == "" || seenAmounts[key] || member.Money.Validate() != nil || member.Money.Sign() <= 0 || a.Mode != AllocationComposite && !inputMembers[member.MemberID] {
			return ErrInvalidAllocation
		}
		seenAmounts[key] = true
		allocatedMembers[member.MemberID] = true
	}
	if a.Purpose == AllocationPersonal && a.Mode == AllocationComposite && len(allocatedMembers) != 1 {
		return ErrInvalidAllocation
	}
	seenAssets := map[money.Asset]bool{}
	for _, amount := range a.Unallocated {
		if amount.Validate() != nil || amount.Sign() <= 0 || seenAssets[amount.Asset()] {
			return ErrInvalidAllocation
		}
		seenAssets[amount.Asset()] = true
	}
	if a.State == AllocationResolved && len(a.Unallocated) != 0 || a.State == AllocationPartial && (len(a.Members) == 0 || len(a.Unallocated) == 0) || a.State == AllocationUnresolved && len(a.Members) != 0 {
		return ErrInvalidAllocation
	}
	seenRules := map[string]bool{}
	for _, ref := range a.RuleRefs {
		if ref.ID == "" || ref.Revision < 1 || ref.Revision > 9007199254740991 || seenRules[ref.ID] {
			return ErrInvalidAllocation
		}
		seenRules[ref.ID] = true
	}
	return nil
}

func validateAllocationInput(input AllocationInput) error {
	if !slices.Contains([]AllocationMode{AllocationByAmounts, AllocationByShares, AllocationEqual, AllocationUnknown}, input.Mode) || utf8.RuneCountInString(input.Reason) > 2000 {
		return ErrInvalidAllocation
	}
	if input.Mode == AllocationUnknown {
		if input.Purpose != "" || len(input.Members) != 0 || strings.TrimSpace(input.Reason) == "" {
			return ErrInvalidAllocation
		}
	} else if !slices.Contains([]AllocationPurpose{AllocationPersonal, AllocationShared}, input.Purpose) || len(input.Members) == 0 {
		return ErrInvalidAllocation
	}
	if !slices.Contains([]AllocationOrigin{AllocationExplicitPurchase, AllocationExplicitItem, AllocationRule, AllocationEqualDefault, AllocationUnknownOrigin}, input.Origin) {
		return ErrInvalidAllocation
	}
	seen := map[string]bool{}
	weights := make([]money.Weight, 0, len(input.Members))
	members := map[household.MembershipID]bool{}
	for _, member := range input.Members {
		key := string(member.MemberID)
		if input.Mode == AllocationByAmounts && member.Amount != nil {
			key += "\x00" + string(member.Amount.Asset())
		}
		if member.MemberID == "" || seen[key] {
			return ErrInvalidAllocation
		}
		seen[key] = true
		members[member.MemberID] = true
		switch input.Mode {
		case AllocationByAmounts:
			if member.Amount == nil || member.Share != "" || member.Amount.Validate() != nil || member.Amount.Sign() < 0 {
				return ErrInvalidAllocation
			}
		case AllocationByShares:
			if member.Amount != nil || !validPositiveDecimal(member.Share) {
				return ErrInvalidAllocation
			}
			weights = append(weights, money.Weight{ID: string(member.MemberID), Value: member.Share})
		case AllocationEqual:
			if member.Amount != nil || member.Share != "" {
				return ErrInvalidAllocation
			}
		case AllocationUnknown:
			return ErrInvalidAllocation
		}
	}
	if input.Purpose == AllocationPersonal && len(members) != 1 || input.Mode == AllocationByShares && !sumIs(weights, "100") {
		return ErrInvalidAllocation
	}
	seenRules := map[string]bool{}
	for _, ref := range input.RuleRefs {
		if ref.ID == "" || ref.Revision < 1 || ref.Revision > 9007199254740991 || seenRules[ref.ID] {
			return ErrInvalidAllocation
		}
		seenRules[ref.ID] = true
	}
	return nil
}

func (r Revision) validateAllocation() error {
	if r.Allocation.State == "" {
		return nil
	}
	if err := r.Allocation.Validate(); err != nil {
		return err
	}
	components, itemAmounts, err := r.allocationComponents()
	if err != nil {
		return err
	}
	if len(components) == 0 {
		if r.Allocation.State != AllocationNotApplicable {
			return ErrInvalidAllocation
		}
	} else {
		if r.Allocation.State == AllocationNotApplicable || !sameAllocationTotals(componentTotals(components), snapshotTotals(r.Allocation)) {
			return ErrInvalidAllocation
		}
	}
	for _, item := range r.ReceiptItems {
		net, err := item.Net()
		if err != nil {
			return err
		}
		_, allocatable := itemAmounts[item.ID]
		if net.Sign() == 0 || !allocatable {
			if item.Allocation.State != AllocationNotApplicable {
				return ErrInvalidAllocation
			}
			continue
		}
		if item.Allocation.State == "" || item.Allocation.State == AllocationNotApplicable || !sameAllocationTotals(map[money.Asset]money.Money{net.Asset(): net}, snapshotTotals(item.Allocation)) {
			return ErrInvalidAllocation
		}
	}
	return nil
}

func componentTotals(components []allocationComponent) map[money.Asset]money.Money {
	result := map[money.Asset]money.Money{}
	for _, component := range components {
		addAllocationTotal(result, component.Money)
	}
	return result
}

func snapshotTotals(snapshot AllocationSnapshot) map[money.Asset]money.Money {
	result := map[money.Asset]money.Money{}
	for _, member := range snapshot.Members {
		addAllocationTotal(result, member.Money)
	}
	for _, amount := range snapshot.Unallocated {
		addAllocationTotal(result, amount)
	}
	return result
}

func addAllocationTotal(target map[money.Asset]money.Money, amount money.Money) {
	current, exists := target[amount.Asset()]
	if !exists {
		target[amount.Asset()] = amount
		return
	}
	current, _ = current.Add(amount)
	target[amount.Asset()] = current
}

func sameAllocationTotals(left, right map[money.Asset]money.Money) bool {
	if len(left) != len(right) {
		return false
	}
	for asset, amount := range left {
		other, exists := right[asset]
		if !exists {
			return false
		}
		compared, err := amount.Compare(other)
		if err != nil || compared != 0 {
			return false
		}
	}
	return true
}

func (r Revision) WithAllocation(input AllocationInput, items []ItemAllocationInput, active []household.MembershipID) (Revision, error) {
	next := r.Clone()
	for index, item := range next.ReceiptItems {
		net, netErr := item.Net()
		if netErr != nil {
			return r, netErr
		}
		if net.Sign() == 0 {
			allocation := NotApplicableAllocation()
			switch {
			case item.Allocation.Basis != nil:
				basis := cloneAllocationInput(*item.Allocation.Basis)
				allocation.Basis = &basis
			case item.Allocation.State != "" && item.Allocation.State != AllocationNotApplicable:
				basis := allocationInput(item.Allocation)
				allocation.Basis = &basis
			}
			next.ReceiptItems[index].Allocation = allocation
		}
	}
	components, itemAmounts, err := next.allocationComponents()
	if err != nil {
		return r, err
	}
	if len(components) == 0 {
		if r.Allocation.Basis != nil && r.Participation.GroupID != "" && input.Mode != "" {
			participation := next.Participation
			next.Participation = Participation{}
			allocated, allocateErr := next.WithAllocation(input, items, active)
			if allocateErr != nil {
				return r, allocateErr
			}
			allocated.Participation = participation
			return allocated.RefreshAllocation()
		}
		if input.Mode != "" || len(items) != 0 {
			return r, ErrInvalidAllocation
		}
		next.Allocation = NotApplicableAllocation()
		for index := range next.ReceiptItems {
			next.ReceiptItems[index].Allocation = NotApplicableAllocation()
		}
		if err := next.validateAllocation(); err != nil {
			return r, err
		}
		return next, nil
	}
	activeSet := map[household.MembershipID]bool{}
	requiresActiveMembers := input.requiresActiveMembers()
	for _, item := range items {
		requiresActiveMembers = requiresActiveMembers || item.Allocation.requiresActiveMembers()
	}
	if requiresActiveMembers {
		activeSet, err = activeMembers(active)
		if err != nil {
			return r, err
		}
	}
	itemInputs := make(map[string]AllocationInput, len(items))
	for _, item := range items {
		if _, exists := itemInputs[item.ItemID]; exists || itemAmounts[item.ItemID].Validate() != nil {
			return r, ErrInvalidAllocation
		}
		if item.Allocation.Origin != AllocationRule {
			item.Allocation.Origin = AllocationExplicitItem
		}
		itemInputs[item.ItemID] = item.Allocation
	}
	if input.Origin == "" {
		input.Origin = AllocationExplicitPurchase
		if input.Mode == AllocationUnknown {
			input.Origin = AllocationUnknownOrigin
		}
	}
	memberTotals := map[string]MemberAmount{}
	unallocated := map[money.Asset]money.Money{}
	origins := map[AllocationOrigin]bool{}
	modes := map[AllocationMode]bool{}
	purposes := map[AllocationPurpose]bool{}
	intendedMembers := map[household.MembershipID]bool{}
	rules := map[string]AllocationRuleRef{}
	parts, err := allocateParts(components, input, itemInputs, activeSet)
	if err != nil {
		return r, err
	}
	for index, component := range components {
		part := parts[index]
		origins[part.Origin] = true
		modes[part.Mode] = true
		if part.Purpose != "" {
			purposes[part.Purpose] = true
		}
		for _, member := range part.Inputs {
			intendedMembers[member.MemberID] = true
		}
		for _, ref := range part.RuleRefs {
			rules[ref.ID] = ref
		}
		for _, amount := range part.Members {
			key := string(amount.MemberID) + "\x00" + string(amount.Money.Asset())
			current, ok := memberTotals[key]
			if !ok {
				memberTotals[key] = amount
				continue
			}
			current.Money, err = current.Money.Add(amount.Money)
			if err != nil {
				return r, err
			}
			memberTotals[key] = current
		}
		for _, amount := range part.Unallocated {
			current, ok := unallocated[amount.Asset()]
			if !ok {
				unallocated[amount.Asset()] = amount
				continue
			}
			current, err = current.Add(amount)
			if err != nil {
				return r, err
			}
			unallocated[amount.Asset()] = current
		}
		if component.ItemIndex >= 0 {
			if basis, ok := itemInputs[component.ID]; ok {
				basis = cloneAllocationInput(basis)
				part.Basis = &basis
			}
			next.ReceiptItems[component.ItemIndex].Allocation = part
		}
	}
	result := AllocationSnapshot{Purpose: input.Purpose, Mode: input.Mode, Origin: input.Origin, Reason: input.Reason, Inputs: cloneAllocationInputs(input.Members)}
	for _, value := range memberTotals {
		result.Members = append(result.Members, value)
	}
	for _, value := range unallocated {
		result.Unallocated = append(result.Unallocated, value)
	}
	for _, value := range rules {
		result.RuleRefs = append(result.RuleRefs, value)
	}
	sort.Slice(result.Members, func(i, j int) bool {
		if result.Members[i].MemberID == result.Members[j].MemberID {
			return result.Members[i].Money.Asset() < result.Members[j].Money.Asset()
		}
		return result.Members[i].MemberID < result.Members[j].MemberID
	})
	sort.Slice(result.Unallocated, func(i, j int) bool { return result.Unallocated[i].Asset() < result.Unallocated[j].Asset() })
	sort.Slice(result.RuleRefs, func(i, j int) bool { return result.RuleRefs[i].ID < result.RuleRefs[j].ID })
	if len(origins) == 1 {
		for origin := range origins {
			result.Origin = origin
		}
	} else if len(origins) > 1 {
		result.Origin = AllocationMixed
	}
	if len(modes) > 1 || len(itemInputs) > 0 {
		result.Mode = AllocationComposite
		result.Inputs = nil
		basis := cloneAllocationInput(input)
		result.Basis = &basis
	}
	if len(result.Unallocated) == 0 {
		result.State = AllocationResolved
	} else if len(result.Members) == 0 {
		result.State = AllocationUnresolved
	} else {
		result.State = AllocationPartial
	}
	if result.Mode == AllocationComposite {
		if purposes[AllocationShared] || len(intendedMembers) > 1 {
			result.Purpose = AllocationShared
		} else if purposes[AllocationPersonal] || len(intendedMembers) == 1 {
			result.Purpose = AllocationPersonal
		}
	}
	if result.State == AllocationUnresolved {
		result.Mode, result.Purpose, result.Origin = AllocationUnknown, "", AllocationUnknownOrigin
		result.Inputs = nil
		if strings.TrimSpace(result.Reason) == "" {
			result.Reason = "allocation_unresolved"
		}
	}
	next.Allocation = result
	if err := next.validateAllocation(); err != nil {
		return r, err
	}
	return next, nil
}

// RefreshAllocation rebuilds a snapshot from its persisted purchase and item
// bases after its financial contribution or exact amounts changed.
func (r Revision) RefreshAllocation() (Revision, error) {
	components, itemAmounts, err := r.allocationComponents()
	if err != nil {
		return r, err
	}
	fallback, items, active, err := r.allocationBases()
	if err != nil {
		return r, err
	}
	if len(components) == 0 {
		return r.suspendAllocation(fallback, items)
	}
	if fallback.Mode == "" {
		fallback = AllocationInput{Mode: AllocationUnknown, Reason: "allocation_unresolved"}
	}
	allocatableItems := items[:0]
	for _, item := range items {
		if _, ok := itemAmounts[item.ItemID]; ok {
			allocatableItems = append(allocatableItems, item)
		}
	}
	return r.WithAllocation(fallback, allocatableItems, active)
}

func (r Revision) allocationBases() (AllocationInput, []ItemAllocationInput, []household.MembershipID, error) {
	fallback := AllocationInput{}
	if r.Allocation.Basis != nil {
		fallback = cloneAllocationInput(*r.Allocation.Basis)
	} else if r.Allocation.State != "" && r.Allocation.State != AllocationNotApplicable {
		fallback = allocationInput(r.Allocation)
	}
	items := make([]ItemAllocationInput, 0, len(r.ReceiptItems))
	members := map[household.MembershipID]bool{}
	for _, member := range fallback.Members {
		members[member.MemberID] = true
	}
	for _, item := range r.ReceiptItems {
		var basis AllocationInput
		switch {
		case item.Allocation.Basis != nil:
			basis = cloneAllocationInput(*item.Allocation.Basis)
		case item.Allocation.State != "" && item.Allocation.State != AllocationNotApplicable && item.Allocation.Origin == AllocationExplicitItem:
			basis = allocationInput(item.Allocation)
		default:
			continue
		}
		items = append(items, ItemAllocationInput{ItemID: item.ID, Allocation: basis})
		for _, member := range basis.Members {
			members[member.MemberID] = true
		}
	}
	active := make([]household.MembershipID, 0, len(members))
	for memberID := range members {
		active = append(active, memberID)
	}
	slices.Sort(active)
	return fallback, items, active, nil
}

// AllocationBases returns the purchase basis and inputs for contributing items.
// Dormant zero-net bases stay on the revision and are retained by WithAllocation.
func (r Revision) AllocationBases() (AllocationInput, []ItemAllocationInput, error) {
	fallback, items, _, err := r.allocationBases()
	if err != nil {
		return fallback, nil, err
	}
	_, amounts, err := r.allocationComponents()
	if err != nil {
		return fallback, nil, err
	}
	active := items[:0]
	for _, item := range items {
		if _, ok := amounts[item.ItemID]; ok {
			active = append(active, item)
		}
	}
	return fallback, active, nil
}

func (r Revision) suspendAllocation(fallback AllocationInput, items []ItemAllocationInput) (Revision, error) {
	next := r.Clone()
	next.Allocation = NotApplicableAllocation()
	if fallback.Mode != "" {
		basis := cloneAllocationInput(fallback)
		next.Allocation.Basis = &basis
	}
	itemBases := make(map[string]AllocationInput, len(items))
	for _, item := range items {
		itemBases[item.ItemID] = cloneAllocationInput(item.Allocation)
	}
	for index := range next.ReceiptItems {
		next.ReceiptItems[index].Allocation = NotApplicableAllocation()
		if basis, ok := itemBases[next.ReceiptItems[index].ID]; ok {
			next.ReceiptItems[index].Allocation.Basis = &basis
		}
	}
	if err := next.validateAllocation(); err != nil {
		return r, err
	}
	return next, nil
}

func allocateParts(components []allocationComponent, input AllocationInput, itemInputs map[string]AllocationInput, active map[household.MembershipID]bool) ([]AllocationSnapshot, error) {
	parts := make([]AllocationSnapshot, len(components))
	grouped := map[money.Asset][]int{}
	for index, component := range components {
		if override, ok := itemInputs[component.ID]; ok {
			part, err := allocateComponent(component.Money, override, active)
			if err != nil {
				return nil, err
			}
			parts[index] = part
			continue
		}
		if input.Mode != AllocationByAmounts {
			part, err := allocateComponent(component.Money, input, active)
			if err != nil {
				return nil, err
			}
			parts[index] = part
			continue
		}
		grouped[component.Money.Asset()] = append(grouped[component.Money.Asset()], index)
	}
	if input.Mode != AllocationByAmounts {
		return parts, nil
	}
	for _, member := range input.Members {
		if member.Amount == nil || len(grouped[member.Amount.Asset()]) == 0 {
			return nil, ErrInvalidAllocation
		}
	}
	assets := make([]money.Asset, 0, len(grouped))
	for asset := range grouped {
		assets = append(assets, asset)
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i] < assets[j] })
	for _, asset := range assets {
		indexes := grouped[asset]
		total := components[indexes[0]].Money
		for _, index := range indexes[1:] {
			var err error
			total, err = total.Add(components[index].Money)
			if err != nil {
				return nil, err
			}
		}
		assetInput := input
		assetInput.Members = nil
		for _, member := range input.Members {
			if member.Amount.Asset() == asset {
				assetInput.Members = append(assetInput.Members, member)
			}
		}
		group, err := allocateComponent(total, assetInput, active)
		if err != nil {
			return nil, err
		}
		distributed, err := distributeAmountGroup(group, components, indexes)
		if err != nil {
			return nil, err
		}
		for position, index := range indexes {
			parts[index] = distributed[position]
		}
	}
	return parts, nil
}

func distributeAmountGroup(group AllocationSnapshot, components []allocationComponent, indexes []int) ([]AllocationSnapshot, error) {
	if len(indexes) == 1 {
		return []AllocationSnapshot{group}, nil
	}
	remaining := map[household.MembershipID]money.Money{}
	memberOrder := make([]household.MembershipID, 0, len(group.Inputs))
	for _, input := range group.Inputs {
		remaining[input.MemberID] = *input.Amount
		memberOrder = append(memberOrder, input.MemberID)
	}
	slices.Sort(memberOrder)
	result := make([]AllocationSnapshot, len(indexes))
	for position, index := range indexes {
		component := components[index]
		allocated := map[household.MembershipID]money.Money{}
		if position == len(indexes)-1 {
			for _, memberID := range memberOrder {
				allocated[memberID] = remaining[memberID]
			}
		} else {
			weights := make([]money.Weight, 0, len(memberOrder))
			for _, memberID := range memberOrder {
				if remaining[memberID].Sign() > 0 {
					weights = append(weights, money.Weight{ID: string(memberID), Value: remaining[memberID].Amount()})
				}
			}
			values, err := component.Money.Allocate(weights, decimalScale(component.Money.Amount()))
			if err != nil {
				return nil, ErrInvalidAllocation
			}
			for _, value := range values {
				allocated[household.MembershipID(value.ID)] = value.Money
			}
		}
		part := AllocationSnapshot{State: AllocationResolved, Purpose: group.Purpose, Mode: AllocationByAmounts, Origin: group.Origin, Reason: group.Reason, RuleRefs: slices.Clone(group.RuleRefs)}
		zero, _ := money.NewMoney("0", component.Money.Asset())
		partTotal := zero
		for _, memberID := range memberOrder {
			amount, ok := allocated[memberID]
			if !ok {
				amount = zero
			}
			value := amount
			part.Inputs = append(part.Inputs, AllocationMemberInput{MemberID: memberID, Amount: &value})
			if amount.Sign() > 0 {
				part.Members = append(part.Members, MemberAmount{MemberID: memberID, Money: amount})
			}
			var err error
			partTotal, err = partTotal.Add(amount)
			if err != nil {
				return nil, err
			}
			remaining[memberID], err = remaining[memberID].Subtract(amount)
			if err != nil || remaining[memberID].Sign() < 0 {
				return nil, ErrInvalidAllocation
			}
		}
		if compared, err := partTotal.Compare(component.Money); err != nil || compared != 0 || part.Validate() != nil {
			return nil, ErrInvalidAllocation
		}
		result[position] = part
	}
	return result, nil
}

type allocationComponent struct {
	ID        string
	Money     money.Money
	ItemIndex int
}

func (r Revision) allocationComponents() ([]allocationComponent, map[string]money.Money, error) {
	items := map[string]money.Money{}
	components := []allocationComponent{}
	if r.Type == Income {
		return components, items, nil
	}
	if r.Type == Expense && len(r.ReceiptItems) > 0 {
		principalContributes := false
		for index, posting := range r.Postings {
			if posting.Role == Principal && r.Contributes(index) && !r.InternalPrincipal(index) {
				principalContributes = true
				break
			}
		}
		if principalContributes {
			for index, item := range r.ReceiptItems {
				amount, err := item.Net()
				if err != nil {
					return nil, nil, err
				}
				if amount.Sign() == 0 {
					continue
				}
				items[item.ID] = amount
				components = append(components, allocationComponent{ID: item.ID, Money: amount, ItemIndex: index})
			}
		}
	}
	for index, posting := range r.Postings {
		if posting.Money.Sign() >= 0 || !posting.MovesMoney() || !r.Contributes(index) || r.InternalPrincipal(index) {
			continue
		}
		if posting.Role == Principal && (r.Type != Expense || len(r.ReceiptItems) > 0) || posting.Role != Principal && posting.Role != Fee && posting.Role != Interest {
			continue
		}
		zero, _ := money.NewMoney("0", posting.Money.Asset())
		amount, err := zero.Subtract(posting.Money)
		if err != nil {
			return nil, nil, err
		}
		components = append(components, allocationComponent{ID: "posting:" + string(posting.Role) + ":" + strconv.Itoa(index), Money: amount, ItemIndex: -1})
	}
	return components, items, nil
}

func allocateComponent(total money.Money, input AllocationInput, active map[household.MembershipID]bool) (AllocationSnapshot, error) {
	if input.Mode == "" || input.Mode == AllocationUnknown {
		if strings.TrimSpace(input.Reason) == "" {
			input.Reason = "allocation_unresolved"
		}
		origin := input.Origin
		if origin == "" {
			origin = AllocationUnknownOrigin
		}
		return AllocationSnapshot{State: AllocationUnresolved, Mode: AllocationUnknown, Origin: origin, Reason: input.Reason, Unallocated: []money.Money{total}, RuleRefs: slices.Clone(input.RuleRefs)}, nil
	}
	if input.Origin == "" {
		input.Origin = AllocationExplicitPurchase
	}
	if !slices.Contains([]AllocationPurpose{AllocationPersonal, AllocationShared}, input.Purpose) || len(input.Members) == 0 {
		return AllocationSnapshot{}, ErrInvalidAllocation
	}
	seen := map[household.MembershipID]bool{}
	weights := make([]money.Weight, 0, len(input.Members))
	for _, member := range input.Members {
		if member.MemberID == "" || !active[member.MemberID] || seen[member.MemberID] {
			return AllocationSnapshot{}, ErrInvalidAllocation
		}
		seen[member.MemberID] = true
		value := member.Share
		if input.Mode == AllocationByAmounts {
			if member.Amount == nil || member.Amount.Validate() != nil || member.Amount.Asset() != total.Asset() || member.Amount.Sign() < 0 || member.Share != "" {
				return AllocationSnapshot{}, ErrInvalidAllocation
			}
			value = member.Amount.Amount()
		} else if input.Mode == AllocationEqual {
			if member.Amount != nil || member.Share != "" {
				return AllocationSnapshot{}, ErrInvalidAllocation
			}
			value = "1"
		} else if member.Amount != nil || !validPositiveDecimal(value) {
			return AllocationSnapshot{}, ErrInvalidAllocation
		}
		weights = append(weights, money.Weight{ID: string(member.MemberID), Value: value})
	}
	if input.Purpose == AllocationPersonal && len(weights) != 1 || input.Purpose == AllocationShared && len(weights) != len(active) {
		return AllocationSnapshot{}, ErrInvalidAllocation
	}
	if input.Mode == AllocationByShares && !sumIs(weights, "100") || input.Mode == AllocationByAmounts && !sumIs(weights, total.Amount()) || !slices.Contains([]AllocationMode{AllocationByAmounts, AllocationByShares, AllocationEqual}, input.Mode) {
		return AllocationSnapshot{}, ErrInvalidAllocation
	}
	result := AllocationSnapshot{State: AllocationResolved, Purpose: input.Purpose, Mode: input.Mode, Origin: input.Origin, Reason: input.Reason, Inputs: cloneAllocationInputs(input.Members), RuleRefs: slices.Clone(input.RuleRefs)}
	if input.Mode == AllocationByAmounts {
		for _, member := range input.Members {
			if member.Amount.Sign() > 0 {
				result.Members = append(result.Members, MemberAmount{MemberID: member.MemberID, Money: *member.Amount})
			}
		}
	} else {
		allocated, err := total.Allocate(weights, decimalScale(total.Amount()))
		if err != nil {
			return AllocationSnapshot{}, ErrInvalidAllocation
		}
		for _, value := range allocated {
			if value.Money.Sign() == 0 {
				continue
			}
			result.Members = append(result.Members, MemberAmount{MemberID: household.MembershipID(value.ID), Money: value.Money})
		}
	}
	return result, result.Validate()
}

func (a AllocationInput) requiresActiveMembers() bool {
	return a.Mode != "" && a.Mode != AllocationUnknown
}

func activeMembers(active []household.MembershipID) (map[household.MembershipID]bool, error) {
	if len(active) == 0 || len(active) > 1000 {
		return nil, ErrInvalidAllocation
	}
	result := make(map[household.MembershipID]bool, len(active))
	for _, id := range active {
		if id == "" || result[id] {
			return nil, ErrInvalidAllocation
		}
		result[id] = true
	}
	return result, nil
}

func cloneAllocationInputs(values []AllocationMemberInput) []AllocationMemberInput {
	result := slices.Clone(values)
	for i := range result {
		if result[i].Amount != nil {
			amount := *result[i].Amount
			result[i].Amount = &amount
		}
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].MemberID != result[j].MemberID {
			return result[i].MemberID < result[j].MemberID
		}
		left, right := money.Asset(""), money.Asset("")
		if result[i].Amount != nil {
			left = result[i].Amount.Asset()
		}
		if result[j].Amount != nil {
			right = result[j].Amount.Asset()
		}
		return left < right
	})
	return result
}

func cloneAllocationInput(value AllocationInput) AllocationInput {
	value.Members = cloneAllocationInputs(value.Members)
	value.RuleRefs = slices.Clone(value.RuleRefs)
	sort.Slice(value.RuleRefs, func(i, j int) bool { return value.RuleRefs[i].ID < value.RuleRefs[j].ID })
	return value
}

func validPositiveDecimal(value string) bool {
	if len(value) == 0 || len(value) > money.MaxDecimalLength || !allocationDecimalSyntax.MatchString(value) {
		return false
	}
	parsed, ok := new(big.Rat).SetString(value)
	return ok && parsed.Sign() > 0
}

func sumIs(weights []money.Weight, expected string) bool {
	total := new(big.Rat)
	for _, weight := range weights {
		value, ok := new(big.Rat).SetString(weight.Value)
		if !ok {
			return false
		}
		total.Add(total, value)
	}
	want, ok := new(big.Rat).SetString(expected)
	return ok && total.Cmp(want) == 0
}

func decimalScale(value string) int32 {
	if index := strings.IndexByte(value, '.'); index >= 0 {
		return int32(len(value) - index - 1)
	}
	return 0
}
