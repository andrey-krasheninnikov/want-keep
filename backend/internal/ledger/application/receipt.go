package application

import (
	"context"
	attachmentdomain "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

// RecordReceipt accepts verified document facts without marking them as human overrides.
func (s *Service) RecordReceipt(ctx context.Context, p household.Principal, r ledger.Revision) (command.Result, error) {
	if r.Origin != "receipt" || r.Revision != 1 || r.Type != ledger.Expense || r.State != ledger.Posted || r.HumanOverride || len(r.Protections) != 0 || r.AttachmentID == "" || len(r.Postings) != 1 {
		return command.Result{}, commands.Rejection{Code: "invalid_transaction"}
	}
	attachment, err := s.repository.Attachment(ctx, p, r.AttachmentID)
	if err != nil || attachment.State != attachmentdomain.Accepted || !attachment.OriginalReady || attachment.Upload.AccountID != r.Postings[0].AccountID {
		return command.Result{}, commands.Rejection{Code: "invalid_attachment"}
	}
	if err = s.requireActiveClassification(ctx, p, r); err != nil {
		return command.Result{}, err
	}
	r, err = r.WithAllocation(ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: "receipt_allocation_unresolved"}, nil, nil)
	if err != nil {
		return command.Result{}, err
	}
	if s.allocations != nil {
		// The household lock keeps current rules stable until the first-fact boundary is saved.
		r, _, err = s.resolveAllocationAt(ctx, p, r, command.MaxRevision)
		if err != nil {
			return command.Result{}, err
		}
	}
	return s.append(ctx, p, r)
}

// AttachReceiptClassification fills unprotected item details without replacing a confirmed classification.
func (s *Service) AttachReceiptClassification(ctx context.Context, p household.Principal, id string, expected uint64, items []ledger.ReceiptItem) (command.Result, error) {
	current, err := s.current(ctx, p, id, expected)
	if err != nil {
		return command.Result{}, err
	}
	unchanged := command.Result{ResourceType: "transaction", ResourceID: id, Revision: expected}
	if len(items) == 0 || len(current.ReceiptItems) > 0 || current.Type != ledger.Expense {
		return unchanged, nil
	}
	for _, field := range []ledger.Field{ledger.LegacyField, ledger.ReceiptItemsField, ledger.CategoryField} {
		if _, protected := current.Protections[field]; protected {
			return unchanged, nil
		}
	}
	if current.HumanOverride && len(current.Protections) == 0 {
		return unchanged, nil
	}
	discount, err := money.NewMoney("0", items[0].Gross.Asset())
	if err != nil {
		return command.Result{}, err
	}
	inputs := []ledger.ReceiptItemInput{}
	for _, item := range items {
		d := item.Discount
		inputs = append(inputs, ledger.ReceiptItemInput{ID: item.ID, Name: item.Name, Quantity: item.Quantity, Gross: item.Gross, Discount: &d, CategoryID: item.CategoryID})
		discount, err = discount.Add(d)
		if err != nil {
			return command.Result{}, err
		}
	}
	empty := ""
	change := ledger.Correction{ReceiptItems: &ledger.ReceiptItemsCorrection{Items: inputs, TotalDiscount: discount}, CategoryID: &empty}
	return s.applyChanges(ctx, p, []Change{{OperationID: id, Expected: expected, Correction: change}}, "receipt_items_confirmed", "automated")
}
