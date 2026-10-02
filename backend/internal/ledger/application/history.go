package application

import (
	"context"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type HistoryRepository interface {
	DecisionRepository
	HistoryRevisions(context.Context, household.Principal, string, uint64, int) ([]ledger.Revision, uint64, error)
}
type HistoryEntry struct {
	Current    View
	Before     *View
	Decision   *ledger.Decision
	Affected   []ExpectedRevision
	UndoReason string
}

func (q *Queries) Revision(ctx context.Context, p household.Principal, id string, revision uint64) (View, error) {
	r, err := q.repository.LedgerRevision(ctx, p, id, revision)
	if err != nil {
		return View{}, err
	}
	return q.viewAt(ctx, p, r)
}
func (q *Queries) History(ctx context.Context, p household.Principal, id string, before uint64, limit int) ([]HistoryEntry, uint64, error) {
	if limit < 1 || limit > 100 {
		return nil, 0, ledger.ErrInvalidRevision
	}
	if _, err := q.Read(ctx, p, id); err != nil {
		return nil, 0, err
	}
	revs, next, err := q.repository.HistoryRevisions(ctx, p, id, before, limit)
	if err != nil {
		return nil, 0, err
	}
	points := make([]ledger.Revision, 0, len(revs)*2)
	priors := make(map[uint64]ledger.Revision, len(revs))
	points = append(points, revs...)
	for _, r := range revs {
		if r.Revision <= 1 {
			continue
		}
		prior, loadErr := q.repository.LedgerRevision(ctx, p, id, r.Revision-1)
		if loadErr != nil {
			return nil, 0, loadErr
		}
		priors[r.Revision] = prior
		points = append(points, prior)
	}
	refunds, err := q.repository.RefundsForRevisions(ctx, p, points)
	if err != nil {
		return nil, 0, err
	}
	out := []HistoryEntry{}
	for _, r := range revs {
		view, err := q.viewWithRefunds(ctx, p, r, refunds[r.Revision])
		if err != nil {
			return nil, 0, err
		}
		entry := HistoryEntry{Current: view, UndoReason: "legacy"}
		if r.Revision > 1 {
			priorRevision := priors[r.Revision]
			prior, err := q.viewWithRefunds(ctx, p, priorRevision, refunds[priorRevision.Revision])
			if err != nil {
				return nil, 0, err
			}
			entry.Before = &prior
		}
		if r.DecisionID != "" {
			d, err := q.repository.Decision(ctx, p, r.DecisionID)
			if err != nil {
				return nil, 0, err
			}
			entry.Decision = &d
			entry.UndoReason, entry.Affected, err = q.undoAvailability(ctx, p, d)
			if err != nil {
				return nil, 0, err
			}
		}
		out = append(out, entry)
	}
	return out, next, nil
}
func (q *Queries) undoAvailability(ctx context.Context, p household.Principal, d ledger.Decision) (string, []ExpectedRevision, error) {
	undone, err := q.repository.DecisionUndone(ctx, p, d.ID)
	if err != nil {
		return "", nil, err
	}
	if undone {
		return "already_undone", nil, nil
	}
	affected := []ExpectedRevision{}
	for _, e := range d.Entries {
		r, ok, err := q.repository.CurrentLedgerRevision(ctx, p, e.OperationID)
		if err != nil {
			return "", nil, err
		}
		if !ok {
			return "invalid_restore", nil, nil
		}
		affected = append(affected, ExpectedRevision{e.OperationID, r.Revision})
		for _, f := range e.Fields {
			if f != ledger.ContributionField && r.FieldVersions[f] != e.After {
				return "superseded", affected, nil
			}
		}
		before, err := q.repository.LedgerRevision(ctx, p, e.OperationID, e.Before)
		if err != nil {
			return "", nil, err
		}
		source, err := q.repository.LatestSourceFact(ctx, p, e.OperationID)
		if err == ledger.ErrSourceAmbiguous {
			return "source_ambiguous", affected, nil
		}
		if err != nil {
			return "", nil, err
		}
		next, err := (decisionRestorer{q.repository}).restore(ctx, p, r, e, before, source)
		if err != nil {
			return "invalid_restore", affected, nil
		}
		next, err = next.InTimezone(r.Timezone)
		if err != nil {
			return "invalid_restore", affected, nil
		}
		if next.Validate() != nil {
			return "invalid_restore", affected, nil
		}
	}
	return "available", affected, nil
}
