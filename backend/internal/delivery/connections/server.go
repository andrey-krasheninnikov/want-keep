package connections

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	application "github.com/pchkauu/want-keep/backend/internal/connections/application"
	domain "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/security"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	identitydomain "github.com/pchkauu/want-keep/backend/internal/identity/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Sessions interface {
	security.Sessions
	commands.SessionTransactions
}
type Server struct {
	service  *application.Service
	sessions Sessions
	queries  *commands.Queries
	guard    *security.Guard
	boundary *contract.Boundary
	now      func() calendar.Instant
	mux      *http.ServeMux
}

func New(service *application.Service, sessions Sessions, queries *commands.Queries, config security.Config, now func() calendar.Instant) (*Server, error) {
	guard, err := security.New(config)
	if err != nil {
		return nil, err
	}
	boundary, err := contract.NewBoundary()
	if err != nil {
		return nil, err
	}
	if service == nil || service.Repository == nil || sessions == nil || queries == nil || now == nil {
		return nil, domain.ErrInvalidConnection
	}
	s := &Server{service: service, sessions: sessions, queries: queries, guard: guard, boundary: boundary, now: now, mux: http.NewServeMux()}
	s.mux.HandleFunc("GET /api/v1/connections", s.list)
	s.mux.HandleFunc("GET /api/v1/connections/{connectionId}", s.read)
	for _, route := range []string{"POST /api/v1/connections", "DELETE /api/v1/connections/{connectionId}", "POST /api/v1/connections/{connectionId}/sync", "POST /api/v1/connections/{connectionId}/reauth"} {
		s.mux.HandleFunc(route, s.mutate)
	}
	s.mux.HandleFunc("POST /api/v1/connections/{connectionId}/authorization", s.authorize)
	s.mux.HandleFunc("GET /api/v1/connections/raiffeisen/callback", s.callback)
	return s, nil
}
func (s *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if err := s.guard.Check(w, r); err != nil {
		s.problem(w, err)
		return
	}
	r.Body = http.MaxBytesReader(w, r.Body, 64*1024)
	s.mux.ServeHTTP(w, r)
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
	out := (contract.ErrorConverter{}).ToResponse(err, uuid.NewString())
	var rejection commands.Rejection
	if errors.As(err, &rejection) {
		switch rejection.Code {
		case "version_conflict", "provider_not_admitted":
			out.Status = 409
		case "not_found":
			out.Status = 404
		case "feature_unavailable", "invalid_request":
			out.Status = 422
		}
		out.Body.Code = generated.ErrorCode(rejection.Code)
		out.Body.Message = "Connection request was rejected; review its state and permissions."
		if rejection.CurrentRevision != 0 {
			revision := int64(rejection.CurrentRevision)
			out.Body.CurrentRevision = &revision
		}
	}
	switch {
	case errors.Is(err, domain.ErrConnectionNotFound):
		out.Status, out.Body.Code, out.Body.Message = 404, "not_found", "Connection not found."
	case errors.Is(err, domain.ErrConnectionVersion):
		out.Status, out.Body.Code, out.Body.Message = 409, "version_conflict", "Connection changed; refresh its current version."
	case errors.Is(err, domain.ErrOAuthUnavailable):
		out.Status, out.Body.Code, out.Body.Message = 503, "service_unavailable", "Bank authorization configuration is not ready."
	case errors.Is(err, domain.ErrOAuthUnknown):
		out.Status, out.Body.Code, out.Body.Message = 409, "source_reauth_required", "Bank authorization outcome is unresolved. Request a new authorization before retrying."
	case errors.Is(err, domain.ErrOAuthAttempt), errors.Is(err, domain.ErrInvalidConnection):
		out.Status, out.Body.Code, out.Body.Message = 400, "invalid_request", "Check connection and authorization details."
	case errors.Is(err, domain.ErrSecretAccess):
		out.Status, out.Body.Code, out.Body.Message = 409, "source_reauth_required", "The external account owner must authorize the connection."
	case errors.Is(err, identitydomain.ErrUnauthorized):
		out.Status, out.Body.Code, out.Body.Message = 401, "unauthorized", "Sign in to continue."
	}
	s.write(w, out.Status, out.Body)
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
func validID(id string) bool {
	v, err := uuid.Parse(id)
	return err == nil && v.Version() == 4 && v.String() == id
}
func (s *Server) toDTO(c domain.ConnectionRecord) (generated.Connection, error) {
	gate, err := s.boundary.AdmissionToDTO(c.Admission)
	if err != nil {
		return generated.Connection{}, err
	}
	state := reporting.CoverageState(c.Coverage)
	gaps := c.Gaps
	if state == "" {
		state = reporting.NoCoverage
	}
	if state != reporting.Complete && len(gaps) == 0 {
		gaps = []string{"history_not_loaded"}
	}
	coverage, err := reporting.NewCoverage(state, gaps)
	if err != nil {
		return generated.Connection{}, err
	}
	mapped, err := s.boundary.CoverageToDTO(coverage)
	if err != nil {
		return generated.Connection{}, err
	}
	quality := generated.DataQuality{Coverage: mapped, Freshness: "unknown"}
	if !c.LastSuccess.IsZero() {
		at := c.LastSuccess.UTC().Format(time.RFC3339Nano)
		quality.FetchedAt = &at
		quality.Freshness = "stale"
		if s.now().Time().Sub(c.LastSuccess) < time.Hour {
			quality.Freshness = "fresh"
		}
	}
	var from *string
	if c.HistoryFrom.String() != "" {
		value := c.HistoryFrom.String()
		from = &value
	}
	products := []string{}
	if c.Provider == "raiffeisen" {
		products = []string{"current"}
	}
	return generated.Connection{Id: c.ID, Revision: int64(c.Revision), Provider: generated.Provider(c.Provider), ExternalAccountOwnerId: string(c.OwnerID), HistoryFrom: from, Generation: int64(c.Generation), State: generated.ConnectionState(c.State), Products: products, DeploymentGate: gate, Quality: quality}, nil
}
func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id := r.PathValue("connectionId")
	if !validID(id) {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	var out generated.Connection
	err = s.service.Repository.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		c, err := s.service.Read(ctx, a.Principal, id)
		if err != nil {
			return err
		}
		out, err = s.toDTO(c)
		return err
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func cursorFor(a identity.Access, id string) string {
	payload := base64.RawURLEncoding.EncodeToString([]byte(id))
	mac := hmac.New(sha256.New, []byte(a.Token))
	_, _ = mac.Write([]byte("connections/" + string(a.Principal.HouseholdID()) + "/" + payload))
	return payload + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	limit := 50
	values := r.URL.Query()
	for key, v := range values {
		if len(v) != 1 || (key != "limit" && key != "cursor") {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if v := values.Get("limit"); v != "" {
		limit, err = strconv.Atoi(v)
		if err != nil || limit < 1 || limit > 100 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	after := ""
	if v := values.Get("cursor"); v != "" {
		payload, _, found := strings.Cut(v, ".")
		raw, err := base64.RawURLEncoding.DecodeString(payload)
		after = string(raw)
		if !found || err != nil || !validID(after) || !hmac.Equal([]byte(v), []byte(cursorFor(a, after))) {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	out := generated.ConnectionPage{Items: []generated.Connection{}}
	err = s.service.Repository.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		items, err := s.service.List(ctx, a.Principal, after, limit+1)
		if err != nil {
			return err
		}
		if len(items) > limit {
			cursor := cursorFor(a, items[limit-1].ID)
			out.NextCursor = &cursor
			items = items[:limit]
		}
		for _, c := range items {
			dto, err := s.toDTO(c)
			if err != nil {
				return err
			}
			out.Items = append(out.Items, dto)
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		out.Quality.Coverage, err = s.boundary.CoverageToDTO(coverage)
		out.Quality.Freshness = "unknown"
		return err
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, out)
}
func (s *Server) mutate(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	action, id := "create", r.PathValue("connectionId")
	revision := uint64(0)
	in := application.CreateInput{}
	var payload any
	if id == "" {
		var dto generated.ConnectionCreate
		if err = s.decode(r, "ConnectionCreate", &dto); err != nil {
			s.problem(w, err)
			return
		}
		date, err := calendar.ParseDate(dto.HistoryFrom)
		if err != nil {
			s.problem(w, err)
			return
		}
		in = application.CreateInput{Provider: string(dto.Provider), OwnerID: household.UserID(dto.ExternalAccountOwnerId), HistoryFrom: date, Products: dto.Products}
		payload = dto
	} else {
		if !validID(id) {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		var dto generated.ConnectionAction
		if err = s.decode(r, "ConnectionAction", &dto); err != nil {
			s.problem(w, err)
			return
		}
		revision = uint64(dto.ExpectedRevision)
		payload = dto
		action = "disconnect"
		if r.Method != "DELETE" {
			action = r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
		}
	}
	if len(r.Header.Values("Idempotency-Key")) != 1 {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	body, _ := json.Marshal(struct {
		ID    string
		Input any
	}{id, payload})
	hash := sha256.Sum256(body)
	request := commands.Request{ID: r.Header.Get("Idempotency-Key"), Kind: "connections." + action, PayloadHash: hex.EncodeToString(hash[:])}
	c, err := s.service.Execute(r.Context(), a, request, action, id, revision, in)
	if err != nil {
		s.problem(w, err)
		return
	}
	err = s.service.Repository.WithinFinancialRead(r.Context(), a.Principal, func(ctx context.Context) error {
		var err error
		c, err = s.queries.Read(ctx, a.Principal, c.ID(), s.now())
		if errors.Is(err, command.ErrCommandExpired) {
			return nil
		}
		return err
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	if errors.Is(c.RequireDetail(a.Principal, s.now()), command.ErrCommandExpired) {
		snapshot := c.Snapshot()
		out, err := s.boundary.ExpiredCommandToDTO(uuid.NewString(), &command.Outcome{CommandID: c.ID(), Status: snapshot.Status, Result: snapshot.Result, FailureCode: snapshot.ErrorCode, CurrentRevision: snapshot.CurrentRevision})
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
func (s *Server) authorize(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, true)
	if err != nil {
		s.problem(w, err)
		return
	}
	id := r.PathValue("connectionId")
	if !validID(id) {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	var dto generated.ConnectionAction
	if err = s.decode(r, "ConnectionAction", &dto); err != nil {
		s.problem(w, err)
		return
	}
	out, err := s.service.Begin(r.Context(), a, id, uint64(dto.ExpectedRevision))
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, 200, generated.AuthorizationAttempt{Id: out.ID, ConnectionId: id, ExpiresAt: out.ExpiresAt.UTC().Format(time.RFC3339Nano), State: "ready", AuthorizationUrl: &out.URL})
}
func (s *Server) callback(w http.ResponseWriter, r *http.Request) {
	a, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	q := r.URL.Query()
	for key, v := range q {
		if len(v) != 1 || (key != "state" && key != "code" && key != "error") {
			s.problem(w, domain.ErrOAuthAttempt)
			return
		}
	}
	id, err := s.service.Callback(r.Context(), a, q.Get("state"), q.Get("code"), q.Get("error") != "")
	if err != nil {
		s.problem(w, err)
		return
	}
	http.Redirect(w, r, "/connections/"+id, http.StatusSeeOther)
}
