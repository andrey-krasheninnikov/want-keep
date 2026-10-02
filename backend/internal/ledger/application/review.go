package application

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"slices"
	"sort"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type ReviewResult struct {
	OperationID                   string
	Revision                      uint64
	ActorID                       household.UserID
	State, Rationale, PayloadHash string
	At                            calendar.Instant
	Evidence                      []ledger.Evidence
	Proposal                      *ledger.ClassificationProposal
}
type ReviewRepository interface {
	ReviewResult(context.Context, household.Principal, string, uint64) (ReviewResult, bool, error)
	SaveReviewResult(context.Context, ReviewResult) error
}
type ReviewInput struct {
	OperationID      string
	Revision         uint64
	State, Rationale string
	Correction       *ledger.Correction
	Evidence         []ledger.Evidence
	Proposal         *ledger.ClassificationProposal
}

// CompleteReview accepts a result from a trusted worker in a household transaction.
// The initiator is the trusted principal, never a model-supplied actor.
func (s *Service) CompleteReview(ctx context.Context, p household.Principal, in ReviewInput) error {
	if !slices.Contains([]string{"reviewed", "clarification", "failed"}, in.State) || utf8.RuneCountInString(in.Rationale) < 1 || utf8.RuneCountInString(in.Rationale) > 2000 {
		return commands.Rejection{Code: "invalid_request"}
	}
	if in.Correction != nil && (in.State != "reviewed" || in.Correction.Principal != nil || in.Correction.Fees != nil || in.Correction.OccurredAt != nil) {
		return commands.Rejection{Code: "source_conflict"}
	}
	if in.Correction != nil && (in.Correction.CategoryID != nil || in.Correction.MerchantID != nil || in.Correction.ReceiptItems != nil) {
		return commands.Rejection{Code: "clarification_required"}
	}
	if len(in.Evidence) > 100 {
		return commands.Rejection{Code: "invalid_request"}
	}
	seen := map[ledger.Evidence]bool{}
	for _, e := range in.Evidence {
		if e.Validate() != nil || seen[e] {
			return commands.Rejection{Code: "invalid_request"}
		}
		seen[e] = true
	}
	// Only text and payer changes reach this encoder; money is never JSON-marshaled from domain objects.
	payload := struct {
		State, Rationale string
		Merchant, Note   *string
		Payer            *ledger.PayerChange
		Allocation       *reviewAllocationChangePayload
		Evidence         []ledger.Evidence
		Proposal         *classificationProposalPayload
	}{State: in.State, Rationale: in.Rationale, Evidence: in.Evidence}
	if in.Correction != nil {
		payload.Merchant = in.Correction.Merchant
		payload.Note = in.Correction.Note
		payload.Payer = in.Correction.Payer
		if in.Correction.Allocation != nil {
			payload.Allocation = reviewAllocationChange(in.Correction.Allocation)
		}
	}
	if in.Proposal != nil {
		payload.Proposal = proposalPayload(in.Proposal)
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	hash := sha256.Sum256(data)
	digest := hex.EncodeToString(hash[:])
	prior, exists, err := s.repository.ReviewResult(ctx, p, in.OperationID, in.Revision)
	if err != nil {
		return err
	}
	if exists {
		if prior.ActorID != p.UserID() || prior.PayloadHash != digest {
			return commands.Rejection{Code: "idempotency_conflict"}
		}
		return nil
	}
	current, found, err := s.repository.CurrentLedgerRevision(ctx, p, in.OperationID)
	if err != nil {
		return err
	}
	if !found {
		return commands.Rejection{Code: "not_found"}
	}
	if current.Revision != in.Revision {
		return commands.Rejection{Code: "version_conflict"}
	}
	if in.Proposal != nil {
		if err = in.Proposal.ValidateFor(current); err != nil {
			return s.reject(err)
		}
		candidate := current.Clone()
		candidate.CategoryID, candidate.MerchantID, candidate.ReceiptItems = in.Proposal.CategoryID, in.Proposal.MerchantID, slices.Clone(in.Proposal.ReceiptItems)
		if err = s.requireActiveClassification(ctx, p, candidate); err != nil {
			return err
		}
	}
	if err = s.repository.SaveReviewResult(ctx, ReviewResult{OperationID: in.OperationID, Revision: in.Revision, ActorID: p.UserID(), State: in.State, Rationale: in.Rationale, PayloadHash: digest, At: s.now(), Evidence: in.Evidence, Proposal: in.Proposal}); err != nil {
		return err
	}
	if in.Correction != nil {
		if _, err = s.applyChanges(ctx, p, []Change{{OperationID: in.OperationID, Expected: in.Revision, Correction: *in.Correction}}, in.Rationale, "automated"); err != nil {
			var rejection commands.Rejection
			if !errors.As(err, &rejection) || rejection.Code != "no_change" {
				return err
			}
		}
	}
	return nil
}

type reviewAllocationChangePayload struct {
	Allocation reviewAllocationInputPayload
	Items      []reviewItemAllocationPayload
	Members    []string
}

type reviewAllocationInputPayload struct {
	Mode, Purpose, Reason, Origin string
	Members                       []reviewAllocationMemberPayload
	Rules                         []reviewAllocationRulePayload
}

type reviewAllocationMemberPayload struct {
	MemberID, Share string
	Amount          *reviewMoneyPayload
}

type reviewMoneyPayload struct{ Amount, Asset string }
type reviewAllocationRulePayload struct {
	ID       string
	Revision uint64
}
type reviewItemAllocationPayload struct {
	ItemID     string
	Allocation reviewAllocationInputPayload
}

func reviewAllocationChange(value *ledger.AllocationChange) *reviewAllocationChangePayload {
	out := &reviewAllocationChangePayload{Allocation: reviewAllocationInput(value.Allocation)}
	for _, item := range value.Items {
		out.Items = append(out.Items, reviewItemAllocationPayload{ItemID: item.ItemID, Allocation: reviewAllocationInput(item.Allocation)})
	}
	sort.Slice(out.Items, func(i, j int) bool { return out.Items[i].ItemID < out.Items[j].ItemID })
	for _, member := range value.Members {
		out.Members = append(out.Members, string(member))
	}
	sort.Strings(out.Members)
	return out
}

func reviewAllocationInput(value ledger.AllocationInput) reviewAllocationInputPayload {
	out := reviewAllocationInputPayload{Mode: string(value.Mode), Purpose: string(value.Purpose), Reason: value.Reason, Origin: string(value.Origin)}
	for _, member := range value.Members {
		item := reviewAllocationMemberPayload{MemberID: string(member.MemberID), Share: member.Share}
		if member.Amount != nil {
			item.Amount = &reviewMoneyPayload{Amount: member.Amount.Amount(), Asset: string(member.Amount.Asset())}
		}
		out.Members = append(out.Members, item)
	}
	sort.Slice(out.Members, func(i, j int) bool {
		if out.Members[i].MemberID != out.Members[j].MemberID {
			return out.Members[i].MemberID < out.Members[j].MemberID
		}
		left, right := "", ""
		if out.Members[i].Amount != nil {
			left = out.Members[i].Amount.Asset
		}
		if out.Members[j].Amount != nil {
			right = out.Members[j].Amount.Asset
		}
		return left < right
	})
	for _, ref := range value.RuleRefs {
		out.Rules = append(out.Rules, reviewAllocationRulePayload{ID: ref.ID, Revision: ref.Revision})
	}
	sort.Slice(out.Rules, func(i, j int) bool { return out.Rules[i].ID < out.Rules[j].ID })
	return out
}

type classificationProposalPayload struct {
	CategoryID, MerchantID, MerchantAlias string
	Items                                 []classificationItemPayload
}

type classificationItemPayload struct {
	ID, Name, Quantity, CategoryID string
	Gross, Discount, Asset         string
}

func proposalPayload(value *ledger.ClassificationProposal) *classificationProposalPayload {
	out := &classificationProposalPayload{CategoryID: value.CategoryID, MerchantID: value.MerchantID, MerchantAlias: value.MerchantAlias}
	for _, item := range value.ReceiptItems {
		out.Items = append(out.Items, classificationItemPayload{ID: item.ID, Name: item.Name, Quantity: item.Quantity, CategoryID: item.CategoryID, Gross: item.Gross.Amount(), Discount: item.Discount.Amount(), Asset: string(item.Gross.Asset())})
	}
	return out
}

// ApplyValidatedReview accepts only classification and allocation after reference validation.
func (s *Service) ApplyValidatedReview(ctx context.Context, p household.Principal, id string, expected uint64, change ledger.Correction, reason string) (command.Result, error) {
	if change.Principal != nil || change.Fees != nil || change.OccurredAt != nil || change.Payer != nil || change.Merchant != nil || change.Note != nil || change.ReceiptItems != nil {
		return command.Result{}, commands.Rejection{Code: "source_conflict"}
	}
	return s.applyChanges(ctx, p, []Change{{OperationID: id, Expected: expected, Correction: change}}, reason, "automated")
}

// ReviewRuleAllocation uses the same item precedence as trusted classification.
func (s *Service) ReviewRuleAllocation(ctx context.Context, p household.Principal, current ledger.Revision, classification ledger.Correction, boundary uint64) (ledger.AllocationChange, bool, error) {
	current = current.Clone()
	if classification.CategoryID != nil {
		current.CategoryID = *classification.CategoryID
	}
	if classification.MerchantID != nil {
		current.MerchantID = *classification.MerchantID
	}
	resolved, _, err := s.resolveAllocationAt(ctx, p, current, boundary)
	if err != nil {
		return ledger.AllocationChange{}, false, err
	}
	fallback, items, err := resolved.AllocationBases()
	if err != nil {
		return ledger.AllocationChange{}, false, err
	}
	members, err := s.allocations.ActiveMemberIDs(ctx, p)
	if err != nil {
		return ledger.AllocationChange{}, false, err
	}
	return ledger.AllocationChange{Allocation: fallback, Items: items, Members: members}, resolved.Allocation.State != ledger.AllocationUnresolved && resolved.Allocation.State != ledger.AllocationPartial, nil
}
