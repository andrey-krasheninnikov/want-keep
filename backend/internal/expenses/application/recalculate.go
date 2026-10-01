package application

import (
	"context"
	"errors"
	"sort"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type refundChange struct {
	refund   ledger.Revision
	items    []expenses.ItemPortion
	expected uint64
	reason   string
}

func recalculate(ctx context.Context, repository Repository, principal household.Principal, purchase ledger.Revision, current []expenses.Refund, revisions map[string]ledger.Revision, basis *expenses.ValuationBasis, change *refundChange, actor household.UserID, recordedAt calendar.Instant) (expenses.Refund, error) {
	links := make(map[string]expenses.Refund, len(current)+1)
	for _, link := range current {
		links[link.OperationID] = link
	}
	changedID := ""
	if change != nil {
		changedID = change.refund.OperationID
		link := links[changedID]
		link.OperationID = changedID
		link.PurchaseID = purchase.OperationID
		link.Items = append([]expenses.ItemPortion(nil), change.items...)
		links[changedID] = link
		revisions[changedID] = change.refund
	}

	purchaseAmount, err := principalAmount(purchase, -1)
	if err != nil || purchase.Type != ledger.Expense {
		return expenses.Refund{}, expenses.ErrInvalidRefund
	}
	zero, _ := money.NewMoney("0", purchaseAmount.Asset())
	total := zero
	itemTotals := map[string]money.Money{}
	activeAmounts := map[string]money.Money{}
	ids := make([]string, 0, len(links))
	for id := range links {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	for _, id := range ids {
		link, revision := links[id], revisions[id]
		amount, amountErr := principalAmount(revision, 1)
		if amountErr != nil || revision.Type != ledger.Refund || amount.Asset() != purchaseAmount.Asset() {
			return expenses.Refund{}, expenses.ErrInvalidRefund
		}
		if !active(purchase, revision) {
			continue
		}
		activeAmounts[id] = amount
		total, err = total.Add(amount)
		if err != nil {
			return expenses.Refund{}, expenses.ErrInvalidRefund
		}
		for _, item := range link.Items {
			itemTotals[item.ItemID], err = addAmount(itemTotals[item.ItemID], item.Amount)
			if err != nil {
				return expenses.Refund{}, expenses.ErrInvalidRefund
			}
		}
	}
	if compared, _ := total.Compare(purchaseAmount); compared > 0 {
		return expenses.Refund{}, expenses.ErrRefundExceedsPurchase
	}
	valuations := map[string]expenses.ValuationShare{}
	if basis != nil {
		valuations, err = expenses.AllocateValuations(*basis, purchaseAmount, activeAmounts)
		if err != nil {
			return expenses.Refund{}, err
		}
	}

	var changed expenses.Refund
	updates := make([]expenses.Refund, 0, len(ids))
	expectedRevisions := make(map[string]uint64, len(ids))
	for _, id := range ids {
		link, revision := links[id], revisions[id]
		refunded := total
		refundedItems := cloneAmounts(itemTotals)
		if amount, ok := activeAmounts[id]; ok {
			refunded, _ = refunded.Subtract(amount)
			for _, item := range link.Items {
				left, subtractErr := refundedItems[item.ItemID].Subtract(item.Amount)
				if subtractErr != nil {
					return expenses.Refund{}, expenses.ErrInvalidRefund
				}
				if left.Sign() == 0 {
					delete(refundedItems, item.ItemID)
				} else {
					refundedItems[item.ItemID] = left
				}
			}
		}
		expected, reason := link.Revision, "refund_recalculation"
		items := link.Items
		force := id == changedID
		if force {
			expected, reason, items = change.expected, change.reason, change.items
		}
		var valuation *expenses.ValuationShare
		if value, ok := valuations[id]; ok {
			valuation = &value
		}
		next, calculateErr := expenses.Calculate(purchase, revision, items, refunded, refundedItems, valuation, expected+1, reason, actor, recordedAt)
		if errors.Is(calculateErr, expenses.ErrClarificationRequired) {
			next, calculateErr = expenses.Clarify(purchase, revision, items, refunded, expected+1, reason, actor, recordedAt)
		}
		if calculateErr != nil {
			return expenses.Refund{}, calculateErr
		}
		if force {
			changed = next
		}
		if !force && link.SameCalculation(next) {
			continue
		}
		updates = append(updates, next)
		expectedRevisions[next.OperationID] = expected
	}
	if len(updates) > 0 {
		if err = repository.SaveRefunds(ctx, principal, updates, expectedRevisions); err != nil {
			return expenses.Refund{}, err
		}
	}
	return changed, nil
}

func active(purchase, refund ledger.Revision) bool {
	return purchase.State == ledger.Posted && purchase.Accounting() == ledger.IncludedInAccounting && refund.State == ledger.Posted && refund.Accounting() == ledger.IncludedInAccounting
}

func principalAmount(revision ledger.Revision, sign int) (money.Money, error) {
	var found *money.Money
	for _, posting := range revision.Postings {
		if posting.Role != ledger.Principal || !posting.MovesMoney() {
			continue
		}
		if found != nil {
			return money.Money{}, expenses.ErrInvalidRefund
		}
		value := posting.Money
		found = &value
	}
	if found == nil || found.Sign() != sign {
		return money.Money{}, expenses.ErrInvalidRefund
	}
	if sign < 0 {
		zero, _ := money.NewMoney("0", found.Asset())
		return zero.Subtract(*found)
	}
	return *found, nil
}

func addAmount(current, value money.Money) (money.Money, error) {
	if current.Validate() != nil {
		current, _ = money.NewMoney("0", value.Asset())
	}
	return current.Add(value)
}

func cloneAmounts(values map[string]money.Money) map[string]money.Money {
	result := make(map[string]money.Money, len(values))
	for id, value := range values {
		result[id] = value
	}
	return result
}
