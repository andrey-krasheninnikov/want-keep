package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	receiptapp "github.com/pchkauu/want-keep/backend/internal/receipts/application"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

func (s *Store) ReceiptByHash(ctx context.Context, p household.Principal, accountID, hash string) (receipt.Receipt, bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return receipt.Receipt{}, false, err
	}
	var id string
	err = q.QueryRow(ctx, `SELECT id FROM want_keep.receipts WHERE household_id=$1 AND account_id=$2 AND content_hash=$3`, p.HouseholdID(), accountID, hash).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return receipt.Receipt{}, false, nil
	}
	if err != nil {
		return receipt.Receipt{}, false, err
	}
	r, err := s.Receipt(ctx, p, id)
	return r, err == nil, err
}
func (s *Store) Receipt(ctx context.Context, p household.Principal, id string) (receipt.Receipt, error) {
	r := receipt.Receipt{ID: id, HouseholdID: p.HouseholdID(), Candidates: []receipt.Candidate{}, Pages: []receipt.Page{}}
	q, err := s.reader(ctx, p)
	if err != nil {
		return r, err
	}
	var candidates []byte
	var ns int16
	err = q.QueryRow(ctx, `SELECT account_id,attachment_id,actor_id,content_hash,revision,state,reason,COALESCE(transaction_id::text,''),COALESCE(transaction_revision,0),candidates,created_at,created_ns FROM want_keep.receipts WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&r.AccountID, &r.AttachmentID, &r.ActorID, &r.Hash, &r.Revision, &r.State, &r.Reason, &r.TransactionID, &r.TransactionRevision, &candidates, &r.CreatedAt, &ns)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, receipt.ErrNotFound
	}
	if err != nil {
		return r, err
	}
	r.CreatedAt = r.CreatedAt.Add(time.Duration(ns))
	if err = json.Unmarshal(candidates, &r.Candidates); err != nil {
		return r, err
	}
	var snapshot []byte
	if err = q.QueryRow(ctx, `SELECT snapshot FROM want_keep.receipt_revisions WHERE household_id=$1 AND receipt_id=$2 AND revision=$3`, p.HouseholdID(), id, r.Revision).Scan(&snapshot); err != nil {
		return r, err
	}
	var stored receiptSnapshot
	if err = json.Unmarshal(snapshot, &stored); err != nil {
		return r, err
	}
	r.Confirmed = stored.Confirmed
	r.CandidateFingerprint = stored.CandidateFingerprint
	r.CandidatesComplete = stored.CandidatesComplete
	rows, err := q.Query(ctx, `SELECT p.id,p.page,p.content_hash,COALESCE(o.attempt_id::text,''),o.extraction FROM want_keep.receipt_pages p LEFT JOIN want_keep.receipt_page_outcomes o ON (o.household_id,o.page_id,o.page_revision)=(p.household_id,p.id,p.revision) WHERE p.household_id=$1 AND p.receipt_id=$2 ORDER BY p.page`, p.HouseholdID(), id)
	if err != nil {
		return r, err
	}
	defer rows.Close()
	for rows.Next() {
		var page receipt.Page
		var data []byte
		if err = rows.Scan(&page.ID, &page.Number, &page.Hash, &page.AttemptID, &data); err != nil {
			return r, err
		}
		if data != nil {
			var e receipt.Extraction
			if err = json.Unmarshal(data, &e); err != nil {
				return r, err
			}
			page.Extraction = &e
		}
		r.Pages = append(r.Pages, page)
	}
	return r, rows.Err()
}
func (s *Store) CreateReceipt(ctx context.Context, p household.Principal, r receipt.Receipt) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if r.HouseholdID != p.HouseholdID() || r.ActorID != p.UserID() || r.Revision != 1 || len(r.Pages) < 1 || len(r.Pages) > 10 {
		return receipt.ErrInvalid
	}
	at, ns := splitAICallTime(r.CreatedAt)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipts(household_id,id,account_id,attachment_id,actor_id,content_hash,revision,state,created_at,created_ns) VALUES($1,$2,$3,$4,$5,$6,1,$7,$8,$9)`, p.HouseholdID(), r.ID, r.AccountID, r.AttachmentID, p.UserID(), r.Hash, r.State, at, ns)
	if err != nil {
		return err
	}
	if err = s.receiptRevision(ctx, p, r); err != nil {
		return err
	}
	for _, page := range r.Pages {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_pages(household_id,id,receipt_id,page,content_hash) VALUES($1,$2,$3,$4,$5)`, p.HouseholdID(), page.ID, r.ID, page.Number, page.Hash); err != nil {
			return err
		}
		if err = s.ensureAISubject(ctx, p, jobs.Job{Kind: jobs.ReceiptExtract, ResourceID: page.ID, ResourceRevision: 1}); err != nil {
			return err
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline,resource_id,resource_revision) VALUES($1,$2,$3,'receipt_extract','ready',5,clock_timestamp(),clock_timestamp()+INTERVAL '24 hours',$4,1)`, p.HouseholdID(), newID(), p.UserID(), page.ID); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) AddReceiptEvidence(ctx context.Context, p household.Principal, id string, a attachment.Attachment) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if a.HouseholdID != p.HouseholdID() {
		return household.ErrForbidden
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_evidence(household_id,receipt_id,attachment_id,actor_id) VALUES($1,$2,$3,$4) ON CONFLICT(household_id,receipt_id,attachment_id) DO NOTHING`, p.HouseholdID(), id, a.Upload.ID, a.ActorID)
	return err
}

type receiptSnapshot struct {
	State                 receipt.State
	Reason, TransactionID string
	TransactionRevision   uint64
	Candidates            []receipt.Candidate
	Confirmed             *receipt.Extraction
	CandidateFingerprint  string
	CandidatesComplete    bool
}

func (s *Store) receiptRevision(ctx context.Context, p household.Principal, r receipt.Receipt) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(receiptSnapshot{r.State, r.Reason, r.TransactionID, r.TransactionRevision, r.Candidates, r.Confirmed, r.CandidateFingerprint, r.CandidatesComplete})
	if err != nil {
		return err
	}
	at, ns := splitAICallTime(time.Now().UTC())
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_revisions(household_id,receipt_id,revision,actor_id,state,reason,snapshot,command_id,recorded_at,recorded_ns) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid,$9,$10)`, p.HouseholdID(), r.ID, r.Revision, p.UserID(), r.State, r.Reason, data, commands.CurrentCommandID(ctx), at, ns)
	return err
}
func (s *Store) SaveReceipt(ctx context.Context, p household.Principal, r receipt.Receipt, expected uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if r.Revision != expected+1 || r.Revision > command.MaxRevision {
		return command.ErrVersionConflict
	}
	candidates, err := json.Marshal(r.Candidates)
	if err != nil {
		return err
	}
	if r.Candidates == nil {
		candidates = []byte(`[]`)
	}
	tag, err := scope.tx.Exec(ctx, `UPDATE want_keep.receipts SET revision=$3,state=$4,reason=$5,transaction_id=NULLIF($6,'')::uuid,transaction_revision=NULLIF($7,0),candidates=$8 WHERE household_id=$1 AND id=$2 AND revision=$9`, p.HouseholdID(), r.ID, r.Revision, r.State, r.Reason, r.TransactionID, r.TransactionRevision, candidates, expected)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return command.ErrVersionConflict
	}
	return s.receiptRevision(ctx, p, r)
}
func (s *Store) SaveReceiptAnswer(ctx context.Context, p household.Principal, r receipt.Receipt, a receipt.Answer) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	data, err := json.Marshal(a)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_answers(household_id,receipt_id,revision,actor_id,answer) VALUES($1,$2,$3,$4,$5)`, p.HouseholdID(), r.ID, r.Revision, p.UserID(), data)
	return err
}
func (s *Store) ReceiptForPage(ctx context.Context, p household.Principal, pageID string) (receipt.Receipt, int, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return receipt.Receipt{}, 0, err
	}
	var id string
	var page int
	if err = q.QueryRow(ctx, `SELECT receipt_id,page FROM want_keep.receipt_pages WHERE household_id=$1 AND id=$2`, p.HouseholdID(), pageID).Scan(&id, &page); err != nil {
		return receipt.Receipt{}, 0, err
	}
	r, err := s.Receipt(ctx, p, id)
	return r, page, err
}
func (s *Store) ReceiptValidation(ctx context.Context, p household.Principal, j jobs.Job) (receiptapp.Validation, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return receiptapp.Validation{}, err
	}
	if _, err = s.FenceJob(ctx, p, j); err != nil {
		return receiptapp.Validation{}, err
	}
	v := receiptapp.Validation{}
	err = scope.tx.QueryRow(ctx, `SELECT a.resource_id,a.id,l.state,COALESCE(l.structured_output,'null'::jsonb),l.code FROM want_keep.receipt_validation_jobs v JOIN want_keep.ai_attempts a ON (a.household_id,a.id)=(v.household_id,v.attempt_id) JOIN LATERAL(SELECT state,structured_output,code FROM want_keep.ai_attempt_states WHERE household_id=a.household_id AND attempt_id=a.id ORDER BY revision DESC LIMIT 1) l ON true WHERE v.household_id=$1 AND v.job_id=$2 AND a.resource_id=$3 AND a.resource_revision=$4`, p.HouseholdID(), j.ID, j.ResourceID, j.ResourceRevision).Scan(&v.PageID, &v.AttemptID, &v.State, &v.Output, &v.Code)
	return v, err
}
func (s *Store) SaveReceiptPageOutcome(ctx context.Context, p household.Principal, v receiptapp.Validation, e *receipt.Extraction) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	var data any
	if e != nil {
		b, err := json.Marshal(e)
		if err != nil {
			return err
		}
		data = b
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_page_outcomes(household_id,page_id,page_revision,attempt_id,extraction,code) VALUES($1,$2,1,$3,$4,$5) ON CONFLICT(household_id,page_id,page_revision) DO NOTHING`, p.HouseholdID(), v.PageID, v.AttemptID, data, v.Code)
	return err
}
func (s *Store) ReceiptCandidates(ctx context.Context, p household.Principal, r receipt.Receipt, f receipt.Facts, selected string) (receipt.CandidateSet, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return receipt.CandidateSet{}, err
	}
	amount := f.Total
	if f.Kind == ledger.Expense {
		zero, _ := money.NewMoney("0", f.Total.Asset())
		amount, err = zero.Subtract(f.Total)
		if err != nil {
			return receipt.CandidateSet{}, err
		}
	}
	var data []byte
	var count int64
	result := receipt.CandidateSet{}
	err = q.QueryRow(ctx, `WITH candidates AS (
 SELECT o.id,o.revision FROM want_keep.operations o
 JOIN want_keep.operation_revisions v ON(v.household_id,v.operation_id,v.revision)=(o.household_id,o.id,o.revision)
 WHERE o.household_id=$1 AND v.economic_type=$2 AND v.cash_date BETWEEN $3::date-7 AND $3::date+7
 AND EXISTS(SELECT 1 FROM want_keep.postings p WHERE (p.household_id,p.operation_id,p.revision)=(o.household_id,o.id,o.revision) AND p.account_id=$4 AND p.asset=$5 AND p.amount=$6::numeric AND p.role='principal')
 AND NOT EXISTS(SELECT 1 FROM want_keep.ledger_participations p WHERE (p.household_id,p.operation_id,p.revision)=(o.household_id,o.id,o.revision) AND p.state='waiting')
 ), summary AS (
 SELECT count(*) AS count,encode(sha256(convert_to(COALESCE(string_agg(id::text||':'||revision::text,',' ORDER BY id),''),'UTF8')),'hex') AS fingerprint FROM candidates
 ) SELECT summary.count,summary.fingerprint,
 COALESCE((SELECT jsonb_agg(jsonb_build_object('ID',page.id,'Revision',page.revision) ORDER BY page.id) FROM (SELECT id,revision FROM candidates WHERE $7='' OR id=NULLIF($7,'')::uuid ORDER BY id LIMIT 20) page),'[]'::jsonb)
 FROM summary`, p.HouseholdID(), f.Kind, f.Date.String(), r.AccountID, f.Total.Asset(), amount.Amount(), selected).Scan(&count, &result.Fingerprint, &data)
	if err != nil {
		return result, err
	}
	result.Complete = count <= 20
	err = json.Unmarshal(data, &result.Items)
	return result, err
}

func (s *Store) ReceiptProcessingState(ctx context.Context, p household.Principal, id string) (receipt.State, string, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return "", "", err
	}
	var waiting, unknown, failed, terminal bool
	err = q.QueryRow(ctx, `SELECT COALESCE(bool_or(j.state='waiting'),false),COALESCE(bool_or(j.state='unresolved'),false),COALESCE(bool_or(j.state IN ('failed','cancelled')),false),COALESCE(bool_and(j.state IN ('succeeded','failed','cancelled')),false) FROM want_keep.jobs j JOIN want_keep.receipt_pages page ON(page.household_id,page.id)=(j.household_id,j.resource_id) WHERE j.household_id=$1 AND page.receipt_id=$2 AND j.kind IN ('receipt_extract','receipt_validation')`, p.HouseholdID(), id).Scan(&waiting, &unknown, &failed, &terminal)
	if err != nil {
		return "", "", err
	}
	if unknown {
		return receipt.WaitingAI, "The provider outcome is unknown; processing waits for reconciliation.", nil
	}
	if waiting {
		return receipt.WaitingAI, "Receipt processing is waiting for the provider, private storage or available budget.", nil
	}
	if failed && terminal {
		return receipt.Clarification, "Receipt processing did not complete. Confirm the payment facts.", nil
	}
	return receipt.Processing, "", nil
}

func (s *Store) ReceiptPageProjection(ctx context.Context, p household.Principal, r receipt.Receipt, number int) (json.RawMessage, error) {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return nil, err
	}
	if number < 1 || number > len(r.Pages) {
		return nil, receipt.ErrInvalid
	}
	page := r.Pages[number-1]
	var input []byte
	err = scope.tx.QueryRow(ctx, `SELECT input FROM want_keep.receipt_page_contexts WHERE household_id=$1 AND page_id=$2 AND page_revision=1`, p.HouseholdID(), page.ID).Scan(&input)
	if err == nil {
		return input, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return nil, err
	}
	categories, next, err := s.Categories(ctx, p, category.Filter{State: category.Active}, "", 100)
	if err != nil {
		return nil, err
	}
	type option struct{ Alias, Name, English string }
	options := []option{}
	refs := map[string]receiptapp.CategoryReference{}
	for index, value := range categories {
		alias := fmt.Sprintf("category-%d", index+1)
		options = append(options, option{alias, value.DisplayName(), value.NameEN})
		refs[alias] = receiptapp.CategoryReference{ID: value.ID, Revision: value.Revision}
	}
	input, err = json.Marshal([]struct {
		ID         string   `json:"id"`
		Source     string   `json:"source"`
		Page       int      `json:"page"`
		Hash       string   `json:"hash"`
		Categories []option `json:"categories"`
		Complete   bool     `json:"categoriesComplete"`
	}{{"case-1", "receipt_page", number, page.Hash, options, next == ""}})
	if err != nil {
		return nil, err
	}
	references, err := json.Marshal(refs)
	if err != nil {
		return nil, err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_page_contexts(household_id,page_id,page_revision,input,references_map) VALUES($1,$2,1,$3,$4)`, p.HouseholdID(), page.ID, input, references)
	return input, err
}
func (s *Store) ReceiptPageReferences(ctx context.Context, p household.Principal, pageID string) (map[string]receiptapp.CategoryReference, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	var data []byte
	if err = q.QueryRow(ctx, `SELECT references_map FROM want_keep.receipt_page_contexts WHERE household_id=$1 AND page_id=$2 AND page_revision=1`, p.HouseholdID(), pageID).Scan(&data); err != nil {
		return nil, err
	}
	var refs map[string]receiptapp.CategoryReference
	err = json.Unmarshal(data, &refs)
	return refs, err
}
