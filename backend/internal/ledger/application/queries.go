package application

import (
	"context"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Filter struct {
	AccountID, CategoryID, MerchantID string
	Search, ItemSearch                string
	From, To                          calendar.Date
	Type                              ledger.Type
	State                             ledger.State
}

func (f Filter) Validate() error {
	if utf8.RuneCountInString(f.Search) > 200 || utf8.RuneCountInString(f.ItemSearch) > 200 || f.Type != "" && !f.Type.Valid() || f.State != "" && !f.State.Valid() || f.From.String() != "" && f.To.String() != "" && f.From.String() > f.To.String() {
		return ledger.ErrInvalidRevision
	}
	return nil
}

type Cursor struct {
	At calendar.Instant
	ID string
}
type SourceReference struct {
	Key          ledger.SourceKey
	Revision     uint64
	ConnectionID string
}
type View struct {
	Revision         ledger.Revision
	Sources          []SourceReference
	Coverage         reporting.Coverage
	SourceFacts      []SourceFact
	Review           *ReviewResult
	Refunds          []expenses.Refund
	ReviewReferences []ReviewReference
}
type ReviewReference struct {
	Kind, ID string
	Revision uint64
	State    string
}
type SourceFact struct {
	SourceID string
	Revision uint64
	Fact     ledger.Revision
	Conflict string
}
type QueryRepository interface {
	HistoryRepository
	ReviewRepository
	TransactionReviewStatus(context.Context, household.Principal, string, uint64) (string, []ReviewReference, error)
	TransactionCoverage(context.Context, household.Principal) (reporting.Coverage, error)
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	TransactionReferences(context.Context, household.Principal, Filter, Cursor, int) ([]ledger.Revision, *Cursor, error)
	TransactionSources(context.Context, household.Principal, string, uint64) ([]SourceReference, error)
	TransactionSourceFacts(context.Context, household.Principal, string, uint64) ([]SourceFact, error)
	TransactionMatchingConflict(context.Context, household.Principal, string, uint64) (bool, error)
	RefundsForOperation(context.Context, household.Principal, string) ([]expenses.Refund, error)
	RefundsForOperationAt(context.Context, household.Principal, string, uint64, calendar.Instant) ([]expenses.Refund, error)
	RefundsForOperations(context.Context, household.Principal, []string) (map[string][]expenses.Refund, error)
	RefundsForRevisions(context.Context, household.Principal, []ledger.Revision) (map[uint64][]expenses.Refund, error)
	RefundHistory(context.Context, household.Principal, string, uint64, int) ([]expenses.Refund, uint64, error)
}
type Queries struct{ repository QueryRepository }

func NewQueries(r QueryRepository) *Queries { return &Queries{r} }
func (q *Queries) RefundHistory(ctx context.Context, p household.Principal, id string, before uint64, limit int) ([]expenses.Refund, uint64, error) {
	return q.repository.RefundHistory(ctx, p, id, before, limit)
}
func (q *Queries) Read(ctx context.Context, p household.Principal, id string) (View, error) {
	r, exists, err := q.repository.CurrentLedgerRevision(ctx, p, id)
	if err != nil {
		return View{}, err
	}
	if !exists {
		return View{}, ledger.ErrNotFound
	}
	return q.view(ctx, p, r)
}
func (q *Queries) List(ctx context.Context, p household.Principal, f Filter, c Cursor, limit int) ([]View, *Cursor, error) {
	if err := f.Validate(); err != nil {
		return nil, nil, err
	}
	if limit < 1 || limit > 100 {
		return nil, nil, ledger.ErrInvalidRevision
	}
	revisions, next, err := q.repository.TransactionReferences(ctx, p, f, c, limit)
	if err != nil {
		return nil, nil, err
	}
	out := make([]View, 0, len(revisions))
	ids := make([]string, len(revisions))
	for i := range revisions {
		ids[i] = revisions[i].OperationID
	}
	refunds, err := q.repository.RefundsForOperations(ctx, p, ids)
	if err != nil {
		return nil, nil, err
	}
	for _, r := range revisions {
		v, err := q.viewWithRefunds(ctx, p, r, refunds[r.OperationID])
		if err != nil {
			return nil, nil, err
		}
		out = append(out, v)
	}
	return out, next, nil
}
func (q *Queries) view(ctx context.Context, p household.Principal, r ledger.Revision) (View, error) {
	refunds, err := q.repository.RefundsForOperation(ctx, p, r.OperationID)
	if err != nil {
		return View{}, err
	}
	return q.viewWithRefunds(ctx, p, r, refunds)
}

func (q *Queries) viewAt(ctx context.Context, p household.Principal, r ledger.Revision) (View, error) {
	refunds, err := q.repository.RefundsForOperationAt(ctx, p, r.OperationID, r.Revision, r.RecordedAt)
	if err != nil {
		return View{}, err
	}
	return q.viewWithRefunds(ctx, p, r, refunds)
}

func (q *Queries) viewWithRefunds(ctx context.Context, p household.Principal, r ledger.Revision, refunds []expenses.Refund) (View, error) {
	review, reviewed, err := q.repository.ReviewResult(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	status, refs, err := q.repository.TransactionReviewStatus(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	if status != "" {
		r.ReviewState = status
	}
	facts, err := q.repository.TransactionSourceFacts(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	sources, err := q.repository.TransactionSources(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	reasons := []string{}
	matchingConflict, err := q.repository.TransactionMatchingConflict(ctx, p, r.OperationID, r.Revision)
	if err != nil {
		return View{}, err
	}
	r.SourceConflict = r.SourceConflict || matchingConflict
	if r.Participation.AwaitingDecision() {
		reasons = append(reasons, "matching_unresolved")
	}
	if r.SourceConflict {
		reasons = append(reasons, "source_conflict")
	}
	if r.Origin != "manual" {
		reasons = append(reasons, "source_history_not_reconciled")
	}
	if r.FeeKnowledge != ledger.KnownFees {
		reasons = append(reasons, "fees_unknown")
	}
	for _, refund := range refunds {
		if refund.State == expenses.Clarification {
			reasons = append(reasons, "refund_clarification")
		}
	}
	for _, posting := range r.Postings {
		if posting.Funding == ledger.UnknownFunds {
			reasons = append(reasons, "funding_split_unknown")
			break
		}
	}
	state := reporting.Complete
	if len(reasons) > 0 {
		state = reporting.Partial
	}
	coverage, err := reporting.NewCoverage(state, reasons)
	v := View{Revision: r, Sources: sources, Coverage: coverage, SourceFacts: facts, Refunds: refunds, ReviewReferences: refs}
	if reviewed {
		v.Review = &review
	}
	return v, err
}

func (q *Queries) Coverage(ctx context.Context, p household.Principal) (reporting.Coverage, error) {
	return q.repository.TransactionCoverage(ctx, p)
}
