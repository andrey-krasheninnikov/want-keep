package storage

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
)

func (s *Store) ValidationSource(ctx context.Context, p household.Principal, job jobs.Job) (aiapp.ValidationSource, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return aiapp.ValidationSource{}, err
	}
	var value aiapp.ValidationSource
	var providerJob string
	err = q.QueryRow(ctx, `SELECT a.id,a.job_id,state.state,state.structured_output FROM want_keep.review_validation_jobs v JOIN want_keep.ai_attempts a ON (a.household_id,a.id)=(v.household_id,v.attempt_id) JOIN LATERAL(SELECT state,structured_output FROM want_keep.ai_attempt_states WHERE household_id=a.household_id AND attempt_id=a.id ORDER BY revision DESC LIMIT 1) state ON true WHERE v.household_id=$1 AND v.job_id=$2`, p.HouseholdID(), job.ID).Scan(&value.AttemptID, &providerJob, &value.State, &value.Output)
	if err != nil {
		return value, err
	}
	value.Context, err = s.ReviewContext(ctx, p, providerJob)
	if errors.Is(err, pgx.ErrNoRows) {
		err = nil
	}
	return value, err
}
func (s *Store) SaveReviewValidation(ctx context.Context, p household.Principal, v ai.ReviewValidation) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_validations(household_id,attempt_id,operation_id,operation_revision,state,code,rationale,proposal_id,clarification_id) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,NULLIF($9,'')::uuid) ON CONFLICT(household_id,attempt_id) DO NOTHING`, p.HouseholdID(), v.AttemptID, v.OperationID, v.Revision, v.State, v.Code, v.Rationale, v.ProposalID, v.ClarificationID)
	return err
}
func (s *Store) SaveReviewProposal(ctx context.Context, p household.Principal, v ai.ReviewProposal) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	payload, err := json.Marshal(v)
	if err != nil {
		return err
	}
	projection, err := json.Marshal(v.Context)
	if err != nil {
		return err
	}
	if v.Revision == 1 {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_proposals(household_id,id,revision,operation_id,operation_revision,state,payload,context) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, p.HouseholdID(), v.ID, v.Revision, v.OperationID, v.OperationRevision, v.State, payload, projection)
	} else {
		tag, e := scope.tx.Exec(ctx, `UPDATE want_keep.review_proposals SET revision=$3,state=$4,payload=$5 WHERE household_id=$1 AND id=$2 AND revision=$3-1`, p.HouseholdID(), v.ID, v.Revision, v.State, payload)
		err = e
		if err == nil && tag.RowsAffected() != 1 {
			return commands.Rejection{Code: "version_conflict"}
		}
	}
	if err != nil {
		return err
	}
	return s.saveReviewState(ctx, p, "proposal", v.ID, v.Revision, payload)
}
func (s *Store) ReviewProposal(ctx context.Context, p household.Principal, id string) (ai.ReviewProposal, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ai.ReviewProposal{}, err
	}
	var raw, projection []byte
	err = q.QueryRow(ctx, `SELECT payload,context FROM want_keep.review_proposals WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&raw, &projection)
	if errors.Is(err, pgx.ErrNoRows) {
		return ai.ReviewProposal{}, commands.Rejection{Code: "not_found"}
	}
	var out ai.ReviewProposal
	if err == nil {
		err = json.Unmarshal(raw, &out)
	}
	if err == nil {
		err = json.Unmarshal(projection, &out.Context)
	}
	if err == nil && out.State == "pending" {
		current, found, e := s.CurrentLedgerRevision(ctx, p, out.OperationID)
		if e != nil {
			return out, e
		}
		if !found || current.Revision != out.OperationRevision {
			out.State = "superseded"
		}
	}
	return out, err
}

type clarificationPayload struct {
	Value    ai.Clarification              `json:"value"`
	Commands map[string][]ai.ReviewCommand `json:"choiceCommands"`
}

func (s *Store) SaveClarification(ctx context.Context, p household.Principal, v ai.Clarification) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	payload := clarificationPayload{Value: v, Commands: map[string][]ai.ReviewCommand{}}
	for _, c := range v.Choices {
		payload.Commands[c.ID] = c.Commands
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	projection, err := json.Marshal(v.Context)
	if err != nil {
		return err
	}
	if v.Revision == 1 {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_clarifications(household_id,id,revision,operation_id,operation_revision,state,payload,context) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, p.HouseholdID(), v.ID, v.Revision, v.OperationID, v.OperationRevision, v.State, raw, projection)
	} else {
		tag, e := scope.tx.Exec(ctx, `UPDATE want_keep.review_clarifications SET revision=$3,state=$4,payload=$5 WHERE household_id=$1 AND id=$2 AND revision=$3-1`, p.HouseholdID(), v.ID, v.Revision, v.State, raw)
		err = e
		if err == nil && tag.RowsAffected() != 1 {
			return commands.Rejection{Code: "version_conflict"}
		}
	}
	if err != nil {
		return err
	}
	return s.saveReviewState(ctx, p, "clarification", v.ID, v.Revision, raw)
}
func decodeClarification(raw, projection []byte) (ai.Clarification, error) {
	var payload clarificationPayload
	err := json.Unmarshal(raw, &payload)
	if err == nil {
		err = json.Unmarshal(projection, &payload.Value.Context)
	}
	for i := range payload.Value.Choices {
		payload.Value.Choices[i].Commands = payload.Commands[payload.Value.Choices[i].ID]
	}
	return payload.Value, err
}
func (s *Store) Clarification(ctx context.Context, p household.Principal, id string) (ai.Clarification, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ai.Clarification{}, err
	}
	var raw, projection []byte
	err = q.QueryRow(ctx, `SELECT payload,context FROM want_keep.review_clarifications WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&raw, &projection)
	if errors.Is(err, pgx.ErrNoRows) {
		return ai.Clarification{}, commands.Rejection{Code: "not_found"}
	}
	if err != nil {
		return ai.Clarification{}, err
	}
	return decodeClarification(raw, projection)
}
func (s *Store) Clarifications(ctx context.Context, p household.Principal, after string, limit int) ([]ai.Clarification, string, error) {
	if limit < 1 || limit > 100 {
		return nil, "", commands.Rejection{Code: "invalid_request"}
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, "", err
	}
	rows, err := q.Query(ctx, `SELECT c.payload,c.context FROM want_keep.review_clarifications c JOIN want_keep.operations o ON (o.household_id,o.id,o.revision)=(c.household_id,c.operation_id,c.operation_revision) WHERE c.household_id=$1 AND c.state IN ('open','checking') AND ($2='' OR c.id>NULLIF($2,'')::uuid) ORDER BY c.id LIMIT $3`, p.HouseholdID(), after, limit+1)
	if err != nil {
		return nil, "", err
	}
	defer rows.Close()
	out := []ai.Clarification{}
	for rows.Next() {
		var raw, projection []byte
		if err = rows.Scan(&raw, &projection); err != nil {
			return nil, "", err
		}
		v, e := decodeClarification(raw, projection)
		if e != nil {
			return nil, "", e
		}
		out = append(out, v)
	}
	if err = rows.Err(); err != nil {
		return nil, "", err
	}
	next := ""
	if len(out) > limit {
		out = out[:limit]
		next = out[limit-1].ID
	}
	return out, next, nil
}
func (s *Store) SaveReviewAnswer(ctx context.Context, p household.Principal, v ai.Clarification, in ai.ReviewAnswer) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	var jobID any
	if in.Text != "" {
		id := newID()
		jobID = id
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline,resource_id,resource_revision) VALUES($1,$2,$3,'ai_answer','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours',$4,$5)`, p.HouseholdID(), id, p.UserID(), v.OperationID, v.OperationRevision)
		if err != nil {
			return err
		}
		projection := v.Context
		projection.JobID = id
		projection.ClarificationID = v.ID
		projection.AnswerRevision = v.Revision
		var cases []aiReviewCase
		if err = json.Unmarshal(projection.Input, &cases); err != nil || len(cases) != 1 {
			return ai.ErrReviewCommand
		}
		text, err := json.Marshal(struct {
			Original string `json:"original"`
			Question string `json:"question"`
			Answer   string `json:"answer"`
		}{cases[0].Text, v.Question, in.Text})
		if err != nil {
			return err
		}
		cases[0].Text = string(text)
		projection.Input, err = json.Marshal(cases)
		if err != nil {
			return err
		}
		if err = s.SaveReviewContext(ctx, p, projection); err != nil {
			return err
		}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_answers(household_id,clarification_id,revision,actor_id,choice_id,answer_text,job_id) VALUES($1,$2,$3,$4,$5,$6,$7)`, p.HouseholdID(), v.ID, v.Revision, p.UserID(), in.ChoiceID, in.Text, jobID)
	return err
}

func (s *Store) TransactionReviewStatus(ctx context.Context, p household.Principal, id string, revision uint64) (string, []journal.ReviewReference, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return "", nil, err
	}
	var state, clarificationID string
	err = q.QueryRow(ctx, `SELECT v.state,COALESCE(v.clarification_id::text,'') FROM want_keep.review_validations v JOIN want_keep.ai_attempts a ON (a.household_id,a.id)=(v.household_id,v.attempt_id) WHERE v.household_id=$1 AND v.operation_id=$2 AND v.operation_revision=$3 ORDER BY a.created_at DESC,a.created_ns DESC,a.id DESC LIMIT 1`, p.HouseholdID(), id, revision).Scan(&state, &clarificationID)
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return "", nil, err
	}
	status := ""
	switch state {
	case "applied":
		status = "reviewed"
	case "clarification", "proposed":
		status = "clarification"
	case "rejected", "stale", "failed":
		status = "failed"
	}
	if status == "clarification" && clarificationID != "" {
		var question string
		err := q.QueryRow(ctx, `SELECT state FROM want_keep.review_clarifications WHERE household_id=$1 AND id=$2`, p.HouseholdID(), clarificationID).Scan(&question)
		if err != nil {
			return "", nil, err
		}
		if question == "answered" {
			status = "reviewed"
		}
		if question == "checking" {
			status = "waiting"
		}
	}
	rows, err := q.Query(ctx, `SELECT 'proposal',id,revision,state FROM want_keep.review_proposals WHERE household_id=$1 AND operation_id=$2 AND operation_revision=$3 UNION ALL SELECT 'clarification',id,revision,state FROM want_keep.review_clarifications WHERE household_id=$1 AND operation_id=$2 AND operation_revision=$3 ORDER BY 1,2 LIMIT 100`, p.HouseholdID(), id, revision)
	if err != nil {
		return "", nil, err
	}
	defer rows.Close()
	refs := []journal.ReviewReference{}
	for rows.Next() {
		var ref journal.ReviewReference
		if err = rows.Scan(&ref.Kind, &ref.ID, &ref.Revision, &ref.State); err != nil {
			return "", nil, err
		}
		refs = append(refs, ref)
	}
	return status, refs, rows.Err()
}

func (s *Store) saveReviewState(ctx context.Context, p household.Principal, kind, id string, revision uint64, payload []byte) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	var proposal, clarification any
	if kind == "proposal" {
		proposal = id
	} else {
		clarification = id
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.review_state_history(household_id,kind,id,revision,actor_id,payload,proposal_id,clarification_id) VALUES($1,$2,$3,$4,$5,$6,$7,$8)`, p.HouseholdID(), kind, id, revision, p.UserID(), payload, proposal, clarification)
	return err
}

func (s *Store) ProposalClarifications(ctx context.Context, p household.Principal, id string) ([]ai.Clarification, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT payload,context FROM want_keep.review_clarifications WHERE household_id=$1 AND payload->'value'->>'proposalId'=$2 AND state IN ('open','checking') ORDER BY id`, p.HouseholdID(), id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []ai.Clarification{}
	for rows.Next() {
		var raw, projection []byte
		if err = rows.Scan(&raw, &projection); err != nil {
			return nil, err
		}
		v, e := decodeClarification(raw, projection)
		if e != nil {
			return nil, e
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
