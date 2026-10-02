package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
)

// ReviewInput persists a frozen projection before any external counting or generation.
func (s *Store) ReviewInput(ctx context.Context, p household.Principal, job jobs.Job) (input json.RawMessage, err error) {
	err = s.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, err := s.FenceJob(ctx, p, job); err != nil {
			return err
		}
		saved, err := s.ReviewContext(ctx, p, job.ID)
		if err == nil {
			input = saved.Input
			return nil
		}
		if !errors.Is(err, pgx.ErrNoRows) {
			return err
		}
		if job.Kind == jobs.AIAnswer {
			return ai.ErrReviewCommand
		}
		raw, err := s.buildReviewInput(ctx, p, job)
		if err != nil {
			return err
		}
		var cases []aiReviewCase
		if err = json.Unmarshal(raw, &cases); err != nil {
			return err
		}
		context := ai.ReviewContext{JobID: job.ID, OperationID: job.ResourceID, Revision: job.ResourceRevision, References: map[string]ai.Reference{"ledger_revision": {Kind: "evidence", ID: job.ResourceID, Revision: job.ResourceRevision}}}
		categories, next, err := s.Categories(ctx, p, category.Filter{State: category.Active}, "", 100)
		if err != nil {
			return err
		}
		merchants, merchantNext, err := s.Merchants(ctx, p, category.Filter{State: category.Active}, "", 100)
		if err != nil {
			return err
		}
		boundary, err := s.FirstLedgerRuleBoundary(ctx, p, job.ResourceID)
		if err != nil {
			return err
		}
		context.RuleBoundary = boundary
		rules, err := s.reviewRules(ctx, p, boundary)
		if err != nil {
			return err
		}
		rulesComplete := len(rules) <= 100
		if !rulesComplete {
			rules = rules[:100]
		}
		for i, rule := range rules {
			context.References[fmt.Sprintf("rule-%d", i+1)] = ai.Reference{Kind: "rule", ID: rule.ID, Revision: rule.Revision}
		}
		members, err := s.HouseholdMemberships(ctx, p)
		if err != nil {
			return err
		}
		names, err := s.reviewMemberNames(ctx, p)
		if err != nil {
			return err
		}
		catalog := map[string][]map[string]any{"categories": {}, "merchants": {}, "members": {}, "candidates": {}, "rules": {}}
		for i, c := range categories {
			alias := fmt.Sprintf("category-%d", i+1)
			context.References[alias] = ai.Reference{Kind: "category", ID: c.ID, Revision: c.Revision, Label: c.DisplayName()}
			catalog["categories"] = append(catalog["categories"], map[string]any{"ref": alias, "name": c.DisplayName()})
		}
		for i, m := range merchants {
			alias := fmt.Sprintf("merchant-%d", i+1)
			context.References[alias] = ai.Reference{Kind: "merchant", ID: m.ID, Revision: m.Revision, Label: m.Name}
			catalog["merchants"] = append(catalog["merchants"], map[string]any{"ref": alias, "name": m.Name})
		}
		for i, m := range members {
			if !m.Active {
				continue
			}
			alias := fmt.Sprintf("member-%d", i+1)
			context.References[alias] = ai.Reference{Kind: "member", ID: string(m.ID), Revision: 1, Label: names[string(m.ID)]}
			catalog["members"] = append(catalog["members"], map[string]any{"ref": alias})
		}
		current, err := s.LedgerRevision(ctx, p, job.ResourceID, job.ResourceRevision)
		if err != nil {
			return err
		}
		candidates, complete, err := s.MatchingReferences(ctx, p, current, false, 20)
		if err != nil {
			return err
		}
		for i, c := range candidates {
			alias := fmt.Sprintf("candidate-%d", i+1)
			context.References[alias] = ai.Reference{Kind: "candidate", ID: c.OperationID, Revision: c.Revision, Label: alias}
			catalog["candidates"] = append(catalog["candidates"], map[string]any{"ref": alias, "type": c.Type, "date": c.CashDate.String(), "merchant": c.Merchant, "state": c.State})
		}
		aliases := map[string]string{}
		for alias, ref := range context.References {
			aliases[ref.Kind+":"+ref.ID] = alias
		}
		for i, rule := range rules {
			shares := []map[string]string{}
			for _, share := range rule.Shares {
				alias, ok := aliases["member:"+string(share.MemberID)]
				if !ok {
					continue
				}
				shares = append(shares, map[string]string{"member": alias, "share": share.Value})
			}
			catalog["rules"] = append(catalog["rules"], map[string]any{"ref": fmt.Sprintf("rule-%d", i+1), "state": rule.State, "priority": rule.Priority, "category": aliases["category:"+rule.Condition.CategoryID], "merchant": aliases["merchant:"+rule.Condition.MerchantID], "shares": shares})
		}
		currentAliases := map[string]string{}
		for alias, ref := range context.References {
			if ref.Kind == "category" && ref.ID == current.CategoryID {
				currentAliases["category"] = alias
			}
			if ref.Kind == "merchant" && ref.ID == current.MerchantID {
				currentAliases["merchant"] = alias
			}
		}
		text, err := json.Marshal(struct {
			Transaction           json.RawMessage   `json:"transaction"`
			Catalog               any               `json:"catalog"`
			CatalogComplete       bool              `json:"catalogComplete"`
			CandidatesComplete    bool              `json:"candidatesComplete"`
			CurrentClassification map[string]string `json:"currentClassification"`
			AllocationState       string            `json:"allocationState"`
			AllocationPurpose     string            `json:"allocationPurpose"`
		}{json.RawMessage(cases[0].Text), catalog, next == "" && merchantNext == "" && rulesComplete, complete, currentAliases, string(current.Allocation.State), string(current.Allocation.Purpose)})
		if err != nil {
			return err
		}
		cases[0].Text = string(text)
		cases[0].Members = []string{}
		for _, m := range catalog["members"] {
			cases[0].Members = append(cases[0].Members, m["ref"].(string))
		}
		input, err = json.Marshal(cases)
		if err != nil {
			return err
		}
		context.Input = input
		return s.SaveReviewContext(ctx, p, context)
	})
	return input, err
}

func (s *Store) ReviewContext(ctx context.Context, p household.Principal, id string) (ai.ReviewContext, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ai.ReviewContext{}, err
	}
	var raw []byte
	err = q.QueryRow(ctx, `SELECT context FROM want_keep.review_contexts WHERE household_id=$1 AND job_id=$2`, p.HouseholdID(), id).Scan(&raw)
	var out ai.ReviewContext
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	return out, err
}
func (s *Store) SaveReviewContext(ctx context.Context, p household.Principal, value ai.ReviewContext) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	raw, err := json.Marshal(value)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_contexts(household_id,job_id,operation_id,operation_revision,context) VALUES($1,$2,$3,$4,$5)`, p.HouseholdID(), value.JobID, value.OperationID, value.Revision, raw)
	return err
}
func (s *Store) enqueueAIValidation(ctx context.Context, p household.Principal, job jobs.Job, attempt string) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	id := newID()
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline,resource_id,resource_revision) VALUES($1,$2,$3,'ai_validation','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours',$4,$5)`, p.HouseholdID(), id, p.UserID(), job.ResourceID, job.ResourceRevision)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_validation_jobs(household_id,job_id,attempt_id) VALUES($1,$2,$3)`, p.HouseholdID(), id, attempt)
	return err
}

// reviewRules returns only revisions that existed at the first financial fact.
func (s *Store) reviewRules(ctx context.Context, p household.Principal, boundary uint64) ([]allocation.Rule, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `WITH historical AS (
 SELECT DISTINCT ON (rule_id) household_id,rule_id,revision,priority,state,merchant_id,category_id,actor_id,recorded_at,recorded_ns
 FROM want_keep.allocation_rule_revisions WHERE household_id=$1 AND rule_sequence<=$2 ORDER BY rule_id,rule_sequence DESC
 )
 SELECT e.rule_id,e.revision,e.priority,e.state,COALESCE(e.merchant_id::text,''),COALESCE(e.category_id::text,''),e.actor_id,e.recorded_at,e.recorded_ns,
 array_agg(s.member_id::text ORDER BY s.position),array_agg(s.share::text ORDER BY s.position)
 FROM historical e JOIN want_keep.allocation_rule_shares s ON (s.household_id,s.rule_id,s.revision)=(e.household_id,e.rule_id,e.revision)
 WHERE e.state='active' GROUP BY e.household_id,e.rule_id,e.revision,e.priority,e.state,e.merchant_id,e.category_id,e.actor_id,e.recorded_at,e.recorded_ns
 ORDER BY e.priority,e.rule_id LIMIT 101`, p.HouseholdID(), boundary)
	if err != nil {
		return nil, err
	}
	return scanAllocationRules(rows, p.HouseholdID())
}

func (s *Store) reviewMemberNames(ctx context.Context, p household.Principal) (map[string]string, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT m.id,u.name FROM want_keep.memberships m JOIN want_keep.users u ON u.id=m.user_id WHERE m.household_id=$1 AND m.active ORDER BY m.id`, p.HouseholdID())
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	names := map[string]string{}
	for rows.Next() {
		var id, name string
		if err = rows.Scan(&id, &name); err != nil {
			return nil, err
		}
		names[id] = name
	}
	return names, rows.Err()
}
