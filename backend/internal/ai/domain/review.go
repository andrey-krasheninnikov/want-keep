package domain

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"slices"
	"unicode/utf8"
)

const ReviewContractVersion = "transaction_review_v1"

var ErrReviewCommand = errors.New("invalid transaction review command")

type Reference struct {
	Kind     string `json:"kind"`
	ID       string `json:"id"`
	Revision uint64 `json:"revision"`
	Label    string `json:"label"`
}

type ReviewContext struct {
	JobID           string               `json:"jobId"`
	OperationID     string               `json:"operationId"`
	Revision        uint64               `json:"revision"`
	References      map[string]Reference `json:"references"`
	Input           json.RawMessage      `json:"input"`
	ClarificationID string               `json:"clarificationId,omitempty"`
	AnswerRevision  uint64               `json:"answerRevision,omitempty"`
}

type ReviewCommand struct {
	Kind         string   `json:"kind"`
	Category     *string  `json:"category"`
	Merchant     *string  `json:"merchant"`
	Distribution *string  `json:"distribution"`
	Member       *string  `json:"member"`
	Candidate    *string  `json:"candidate"`
	Question     *string  `json:"question"`
	Evidence     []string `json:"evidence"`
	Reason       string   `json:"reason"`
}

type ReviewOutput struct {
	Version  string          `json:"version"`
	CaseID   string          `json:"caseId"`
	Commands []ReviewCommand `json:"commands"`
}

// DecodeReviewOutput accepts only bounded commands. Monetary facts cannot be assigned.
func DecodeReviewOutput(data []byte) (ReviewOutput, error) {
	var out ReviewOutput
	if len(data) == 0 || len(data) > 65536 {
		return out, ErrReviewCommand
	}
	var raw struct {
		Commands []map[string]json.RawMessage `json:"commands"`
	}
	if json.Unmarshal(data, &raw) != nil {
		return out, ErrReviewCommand
	}
	for _, c := range raw.Commands {
		for _, field := range []string{"kind", "category", "merchant", "distribution", "member", "candidate", "question", "evidence", "reason"} {
			if _, ok := c[field]; !ok {
				return out, ErrReviewCommand
			}
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&out) != nil || decoder.Decode(new(any)) != io.EOF || out.Version != ReviewContractVersion || out.CaseID != "case-1" || len(out.Commands) < 1 || len(out.Commands) > 8 {
		return ReviewOutput{}, ErrReviewCommand
	}
	for _, command := range out.Commands {
		if !slices.Contains([]string{"classification", "distribution", "link", "clarify", "no_change", "reject", "budget", "goal"}, command.Kind) || utf8.RuneCountInString(command.Reason) < 1 || utf8.RuneCountInString(command.Reason) > 2000 || len(command.Evidence) < 1 || len(command.Evidence) > 20 {
			return ReviewOutput{}, ErrReviewCommand
		}
		if !slices.Contains(command.Evidence, "ledger_revision") {
			return ReviewOutput{}, ErrReviewCommand
		}
		for _, ref := range command.Evidence {
			if len(ref) > 100 || ref == "" {
				return ReviewOutput{}, ErrReviewCommand
			}
		}
		switch command.Kind {
		case "classification":
			if command.Category == nil && command.Merchant == nil || command.Distribution != nil || command.Member != nil || command.Candidate != nil || command.Question != nil {
				return ReviewOutput{}, ErrReviewCommand
			}
		case "distribution":
			if command.Distribution == nil || !slices.Contains([]string{"rule", "personal", "joint"}, *command.Distribution) || command.Category != nil || command.Merchant != nil || command.Candidate != nil || command.Question != nil || (*command.Distribution == "personal") != (command.Member != nil) {
				return ReviewOutput{}, ErrReviewCommand
			}
		case "link":
			if command.Candidate == nil || command.Category != nil || command.Merchant != nil || command.Distribution != nil || command.Member != nil || command.Question != nil {
				return ReviewOutput{}, ErrReviewCommand
			}
		case "clarify":
			if command.Question == nil || utf8.RuneCountInString(*command.Question) < 1 || utf8.RuneCountInString(*command.Question) > 2000 || command.Category != nil || command.Merchant != nil || command.Distribution != nil || command.Member != nil || command.Candidate != nil {
				return ReviewOutput{}, ErrReviewCommand
			}
		default:
			if command.Category != nil || command.Merchant != nil || command.Distribution != nil || command.Member != nil || command.Candidate != nil || command.Question != nil {
				return ReviewOutput{}, ErrReviewCommand
			}
		}
	}
	return out, nil
}

func (c ReviewContext) Resolve(alias, kind string) (Reference, error) {
	ref, ok := c.References[alias]
	if !ok || ref.Kind != kind || ref.ID == "" || ref.Revision == 0 {
		return Reference{}, ErrReviewCommand
	}
	return ref, nil
}

type ReviewValidation struct {
	AttemptID, OperationID      string
	Revision                    uint64
	State, Code, Rationale      string
	ProposalID, ClarificationID string
}

type ReviewProposal struct {
	ID                string          `json:"id"`
	Revision          uint64          `json:"revision"`
	OperationID       string          `json:"operationId"`
	OperationRevision uint64          `json:"operationRevision"`
	State             string          `json:"state"`
	Commands          []ReviewCommand `json:"commands"`
	Context           ReviewContext   `json:"-"`
	Rationale         string          `json:"rationale"`
}

type Clarification struct {
	ID                string        `json:"id"`
	Revision          uint64        `json:"revision"`
	OperationID       string        `json:"operationId"`
	OperationRevision uint64        `json:"operationRevision"`
	State             string        `json:"state"`
	Question          string        `json:"question"`
	Choices           []Choice      `json:"choices"`
	ProposalID        string        `json:"proposalId,omitempty"`
	Context           ReviewContext `json:"-"`
}

type Choice struct {
	ID       string          `json:"id"`
	Label    string          `json:"label"`
	Commands []ReviewCommand `json:"-"`
}

type ReviewAnswer struct {
	ClarificationID         string
	ExpectedRevision        uint64
	SubjectExpectedRevision uint64
	ChoiceID, Text          string
}
