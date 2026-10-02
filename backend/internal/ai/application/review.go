package application

import (
	"context"
	"encoding/json"
	"errors"
	"slices"
	"unicode/utf8"

	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matchingapp "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

type ValidationSource struct {
	AttemptID string
	Context   ai.ReviewContext
	State     ai.State
	Output    json.RawMessage
}

type ReviewRepository interface {
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	ValidationSource(context.Context, household.Principal, jobs.Job) (ValidationSource, error)
	SaveReviewValidation(context.Context, household.Principal, ai.ReviewValidation) error
	SaveReviewProposal(context.Context, household.Principal, ai.ReviewProposal) error
	ReviewProposal(context.Context, household.Principal, string) (ai.ReviewProposal, error)
	SaveClarification(context.Context, household.Principal, ai.Clarification) error
	Clarification(context.Context, household.Principal, string) (ai.Clarification, error)
	Clarifications(context.Context, household.Principal, string, int) ([]ai.Clarification, string, error)
	SaveReviewAnswer(context.Context, household.Principal, ai.Clarification, ai.ReviewAnswer) error
	ProposalClarifications(context.Context, household.Principal, string) ([]ai.Clarification, error)
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	Category(context.Context, household.Principal, string) (category.Category, error)
	Merchant(context.Context, household.Principal, string) (category.Merchant, error)
}

type ReviewService struct {
	repository  ReviewRepository
	ledger      *journal.Service
	allocations journal.AllocationResolver
	matching    *matchingapp.Service
	newID       func() string
}

func NewReviewService(r ReviewRepository, l *journal.Service, a journal.AllocationResolver, m *matchingapp.Service, newID func() string) *ReviewService {
	return &ReviewService{r, l, a, m, newID}
}

// Prepare uses no provider IO: the durable source is an already settled response.
func (s *ReviewService) Prepare(_ context.Context, x jobapp.Execution) (jobapp.Result, error) {
	if x.Job.Kind != jobs.AIValidation {
		return jobapp.Result{}, jobs.ErrInvalidJob
	}
	return jobapp.Result{State: jobs.Succeeded, Apply: func(ctx context.Context, p household.Principal) error { return s.Validate(ctx, p, x.Job) }}, nil
}

func (s *ReviewService) Validate(ctx context.Context, p household.Principal, job jobs.Job) error {
	source, err := s.repository.ValidationSource(ctx, p, job)
	if err != nil {
		return err
	}
	result := ai.ReviewValidation{AttemptID: source.AttemptID, OperationID: job.ResourceID, Revision: job.ResourceRevision, State: "failed", Rationale: "The provider response could not be validated."}
	current, found, err := s.repository.CurrentLedgerRevision(ctx, p, job.ResourceID)
	if err != nil {
		return err
	}
	if !found || current.Revision != job.ResourceRevision {
		result.State, result.Code, result.Rationale = "stale", "version_conflict", "The transaction changed before this response was applied."
		return s.saveValidation(ctx, p, source.Context, result)
	}
	out, err := ai.DecodeReviewOutput(source.Output)
	if source.State != ai.Completed || err != nil || source.Context.OperationID != job.ResourceID || source.Context.Revision != job.ResourceRevision {
		result.Code = "invalid_request"
		if source.Context.ClarificationID == "" {
			if err = s.ledger.CompleteReview(ctx, p, journal.ReviewInput{OperationID: current.OperationID, Revision: current.Revision, State: "failed", Rationale: result.Rationale}); err != nil {
				return err
			}
		}
		return s.saveValidation(ctx, p, source.Context, result)
	}
	result.Rationale = out.Commands[0].Reason
	change, needsApproval, err := s.prepareCommands(ctx, p, source.Context, current, out.Commands, false)
	if err != nil {
		result.State, result.Code = "rejected", rejectionCode(err)
		return s.saveValidation(ctx, p, source.Context, result)
	}
	if needsApproval {
		question := "Confirm the purpose or suggested change for this transaction."
		for _, c := range out.Commands {
			if c.Question != nil {
				question = *c.Question
			}
		}
		proposal := ai.ReviewProposal{ID: s.newID(), Revision: 1, OperationID: current.OperationID, OperationRevision: current.Revision, State: "pending", Commands: out.Commands, Context: source.Context, Rationale: result.Rationale}
		if err = s.repository.SaveReviewProposal(ctx, p, proposal); err != nil {
			return err
		}
		clarification := ai.Clarification{ID: s.newID(), Revision: 1, OperationID: current.OperationID, OperationRevision: current.Revision, State: "open", Question: question, ProposalID: proposal.ID, Context: source.Context, Choices: s.choices(source.Context, out.Commands)}
		if err = s.repository.SaveClarification(ctx, p, clarification); err != nil {
			return err
		}
		result.State, result.ProposalID, result.ClarificationID = "clarification", proposal.ID, clarification.ID
		if source.Context.ClarificationID == "" {
			if err = s.ledger.CompleteReview(ctx, p, journal.ReviewInput{OperationID: current.OperationID, Revision: current.Revision, State: "clarification", Rationale: result.Rationale}); err != nil {
				return err
			}
		}
	} else {
		err = s.repository.WithinHousehold(ctx, p, func(ctx context.Context) error {
			if source.Context.ClarificationID == "" {
				if err := s.ledger.CompleteReview(ctx, p, journal.ReviewInput{OperationID: current.OperationID, Revision: current.Revision, State: "reviewed", Rationale: result.Rationale}); err != nil {
					return err
				}
			}
			if change.CategoryID != nil || change.MerchantID != nil || change.Allocation != nil {
				_, err := s.ledger.ApplyValidatedReview(ctx, p, current.OperationID, current.Revision, change, result.Rationale)
				if err != nil && rejectionCode(err) != "no_change" {
					return err
				}
			}
			return nil
		})
		if err != nil {
			var rejection commands.Rejection
			if !errors.As(err, &rejection) {
				return err
			}
			result.State, result.Code = "rejected", rejection.Code
		} else {
			result.State = "applied"
		}
	}
	return s.saveValidation(ctx, p, source.Context, result)

}

func (s *ReviewService) prepareCommands(ctx context.Context, p household.Principal, projection ai.ReviewContext, current ledger.Revision, values []ai.ReviewCommand, approved bool) (ledger.Correction, bool, error) {
	change := ledger.Correction{}
	approval := false
	seen := map[string]bool{}
	for _, c := range values {
		if seen[c.Kind] {
			return change, false, commands.Rejection{Code: "invalid_request"}
		}
		seen[c.Kind] = true
		if len(c.Evidence) == 0 || !slices.Contains(c.Evidence, "ledger_revision") {
			return change, false, commands.Rejection{Code: "invalid_request"}
		}
		for _, e := range c.Evidence {
			if _, err := projection.Resolve(e, "evidence"); err != nil {
				return change, false, commands.Rejection{Code: "invalid_request"}
			}
		}
		switch c.Kind {
		case "classification":
			if c.Category != nil {
				ref, err := projection.Resolve(*c.Category, "category")
				if err != nil {
					return change, false, commands.Rejection{Code: "invalid_request"}
				}
				value, err := s.repository.Category(ctx, p, ref.ID)
				if err != nil {
					return change, false, err
				}
				if value.Revision != ref.Revision || value.State != category.Active {
					return change, false, commands.Rejection{Code: "version_conflict"}
				}
				change.CategoryID = &ref.ID
			}
			if c.Merchant != nil {
				ref, err := projection.Resolve(*c.Merchant, "merchant")
				if err != nil {
					return change, false, commands.Rejection{Code: "invalid_request"}
				}
				value, err := s.repository.Merchant(ctx, p, ref.ID)
				if err != nil {
					return change, false, err
				}
				if value.Revision != ref.Revision || value.State != category.Active {
					return change, false, commands.Rejection{Code: "version_conflict"}
				}
				change.MerchantID = &ref.ID
			}
		case "distribution":
			if *c.Distribution != "rule" {
				approval = !approved
			}
		case "link":
			ref, err := projection.Resolve(*c.Candidate, "candidate")
			if err != nil {
				return change, false, commands.Rejection{Code: "invalid_request"}
			}
			value, found, err := s.repository.CurrentLedgerRevision(ctx, p, ref.ID)
			if err != nil {
				return change, false, err
			}
			if !found || value.Revision != ref.Revision {
				return change, false, commands.Rejection{Code: "version_conflict"}
			}
			approval = !approved
		case "clarify":
			approval = !approved
		case "budget", "goal":
			if approved {
				return change, false, commands.Rejection{Code: "feature_unavailable"}
			}
			approval = true
		case "reject":
			return change, false, commands.Rejection{Code: "invalid_request"}
		case "no_change":
			if len(values) != 1 {
				return change, false, commands.Rejection{Code: "invalid_request"}
			}
		}
	}
	for _, c := range values {
		if c.Kind != "distribution" {
			continue
		}
		var input ledger.AllocationInput
		if *c.Distribution == "rule" {
			categoryID, merchantID := current.CategoryID, current.MerchantID
			if change.CategoryID != nil {
				categoryID = *change.CategoryID
			}
			if change.MerchantID != nil {
				merchantID = *change.MerchantID
			}
			var matched bool
			var err error
			input, matched, err = s.allocations.Resolve(ctx, p, merchantID, categoryID)
			if err != nil {
				return change, false, err
			}
			for _, rule := range input.RuleRefs {
				found := false
				for _, ref := range projection.References {
					if ref.Kind == "rule" && ref.ID == rule.ID && ref.Revision == rule.Revision {
						found = true
						break
					}
				}
				if !found {
					return change, false, commands.Rejection{Code: "version_conflict"}
				}
			}
			if !matched {
				approval = true
				continue
			}
		} else if approved {
			input = ledger.AllocationInput{Mode: ledger.AllocationEqual, Purpose: ledger.AllocationShared, Reason: c.Reason, Origin: ledger.AllocationExplicitPurchase}
			if *c.Distribution == "personal" {
				ref, err := projection.Resolve(*c.Member, "member")
				if err != nil {
					return change, false, commands.Rejection{Code: "invalid_request"}
				}
				input.Mode = ledger.AllocationByShares
				input.Purpose = ledger.AllocationPersonal
				input.Members = []ledger.AllocationMemberInput{{MemberID: household.MembershipID(ref.ID), Share: "100"}}
			}
		} else {
			continue
		}
		change.Allocation = &ledger.AllocationChange{Allocation: input}
	}
	if !approved {
		fields := []ledger.Field{}
		if change.CategoryID != nil {
			fields = append(fields, ledger.CategoryField)
		}
		if change.MerchantID != nil {
			fields = append(fields, ledger.MerchantIDField)
		}
		if change.Allocation != nil {
			fields = append(fields, ledger.AllocationField)
		}
		for _, f := range fields {
			if _, ok := current.Protections[f]; ok {
				return change, false, commands.Rejection{Code: "protected_field"}
			}
		}
		if len(fields) > 0 {
			if _, ok := current.Protections[ledger.LegacyField]; ok || current.HumanOverride && len(current.Protections) == 0 {
				return change, false, commands.Rejection{Code: "protected_field"}
			}
		}
	}
	return change, approval, nil
}

func rejectionCode(err error) string {
	var r commands.Rejection
	if errors.As(err, &r) {
		return r.Code
	}
	return "invalid_request"
}

func (s *ReviewService) choices(projection ai.ReviewContext, values []ai.ReviewCommand) []ai.Choice {
	choices := []ai.Choice{{ID: "keep", Label: "Keep current values", Commands: []ai.ReviewCommand{{Kind: "no_change", Evidence: []string{"ledger_revision"}, Reason: "User confirmed current values."}}}}
	actionable := []ai.ReviewCommand{}
	for _, c := range values {
		if c.Kind != "clarify" {
			actionable = append(actionable, c)
		}
	}
	if len(actionable) > 0 {
		choices = append(choices, ai.Choice{ID: "apply", Label: "Apply the proposed change", Commands: actionable})
	}
	joint := "joint"
	choices = append(choices, ai.Choice{ID: "joint", Label: "Shared purchase", Commands: []ai.ReviewCommand{{Kind: "distribution", Distribution: &joint, Evidence: []string{"ledger_revision"}, Reason: "User confirmed shared purpose."}}})
	for alias, ref := range projection.References {
		if ref.Kind == "member" {
			personal := "personal"
			member := alias
			choices = append(choices, ai.Choice{ID: alias, Label: "Personal purchase: " + ref.Label, Commands: []ai.ReviewCommand{{Kind: "distribution", Distribution: &personal, Member: &member, Evidence: []string{"ledger_revision"}, Reason: "User confirmed personal purpose."}}})
		}
	}
	slices.SortFunc(choices, func(a, b ai.Choice) int {
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})
	return choices
}

func (s *ReviewService) Proposal(ctx context.Context, p household.Principal, id string) (ai.ReviewProposal, error) {
	return s.repository.ReviewProposal(ctx, p, id)
}
func (s *ReviewService) Clarifications(ctx context.Context, p household.Principal, after string, limit int) ([]ai.Clarification, string, error) {
	return s.repository.Clarifications(ctx, p, after, limit)
}

func (s *ReviewService) Apply(ctx context.Context, p household.Principal, id string, expected uint64, subject uint64, decision string) (command.Result, error) {
	proposal, err := s.repository.ReviewProposal(ctx, p, id)
	if err != nil {
		return command.Result{}, err
	}
	if proposal.Revision != expected || proposal.State != "pending" || proposal.OperationRevision != subject || expected >= command.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	if decision == "reject" {
		err = s.finishProposal(ctx, p, proposal, "rejected")
		proposal.Revision++
		return command.Result{ResourceType: "proposal", ResourceID: id, Revision: proposal.Revision}, err
	}
	if decision != "apply" {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	result, err := s.applyApproved(ctx, p, proposal.Context, proposal.Commands, proposal.Rationale)
	if err != nil {
		return command.Result{}, err
	}
	if err = s.finishProposal(ctx, p, proposal, "applied"); err != nil {
		return command.Result{}, err
	}
	return result, nil
}

func (s *ReviewService) applyApproved(ctx context.Context, p household.Principal, projection ai.ReviewContext, values []ai.ReviewCommand, reason string) (command.Result, error) {
	current, found, err := s.repository.CurrentLedgerRevision(ctx, p, projection.OperationID)
	if err != nil {
		return command.Result{}, err
	}
	if !found {
		return command.Result{}, commands.Rejection{Code: "not_found"}
	}
	if current.Revision != projection.Revision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	change, _, err := s.prepareCommands(ctx, p, projection, current, values, true)
	if err != nil {
		return command.Result{}, err
	}
	var link *ai.ReviewCommand
	for _, c := range values {
		if c.Kind == "link" {
			copy := c
			link = &copy
		}
	}
	if link != nil {
		// A link may alter several revisions, so it is its own complete decision.
		if change.CategoryID != nil || change.MerchantID != nil || change.Allocation != nil {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		ref, _ := projection.Resolve(*link.Candidate, "candidate")
		kind := matching.Payment
		if current.Type == ledger.Transfer {
			kind = matching.Transfer
		}
		if current.Type == ledger.Exchange {
			kind = matching.Exchange
		}
		return s.matching.Link(ctx, p, matchingapp.LinkInput{Kind: kind, PrimaryID: current.OperationID, Members: []matching.Member{{OperationID: current.OperationID, Revision: current.Revision}, {OperationID: ref.ID, Revision: ref.Revision}}, Reason: reason})
	}
	if change.CategoryID == nil && change.MerchantID == nil && change.Allocation == nil {
		return command.Result{ResourceType: "transaction", ResourceID: current.OperationID, Revision: current.Revision}, nil
	}
	result, err := s.ledger.Correct(ctx, p, journal.Change{OperationID: current.OperationID, Expected: current.Revision, Correction: change}, reason)
	if rejectionCode(err) == "no_change" {
		return command.Result{ResourceType: "transaction", ResourceID: current.OperationID, Revision: current.Revision}, nil
	}
	return result, err
}

func (s *ReviewService) Answer(ctx context.Context, p household.Principal, in ai.ReviewAnswer) (command.Result, error) {
	value, err := s.repository.Clarification(ctx, p, in.ClarificationID)
	if err != nil {
		return command.Result{}, err
	}
	if value.State != "open" || value.Revision != in.ExpectedRevision || value.Revision >= command.MaxRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	current, found, err := s.repository.CurrentLedgerRevision(ctx, p, value.OperationID)
	if err != nil {
		return command.Result{}, err
	}
	if !found || current.Revision != value.OperationRevision || in.SubjectExpectedRevision != value.OperationRevision {
		return command.Result{}, commands.Rejection{Code: "version_conflict"}
	}
	if (in.ChoiceID == "") == (in.Text == "") || utf8.RuneCountInString(in.Text) > 2000 {
		return command.Result{}, commands.Rejection{Code: "invalid_request"}
	}
	value.Revision++
	if in.ChoiceID != "" {
		index := slices.IndexFunc(value.Choices, func(c ai.Choice) bool { return c.ID == in.ChoiceID })
		if index < 0 {
			return command.Result{}, commands.Rejection{Code: "invalid_request"}
		}
		_, err = s.applyApproved(ctx, p, value.Context, value.Choices[index].Commands, "User answered the transaction clarification.")
		if err != nil {
			return command.Result{}, err
		}
		value.State = "answered"
	} else {
		value.State = "checking"
	}
	if err = s.repository.SaveClarification(ctx, p, value); err != nil {
		return command.Result{}, err
	}
	if err = s.repository.SaveReviewAnswer(ctx, p, value, in); err != nil {
		return command.Result{}, err
	}
	if in.ChoiceID != "" && value.ProposalID != "" {
		proposal, err := s.repository.ReviewProposal(ctx, p, value.ProposalID)
		if err != nil {
			return command.Result{}, err
		}
		state := "applied"
		if in.ChoiceID == "keep" {
			state = "rejected"
		}
		if err = s.finishProposal(ctx, p, proposal, state); err != nil {
			return command.Result{}, err
		}
	}
	return command.Result{ResourceType: "clarification", ResourceID: value.ID, Revision: value.Revision}, nil
}

func (s *ReviewService) finishProposal(ctx context.Context, p household.Principal, proposal ai.ReviewProposal, state string) error {
	if proposal.Revision >= command.MaxRevision {
		return commands.Rejection{Code: "version_conflict"}
	}
	proposal.Revision++
	proposal.State = state
	if err := s.repository.SaveReviewProposal(ctx, p, proposal); err != nil {
		return err
	}
	questions, err := s.repository.ProposalClarifications(ctx, p, proposal.ID)
	if err != nil {
		return err
	}
	for _, q := range questions {
		if q.Revision >= command.MaxRevision {
			return commands.Rejection{Code: "version_conflict"}
		}
		q.Revision++
		q.State = "answered"
		if err = s.repository.SaveClarification(ctx, p, q); err != nil {
			return err
		}
	}
	return nil
}

func (s *ReviewService) saveValidation(ctx context.Context, p household.Principal, projection ai.ReviewContext, result ai.ReviewValidation) error {
	if projection.ClarificationID != "" && result.State != "stale" {
		q, err := s.repository.Clarification(ctx, p, projection.ClarificationID)
		if err != nil {
			return err
		}
		if q.State != "checking" || q.Revision != projection.AnswerRevision || q.Revision >= command.MaxRevision {
			return commands.Rejection{Code: "version_conflict"}
		}
		q.Revision++
		q.State = "open"
		if result.State == "applied" || result.State == "clarification" {
			q.State = "answered"
		}
		if err = s.repository.SaveClarification(ctx, p, q); err != nil {
			return err
		}
		if q.State == "answered" && q.ProposalID != "" {
			proposal, err := s.repository.ReviewProposal(ctx, p, q.ProposalID)
			if err != nil {
				return err
			}
			state := "applied"
			if result.State == "clarification" {
				state = "superseded"
			}
			if err = s.finishProposal(ctx, p, proposal, state); err != nil {
				return err
			}
		}
	}
	return s.repository.SaveReviewValidation(ctx, p, result)
}
