package review

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func reviewCursor(access identity.Access, id string) string {
	raw, _ := json.Marshal([]string{string(access.Principal.HouseholdID()), string(access.Principal.UserID()), id})
	encoded := base64.RawURLEncoding.EncodeToString(raw)
	hash := hmac.New(sha256.New, []byte(access.Token))
	_, _ = hash.Write([]byte("clarifications/" + encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(hash.Sum(nil))
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	values := r.URL.Query()
	limit := 50
	after := ""
	for k, v := range values {
		if len(v) != 1 || (k != "cursor" && k != "limit") {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if value := values.Get("limit"); value != "" {
		limit, err = strconv.Atoi(value)
		if err != nil || limit < 1 || limit > 100 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if cursor := values.Get("cursor"); cursor != "" {
		encoded, _, ok := strings.Cut(cursor, ".")
		raw, e := base64.RawURLEncoding.DecodeString(encoded)
		var scope []string
		if !ok || e != nil || len(cursor) > 2048 || json.Unmarshal(raw, &scope) != nil || len(scope) != 3 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		after = scope[2]
		id, e := uuid.Parse(after)
		if e != nil || id.Version() != 4 || id.String() != after || !hmac.Equal([]byte(cursor), []byte(reviewCursor(access, after))) {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	quality, err := s.boundary.CoverageToDTO(coverage)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.ClarificationPage{Items: []generated.Clarification{}, Quality: generated.DataQuality{Coverage: quality, Freshness: "fresh"}}
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		items, next, err := s.service.Clarifications(ctx, access.Principal, after, limit)
		if err != nil {
			return err
		}
		for _, c := range items {
			out.Items = append(out.Items, clarificationDTO(c))
		}
		if next != "" {
			cursor := reviewCursor(access, next)
			out.NextCursor = &cursor
		}
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func clarificationDTO(c ai.Clarification) generated.Clarification {
	out := generated.Clarification{Id: c.ID, Revision: generated.Revision(c.Revision), OperationId: c.OperationID, OperationRevision: generated.Revision(c.OperationRevision), State: generated.ClarificationState(c.State), Question: c.Question, Choices: []generated.ClarificationChoice{}}
	if c.ProposalID != "" {
		out.ProposalId = &c.ProposalID
	}
	for _, choice := range c.Choices {
		out.Choices = append(out.Choices, generated.ClarificationChoice{Id: choice.ID, Label: choice.Label})
	}
	return out
}
func (s *Server) proposal(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := resourceID(r, "proposalId")
	if err != nil {
		s.problem(w, err)
		return
	}
	var out generated.Proposal
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		proposal, err := s.service.Proposal(ctx, access.Principal, id)
		if err != nil {
			return err
		}
		out = generated.Proposal{Id: proposal.ID, Revision: generated.Revision(proposal.Revision), OperationId: proposal.OperationID, OperationRevision: generated.Revision(proposal.OperationRevision), State: generated.ProposalState(proposal.State), Summary: proposal.Rationale, Changes: []generated.ReviewChange{}}
		for _, c := range proposal.Commands {
			value := generated.ReviewChange{Kind: generated.ReviewChangeKind(c.Kind), Reason: c.Reason}
			if c.Category != nil {
				ref := proposal.Context.References[*c.Category]
				label := ref.Label
				value.Category = &label
			}
			if c.Merchant != nil {
				ref := proposal.Context.References[*c.Merchant]
				label := ref.Label
				value.Merchant = &label
			}
			if c.Distribution != nil {
				mode := generated.ReviewChangeDistribution(*c.Distribution)
				value.Distribution = &mode
			}
			out.Changes = append(out.Changes, value)
		}
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
