package ledger

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/application"
)

type Sessions interface {
	security.Sessions
	commands.SessionTransactions
}
type ReadTransactions interface {
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
}
type Server struct {
	matching  *matching.Service
	refunds   *expenses.Service
	service   *application.Service
	queries   *application.Queries
	mutations *commands.Authenticated
	commands  *commands.Queries
	sessions  Sessions
	reads     ReadTransactions
	guard     *security.Guard
	boundary  *contract.Boundary
	now       func() calendar.Instant
	mux       *http.ServeMux
}

func New(s *application.Service, m *matching.Service, q *application.Queries, e *commands.Executor, cq *commands.Queries, sessions Sessions, reads ReadTransactions, c security.Config, now func() calendar.Instant) (*Server, error) {
	if s == nil || m == nil || q == nil || e == nil || cq == nil || sessions == nil || reads == nil || now == nil {
		return nil, ledger.ErrInvalidRevision
	}
	g, err := security.New(c)
	if err != nil {
		return nil, err
	}
	b, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	h := &Server{service: s, matching: m, queries: q, mutations: commands.NewAuthenticated(e, sessions), commands: cq, sessions: sessions, reads: reads, guard: g, boundary: b, now: now, mux: http.NewServeMux()}
	h.mux.HandleFunc("GET /api/v1/transactions", h.list)
	h.mux.HandleFunc("POST /api/v1/transactions", h.create)
	h.mux.HandleFunc("GET /api/v1/transactions/{transactionId}", h.read)
	h.mux.HandleFunc("POST /api/v1/transfers", h.transfer)
	h.mux.HandleFunc("POST /api/v1/refunds", h.refund)
	h.mux.HandleFunc("POST /api/v1/transactions/{transactionId}/corrections", h.correct)
	h.mux.HandleFunc("POST /api/v1/transactions/{transactionId}/undo", h.undo)
	h.mux.HandleFunc("POST /api/v1/transactions/{transactionId}/exclude", h.exclude)
	h.mux.HandleFunc("GET /api/v1/transactions/{transactionId}/history", h.history)
	h.mux.HandleFunc("GET /api/v1/transactions/{transactionId}/refund-history", h.refundHistory)
	h.mux.HandleFunc("GET /api/v1/transactions/{transactionId}/revisions/{revision}", h.revision)
	h.mux.HandleFunc("POST /api/v1/transactions/{transactionId}/links", h.link)
	h.mux.HandleFunc("POST /api/v1/transactions/{transactionId}/allocations", h.allocate)
	h.mux.HandleFunc("GET /api/v1/matching", h.matchingList)
	h.mux.HandleFunc("GET /api/v1/matching/{matchingId}", h.matchingRead)
	h.mux.HandleFunc("POST /api/v1/matching/{matchingId}/resolve", h.matchingResolve)
	return h, nil
}

func NewWithRefunds(s *application.Service, m *matching.Service, refunds *expenses.Service, q *application.Queries, e *commands.Executor, cq *commands.Queries, sessions Sessions, reads ReadTransactions, c security.Config, now func() calendar.Instant) (*Server, error) {
	if refunds == nil {
		return nil, ledger.ErrInvalidRevision
	}
	h, err := New(s, m, q, e, cq, sessions, reads, c, now)
	if err != nil {
		return nil, err
	}
	h.refunds = refunds
	return h, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 128*1024)
	s.mux.ServeHTTP(w, r)
}
func (s *Server) decode(r *http.Request, name string, out any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return contract.ErrInvalidRequest
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return contract.ErrInvalidRequest
	}
	return s.boundary.Decode(name, data, out)
}
func (s *Server) write(w http.ResponseWriter, status int, value any) {
	data, err := json.Marshal(value)
	if err != nil {
		s.problem(w, err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(data)
}
func (s *Server) problem(w http.ResponseWriter, err error) {
	r := (contract.ErrorConverter{}).ToResponse(err, uuid.NewString())
	switch {
	case errors.Is(err, ledger.ErrMatchingConflict):
		r.Status, r.Body.Code, r.Body.Message = 409, "matching_conflict", "Review the current participants and amounts before linking."
	case errors.Is(err, ledger.ErrNotFound):
		r.Status, r.Body.Code, r.Body.Message = 404, "not_found", "Transaction not found."
	case errors.Is(err, ledger.ErrFeatureUnavailable):
		r.Status, r.Body.Code, r.Body.Message = 422, "feature_unavailable", "This feature is not available yet. No change was applied."
	case errors.Is(err, ledger.ErrInvalidRevision):
		r.Status, r.Body.Code, r.Body.Message = 422, "invalid_transaction", "Check the transaction amounts, accounts and economic meaning."
	case errors.Is(err, ledger.ErrInvalidTransition):
		r.Status, r.Body.Code, r.Body.Message = 409, "invalid_transition", "This transaction cannot enter the requested state."
	case errors.Is(err, identity.ErrUnauthorized):
		r.Status, r.Body.Code, r.Body.Message = 401, "unauthorized", "Sign in to continue."
	case errors.Is(err, identity.ErrAttempt):
		r.Status, r.Body.Code, r.Body.Message = 400, "invalid_request", "Check the request fields."
	}
	s.write(w, r.Status, r.Body)
}
func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id := r.PathValue("transactionId")
	parsed, err := uuid.Parse(id)
	if err != nil || parsed.Version() != 4 || parsed.String() != id {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	var out any
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		view, e := s.queries.Read(ctx, a.Principal, id)
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
