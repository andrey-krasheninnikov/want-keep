package ledger

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"net/http"
	"strconv"
	"strings"

	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
)

type historyPage struct {
	access        identity.Access
	transactionID string
	purpose       string
}

func (p historyPage) cursor(revision uint64) string {
	payload := strconv.FormatUint(revision, 10)
	purpose := p.purpose
	if purpose == "" {
		purpose = "ledger-history"
	}
	h := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = h.Write([]byte(purpose + "/" + string(p.access.Principal.HouseholdID()) + "/" + string(p.access.Principal.UserID()) + "/" + p.transactionID + "/" + payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(h.Sum(nil))
}

func (p historyPage) input(r *http.Request) (int, uint64, error) {
	limit := 50
	var before uint64
	for key, values := range r.URL.Query() {
		if len(values) != 1 || key != "limit" && key != "cursor" {
			return 0, 0, contract.ErrInvalidRequest
		}
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil || parsed < 1 || parsed > 100 {
			return 0, 0, contract.ErrInvalidRequest
		}
		limit = parsed
	}
	if value := r.URL.Query().Get("cursor"); value != "" {
		raw, _, ok := strings.Cut(value, ".")
		parsed, err := strconv.ParseUint(raw, 10, 64)
		if !ok || err != nil || parsed < 1 || parsed > 9007199254740991 || !hmac.Equal([]byte(value), []byte(p.cursor(parsed))) {
			return 0, 0, contract.ErrInvalidRequest
		}
		before = parsed
	}
	return limit, before, nil
}
func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	p := historyPage{access: a, transactionID: id}
	limit, before, err := p.input(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.TransactionHistoryPage{Items: []generated.TransactionHistoryEntry{}}
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		entries, next, e := s.queries.History(ctx, a.Principal, id, before, limit)
		if e != nil {
			return e
		}
		for _, entry := range entries {
			v, e := s.historyDTO(a, entry)
			if e != nil {
				return e
			}
			out.Items = append(out.Items, v)
		}
		if next > 0 {
			cursor := p.cursor(next)
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

func (s *Server) refundHistory(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	p := historyPage{access: a, transactionID: id, purpose: "refund-history"}
	limit, before, err := p.input(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.RefundHistoryPage{Items: []generated.RefundHistoryEntry{}}
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		values, next, readErr := s.queries.RefundHistory(ctx, a.Principal, id, before, limit)
		if readErr != nil {
			return readErr
		}
		for _, value := range values {
			refund, conversionErr := s.refundDTO(value)
			if conversionErr != nil {
				return conversionErr
			}
			out.Items = append(out.Items, generated.RefundHistoryEntry{Refund: refund, ActorId: string(value.ActorID), RecordedAt: value.RecordedAt.String()})
		}
		if next > 0 {
			cursor := p.cursor(next)
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
func (s *Server) revision(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := s.transactionID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	v, err := strconv.ParseUint(r.PathValue("revision"), 10, 64)
	if err != nil || v < 1 || v > 9007199254740991 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	var out generated.Transaction
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		view, e := s.queries.Revision(ctx, a.Principal, id, v)
		if e != nil {
			return e
		}
		out, e = s.transactionDTO(a.Principal, view)
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func (s *Server) historyDTO(a identity.Access, e application.HistoryEntry) (generated.TransactionHistoryEntry, error) {
	r := e.Current.Revision
	transaction, err := s.transactionDTO(a.Principal, e.Current)
	if err != nil {
		return generated.TransactionHistoryEntry{}, err
	}
	out := generated.TransactionHistoryEntry{Transaction: transaction, Reason: r.Reason, Fields: []generated.LedgerField{}, Affected: []generated.DecisionRevision{}, UndoAvailable: e.UndoReason == "available", UndoReason: generated.TransactionHistoryEntryUndoReason(e.UndoReason)}
	out.Evidence = []generated.DecisionEvidence{}
	if r.RecordedAt.String() != "" {
		at := r.RecordedAt.String()
		out.RecordedAt = &at
	}
	if e.Before != nil {
		v, err := s.transactionDTO(a.Principal, *e.Before)
		if err != nil {
			return out, err
		}
		out.Before = &v
	}
	if e.Decision != nil {
		d := e.Decision
		for _, ref := range d.Evidence {
			out.Evidence = append(out.Evidence, generated.DecisionEvidence{Kind: generated.DecisionEvidenceKind(ref.Kind), Id: ref.ID, Revision: int64(ref.Revision)})
		}
		out.DecisionId = &d.ID
		kind := generated.TransactionHistoryEntryDecisionKind(d.Kind)
		out.DecisionKind = &kind
		if d.UndoOf != "" {
			out.UndoOf = &d.UndoOf
		}
		for _, entry := range d.Entries {
			if entry.OperationID == r.OperationID {
				for _, f := range entry.Fields {
					out.Fields = append(out.Fields, generated.LedgerField(f))
				}
			}
		}
	}
	for _, v := range e.Affected {
		out.Affected = append(out.Affected, generated.DecisionRevision{TransactionId: v.OperationID, ExpectedRevision: int64(v.Revision)})
	}
	return out, nil
}
