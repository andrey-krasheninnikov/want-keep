package reimbursements

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
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	identity "github.com/pchkauu/want-keep/backend/internal/identity/application"
	application "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type page struct {
	access  identity.Access
	filter  application.ReimbursementFilter
	after   application.ReimbursementCursor
	limit   int
	history string
	scope   string
}

func (p page) cursor(cursor application.ReimbursementCursor) string {
	payload, _ := json.Marshal([]any{cursor.At.String(), cursor.ID, cursor.Revision})
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	mac := hmac.New(sha256.New, []byte(p.access.Token))
	_, _ = mac.Write([]byte("reimbursements/" + string(p.access.Principal.HouseholdID()) + "/" + string(p.access.Principal.UserID()) + "/" + p.history + "/" + p.scope + "/" + encoded))
	return encoded + "." + base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func (s *Server) page(r *http.Request, historyID string) (page, error) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		return page{}, err
	}
	result := page{access: access, limit: 50, history: historyID}
	values := r.URL.Query()
	for key, entries := range values {
		if len(entries) != 1 || historyID != "" && key != "cursor" && key != "limit" {
			return result, contract.ErrInvalidRequest
		}
		switch key {
		case "cursor", "limit", "memberId", "asset", "state":
		default:
			return result, contract.ErrInvalidRequest
		}
	}
	if value := values.Get("limit"); value != "" {
		result.limit, err = strconv.Atoi(value)
		if err != nil || result.limit < 1 || result.limit > 100 {
			return result, contract.ErrInvalidRequest
		}
	}
	if historyID == "" {
		result.filter = application.ReimbursementFilter{MemberID: householdMembership(values.Get("memberId")), Asset: money.Asset(values.Get("asset")), State: ledger.ReimbursementState(values.Get("state"))}
		if value := values.Get("memberId"); value != "" && !validUUID(value) {
			return result, contract.ErrInvalidRequest
		}
	}
	scope, _ := json.Marshal([]string{string(result.filter.MemberID), string(result.filter.Asset), string(result.filter.State)})
	result.scope = string(scope)
	if value := values.Get("cursor"); value != "" {
		if len(value) > 2048 {
			return result, contract.ErrInvalidRequest
		}
		encoded, _, found := strings.Cut(value, ".")
		raw, decodeErr := base64.RawURLEncoding.DecodeString(encoded)
		var decoded []json.RawMessage
		if !found || decodeErr != nil || json.Unmarshal(raw, &decoded) != nil || len(decoded) != 3 {
			return result, contract.ErrInvalidRequest
		}
		var at, id string
		var revision uint64
		if json.Unmarshal(decoded[0], &at) != nil || json.Unmarshal(decoded[1], &id) != nil || json.Unmarshal(decoded[2], &revision) != nil {
			return result, contract.ErrInvalidRequest
		}
		stamp, parseErr := calendar.ParseInstant(at)
		if parseErr != nil || historyID == "" && !validUUID(id) || historyID != "" && (id != "" || revision < 1) {
			return result, contract.ErrInvalidRequest
		}
		result.after = application.ReimbursementCursor{At: stamp, ID: id, Revision: revision}
		if !hmac.Equal([]byte(value), []byte(result.cursor(result.after))) {
			return result, contract.ErrInvalidRequest
		}
	}
	return result, nil
}

func (s *Server) list(w http.ResponseWriter, r *http.Request) {
	page, err := s.page(r, "")
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.ReimbursementPage{Items: []generated.Reimbursement{}}
	err = s.reads.WithinFinancialRead(r.Context(), page.access.Principal, func(ctx context.Context) error {
		values, next, queryErr := s.service.List(ctx, page.access.Principal, page.filter, page.after, page.limit)
		if queryErr != nil {
			return queryErr
		}
		for _, value := range values {
			dto, mapErr := reimbursementDTO(value)
			if mapErr != nil {
				return mapErr
			}
			out.Items = append(out.Items, dto)
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		out.Quality.Coverage, queryErr = s.boundary.CoverageToDTO(coverage)
		out.Quality.Freshness = generated.Freshness(reporting.UnknownFreshness)
		if next != nil {
			value := page.cursor(*next)
			out.NextCursor = &value
		}
		return queryErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}

func (s *Server) read(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	id, err := reimbursementID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	var out generated.Reimbursement
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		value, queryErr := s.service.Read(ctx, access.Principal, id)
		if queryErr == nil {
			out, queryErr = reimbursementDTO(value)
		}
		return queryErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}

func (s *Server) history(w http.ResponseWriter, r *http.Request) {
	id, err := reimbursementID(r)
	if err != nil {
		s.problem(w, err)
		return
	}
	page, err := s.page(r, id)
	if err != nil {
		s.problem(w, err)
		return
	}
	out := generated.ReimbursementPage{Items: []generated.Reimbursement{}}
	err = s.reads.WithinFinancialRead(r.Context(), page.access.Principal, func(ctx context.Context) error {
		values, next, queryErr := s.service.History(ctx, page.access.Principal, id, page.after, page.limit)
		if queryErr != nil {
			return queryErr
		}
		for _, value := range values {
			dto, mapErr := reimbursementDTO(value)
			if mapErr != nil {
				return mapErr
			}
			out.Items = append(out.Items, dto)
		}
		coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
		out.Quality.Coverage, queryErr = s.boundary.CoverageToDTO(coverage)
		out.Quality.Freshness = generated.Freshness(reporting.UnknownFreshness)
		if next != nil {
			value := page.cursor(*next)
			out.NextCursor = &value
		}
		return queryErr
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}

func validUUID(value string) bool {
	id, err := uuid.Parse(value)
	return err == nil && id.Version() == 4 && id.String() == value
}
