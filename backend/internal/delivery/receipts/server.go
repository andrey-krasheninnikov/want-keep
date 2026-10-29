package receipts

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	ledgerdelivery "github.com/pchkauu/want-keep/backend/internal/delivery/ledger"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	access "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	application "github.com/pchkauu/want-keep/backend/internal/receipts/application"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

type Sessions interface {
	security.Sessions
	commands.SessionTransactions
}
type Reads interface {
	WithinFinancialRead(context.Context, household.Principal, func(context.Context) error) error
}
type Server struct {
	service   *application.Service
	mutations *commands.Authenticated
	commands  *commands.Queries
	sessions  Sessions
	reads     Reads
	guard     *security.Guard
	boundary  *contract.Boundary
	now       func() calendar.Instant
	mux       *http.ServeMux
}

func New(service *application.Service, executor *commands.Executor, queries *commands.Queries, sessions Sessions, reads Reads, config security.Config, now func() calendar.Instant) (*Server, error) {
	if service == nil || executor == nil || queries == nil || sessions == nil || reads == nil || now == nil {
		return nil, receipt.ErrInvalid
	}
	guard, err := security.New(config)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	s := &Server{service, commands.NewAuthenticated(executor, sessions), queries, sessions, reads, guard, boundary, now, http.NewServeMux()}
	s.mux.HandleFunc("POST /api/v1/receipts", s.create)
	s.mux.HandleFunc("GET /api/v1/receipts/{receiptId}", s.read)
	s.mux.HandleFunc("POST /api/v1/receipts/{receiptId}/answers", s.answer)
	s.mux.HandleFunc("POST /api/v1/receipts/{receiptId}/allocations", s.allocate)
	return s, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1024*1024)
	s.mux.ServeHTTP(w, r)
}
func (s *Server) decode(r *http.Request, schema string, out any) error {
	media, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || media != "application/json" {
		return contract.ErrInvalidRequest
	}
	data, err := io.ReadAll(r.Body)
	if err != nil {
		return contract.ErrInvalidRequest
	}
	return s.boundary.Decode(schema, data, out)
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
	problem := (contract.ErrorConverter{}).ToResponse(err, uuid.NewString())
	switch {
	case errors.Is(err, receipt.ErrNotFound):
		problem.Status = 404
		problem.Body.Code = "not_found"
		problem.Body.Message = "Receipt not found."
	case errors.Is(err, receipt.ErrInvalid):
		problem.Status = 400
		problem.Body.Code = "invalid_request"
	case errors.Is(err, access.ErrUnauthorized):
		problem.Status = 401
		problem.Body.Code = "unauthorized"
	}
	s.write(w, problem.Status, problem.Body)
}
func (s *Server) execute(w http.ResponseWriter, r *http.Request, a identity.Access, kind, id string, input any, apply func(context.Context) (command.Result, error)) {
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	data, err := json.Marshal(struct {
		ID    string
		Input any
	}{id, input})
	if err != nil {
		s.problem(w, err)
		return
	}
	hash := sha256.Sum256(data)
	c, err := s.mutations.Execute(r.Context(), a, commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: kind, PayloadHash: hex.EncodeToString(hash[:])}, apply)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		var e error
		c, e = s.commands.Read(ctx, a.Principal, c.ID(), s.now())
		if errors.Is(e, command.ErrCommandExpired) {
			return nil
		}
		return e
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	if errors.Is(c.RequireDetail(a.Principal, s.now()), command.ErrCommandExpired) {
		x := c.Snapshot()
		out, err := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &command.Outcome{CommandID: x.ID, Status: x.Status, Result: x.Result, FailureCode: x.ErrorCode})
		if err != nil {
			s.problem(w, err)
			return
		}
		s.write(w, 410, out)
		return
	}
	out, err := s.boundary.CommandToDTO(c, uuid.NewString())
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 202, out)
}
func (s *Server) create(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.ReceiptCreate
	if err = s.decode(r, "ReceiptCreate", &in); err != nil {
		s.problem(w, err)
		return
	}
	s.execute(w, r, a, "receipts.create", "", in, func(ctx context.Context) (command.Result, error) {
		return s.service.Start(ctx, a.Principal, in.AttachmentId, in.AccountId)
	})
}
func receiptID(r *http.Request) (string, error) {
	id, err := uuid.Parse(r.PathValue("receiptId"))
	if err != nil || id.String() != r.PathValue("receiptId") {
		return "", receipt.ErrNotFound
	}
	return id.String(), nil
}
func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := receiptID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var out generated.Receipt
	err = s.reads.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		value, err := s.service.Read(ctx, a.Principal, id)
		if err != nil {
			return err
		}
		out, err = s.toDTO(value)
		return err
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func (s *Server) answer(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := receiptID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.ReceiptAnswer
	if err = s.decode(r, "ReceiptAnswer", &in); err != nil {
		s.problem(w, err)
		return
	}
	answer := receipt.Answer{ExpectedRevision: uint64(in.ExpectedRevision)}
	if in.Distinct != nil {
		answer.Distinct = bool(*in.Distinct)
	}
	if in.Match != nil {
		answer.Match = &receipt.Candidate{ID: in.Match.Id, Revision: uint64(in.Match.Revision)}
	}
	if in.Purchase != nil {
		answer.Purchase = &receipt.Candidate{ID: in.Purchase.Id, Revision: uint64(in.Purchase.Revision)}
	}
	if in.RefundItems != nil {
		for _, item := range *in.RefundItems {
			answer.RefundItems = append(answer.RefundItems, receipt.RefundItem{ID: item.ItemId, Amount: item.Amount})
		}
	}
	if in.Facts != nil {
		e := factsFromDTO(*in.Facts)
		answer.Facts = &e
	}
	s.execute(w, r, a, "receipts.answer", id, in, func(ctx context.Context) (command.Result, error) {
		return s.service.Answer(ctx, a.Principal, id, answer)
	})
}

func (s *Server) allocate(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := receiptID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var in generated.ReceiptAllocationChange
	if err = s.decode(r, "ReceiptAllocationChange", &in); err != nil {
		s.problem(w, err)
		return
	}
	allocation, err := ledgerdelivery.AllocationFromDTO(in.Allocation)
	if err != nil {
		s.problem(w, err)
		return
	}
	items := []ledger.ItemAllocationInput{}
	if in.Items != nil {
		for _, item := range *in.Items {
			value, err := ledgerdelivery.AllocationFromDTO(item.Allocation)
			if err != nil {
				s.problem(w, err)
				return
			}
			items = append(items, ledger.ItemAllocationInput{ItemID: item.ItemId, Allocation: value})
		}
	}
	s.execute(w, r, a, "receipts.allocate", id, in, func(ctx context.Context) (command.Result, error) {
		return s.service.Allocate(ctx, a.Principal, id, uint64(in.ExpectedRevision), uint64(in.SubjectExpectedRevision), allocation, items, in.Reason)
	})
}
