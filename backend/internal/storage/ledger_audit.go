package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func (s *Store) FirstLedgerRuleBoundary(ctx context.Context, p household.Principal, operationID string) (uint64, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return 0, err
	}
	var boundary uint64
	err = q.QueryRow(ctx, `SELECT allocation_rule_boundary FROM want_keep.ledger_revision_audit WHERE household_id=$1 AND operation_id=$2 AND revision=1`, p.HouseholdID(), operationID).Scan(&boundary)
	return boundary, err
}

func (s *Store) SaveDecision(ctx context.Context, d ledger.Decision) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if err = d.Validate(); err != nil {
		return err
	}
	if d.ActorID != scope.principal.UserID() {
		return household.ErrForbidden
	}
	family := scope.principal.HouseholdID()
	at, ns := splitInstant(d.At)
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_decisions(household_id,id,actor_id,kind,reason,recorded_at,recorded_ns,undo_of) VALUES($1,$2,$3,$4,$5,$6,$7,NULLIF($8,'')::uuid)`, family, d.ID, d.ActorID, d.Kind, d.Reason, at, ns, d.UndoOf)
	if err != nil {
		return err
	}
	for _, e := range d.Entries {
		fields := []string{}
		for _, f := range e.Fields {
			fields = append(fields, string(f))
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_decision_entries(household_id,decision_id,operation_id,before_revision,after_revision,fields) VALUES($1,$2,$3,$4,$5,$6)`, family, d.ID, e.OperationID, e.Before, e.After, fields); err != nil {
			return err
		}
	}
	for _, e := range d.Evidence {
		if err = s.requireDecisionEvidence(ctx, scope, e); err != nil {
			return err
		}
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_decision_evidence(household_id,decision_id,kind,evidence_id,evidence_revision) VALUES($1,$2,$3,$4,$5)`, family, d.ID, e.Kind, e.ID, e.Revision); err != nil {
			return err
		}
	}
	return nil
}
func (s *Store) requireDecisionEvidence(ctx context.Context, scope *transactionScope, e ledger.Evidence) error {
	var exists bool
	var query string
	switch e.Kind {
	case "source":
		query = `SELECT EXISTS(SELECT 1 FROM want_keep.source_revisions WHERE household_id=$1 AND source_id=$2 AND revision=$3)`
	case "attachment":
		query = `SELECT EXISTS(SELECT 1 FROM want_keep.attachments WHERE household_id=$1 AND id=$2 AND $3::bigint=1 AND state='accepted')`
	case "review":
		query = `SELECT EXISTS(SELECT 1 FROM want_keep.ledger_review_results WHERE household_id=$1 AND operation_id=$2 AND revision=$3)`
	default:
		return ledger.ErrInvalidRevision
	}
	if err := scope.tx.QueryRow(ctx, query, scope.principal.HouseholdID(), e.ID, e.Revision).Scan(&exists); err != nil {
		return err
	}
	if !exists {
		return ErrNotFound
	}
	return nil
}
func (s *Store) Decision(ctx context.Context, p household.Principal, id string) (ledger.Decision, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return ledger.Decision{}, err
	}
	d := ledger.Decision{ID: id}
	var at time.Time
	var ns int16
	err = q.QueryRow(ctx, `SELECT actor_id,kind,reason,recorded_at,recorded_ns,COALESCE(undo_of::text,'') FROM want_keep.ledger_decisions WHERE household_id=$1 AND id=$2`, p.HouseholdID(), id).Scan(&d.ActorID, &d.Kind, &d.Reason, &at, &ns, &d.UndoOf)
	if errors.Is(err, pgx.ErrNoRows) {
		return d, errors.Join(ErrNotFound, ledger.ErrNotFound)
	}
	if err != nil {
		return d, err
	}
	d.At, err = restoreInstant(at, ns)
	if err != nil {
		return d, err
	}
	rows, err := q.Query(ctx, `SELECT operation_id,before_revision,after_revision,fields FROM want_keep.ledger_decision_entries WHERE household_id=$1 AND decision_id=$2 ORDER BY operation_id`, p.HouseholdID(), id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var e ledger.DecisionEntry
		var fields []string
		if err = rows.Scan(&e.OperationID, &e.Before, &e.After, &fields); err != nil {
			return d, err
		}
		for _, f := range fields {
			e.Fields = append(e.Fields, ledger.Field(f))
		}
		d.Entries = append(d.Entries, e)
	}
	if err = rows.Err(); err != nil {
		return d, err
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT kind,evidence_id,evidence_revision FROM want_keep.ledger_decision_evidence WHERE household_id=$1 AND decision_id=$2 ORDER BY kind,evidence_id,evidence_revision`, p.HouseholdID(), id)
	if err != nil {
		return d, err
	}
	defer rows.Close()
	for rows.Next() {
		var e ledger.Evidence
		if err = rows.Scan(&e.Kind, &e.ID, &e.Revision); err != nil {
			return d, err
		}
		d.Evidence = append(d.Evidence, e)
	}
	if err = rows.Err(); err != nil {
		return d, err
	}
	return d, d.Validate()
}
func (s *Store) DecisionUndone(ctx context.Context, p household.Principal, id string) (bool, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return false, err
	}
	var exists bool
	err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.ledger_decisions WHERE household_id=$1 AND undo_of=$2)`, p.HouseholdID(), id).Scan(&exists)
	return exists, err
}

func (s *Store) saveLedgerAudit(ctx context.Context, r ledger.Revision) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	family := scope.principal.HouseholdID()
	var at, ns any
	if r.RecordedAt.String() != "" {
		at, ns = splitInstant(r.RecordedAt)
	}
	var boundary uint64
	if r.Revision == 1 {
		err = scope.tx.QueryRow(ctx, `SELECT allocation_rule_sequence FROM want_keep.households WHERE id=$1`, family).Scan(&boundary)
	} else {
		err = scope.tx.QueryRow(ctx, `SELECT allocation_rule_boundary FROM want_keep.ledger_revision_audit WHERE household_id=$1 AND operation_id=$2 AND revision=1`, family, r.OperationID).Scan(&boundary)
	}
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_revision_audit(household_id,operation_id,revision,accounting_state,decision_id,recorded_at,recorded_ns,allocation_rule_boundary) VALUES($1,$2,$3,$4,NULLIF($5,'')::uuid,COALESCE($6,clock_timestamp()),COALESCE($7,0),$8)`, family, r.OperationID, r.Revision, r.Accounting(), r.DecisionID, at, ns, boundary)
	if err != nil {
		return err
	}
	if r.Revision > 1 {
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_revision_sources(household_id,operation_id,revision,source_id,source_revision) SELECT household_id,operation_id,$3,source_id,source_revision FROM want_keep.ledger_revision_sources WHERE household_id=$1 AND operation_id=$2 AND revision=$4`, family, r.OperationID, r.Revision, r.Revision-1)
		if err != nil {
			return err
		}
	}
	fields := map[ledger.Field]uint64{}
	for f, v := range r.FieldVersions {
		fields[f] = v
	}
	for f, p := range r.Protections {
		if fields[f] == 0 {
			fields[f] = p.Revision
		}
	}
	if r.HumanOverride && len(r.Protections) == 0 {
		fields[ledger.LegacyField] = r.Revision
	}
	for f, v := range fields {
		p, protected := r.Protections[f]
		if f == ledger.LegacyField {
			protected = true
			if p.Revision == 0 {
				p.Revision = v
			}
		}
		var protectionRevision any
		if protected {
			protectionRevision = p.Revision
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_field_origins(household_id,operation_id,revision,field,changed_revision,protected,decision_id,protection_revision) VALUES($1,$2,$3,$4,$5,$6,NULLIF($7,'')::uuid,$8)`, family, r.OperationID, r.Revision, f, v, protected, p.DecisionID, protectionRevision)
		if err != nil {
			return err
		}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ledger_review_requests(household_id,operation_id,revision) VALUES($1,$2,$3)`, family, r.OperationID, r.Revision)
	if err != nil {
		return err
	}
	if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.ai_request_subjects(household_id,resource_id,resource_revision,operation_id,operation_revision) VALUES($1,$2,$3,$2,$3)`, family, r.OperationID, r.Revision); err != nil {
		return err
	}
	return s.EmitEvent(ctx, "transaction", r.OperationID, r.Revision, "transaction.changed")
}

func (s *Store) loadLedgerAudit(ctx context.Context, q reader, p household.Principal, r *ledger.Revision) error {
	var at *time.Time
	var ns *int16
	err := q.QueryRow(ctx, `SELECT a.accounting_state,COALESCE(a.decision_id::text,''),a.recorded_at,a.recorded_ns,COALESCE(v.state,'waiting') FROM want_keep.ledger_revision_audit a LEFT JOIN want_keep.ledger_review_results v USING(household_id,operation_id,revision) WHERE a.household_id=$1 AND a.operation_id=$2 AND a.revision=$3`, p.HouseholdID(), r.OperationID, r.Revision).Scan(&r.AccountingState, &r.DecisionID, &at, &ns, &r.ReviewState)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	if at != nil {
		if ns == nil {
			return ErrStorage
		}
		r.RecordedAt, err = restoreInstant(*at, *ns)
		if err != nil {
			return err
		}
	}
	r.Protections = map[ledger.Field]ledger.Protection{}
	r.FieldVersions = map[ledger.Field]uint64{}
	rows, err := q.Query(ctx, `SELECT field,changed_revision,protected,COALESCE(decision_id::text,''),protection_revision FROM want_keep.ledger_field_origins WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), r.OperationID, r.Revision)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var f ledger.Field
		var v uint64
		var protected bool
		var decision string
		var protectionRevision *uint64
		if err = rows.Scan(&f, &v, &protected, &decision, &protectionRevision); err != nil {
			return err
		}
		r.FieldVersions[f] = v
		if protected {
			if protectionRevision == nil {
				return ErrStorage
			}
			r.Protections[f] = ledger.Protection{DecisionID: decision, Revision: *protectionRevision}
		}
	}
	if err = rows.Err(); err != nil {
		return err
	}
	rows.Close()
	source, err := s.sourceFact(ctx, p, r.OperationID, r.Revision)
	if errors.Is(err, ledger.ErrSourceAmbiguous) {
		r.SourceConflict = true
		return nil
	}
	if err != nil {
		return err
	}
	r.SourceConflict = r.ConflictsWithSource(source)
	return err
}
