package storage

import (
	"context"
	"encoding/json"
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	application "github.com/pchkauu/want-keep/backend/internal/matching/application"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
)

func (s *Store) MatchingReferences(ctx context.Context, p household.Principal, r ledger.Revision, exact bool, limit int) ([]ledger.Revision, bool, error) {
	if limit < 1 || limit > 100 {
		return nil, false, matching.ErrInvalid
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, false, err
	}
	var query string
	args := []any{p.HouseholdID(), r.OperationID, limit + 1}
	if exact {
		if r.Correspondence == nil {
			return []ledger.Revision{}, true, nil
		}
		d := r.Correspondence.Digest()
		args = append(args, d[:])
		query = `SELECT DISTINCT o.id,o.revision FROM want_keep.operations o JOIN want_keep.ledger_correspondences c ON(c.household_id,c.operation_id)=(o.household_id,o.id) WHERE o.household_id=$1 AND o.id<>$2 AND c.digest=$4 ORDER BY o.id LIMIT $3`
	} else {
		type leg struct {
			Account string `json:"account"`
			Amount  string `json:"amount"`
			Asset   string `json:"asset"`
			Role    string `json:"role"`
		}
		legs := []leg{}
		for _, posting := range r.Postings {
			if posting.MovesMoney() && (posting.Role == ledger.Principal || posting.Role == ledger.Fee) {
				legs = append(legs, leg{posting.AccountID, posting.Money.Amount(), string(posting.Money.Asset()), string(posting.Role)})
			}
		}
		if len(legs) == 0 {
			return []ledger.Revision{}, true, nil
		}
		encoded, err := json.Marshal(legs)
		if err != nil {
			return nil, false, err
		}
		args = append(args, encoded, r.CashDate.String(), r.Type, r.Origin)
		query = `SELECT o.id,o.revision FROM want_keep.operations o
 JOIN want_keep.operation_revisions v ON(v.household_id,v.operation_id,v.revision)=(o.household_id,o.id,o.revision)
 LEFT JOIN want_keep.transaction_details d ON(d.household_id,d.operation_id,d.revision)=(v.household_id,v.operation_id,v.revision)
 WHERE o.household_id=$1 AND o.id<>$2 AND v.economic_type IN ('income','expense','transfer','exchange','refund') AND (v.economic_type='refund')=($6='refund') AND (v.economic_type<>'refund' OR d.origin<>$7) AND v.cash_date BETWEEN $5::date-7 AND $5::date+7
 AND EXISTS(SELECT 1 FROM want_keep.postings p JOIN jsonb_to_recordset($4::jsonb) AS wanted(account text,amount text,asset text,role text) ON(p.account_id,p.amount,p.asset,p.role)=(wanted.account::uuid,wanted.amount::numeric,wanted.asset,wanted.role)
 WHERE (p.household_id,p.operation_id,p.revision)=(o.household_id,o.id,o.revision))
 AND NOT EXISTS(SELECT 1 FROM want_keep.ledger_participations p WHERE (p.household_id,p.operation_id,p.revision)=(o.household_id,o.id,o.revision) AND p.state='waiting')
 ORDER BY o.id LIMIT $3`

	}
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, false, err
	}
	defer rows.Close()
	refs := []matching.Member{}
	for rows.Next() {
		var m matching.Member
		if err = rows.Scan(&m.OperationID, &m.Revision); err != nil {
			return nil, false, err
		}
		refs = append(refs, m)
	}
	if err = rows.Err(); err != nil {
		return nil, false, err
	}
	rows.Close()
	complete := len(refs) <= limit
	if !complete {
		refs = refs[:limit]
	}
	out := []ledger.Revision{}
	for _, m := range refs {
		v, err := s.LedgerRevision(ctx, p, m.OperationID, m.Revision)
		if err != nil {
			return nil, false, err
		}
		if exact {
			c := r.Correspondence
			digest := c.Digest()
			var verified bool
			err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.ledger_correspondences WHERE household_id=$1 AND operation_id=$2 AND digest=$3 AND kind=$4 AND namespace=$5 AND reference=$6 AND network=$7 AND movement=$8 AND COALESCE(from_account_id::text,'')=$9 AND COALESCE(to_account_id::text,'')=$10)`, p.HouseholdID(), v.OperationID, digest[:], c.Kind, c.Namespace, c.Reference, c.Network, c.Movement, c.FromAccountID, c.ToAccountID).Scan(&verified)
			if err != nil {
				return nil, false, err
			}
			if !verified {
				return nil, false, ledger.ErrSourceAmbiguous
			}
		}

		if !exact && r.Correspondence != nil && v.Correspondence != nil && r.Correspondence.Namespace == v.Correspondence.Namespace && r.Correspondence.Digest() != v.Correspondence.Digest() {
			continue
		}
		out = append(out, v)
	}
	return out, complete, nil
}

func (s *Store) MatchingPage(ctx context.Context, p household.Principal, state matching.State, c application.Cursor, limit int) ([]matching.Group, *application.Cursor, error) {
	if limit < 1 || limit > 100 {
		return nil, nil, matching.ErrInvalid
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, nil, err
	}
	var at any
	var ns any
	if c.ID != "" {
		at, ns = splitInstant(c.At)
	}
	rows, err := q.Query(ctx, `SELECT c.id,first.at,first.at_ns FROM want_keep.matching_cases c JOIN want_keep.matching_revisions r ON(r.household_id,r.id,r.revision)=(c.household_id,c.id,c.revision) JOIN want_keep.matching_revisions first ON(first.household_id,first.id,first.revision)=(c.household_id,c.id,1) WHERE c.household_id=$1 AND($2='' OR r.state=$2) AND($3::timestamptz IS NULL OR(first.at,first.at_ns,c.id)>($3,$4,NULLIF($5,'')::uuid)) ORDER BY first.at,first.at_ns,c.id LIMIT $6`, p.HouseholdID(), state, at, ns, c.ID, limit+1)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()
	refs := []application.Cursor{}
	for rows.Next() {
		var ref application.Cursor
		var t time.Time
		var n int16
		if err = rows.Scan(&ref.ID, &t, &n); err != nil {
			return nil, nil, err
		}
		ref.At, err = restoreInstant(t, n)
		if err != nil {
			return nil, nil, err
		}
		refs = append(refs, ref)
	}
	if err = rows.Err(); err != nil {
		return nil, nil, err
	}
	rows.Close()
	var next *application.Cursor
	if len(refs) > limit {
		value := refs[limit-1]
		next = &value
		refs = refs[:limit]
	}
	out := []matching.Group{}
	for _, ref := range refs {
		g, err := s.MatchingGroup(ctx, p, ref.ID)
		if err != nil {
			return nil, nil, err
		}
		out = append(out, g)
	}
	return out, next, nil
}

func (s *Store) MatchingRejected(ctx context.Context, p household.Principal, ids []string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var found bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.matching_cases c
 JOIN want_keep.matching_revisions r ON(r.household_id,r.id,r.revision)=(c.household_id,c.id,c.revision)
 JOIN want_keep.matching_members m ON(m.household_id,m.group_id,m.group_revision)=(r.household_id,r.id,r.revision)
 JOIN want_keep.matching_candidates candidate ON(candidate.household_id,candidate.group_id,candidate.group_revision)=(r.household_id,r.id,r.revision)
 WHERE c.household_id=$1 AND r.state='separate' AND m.operation_id=ANY($2::uuid[]) AND candidate.operation_id=ANY($2::uuid[]))`, p.HouseholdID(), ids).Scan(&found)
	return found, err
}
