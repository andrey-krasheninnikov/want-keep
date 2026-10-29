package application

import (
	"context"
	"errors"
	"time"

	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	expense "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matchingapp "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	receiptcontract "github.com/pchkauu/want-keep/backend/internal/receipts/contract"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

type Service struct {
	repository Repository
	refunds    *expenses.Service
	matching   *matchingapp.Service
	ledger     *journal.Service
	now        Clock
	newID      func() string
}

func NewService(r Repository, l *journal.Service, refunds *expenses.Service, matcher *matchingapp.Service, now Clock, newID func() string) *Service {
	return &Service{repository: r, refunds: refunds, ledger: l, matching: matcher, now: now, newID: newID}
}
func (s *Service) Start(ctx context.Context, p household.Principal, attachmentID, accountID string) (command.Result, error) {
	a, err := s.repository.Attachment(ctx, p, attachmentID)
	if err != nil {
		return command.Result{}, commands.Rejection{Code: "not_found"}
	}
	if a.Upload.AccountID != accountID {
		return command.Result{}, commands.Rejection{Code: "invalid_attachment"}
	}
	if a.State != attachment.Accepted || !a.OriginalReady || len(a.Pages) == 0 {
		return command.Result{}, commands.Rejection{Code: "attachment_not_ready"}
	}
	if _, err = s.repository.Account(ctx, p, accountID); err != nil {
		return command.Result{}, commands.Rejection{Code: "not_found"}
	}
	r, exists, err := s.repository.ReceiptByHash(ctx, p, accountID, a.Upload.Hash)
	if err != nil {
		return command.Result{}, err
	}
	if !exists {
		r = receipt.Receipt{ID: s.newID(), HouseholdID: p.HouseholdID(), ActorID: p.UserID(), AccountID: accountID, AttachmentID: attachmentID, Hash: a.Upload.Hash, Revision: 1, State: receipt.Processing, CreatedAt: s.now().UTC(), Candidates: []receipt.Candidate{}}
		for _, page := range a.Pages {
			r.Pages = append(r.Pages, receipt.Page{ID: s.newID(), Number: page.Number, Hash: page.Hash})
		}
		if err = s.repository.CreateReceipt(ctx, p, r); err != nil {
			return command.Result{}, err
		}
	}
	if err = s.repository.AddReceiptEvidence(ctx, p, r.ID, a); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "receipt", ResourceID: r.ID, Revision: r.Revision}, nil
}
func (s *Service) Read(ctx context.Context, p household.Principal, id string) (receipt.Receipt, error) {
	r, err := s.repository.Receipt(ctx, p, id)
	if err != nil {
		return r, err
	}
	if r.State == receipt.Processing {
		state, reason, e := s.repository.ReceiptProcessingState(ctx, p, id)
		if e != nil {
			return r, e
		}
		if reason != "" {
			r.State = state
			r.Reason = reason
		}
	}
	a, err := s.repository.Account(ctx, p, r.AccountID)
	if err != nil {
		return r, err
	}
	pages := []receipt.Extraction{}
	if r.Confirmed != nil {
		pages = append(pages, *r.Confirmed)
	} else {
		for _, page := range r.Pages {
			if page.Extraction != nil {
				pages = append(pages, *page.Extraction)
			}
		}
	}
	if len(pages) == len(r.Pages) || r.Confirmed != nil {
		if f, e := receipt.Build(pages, string(a.Asset), r.ID); e == nil {
			r.Facts = &f
		}
	}
	if r.TransactionID != "" && r.Facts != nil {
		current, found, e := s.repository.CurrentLedgerRevision(ctx, p, r.TransactionID)
		if e != nil {
			return r, e
		}
		if found {
			r.TransactionRevision = current.Revision
			if current.Type == ledger.Expense || current.Type == ledger.Refund {
				for _, posting := range current.Postings {
					if posting.Role != ledger.Principal {
						continue
					}
					zero, _ := money.NewMoney("0", posting.Money.Asset())
					principal := posting.Money
					var e error
					if current.Type == ledger.Expense {
						principal, e = zero.Subtract(posting.Money)
					}
					if e != nil {
						return r, e
					}
					r.Facts.Total = principal
					break
				}
				r.Facts.Items = current.ReceiptItems
			}
		}
	}
	return r, nil
}
func (s *Service) Prepare(_ context.Context, x jobapp.Execution) (jobapp.Result, error) {
	if x.Job.Kind != jobs.ReceiptValidation {
		return jobapp.Result{}, jobs.ErrInvalidJob
	}
	return jobapp.Result{State: jobs.Succeeded, Apply: func(ctx context.Context, p household.Principal) error {
		v, err := s.repository.ReceiptValidation(ctx, p, x.Job)
		if err != nil {
			return err
		}
		r, _, err := s.repository.ReceiptForPage(ctx, p, v.PageID)
		if err != nil {
			return err
		}
		var extracted *receipt.Extraction
		if v.State == "completed" {
			e, err := receiptcontract.Decode(v.Output)
			if err == nil {
				refs, refErr := s.repository.ReceiptPageReferences(ctx, p, v.PageID)
				if refErr != nil {
					return refErr
				}
				for index, item := range e.Items {
					if item.CategoryAlias == nil {
						continue
					}
					ref, exists := refs[*item.CategoryAlias]
					if !exists {
						continue
					}
					current, lookupErr := s.repository.Category(ctx, p, ref.ID)
					if lookupErr == nil && current.State == category.Active && current.Revision == ref.Revision {
						e.Items[index].CategoryID = current.ID
					}
				}
				extracted = &e
			} else {
				v.Code = "invalid_extraction"
			}
		}
		if err = s.repository.SaveReceiptPageOutcome(ctx, p, v, extracted); err != nil {
			return err
		}
		if r.State != receipt.Processing {
			return nil
		}
		r, err = s.repository.Receipt(ctx, p, r.ID)
		if err != nil {
			return err
		}
		for _, page := range r.Pages {
			if page.AttemptID == "" {
				return nil
			}
		}
		return s.finish(ctx, p, r, receipt.Answer{})
	}}, nil
}
func (s *Service) finish(ctx context.Context, p household.Principal, r receipt.Receipt, answer receipt.Answer) error {
	distinct, match := answer.Distinct, answer.Match
	expected := r.Revision
	r.Revision++
	a, err := s.repository.Account(ctx, p, r.AccountID)
	if err != nil {
		return err
	}
	pages := []receipt.Extraction{}
	if r.Confirmed != nil {
		pages = append(pages, *r.Confirmed)
	} else {
		for _, page := range r.Pages {
			if page.Extraction == nil {
				r.State = receipt.Clarification
				r.Reason = "A receipt page could not be read. Confirm the payment facts."
				return s.save(ctx, p, r, expected)
			}
			pages = append(pages, *page.Extraction)
		}
	}
	f, err := receipt.Build(pages, string(a.Asset), r.ID)
	if errors.Is(err, receipt.ErrSkipped) {
		r.State = receipt.Skipped
		r.Reason = "The document does not establish a payment."
		return s.save(ctx, p, r, expected)
	}
	if err != nil {
		r.State = receipt.Clarification
		r.Reason = "Confirm the receipt date, currency, paid total and line items; the extracted facts are incomplete or inconsistent."
		return s.save(ctx, p, r, expected)
	}
	selected := ""
	if match != nil {
		selected = match.ID
	}
	candidates, err := s.repository.ReceiptCandidates(ctx, p, r, f, selected)
	if err != nil {
		return err
	}
	changed := r.CandidateFingerprint != candidates.Fingerprint
	r.Candidates = candidates.Items
	r.CandidateFingerprint = candidates.Fingerprint
	r.CandidatesComplete = candidates.Complete
	if distinct && changed {
		r.State = receipt.Clarification
		r.Reason = "The possible payments changed. Confirm the updated matches or a distinct payment again."
		return s.save(ctx, p, r, expected)
	}

	if match != nil {
		confirmed := false
		for _, candidate := range candidates.Items {
			if candidate == *match {
				confirmed = true
			}
		}
		if !confirmed {
			return commands.Rejection{Code: "version_conflict"}
		}
		current, found, err := s.repository.CurrentLedgerRevision(ctx, p, match.ID)
		if err != nil {
			return err
		}
		if !found || current.Revision != match.Revision {
			return command.ErrVersionConflict
		}
		if current.Type != f.Kind {
			return commands.Rejection{Code: "matching_conflict"}
		}
		linked, err := s.ledger.AttachReceiptClassification(ctx, p, current.OperationID, current.Revision, f.Items)
		if err != nil {
			return err
		}
		current.Revision = linked.Revision
		r.TransactionID, r.TransactionRevision = current.OperationID, current.Revision
		r.State = receipt.Linked
		r.Reason = "Receipt evidence linked to the confirmed payment."
		return s.save(ctx, p, r, expected)
	}
	if (len(candidates.Items) > 0 || !candidates.Complete) && !distinct {
		r.State = receipt.Clarification
		r.Reason = "A possible payment already exists. Choose it or confirm a distinct purchase."
		return s.save(ctx, p, r, expected)
	}
	if f.Kind == ledger.Refund && answer.Purchase == nil {
		r.State = receipt.Clarification
		r.Reason = "Identify the original purchase before recording this refund."
		return s.save(ctx, p, r, expected)
	}
	zone, err := s.repository.AccountTimezone(ctx, p)
	if err != nil {
		return err
	}
	localClock := f.Time
	precision := "second"
	if localClock == "" {
		localClock = "00:00:00"
		precision = "day"
	}
	location, err := time.LoadLocation(zone.String())
	if err != nil {
		return err
	}
	local, err := time.ParseInLocation("2006-01-02T15:04:05", f.Date.String()+"T"+localClock, location)
	if err != nil || local.Format("2006-01-02T15:04:05") != f.Date.String()+"T"+localClock || local.After(s.now()) {
		r.State = receipt.Clarification
		r.Reason = "Confirm a valid payment date and local time."
		return s.save(ctx, p, r, expected)
	}
	at, err := calendar.ParseInstant(local.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return err
	}
	if answer.Purchase != nil {
		if f.Kind != ledger.Refund || s.refunds == nil {
			return commands.Rejection{Code: "invalid_request"}
		}
		items := []expense.ItemPortion{}
		for _, item := range answer.RefundItems {
			amount, err := money.NewMoney(item.Amount, f.Total.Asset())
			if err != nil {
				return commands.Rejection{Code: "invalid_money"}
			}
			items = append(items, expense.ItemPortion{ItemID: item.ID, Amount: amount})
		}
		result, err := s.refunds.CreateFromReceipt(ctx, p, expenses.CreateInput{PurchaseID: answer.Purchase.ID, PurchaseExpectedRevision: answer.Purchase.Revision, AccountID: r.AccountID, At: at, Amount: f.Total, Items: items, Reason: "confirmed_receipt_refund"}, r.AttachmentID, precision)
		if err != nil {
			return err
		}
		if distinct {
			result, err = s.confirmDistinct(ctx, p, result.ResourceID, result.Revision)
			if err != nil {
				return err
			}
		}
		r.TransactionID, r.TransactionRevision = result.ResourceID, result.Revision

		r.State = receipt.Recorded
		r.Reason = "The receipt recorded a refund against the confirmed original purchase."
		return s.save(ctx, p, r, expected)
	}
	month, _ := calendar.ParseMonth(f.Date.String()[:7])
	recorded, _ := calendar.ParseInstant(s.now().UTC().Format(time.RFC3339Nano))
	zero, _ := money.NewMoney("0", a.Asset)
	amount, _ := zero.Subtract(f.Total)
	revision := ledger.Revision{OperationID: s.newID(), Revision: 1, ActorID: p.UserID(), Reason: "confirmed_receipt", Type: ledger.Expense, State: ledger.Posted, OccurredAt: at, OccurredPrecision: precision, CashDate: f.Date, ExpenseMonth: month, Timezone: zone, Origin: "receipt", FeeKnowledge: ledger.UnknownFees, Merchant: f.Merchant, AttachmentID: r.AttachmentID, RecordedAt: recorded, PayerState: "unknown", Allocation: ledger.AllocationSnapshot{}, AllocationReason: "receipt_allocation_unresolved", Postings: []ledger.Posting{{AccountID: r.AccountID, Money: amount, Role: ledger.Principal, Funding: ledger.OwnFunds, Treatment: ledger.Movement}}, ReceiptItems: f.Items, Protections: map[ledger.Field]ledger.Protection{}, FieldVersions: map[ledger.Field]uint64{}}
	if a.Product == "credit_card" {
		revision.Postings[0].Funding = ledger.UnknownFunds
	}
	revision, err = revision.WithAllocation(ledger.AllocationInput{Mode: ledger.AllocationUnknown, Reason: "receipt_allocation_unresolved"}, nil, nil)
	if err != nil {
		return err
	}
	result, err := s.ledger.RecordReceipt(ctx, p, revision)
	if err != nil {
		return err
	}
	revision.Revision = result.Revision
	if distinct {
		result, err = s.confirmDistinct(ctx, p, revision.OperationID, revision.Revision)
		if err != nil {
			return err
		}
		revision.Revision = result.Revision
	}

	r.State = receipt.Recorded
	r.Reason = "The confirmed receipt created one payment."
	r.TransactionID = revision.OperationID
	r.TransactionRevision = revision.Revision
	return s.save(ctx, p, r, expected)
}
func (s *Service) Answer(ctx context.Context, p household.Principal, id string, answer receipt.Answer) (command.Result, error) {
	r, err := s.repository.Receipt(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if r.Revision != answer.ExpectedRevision {
		return command.Result{}, command.ErrVersionConflict
	}
	if r.State == receipt.Processing {
		state, _, e := s.repository.ReceiptProcessingState(ctx, p, id)
		if e != nil {
			return command.Result{}, e
		}
		if state == receipt.Clarification {
			r.State = state
		}
	}
	if r.State != receipt.Clarification {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	count := 0
	if answer.Facts != nil {
		count++
	}
	if answer.Match != nil {
		count++
	}
	if answer.Distinct && answer.Purchase == nil {
		count++
	}
	if answer.Purchase != nil {
		count++
	}
	if len(answer.RefundItems) > 0 && answer.Purchase == nil {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if count != 1 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	if answer.Facts != nil {
		r.Confirmed = answer.Facts
	}
	if err = s.finish(ctx, p, r, answer); err != nil {
		return command.Result{}, err
	}
	r, err = s.repository.Receipt(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if err = s.repository.SaveReceiptAnswer(ctx, p, r, answer); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "receipt", ResourceID: id, Revision: r.Revision}, nil
}

func (s *Service) Allocate(ctx context.Context, p household.Principal, id string, expected, subject uint64, allocation ledger.AllocationInput, items []ledger.ItemAllocationInput, reason string) (command.Result, error) {
	r, err := s.repository.Receipt(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if r.Revision != expected {
		return command.Result{}, command.ErrVersionConflict
	}
	if r.TransactionID == "" || r.State != receipt.Recorded && r.State != receipt.Linked {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	result, err := s.ledger.Allocate(ctx, p, r.TransactionID, subject, allocation, items, reason)
	if err != nil {
		return command.Result{}, err
	}
	r.TransactionRevision = result.Revision
	r.Revision++
	r.Reason = "The receipt allocation was updated."
	if err = s.repository.SaveReceipt(ctx, p, r, expected); err != nil {
		return command.Result{}, err
	}
	return command.Result{ResourceType: "receipt", ResourceID: id, Revision: r.Revision}, nil
}

func (s *Service) save(ctx context.Context, p household.Principal, r receipt.Receipt, expected uint64) error {
	if err := s.repository.SaveReceipt(ctx, p, r, expected); err != nil {
		return err
	}
	if r.State == receipt.Recorded || r.State == receipt.Linked {
		return s.repository.EmitEvent(ctx, "receipt", r.ID, r.Revision, "receipt.completed")
	}
	return nil
}

func (s *Service) confirmDistinct(ctx context.Context, p household.Principal, id string, revision uint64) (command.Result, error) {
	result := command.Result{ResourceType: "transaction", ResourceID: id, Revision: revision}
	group, found, err := s.repository.MatchingForOperation(ctx, p, id)
	if err != nil {
		return result, err
	}
	if found && group.State == matching.Clarification {
		return s.matching.Separate(ctx, p, group.ID, group.Revision, group.Members, "receipt_confirmed_distinct")
	}
	return result, nil
}
