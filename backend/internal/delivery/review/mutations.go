package review

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	"net/http"
)

func (s *Server) execute(w http.ResponseWriter, r *http.Request, access identity.Access, kind, resource string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	payload, err := json.Marshal(struct {
		Resource string `json:"resource"`
		Input    any    `json:"input"`
	}{resource, input})
	if err != nil {
		s.problem(w, err)
		return
	}
	hash := sha256.Sum256(payload)
	registered, err := s.mutations.Execute(r.Context(), access, commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: kind, PayloadHash: hex.EncodeToString(hash[:])}, apply)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var readErr error
		registered, readErr = s.commands.Read(ctx, access.Principal, registered.ID(), s.now())
		if errors.Is(readErr, command.ErrCommandExpired) {
			return nil
		}
		return readErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.commandResponse(w, access.Principal, registered)
}

func (s *Server) answer(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := resourceID(r, "clarificationId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.ClarificationAnswer
	if err = s.decode(r, "ClarificationAnswer", &input); err != nil {
		s.problem(w, err)
		return
	}
	answer := ai.ReviewAnswer{ClarificationID: id, ExpectedRevision: uint64(input.ExpectedRevision), SubjectExpectedRevision: uint64(input.SubjectExpectedRevision)}
	if input.ChoiceId != nil {
		answer.ChoiceID = *input.ChoiceId
	}
	if input.Answer != nil {
		answer.Text = *input.Answer
	}
	s.execute(w, r, access, "clarifications.answer", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.Answer(ctx, access.Principal, answer)
	})
}
func (s *Server) decide(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := resourceID(r, "proposalId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var input generated.ProposalDecision
	if err = s.decode(r, "ProposalDecision", &input); err != nil {
		s.problem(w, err)
		return
	}
	s.execute(w, r, access, "proposals.decide", id, input, func(ctx context.Context) (command.Result, error) {
		return s.service.Apply(ctx, access.Principal, id, uint64(input.ExpectedRevision), uint64(input.SubjectExpectedRevision), string(input.Decision))
	})
}
