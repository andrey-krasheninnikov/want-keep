package ledger

import (
	"context"
	"net/http"

	"github.com/google/uuid"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	application "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Server) existingMembers(in []generated.ExistingTransaction) []matching.Member {
	out := make([]matching.Member, 0, len(in))
	for _, v := range in {
		out = append(out, matching.Member{OperationID: v.TransactionId, Revision: uint64(v.ExpectedRevision)})
	}
	return out
}
func (s *Server) matchingMembers(in []generated.DecisionRevision) []matching.Member {
	out := make([]matching.Member, 0, len(in))
	for _, v := range in {
		out = append(out, matching.Member{OperationID: v.TransactionId, Revision: uint64(v.ExpectedRevision)})
	}
	return out
}
func (s *Server) link(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.TransactionLink
	if err = s.decode(r, "TransactionLink", &in); err != nil {
		s.problem(w, err)
		return
	}
	if in.Kind == "refund" {
		if s.refunds == nil || in.Refund == nil || in.Refund.ExpectedRevision < 0 || len(in.ExpectedRevisions) != 2 || in.Refund.PurchaseId == id {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		revisions := map[string]uint64{}
		for _, value := range in.ExpectedRevisions {
			if _, found := revisions[value.TransactionId]; found {
				s.problem(w, contract.ErrInvalidRequest)
				return
			}
			revisions[value.TransactionId] = uint64(value.ExpectedRevision)
		}
		refundRevision, refundFound := revisions[id]
		purchaseRevision, purchaseFound := revisions[in.Refund.PurchaseId]
		if !refundFound || !purchaseFound {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		items, itemErr := s.refundItems(in.Refund.ReturnedItems)
		if itemErr != nil {
			s.problem(w, itemErr)
			return
		}
		input := expenses.LinkInput{RefundID: id, PurchaseID: in.Refund.PurchaseId, RefundExpectedRevision: refundRevision, PurchaseExpectedRevision: purchaseRevision, ExpectedRevision: uint64(in.Refund.ExpectedRevision), Items: items, Reason: in.Reason}
		payload := struct {
			TransactionID string
			Input         generated.TransactionLink
		}{id, in}
		s.execute(w, r, a, "transactions.links", payload, func(ctx context.Context) (command.Result, error) { return s.refunds.Link(ctx, a.Principal, input) })
		return
	}
	if in.Refund != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	kind := matching.Kind(in.Kind)
	if in.Kind == "receipt_match" {
		kind = matching.Payment
	}
	input := application.LinkInput{Kind: kind, PrimaryID: id, Members: s.matchingMembers(in.ExpectedRevisions), Reason: in.Reason}
	payload := struct {
		TransactionID string
		Input         generated.TransactionLink
	}{id, in}
	s.execute(w, r, a, "transactions.links", payload, func(ctx context.Context) (command.Result, error) { return s.matching.Link(ctx, a.Principal, input) })
}
func (s *Server) matchingID(r *http.Request) (string, error) {
	id := r.PathValue("matchingId")
	v, err := uuid.Parse(id)
	if err != nil || v.Version() != 4 || v.String() != id {
		return "", contract.ErrInvalidRequest
	}
	return id, nil
}
func (s *Server) matchingRead(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.matchingID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var out generated.MatchingCase
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		v, err := s.matching.Read(ctx, a.Principal, id)
		if err != nil {
			return err
		}
		out = s.matchingDTO(v)
		return nil
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func (s *Server) matchingResolve(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.matchingID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.MatchingResolution
	if err = s.decode(r, "MatchingResolution", &in); err != nil {
		s.problem(w, err)
		return
	}
	members := s.matchingMembers(in.ExpectedRevisions)
	payload := struct {
		MatchingID string
		Input      generated.MatchingResolution
	}{id, in}
	s.execute(w, r, a, "matching.resolve", payload, func(ctx context.Context) (command.Result, error) {
		if in.Decision == "separate" {
			return s.matching.Separate(ctx, a.Principal, id, uint64(in.ExpectedRevision), members, in.Reason)
		}
		if in.Kind == nil || in.PrimaryId == nil {
			return command.Result{}, contract.ErrInvalidRequest
		}
		return s.matching.Link(ctx, a.Principal, application.LinkInput{GroupID: id, ExpectedRevision: uint64(in.ExpectedRevision), Kind: matching.Kind(*in.Kind), PrimaryID: *in.PrimaryId, Members: members, Reason: in.Reason})
	})
}
func (s *Server) matchingDTO(v application.View) generated.MatchingCase {
	g := v.Group
	out := generated.MatchingCase{Id: g.ID, Revision: int64(g.Revision), PrimaryId: g.PrimaryID, Kind: generated.MatchingKind(g.Kind), State: generated.MatchingState(g.State), ActorId: string(g.ActorID), RecordedAt: g.At.String(), Reason: g.Reason, CandidatesComplete: g.CandidatesComplete, Members: []generated.MatchingMember{}, Candidates: []generated.MatchingCandidate{}}
	if g.DecisionID != "" {
		out.DecisionId = &g.DecisionID
	}
	for _, m := range g.Members {
		member := generated.MatchingMember{TransactionId: m.OperationID, Revision: int64(m.Revision), Evidence: []generated.DecisionEvidence{}}
		for _, e := range v.Evidence[m.OperationID] {
			member.Evidence = append(member.Evidence, generated.DecisionEvidence{Kind: generated.DecisionEvidenceKind(e.Kind), Id: e.ID, Revision: int64(e.Revision)})
		}
		out.Members = append(out.Members, member)
	}
	for _, c := range g.Candidates {
		out.Candidates = append(out.Candidates, generated.MatchingCandidate{TransactionId: c.OperationID, Revision: int64(c.Revision), Reason: c.Reason})
	}
	return out
}
