package domain

import (
	"slices"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestAllocationUsesExactLargestRemaindersForEveryAsset(t *testing.T) {
	members := []household.MembershipID{"member-b", "member-a", "member-c"}
	for _, asset := range []money.Asset{money.RUB, money.USD, money.USDT, money.USDC, money.BTC, money.ETH} {
		amount, err := money.NewMoney("0.00000000000000000001", asset)
		if err != nil {
			t.Fatal(err)
		}
		revision := expenseRevision(amount)
		input := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "33.333333333333333333"}, {MemberID: members[1], Share: "33.333333333333333334"}, {MemberID: members[2], Share: "33.333333333333333333"}}, Origin: AllocationExplicitPurchase}
		allocated, err := revision.WithAllocation(input, nil, members)
		if err != nil {
			t.Fatalf("%s: %v", asset, err)
		}
		if allocated.Allocation.State != AllocationResolved || len(allocated.Allocation.Members) != 1 || allocated.Allocation.Members[0].MemberID != "member-a" || allocated.Allocation.Members[0].Money.Amount() != amount.Amount() {
			t.Fatalf("%s allocation = %+v", asset, allocated.Allocation)
		}
	}
}

func TestMixedReceiptAllocationCountsHouseholdOnce(t *testing.T) {
	paid := mustAllocationMoney(t, "1000", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{
		{ID: "shared", Name: "Shared", Quantity: "1", Gross: mustAllocationMoney(t, "600", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "personal-a", Name: "A", Quantity: "1", Gross: mustAllocationMoney(t, "100", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "personal-b", Name: "B", Quantity: "1", Gross: mustAllocationMoney(t, "300", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	members := []household.MembershipID{"member-a", "member-b"}
	shared := AllocationInput{Mode: AllocationEqual, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0]}, {MemberID: members[1]}}, Origin: AllocationExplicitItem}
	personalA := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationPersonal, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "100", money.RUB)}}, Origin: AllocationExplicitItem}
	personalB := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationPersonal, Members: []AllocationMemberInput{{MemberID: members[1], Amount: allocationMoneyPtr(t, "300", money.RUB)}}, Origin: AllocationExplicitItem}
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationUnknown, Reason: "item allocations"}, []ItemAllocationInput{{ItemID: "shared", Allocation: shared}, {ItemID: "personal-a", Allocation: personalA}, {ItemID: "personal-b", Allocation: personalB}}, members)
	if err != nil {
		t.Fatal(err)
	}
	if allocated.Allocation.State != AllocationResolved || allocated.Allocation.Mode != AllocationComposite || len(allocated.Allocation.Members) != 2 {
		t.Fatalf("allocation = %+v", allocated.Allocation)
	}
	if allocated.Allocation.Members[0].MemberID != members[0] || allocated.Allocation.Members[0].Money.Amount() != "400" || allocated.Allocation.Members[1].MemberID != members[1] || allocated.Allocation.Members[1].Money.Amount() != "600" {
		t.Fatalf("member totals = %+v", allocated.Allocation.Members)
	}
}

func TestAllocationRejectsInvalidMembersAndShareSumWithoutMutation(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "1000", money.RUB))
	input := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: "member-a", Share: "60"}, {MemberID: "member-b", Share: "30"}}}
	updated, err := revision.WithAllocation(input, nil, []household.MembershipID{"member-a", "member-b"})
	if err != ErrInvalidAllocation {
		t.Fatalf("err = %v", err)
	}
	if updated.Allocation.State != "" || revision.Allocation.State != "" {
		t.Fatal("failed allocation mutated revision")
	}
	input.Members[1].Share = "40"
	input.Members[1].MemberID = "foreign"
	if _, err = revision.WithAllocation(input, nil, []household.MembershipID{"member-a", "member-b"}); err != ErrInvalidAllocation {
		t.Fatalf("foreign member err = %v", err)
	}
}

func TestAllocationDistributesFeeInThirdAssetWithoutConversion(t *testing.T) {
	rub := mustAllocationMoney(t, "-9000", money.RUB)
	usdt := mustAllocationMoney(t, "100", money.USDT)
	ethFee := mustAllocationMoney(t, "-0.000000000000000003", money.ETH)
	revision := validAllocationRevision(Exchange, Posted, []Posting{{AccountID: "rub", Money: rub, Role: Principal, Funding: OwnFunds, Treatment: Movement}, {AccountID: "usdt", Money: usdt, Role: Principal, Funding: OwnFunds, Treatment: Movement}, {AccountID: "eth", Money: ethFee, Role: Fee, Funding: OwnFunds, Treatment: Movement}})
	members := []household.MembershipID{"member-a", "member-b"}
	input := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	if len(allocated.Allocation.Members) != 2 || allocated.Allocation.Members[0].Money.Asset() != money.ETH || allocated.Allocation.Members[0].Money.Amount() != "0.000000000000000002" || allocated.Allocation.Members[1].Money.Amount() != "0.000000000000000001" {
		t.Fatalf("third asset = %+v", allocated.Allocation.Members)
	}
}

func TestCorrectionRecalculatesSharesAndRejectsStaleAmounts(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	shares := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(shares, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	changed := []Posting{{AccountID: "account", Money: mustAllocationMoney(t, "-200", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement}}
	corrected, fields, err := allocated.Correct(Correction{Principal: &changed})
	if err != nil || !slices.Contains(fields, AllocationField) || corrected.Allocation.Members[0].Money.Amount() != "120" || corrected.Allocation.Members[1].Money.Amount() != "80" {
		t.Fatalf("share correction = %+v %v %v", corrected.Allocation, fields, err)
	}
	amounts := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "60", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "40", money.RUB)}}}
	allocated, err = revision.WithAllocation(amounts, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = allocated.Correct(Correction{Principal: &changed}); err != ErrInvalidAllocation {
		t.Fatalf("amount correction error = %v", err)
	}
}

func TestSourceMergeCannotEraseOrDetachAllocation(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	source := allocated.Clone()
	source.Allocation = AllocationSnapshot{}
	source.Merchant = "Updated by provider"
	merged, conflict, err := allocated.MergeSource(source)
	if err != nil || conflict || !allocated.FieldEqual(merged, AllocationField) || merged.Validate() != nil {
		t.Fatalf("safe source merge = %+v, conflict=%v, err=%v", merged.Allocation, conflict, err)
	}

	source.Postings[0].Money = mustAllocationMoney(t, "-200", money.RUB)
	merged, _, err = allocated.MergeSource(source)
	if err != nil {
		t.Fatal(err)
	}
	if err := merged.Validate(); err != ErrInvalidAllocation {
		t.Fatalf("stale allocation accepted after source amount change: %v", err)
	}
}

func TestAmountAllocationCoversMultipleComponentsAndAssetsExactly(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	revision.Postings = append(revision.Postings, Posting{AccountID: "account", Money: mustAllocationMoney(t, "-10", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement})
	input := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "70", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "40", money.RUB)}}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil || allocationMemberTotal(allocated.Allocation, members[0], money.RUB) != "70" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "40" {
		t.Fatalf("multi-component amounts = %+v, err=%v", allocated.Allocation, err)
	}

	revision = validAllocationRevision(Transfer, Posted, []Posting{
		{AccountID: "from", Money: mustAllocationMoney(t, "-100", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "to", Money: mustAllocationMoney(t, "100", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "from", Money: mustAllocationMoney(t, "-5", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "eth", Money: mustAllocationMoney(t, "-0.000000000000000003", money.ETH), Role: Fee, Funding: OwnFunds, Treatment: Movement},
	})
	input.Members = []AllocationMemberInput{
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "3", money.RUB)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "2", money.RUB)},
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "0.000000000000000002", money.ETH)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "0.000000000000000001", money.ETH)},
	}
	allocated, err = revision.WithAllocation(input, nil, members)
	if err != nil || allocated.Validate() != nil || allocationMemberTotal(allocated.Allocation, members[0], money.ETH) != "0.000000000000000002" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "2" {
		t.Fatalf("multi-asset amounts = %+v, err=%v", allocated.Allocation, err)
	}
}

func TestAmountAllocationPreservesExplicitPrecision(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "1", money.RUB))
	input := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "0.5", money.RUB)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "0.5", money.RUB)},
	}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil || allocationMemberTotal(allocated.Allocation, members[0], money.RUB) != "0.5" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "0.5" {
		t.Fatalf("precise amount allocation = %+v, err=%v", allocated.Allocation, err)
	}
}

func TestCompositeAllocationPreservesSharedIntentAtOneQuantum(t *testing.T) {
	paid := mustAllocationMoney(t, "0.01", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{{ID: "small", Name: "Small", Quantity: "1", Gross: paid, Discount: mustAllocationMoney(t, "0", money.RUB)}}
	members := []household.MembershipID{"member-a", "member-b"}
	shared := AllocationInput{Mode: AllocationEqual, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0]}, {MemberID: members[1]}}}
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationUnknown, Reason: "item allocation"}, []ItemAllocationInput{{ItemID: "small", Allocation: shared}}, members)
	if err != nil {
		t.Fatal(err)
	}
	if allocated.Allocation.Purpose != AllocationShared || allocated.Allocation.Members[0].MemberID != members[0] || allocated.Allocation.Members[0].Money.Amount() != "0.01" {
		t.Fatalf("shared one-quantum allocation = %+v", allocated.Allocation)
	}
}

func TestCompositeUnresolvedAllocationNormalizesAggregateAndKeepsItem(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	members := []household.MembershipID{"member-a", "member-b"}
	revision.ReceiptItems = []ReceiptItem{
		{ID: "ambiguous", Name: "Ambiguous", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "unknown", Name: "Unknown", Quantity: "1", Gross: mustAllocationMoney(t, "60", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	allocated, err := revision.WithAllocation(
		AllocationInput{Mode: AllocationUnknown, Reason: "purchase_unknown"},
		[]ItemAllocationInput{{ItemID: "ambiguous", Allocation: AllocationInput{Mode: AllocationUnknown, Reason: "item_unknown"}}},
		members,
	)
	if err != nil || allocated.Validate() != nil {
		t.Fatalf("unresolved composite allocation = %+v, err=%v", allocated.Allocation, err)
	}
	if allocated.Allocation.State != AllocationUnresolved || allocated.Allocation.Mode != AllocationUnknown || allocated.Allocation.Basis == nil || allocated.Allocation.Unallocated[0].Amount() != "100" {
		t.Fatalf("unresolved aggregate = %+v", allocated.Allocation)
	}
	if allocated.ReceiptItems[0].Allocation.Reason != "item_unknown" || allocated.ReceiptItems[1].Allocation.Reason != "purchase_unknown" {
		t.Fatalf("unresolved items = %+v", allocated.ReceiptItems)
	}
	refreshed, err := allocated.RefreshAllocation()
	if err != nil || refreshed.ReceiptItems[0].Allocation.Origin != AllocationExplicitItem || refreshed.ReceiptItems[0].Allocation.Reason != "item_unknown" || refreshed.ReceiptItems[1].Allocation.Reason != "purchase_unknown" {
		t.Fatalf("refreshed unresolved items = %+v, err=%v", refreshed.ReceiptItems, err)
	}
}

func TestExplicitUnresolvedItemSurvivesFeeCorrection(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{{ID: "unclear", Name: "Unclear", Quantity: "1", Gross: paid, Discount: mustAllocationMoney(t, "0", money.RUB)}}
	revision.Postings = append(revision.Postings, Posting{AccountID: "account", Money: mustAllocationMoney(t, "-10", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement})
	members := []household.MembershipID{"member-a", "member-b"}
	fallback := AllocationInput{Mode: AllocationEqual, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0]}, {MemberID: members[1]}}}
	allocated, err := revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "unclear", Allocation: AllocationInput{Mode: AllocationUnknown, Reason: "item_needs_clarification"}}}, members)
	if err != nil {
		t.Fatal(err)
	}
	fees := []Posting{{AccountID: "account", Money: mustAllocationMoney(t, "-20", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement}}
	corrected, fields, err := allocated.Correct(Correction{Fees: &fees})
	if err != nil || !slices.Contains(fields, AllocationField) {
		t.Fatalf("fee correction = %+v fields=%v err=%v", corrected.Allocation, fields, err)
	}
	item := corrected.ReceiptItems[0].Allocation
	if item.State != AllocationUnresolved || item.Origin != AllocationExplicitItem || item.Reason != "item_needs_clarification" || item.Unallocated[0].Amount() != "100" {
		t.Fatalf("corrected unresolved item = %+v", item)
	}
	if allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "10" || allocationMemberTotal(corrected.Allocation, members[1], money.RUB) != "10" {
		t.Fatalf("corrected fallback fee = %+v", corrected.Allocation.Members)
	}
}

func TestRuleItemBasisSurvivesFinancialRefresh(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{{ID: "ruled", Name: "Ruled", Quantity: "1", Gross: paid, Discount: mustAllocationMoney(t, "0", money.RUB)}}
	revision.Postings = append(revision.Postings, Posting{AccountID: "account", Money: mustAllocationMoney(t, "-10", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement})
	members := []household.MembershipID{"member-a", "member-b"}
	rule := AllocationInput{
		Mode:     AllocationByShares,
		Purpose:  AllocationShared,
		Origin:   AllocationRule,
		Members:  []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}},
		RuleRefs: []AllocationRuleRef{{ID: "rule", Revision: 3}},
	}
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationUnknown, Reason: "purchase_unknown"}, []ItemAllocationInput{{ItemID: "ruled", Allocation: rule}}, members)
	if err != nil {
		t.Fatal(err)
	}
	fees := []Posting{{AccountID: "account", Money: mustAllocationMoney(t, "-20", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement}}
	corrected, fields, err := allocated.Correct(Correction{Fees: &fees})
	if err != nil || !slices.Contains(fields, AllocationField) {
		t.Fatalf("rule refresh = %+v fields=%v err=%v", corrected.Allocation, fields, err)
	}
	item := corrected.ReceiptItems[0].Allocation
	if item.Origin != AllocationRule || item.Basis == nil || len(item.RuleRefs) != 1 || item.RuleRefs[0].Revision != 3 {
		t.Fatalf("rule item basis = %+v", item)
	}
	if allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "60" || allocationMemberTotal(corrected.Allocation, members[1], money.RUB) != "40" {
		t.Fatalf("rule member totals = %+v", corrected.Allocation.Members)
	}
}

func TestWaitingAllocationSuspendsAndRestoresBasis(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{{ID: "personal", Name: "Personal", Quantity: "1", Gross: paid, Discount: mustAllocationMoney(t, "0", money.RUB)}}
	members := []household.MembershipID{"member-a", "member-b"}
	personal := AllocationInput{Mode: AllocationByShares, Purpose: AllocationPersonal, Origin: AllocationExplicitItem, Members: []AllocationMemberInput{{MemberID: members[0], Share: "100"}}}
	allocated, err := revision.WithAllocation(AllocationInput{Mode: AllocationUnknown, Reason: "purchase_unknown"}, []ItemAllocationInput{{ItemID: "personal", Allocation: personal}}, members)
	if err != nil {
		t.Fatal(err)
	}
	waiting := allocated.Clone()
	waiting.Participation = Participation{GroupID: "matching", Kind: "payment", State: "waiting"}
	waiting, err = waiting.RefreshAllocation()
	if err != nil || waiting.Validate() != nil {
		t.Fatalf("suspend allocation = %+v err=%v", waiting.Allocation, err)
	}
	if waiting.Allocation.State != AllocationNotApplicable || waiting.Allocation.Basis == nil || waiting.ReceiptItems[0].Allocation.State != AllocationNotApplicable || waiting.ReceiptItems[0].Allocation.Basis == nil {
		t.Fatalf("suspended basis = aggregate %+v item %+v", waiting.Allocation, waiting.ReceiptItems[0].Allocation)
	}
	waiting.Participation = Participation{}
	restored, err := waiting.RefreshAllocation()
	if err != nil || restored.Validate() != nil {
		t.Fatalf("restore allocation = %+v err=%v", restored.Allocation, err)
	}
	if restored.ReceiptItems[0].Allocation.Origin != AllocationExplicitItem || allocationMemberTotal(restored.Allocation, members[0], money.RUB) != "100" {
		t.Fatalf("restored basis = aggregate %+v item %+v", restored.Allocation, restored.ReceiptItems[0].Allocation)
	}
}

func TestCompositeShareAllocationRefreshesAndAmountBasisRejectsChanges(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "60", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	members := []household.MembershipID{"member-a", "member-b"}
	fallback := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "50"}, {MemberID: members[1], Share: "50"}}}
	override := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "first", Allocation: override}}, members)
	if err != nil || allocated.Allocation.Basis == nil {
		t.Fatalf("composite allocation = %+v, err=%v", allocated.Allocation, err)
	}
	principal := []Posting{{AccountID: "account", Money: mustAllocationMoney(t, "-200", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement}}
	zero := mustAllocationMoney(t, "0", money.RUB)
	items := ReceiptItemsCorrection{Items: []ReceiptItemInput{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "80", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "120", money.RUB)},
	}, TotalDiscount: zero}
	corrected, fields, err := allocated.Correct(Correction{Principal: &principal, ReceiptItems: &items})
	if err != nil || !slices.Contains(fields, AllocationField) || allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "108" || allocationMemberTotal(corrected.Allocation, members[1], money.RUB) != "92" {
		t.Fatalf("refreshed allocation = %+v, fields=%v, err=%v", corrected.Allocation, fields, err)
	}

	amountOverride := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "24", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "16", money.RUB)}}}
	allocated, err = revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "first", Allocation: amountOverride}}, members)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err = allocated.Correct(Correction{Principal: &principal, ReceiptItems: &items}); err != ErrInvalidAllocation {
		t.Fatalf("amount-based composite refresh error = %v", err)
	}
}

func TestCompositeAmountFallbackRefreshesOnlyExplicitItemBases(t *testing.T) {
	paid := mustAllocationMoney(t, "100", money.RUB)
	revision := expenseRevision(paid)
	revision.ReceiptItems = []ReceiptItem{
		{ID: "explicit", Name: "Explicit", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "fallback-a", Name: "Fallback A", Quantity: "1", Gross: mustAllocationMoney(t, "20", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "fallback-b", Name: "Fallback B", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	members := []household.MembershipID{"member-a", "member-b"}
	fallback := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "30", money.RUB)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "30", money.RUB)},
	}}
	override := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "10", money.RUB)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "30", money.RUB)},
	}}
	allocated, err := revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "explicit", Allocation: override}}, members)
	if err != nil {
		t.Fatal(err)
	}
	zero := mustAllocationMoney(t, "0", money.RUB)
	items := ReceiptItemsCorrection{Items: []ReceiptItemInput{
		{ID: "explicit", Name: "Explicit", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB)},
		{ID: "fallback-a", Name: "Fallback A changed", Quantity: "1", Gross: mustAllocationMoney(t, "30", money.RUB)},
		{ID: "fallback-b", Name: "Fallback B changed", Quantity: "1", Gross: mustAllocationMoney(t, "30", money.RUB)},
	}, TotalDiscount: zero}
	corrected, _, err := allocated.Correct(Correction{ReceiptItems: &items})
	if err != nil || corrected.Validate() != nil {
		t.Fatalf("refreshed composite amount allocation = %+v, err=%v", corrected.Allocation, err)
	}
	if allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "40" || allocationMemberTotal(corrected.Allocation, members[1], money.RUB) != "60" {
		t.Fatalf("refreshed member totals = %+v", corrected.Allocation.Members)
	}
	if corrected.ReceiptItems[0].Allocation.Origin != AllocationExplicitItem || corrected.ReceiptItems[1].Allocation.Origin == AllocationExplicitItem || corrected.ReceiptItems[2].Allocation.Origin == AllocationExplicitItem {
		t.Fatalf("item origins = %+v", corrected.ReceiptItems)
	}
}

func TestRefreshMakesZeroNetItemNotApplicableAndRetainsBasis(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	revision.ReceiptItems = []ReceiptItem{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "60", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	fallback := AllocationInput{Mode: AllocationEqual, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0]}, {MemberID: members[1]}}}
	override := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "50"}, {MemberID: members[1], Share: "50"}}, Origin: AllocationExplicitItem}
	allocated, err := revision.WithAllocation(fallback, []ItemAllocationInput{{ItemID: "first", Allocation: override}}, members)
	if err != nil {
		t.Fatal(err)
	}
	zero := mustAllocationMoney(t, "0", money.RUB)
	fullDiscount := mustAllocationMoney(t, "40", money.RUB)
	items := ReceiptItemsCorrection{Items: []ReceiptItemInput{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "40", money.RUB), Discount: &fullDiscount},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "100", money.RUB), Discount: &zero},
	}, TotalDiscount: fullDiscount}
	corrected, fields, err := allocated.Correct(Correction{ReceiptItems: &items})
	if err != nil || !slices.Contains(fields, AllocationField) || corrected.Validate() != nil {
		t.Fatalf("zero-net refresh = %+v, fields=%v, err=%v", corrected.Allocation, fields, err)
	}
	item := corrected.ReceiptItems[0].Allocation
	if item.State != AllocationNotApplicable || item.Basis == nil || item.Basis.Origin != AllocationExplicitItem || len(item.Members) != 0 {
		t.Fatalf("zero-net item allocation = %+v", item)
	}
	if allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "50" || allocationMemberTotal(corrected.Allocation, members[1], money.RUB) != "50" {
		t.Fatalf("remaining allocation = %+v", corrected.Allocation.Members)
	}
	_, bases, err := corrected.AllocationBases()
	if err != nil {
		t.Fatal(err)
	}
	for _, basis := range bases {
		if basis.ItemID == "first" {
			t.Fatal("dormant basis became an allocation input")
		}
	}
}

func TestMultiAssetAmountAllocationRefreshUsesUniqueMembers(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	revision.ReceiptItems = []ReceiptItem{{ID: "item", Name: "Original", Quantity: "1", Gross: mustAllocationMoney(t, "100", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)}}
	revision.Postings = append(revision.Postings, Posting{AccountID: "eth", Money: mustAllocationMoney(t, "-0.003", money.ETH), Role: Fee, Funding: OwnFunds, Treatment: Movement})
	members := []household.MembershipID{"member-a", "member-b"}
	input := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "60", money.RUB)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "40", money.RUB)},
		{MemberID: members[0], Amount: allocationMoneyPtr(t, "0.002", money.ETH)},
		{MemberID: members[1], Amount: allocationMoneyPtr(t, "0.001", money.ETH)},
	}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	items := ReceiptItemsCorrection{Items: []ReceiptItemInput{{ID: "item", Name: "Renamed", Quantity: "1", Gross: mustAllocationMoney(t, "100", money.RUB)}}, TotalDiscount: mustAllocationMoney(t, "0", money.RUB)}
	corrected, _, err := allocated.Correct(Correction{ReceiptItems: &items})
	if err != nil || corrected.Validate() != nil {
		t.Fatalf("multi-asset refresh = %+v, err=%v", corrected.Allocation, err)
	}
	if allocationMemberTotal(corrected.Allocation, members[0], money.RUB) != "60" || allocationMemberTotal(corrected.Allocation, members[0], money.ETH) != "0.002" {
		t.Fatalf("multi-asset member totals = %+v", corrected.Allocation.Members)
	}
}

func TestAllocationInputOrderIsSemanticNoChange(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	members := []household.MembershipID{"member-a", "member-b"}
	first := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}}}
	allocated, err := revision.WithAllocation(first, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	reordered := AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[1], Share: "40"}, {MemberID: members[0], Share: "60"}}}
	if _, _, err = allocated.Correct(Correction{Allocation: &AllocationChange{Allocation: reordered, Members: members}}); err != ErrNoChange {
		t.Fatalf("reordered allocation error = %v", err)
	}
}

func TestAllocationDecimalScaleIsSemanticNoChange(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	members := []household.MembershipID{"member-a", "member-b"}
	for _, test := range []struct {
		name    string
		first   AllocationInput
		changed AllocationInput
	}{
		{
			name:    "shares",
			first:   AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "50"}, {MemberID: members[1], Share: "50.0"}}},
			changed: AllocationInput{Mode: AllocationByShares, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Share: "50.00"}, {MemberID: members[1], Share: "50"}}},
		},
		{
			name: "amounts",
			first: AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{
				{MemberID: members[0], Amount: allocationMoneyPtr(t, "50", money.RUB)},
				{MemberID: members[1], Amount: allocationMoneyPtr(t, "50.0", money.RUB)},
			}},
			changed: AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{
				{MemberID: members[0], Amount: allocationMoneyPtr(t, "50.00", money.RUB)},
				{MemberID: members[1], Amount: allocationMoneyPtr(t, "50", money.RUB)},
			}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			allocated, err := revision.WithAllocation(test.first, nil, members)
			if err != nil {
				t.Fatal(err)
			}
			if _, _, err = allocated.Correct(Correction{Allocation: &AllocationChange{Allocation: test.changed, Members: members}}); err != ErrNoChange {
				t.Fatalf("decimal scale change error = %v", err)
			}
		})
	}
}

func TestAllocationExcludesMatchedInternalAndNonCarrierPrincipal(t *testing.T) {
	for _, tc := range []struct {
		name         string
		kind         ParticipationKind
		carrierID    string
		contribution ContributionRole
	}{
		{name: "internal transfer", kind: "transfer", carrierID: "operation", contribution: "outgoing"},
		{name: "payment evidence", kind: "payment", carrierID: "other", contribution: "payment"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			revision := expenseRevision(mustAllocationMoney(t, "1000", money.RUB))
			revision.Allocation, revision.Participation = AllocationSnapshot{}, Participation{GroupID: "group", Kind: tc.kind, State: "linked", Parts: []Contribution{{ComponentID: tc.carrierID + ":0", Position: 0, CarrierID: tc.carrierID, CarrierPosition: 0, Role: tc.contribution, State: Posted, At: revision.OccurredAt}}}
			refreshed, err := revision.RefreshAllocation()
			if err != nil || refreshed.Allocation.State != AllocationNotApplicable || refreshed.Validate() != nil {
				t.Fatalf("refreshed = %+v, err=%v", refreshed.Allocation, err)
			}
		})
	}
}

func TestWithAllocationRefreshesSuspendedBasisWithoutRestoringEffect(t *testing.T) {
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	members := []household.MembershipID{"member-a", "member-b"}
	initial := AllocationInput{Mode: AllocationUnknown, Reason: "allocation_unresolved", Origin: AllocationUnknownOrigin}
	allocated, err := revision.WithAllocation(initial, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	allocated.Participation = Participation{GroupID: "matching-group", Kind: "payment", State: "waiting"}
	suspended, err := allocated.RefreshAllocation()
	if err != nil {
		t.Fatal(err)
	}
	updatedInput := AllocationInput{
		Mode: AllocationByShares, Purpose: AllocationShared, Origin: AllocationRule,
		Members:  []AllocationMemberInput{{MemberID: members[0], Share: "60"}, {MemberID: members[1], Share: "40"}},
		RuleRefs: []AllocationRuleRef{{ID: "rule", Revision: 1}},
	}
	updated, err := suspended.WithAllocation(updatedInput, nil, members)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Allocation.State != AllocationNotApplicable || updated.Allocation.Basis == nil || updated.Allocation.Basis.Origin != AllocationRule || len(updated.Allocation.Members) != 0 {
		t.Fatalf("suspended allocation = %+v", updated.Allocation)
	}
	restored := updated.Clone()
	restored.Participation = Participation{}
	restored, err = restored.RefreshAllocation()
	if err != nil {
		t.Fatal(err)
	}
	if restored.Allocation.State != AllocationResolved || restored.Allocation.Origin != AllocationRule || allocationMemberTotal(restored.Allocation, members[0], money.RUB) != "60" || allocationMemberTotal(restored.Allocation, members[1], money.RUB) != "40" {
		t.Fatalf("restored allocation = %+v", restored.Allocation)
	}
}

func TestIncomeAllocationIsNotApplicableEvenWithFee(t *testing.T) {
	revision := validAllocationRevision(Income, Posted, []Posting{
		{AccountID: "account", Money: mustAllocationMoney(t, "100", money.RUB), Role: Principal, Funding: OwnFunds, Treatment: Movement},
		{AccountID: "account", Money: mustAllocationMoney(t, "-5", money.RUB), Role: Fee, Funding: OwnFunds, Treatment: Movement},
	})
	allocated, err := revision.WithAllocation(AllocationInput{}, nil, nil)
	if err != nil || allocated.Allocation.State != AllocationNotApplicable || allocated.Validate() != nil {
		t.Fatalf("income allocation = %+v, err=%v", allocated.Allocation, err)
	}
}

func TestPurchaseAmountFallbackProducesExactItemSnapshots(t *testing.T) {
	members := []household.MembershipID{"member-a", "member-b"}
	revision := expenseRevision(mustAllocationMoney(t, "100", money.RUB))
	revision.ReceiptItems = []ReceiptItem{
		{ID: "first", Name: "First", Quantity: "1", Gross: mustAllocationMoney(t, "33", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
		{ID: "second", Name: "Second", Quantity: "1", Gross: mustAllocationMoney(t, "67", money.RUB), Discount: mustAllocationMoney(t, "0", money.RUB)},
	}
	input := AllocationInput{Mode: AllocationByAmounts, Purpose: AllocationShared, Members: []AllocationMemberInput{{MemberID: members[0], Amount: allocationMoneyPtr(t, "40", money.RUB)}, {MemberID: members[1], Amount: allocationMoneyPtr(t, "60", money.RUB)}}}
	allocated, err := revision.WithAllocation(input, nil, members)
	if err != nil || allocated.Validate() != nil || allocationMemberTotal(allocated.Allocation, members[0], money.RUB) != "40" || allocationMemberTotal(allocated.Allocation, members[1], money.RUB) != "60" {
		t.Fatalf("purchase amount fallback = %+v, err=%v", allocated.Allocation, err)
	}
	for _, item := range allocated.ReceiptItems {
		if item.Allocation.State != AllocationResolved {
			t.Fatalf("item allocation = %+v", item.Allocation)
		}
	}
}

func allocationMemberTotal(snapshot AllocationSnapshot, memberID household.MembershipID, asset money.Asset) string {
	for _, member := range snapshot.Members {
		if member.MemberID == memberID && member.Money.Asset() == asset {
			return member.Money.Amount()
		}
	}
	return "0"
}

func expenseRevision(amount money.Money) Revision {
	zero, _ := money.NewMoney("0", amount.Asset())
	negative, _ := zero.Subtract(amount)
	return validAllocationRevision(Expense, Posted, []Posting{{AccountID: "account", Money: negative, Role: Principal, Funding: OwnFunds, Treatment: Movement}})
}

func validAllocationRevision(kind Type, state State, postings []Posting) Revision {
	at, _ := calendar.ParseInstant("2026-09-08T10:00:00Z")
	zone, _ := calendar.ParseTimezone("Europe/Moscow")
	date, _ := at.DateIn(zone)
	month, _ := calendar.ParseMonth(date.String()[:7])
	return Revision{OperationID: "operation", Revision: 1, ActorID: "actor", Reason: "synthetic", Type: kind, State: state, OccurredAt: at, PostedAt: at, Timezone: zone, CashDate: date, ExpenseMonth: month, Origin: "manual", FeeKnowledge: KnownFees, PayerState: "unknown", Postings: postings}
}

func mustAllocationMoney(t *testing.T, amount string, asset money.Asset) money.Money {
	t.Helper()
	value, err := money.NewMoney(amount, asset)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func allocationMoneyPtr(t *testing.T, amount string, asset money.Asset) *money.Money {
	value := mustAllocationMoney(t, amount, asset)
	return &value
}
