package storage

import (
	"context"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) CurrentRefundRevisions(ctx context.Context, p household.Principal, ids []string) (map[string]ledger.Revision, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	result := make(map[string]ledger.Revision, len(ids))
	if len(ids) == 0 {
		return result, nil
	}
	rows, err := q.Query(ctx, `SELECT o.id,o.revision,r.economic_type,r.state,r.cash_date,a.accounting_state,p.account_id,p.amount::text,p.asset,p.role,p.funding,p.treatment,lp.group_id,lp.kind,lp.state,lc.carrier_id,lc.effect_state FROM want_keep.operations o JOIN want_keep.operation_revisions r ON (r.household_id,r.operation_id,r.revision)=(o.household_id,o.id,o.revision) JOIN want_keep.ledger_revision_audit a ON (a.household_id,a.operation_id,a.revision)=(r.household_id,r.operation_id,r.revision) JOIN want_keep.postings p ON (p.household_id,p.operation_id,p.revision)=(r.household_id,r.operation_id,r.revision) LEFT JOIN want_keep.ledger_participations lp ON (lp.household_id,lp.operation_id,lp.revision)=(r.household_id,r.operation_id,r.revision) LEFT JOIN want_keep.ledger_contributions lc ON (lc.household_id,lc.operation_id,lc.revision,lc.position)=(p.household_id,p.operation_id,p.revision,p.position) WHERE o.household_id=$1 AND o.id=ANY($2::uuid[]) AND p.role='principal' AND p.treatment IN ('','movement') ORDER BY o.id,p.position`, p.HouseholdID(), ids)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var revision ledger.Revision
		var cashDate time.Time
		var posting ledger.Posting
		var amount, asset string
		var groupID, kind, state, carrierID, effectState *string
		if err = rows.Scan(&revision.OperationID, &revision.Revision, &revision.Type, &revision.State, &cashDate, &revision.AccountingState, &posting.AccountID, &amount, &asset, &posting.Role, &posting.Funding, &posting.Treatment, &groupID, &kind, &state, &carrierID, &effectState); err != nil {
			return nil, err
		}
		if _, duplicate := result[revision.OperationID]; duplicate {
			return nil, expenses.ErrInvalidRefund
		}
		revision.CashDate, err = calendar.ParseDate(cashDate.Format(time.DateOnly))
		if err != nil {
			return nil, err
		}
		posting.Money, err = money.NewMoney(amount, money.Asset(asset))
		if err != nil {
			return nil, err
		}
		revision.Postings = []ledger.Posting{posting}
		if groupID != nil {
			if kind == nil || state == nil {
				return nil, expenses.ErrInvalidRefund
			}
			revision.Participation = ledger.Participation{GroupID: *groupID, Kind: ledger.ParticipationKind(*kind), State: ledger.ParticipationState(*state)}
			if revision.Participation.State == "linked" {
				if carrierID == nil || effectState == nil {
					return nil, expenses.ErrInvalidRefund
				}
				revision.Participation.Parts = []ledger.Contribution{{CarrierID: *carrierID, State: ledger.State(*effectState)}}
			}
		}
		result[revision.OperationID] = revision
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	unique := map[string]bool{}
	for _, id := range ids {
		unique[id] = true
	}
	if len(result) != len(unique) {
		return nil, ledger.ErrNotFound
	}
	return result, nil
}

func (s *Store) PurchaseValuation(ctx context.Context, p household.Principal, operationID string, revision uint64) (*expenses.ValuationBasis, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	var ref, nativeAmount, nativeAsset, reportingAmount, reportingAsset string
	var compatible bool
	err = q.QueryRow(ctx, `SELECT h.basis_ref,h.native_amount::text,h.native_asset,h.reporting_amount::text,h.reporting_asset,
		(source_revision.occurred_at,source_revision.occurred_ns)=(current_revision.occurred_at,current_revision.occurred_ns)
		AND EXISTS (SELECT 1 FROM want_keep.postings posting WHERE (posting.household_id,posting.operation_id,posting.revision)=(h.household_id,h.operation_id,current_revision.revision)
			AND posting.role='principal' AND posting.treatment IN ('','movement') AND posting.amount=-h.native_amount AND posting.asset=h.native_asset)
		FROM want_keep.transaction_historical_values h
		JOIN want_keep.operation_revisions source_revision ON (source_revision.household_id,source_revision.operation_id,source_revision.revision)=(h.household_id,h.operation_id,h.operation_revision)
		JOIN want_keep.operation_revisions current_revision ON (current_revision.household_id,current_revision.operation_id,current_revision.revision)=(h.household_id,h.operation_id,$3)
		WHERE h.household_id=$1 AND h.operation_id=$2 AND h.operation_revision<=$3
		ORDER BY h.operation_revision DESC LIMIT 1`, p.HouseholdID(), operationID, revision).Scan(&ref, &nativeAmount, &nativeAsset, &reportingAmount, &reportingAsset, &compatible)
	if errors.Is(err, pgx.ErrNoRows) {
		return s.pinPurchaseValuation(ctx, p, operationID, revision)
	}
	if err != nil {
		return nil, err
	}
	if !compatible {
		return nil, expenses.ErrHistoricalBasisConflict
	}
	native, err := money.NewMoney(nativeAmount, money.Asset(nativeAsset))
	if err != nil {
		return nil, err
	}
	reporting, err := money.NewMoney(reportingAmount, money.Asset(reportingAsset))
	if err != nil {
		return nil, err
	}
	return &expenses.ValuationBasis{Purchase: native, Value: reporting, Ref: ref}, nil
}

func (s *Store) Refund(ctx context.Context, p household.Principal, operationID string) (expenses.Refund, bool, error) {
	values, err := s.RefundsForOperation(ctx, p, operationID)
	if err != nil {
		return expenses.Refund{}, false, err
	}
	for _, value := range values {
		if value.OperationID == operationID {
			return value, true, nil
		}
	}
	return expenses.Refund{}, false, nil
}

func (s *Store) ActiveRefundCountForPurchase(ctx context.Context, p household.Principal, purchaseID string) (int, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return 0, err
	}
	var count int
	err = q.QueryRow(ctx, `SELECT count(*) FROM want_keep.refunds r JOIN want_keep.refund_revisions rr ON (rr.household_id,rr.refund_operation_id,rr.revision)=(r.household_id,r.refund_operation_id,r.revision) WHERE r.household_id=$1 AND r.purchase_operation_id=$2 AND rr.state IN ('applied','clarification')`, p.HouseholdID(), purchaseID).Scan(&count)
	return count, err
}

func (s *Store) RefundsForOperation(ctx context.Context, p household.Principal, operationID string) ([]expenses.Refund, error) {
	values, err := s.RefundsForOperations(ctx, p, []string{operationID})
	return values[operationID], err
}

// RefundsForMatchingGroup locates purchase links even when another member carries cash.
func (s *Store) RefundsForMatchingGroup(ctx context.Context, p household.Principal, groupID string) ([]expenses.Refund, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `SELECT r.refund_operation_id::text FROM want_keep.refunds r
		JOIN want_keep.operations o ON (o.household_id,o.id)=(r.household_id,r.refund_operation_id)
		JOIN want_keep.ledger_participations lp ON (lp.household_id,lp.operation_id,lp.revision)=(o.household_id,o.id,o.revision)
		WHERE r.household_id=$1 AND lp.group_id=$2 AND lp.state='linked'`, p.HouseholdID(), groupID)
	if err != nil {
		return nil, err
	}
	ids := []string{}
	for rows.Next() {
		var id string
		if err = rows.Scan(&id); err != nil {
			rows.Close()
			return nil, err
		}
		ids = append(ids, id)
	}
	rows.Close()
	if err = rows.Err(); err != nil {
		return nil, err
	}
	values, err := s.RefundsForOperations(ctx, p, ids)
	if err != nil {
		return nil, err
	}
	links := make([]expenses.Refund, 0, len(ids))
	for _, id := range ids {
		for _, value := range values[id] {
			if value.OperationID == id {
				links = append(links, value)
			}
		}
	}
	return links, nil
}

func (s *Store) RefundsForOperations(ctx context.Context, p household.Principal, operationIDs []string) (map[string][]expenses.Refund, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	return s.loadRefunds(ctx, q, p, operationIDs, 0, nil, 0, 0)
}

func (s *Store) RefundsForOperationAt(ctx context.Context, p household.Principal, operationID string, operationRevision uint64, at calendar.Instant) ([]expenses.Refund, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	values, err := s.loadRefunds(ctx, q, p, []string{operationID}, operationRevision, &at, 0, 0)
	return values[operationID], err
}

func (s *Store) RefundHistory(ctx context.Context, p household.Principal, operationID string, before uint64, limit int) ([]expenses.Refund, uint64, error) {
	if limit < 1 || limit > 100 || before > 9007199254740991 {
		return nil, 0, expenses.ErrInvalidRefund
	}
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, 0, err
	}
	var found bool
	if err = q.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM want_keep.refunds WHERE household_id=$1 AND refund_operation_id=$2)`, p.HouseholdID(), operationID).Scan(&found); err != nil {
		return nil, 0, err
	}
	if !found {
		return nil, 0, ledger.ErrNotFound
	}
	values, err := s.loadRefunds(ctx, q, p, []string{operationID}, 0, nil, before, limit+1)
	if err != nil {
		return nil, 0, err
	}
	history := values[operationID]
	if len(history) <= limit {
		return history, 0, nil
	}
	history = history[:limit]
	return history, history[len(history)-1].Revision, nil
}

func (s *Store) loadRefunds(ctx context.Context, q reader, p household.Principal, operationIDs []string, operationRevision uint64, at *calendar.Instant, historyBefore uint64, historyLimit int) (map[string][]expenses.Refund, error) {
	result := make(map[string][]expenses.Refund, len(operationIDs))
	if len(operationIDs) == 0 {
		return result, nil
	}
	const columns = `r.refund_operation_id,r.purchase_operation_id,rr.revision,rr.purchase_revision,rr.refund_revision,rr.actor_id,rr.reason,rr.state,rr.expense_month,rr.cash_date,rr.amount::text,rr.remaining::text,rr.asset,rr.valuation_basis_native_amount::text,rr.valuation_basis_native_asset,rr.valuation_basis_reporting_amount::text,rr.valuation_basis_reporting_asset,rr.valuation_basis_ref,rr.valuation_amount::text,rr.valuation_asset,rr.recorded_at,rr.recorded_ns`
	query := `SELECT ` + columns + ` FROM want_keep.refunds r JOIN want_keep.refund_revisions rr ON (rr.household_id,rr.refund_operation_id,rr.revision)=(r.household_id,r.refund_operation_id,r.revision) WHERE r.household_id=$1 AND (r.refund_operation_id=ANY($2::uuid[]) OR r.purchase_operation_id=ANY($2::uuid[])) ORDER BY r.refund_operation_id`
	args := []any{p.HouseholdID(), operationIDs}
	if at != nil {
		if len(operationIDs) != 1 || operationRevision < 1 {
			return nil, expenses.ErrInvalidRefund
		}
		cutoff, ns := splitInstant(*at)
		query = `SELECT ` + columns + ` FROM want_keep.refunds r JOIN LATERAL (SELECT * FROM want_keep.refund_revisions candidate WHERE (candidate.household_id,candidate.refund_operation_id)=(r.household_id,r.refund_operation_id) AND (candidate.recorded_at,candidate.recorded_ns)<=($4,$5) AND ((r.refund_operation_id=$2 AND candidate.refund_revision<=$3) OR (r.purchase_operation_id=$2 AND candidate.purchase_revision<=$3)) ORDER BY candidate.recorded_at DESC,candidate.recorded_ns DESC,candidate.revision DESC LIMIT 1) rr ON true WHERE r.household_id=$1 AND (r.refund_operation_id=$2 OR r.purchase_operation_id=$2) ORDER BY r.refund_operation_id`
		args = []any{p.HouseholdID(), operationIDs[0], operationRevision, cutoff, ns}
	} else if historyLimit > 0 {
		if len(operationIDs) != 1 {
			return nil, expenses.ErrInvalidRefund
		}
		query = `SELECT ` + columns + ` FROM want_keep.refunds r JOIN want_keep.refund_revisions rr ON (rr.household_id,rr.refund_operation_id)=(r.household_id,r.refund_operation_id) WHERE r.household_id=$1 AND r.refund_operation_id=$2 AND ($3::bigint=0 OR rr.revision<$3) ORDER BY rr.revision DESC LIMIT $4`
		args = []any{p.HouseholdID(), operationIDs[0], historyBefore, historyLimit}
	}
	rows, err := q.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	values := []expenses.Refund{}
	for rows.Next() {
		var value expenses.Refund
		var month, cashDate, recordedAt time.Time
		var recordedNS int16
		var amount, remaining, asset string
		var basisNativeAmount, basisNativeAsset, basisValueAmount, basisValueAsset, basisRef, valueAmount, valueAsset *string
		if err = rows.Scan(&value.OperationID, &value.PurchaseID, &value.Revision, &value.PurchaseRevision, &value.RefundRevision, &value.ActorID, &value.Reason, &value.State, &month, &cashDate, &amount, &remaining, &asset, &basisNativeAmount, &basisNativeAsset, &basisValueAmount, &basisValueAsset, &basisRef, &valueAmount, &valueAsset, &recordedAt, &recordedNS); err != nil {
			rows.Close()
			return nil, err
		}
		value.ExpenseMonth, err = calendar.ParseMonth(month.Format("2006-01"))
		if err == nil {
			value.CashDate, err = calendar.ParseDate(cashDate.Format(time.DateOnly))
		}
		if err == nil {
			value.RecordedAt, err = restoreInstant(recordedAt, recordedNS)
		}
		if err == nil {
			value.Amount, err = money.NewMoney(amount, money.Asset(asset))
		}
		if err == nil {
			value.Remaining, err = money.NewMoney(remaining, money.Asset(asset))
		}
		if err != nil {
			rows.Close()
			return nil, err
		}
		if basisNativeAmount != nil && basisNativeAsset != nil && basisValueAmount != nil && basisValueAsset != nil && basisRef != nil && valueAmount != nil && valueAsset != nil {
			basisPurchase, parseErr := money.NewMoney(*basisNativeAmount, money.Asset(*basisNativeAsset))
			if parseErr != nil {
				rows.Close()
				return nil, parseErr
			}
			basisValue, parseErr := money.NewMoney(*basisValueAmount, money.Asset(*basisValueAsset))
			if parseErr != nil {
				rows.Close()
				return nil, parseErr
			}
			valuation, parseErr := money.NewMoney(*valueAmount, money.Asset(*valueAsset))
			if parseErr != nil {
				rows.Close()
				return nil, parseErr
			}
			basis := expenses.ValuationBasis{Purchase: basisPurchase, Value: basisValue, Ref: *basisRef}
			value.Valuation = &expenses.Valuation{Value: valuation, Ref: *basisRef, Basis: &basis}
		}
		values = append(values, value)
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	if len(values) == 0 {
		return result, nil
	}
	ids, revisions := make([]string, len(values)), make([]int64, len(values))
	index := make(map[string]*expenses.Refund, len(values))
	for i := range values {
		ids[i], revisions[i] = values[i].OperationID, int64(values[i].Revision)
		index[refundRevisionKey(values[i].OperationID, values[i].Revision)] = &values[i]
	}
	rows, err = q.Query(ctx, `SELECT i.refund_operation_id,i.revision,i.item_id,i.amount::text,i.asset FROM want_keep.refund_item_portions i JOIN unnest($2::uuid[],$3::bigint[]) selected(refund_operation_id,revision) ON (selected.refund_operation_id,selected.revision)=(i.refund_operation_id,i.revision) WHERE i.household_id=$1 ORDER BY i.refund_operation_id,i.revision,i.position`, p.HouseholdID(), ids, revisions)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, itemID, amount, asset string
		var revision uint64
		if err = rows.Scan(&id, &revision, &itemID, &amount, &asset); err != nil {
			rows.Close()
			return nil, err
		}
		value, parseErr := money.NewMoney(amount, money.Asset(asset))
		if parseErr != nil {
			rows.Close()
			return nil, parseErr
		}
		index[refundRevisionKey(id, revision)].Items = append(index[refundRevisionKey(id, revision)].Items, expenses.ItemPortion{ItemID: itemID, Amount: value})
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	rows, err = q.Query(ctx, `SELECT e.refund_operation_id,e.revision,e.basis,e.dimension,COALESCE(e.member_id::text,e.category_id::text,''),e.amount::text,e.asset FROM want_keep.refund_effects e JOIN unnest($2::uuid[],$3::bigint[]) selected(refund_operation_id,revision) ON (selected.refund_operation_id,selected.revision)=(e.refund_operation_id,e.revision) WHERE e.household_id=$1 ORDER BY e.refund_operation_id,e.revision,e.basis,e.position`, p.HouseholdID(), ids, revisions)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var id, basis, dimension, key, amount, asset string
		var revision uint64
		if err = rows.Scan(&id, &revision, &basis, &dimension, &key, &amount, &asset); err != nil {
			rows.Close()
			return nil, err
		}
		value, parseErr := money.NewMoney(amount, money.Asset(asset))
		if parseErr != nil {
			rows.Close()
			return nil, parseErr
		}
		refund := index[refundRevisionKey(id, revision)]
		switch dimension {
		case "member":
			if basis == "native" {
				refund.Members = append(refund.Members, expenses.MemberAmount{MemberID: household.MembershipID(key), Amount: value})
			} else {
				refund.Valuation.Members = append(refund.Valuation.Members, expenses.MemberAmount{MemberID: household.MembershipID(key), Amount: value})
			}
		case "category":
			if basis == "native" {
				refund.Categories = append(refund.Categories, expenses.CategoryAmount{CategoryID: key, Amount: value})
			} else {
				refund.Valuation.Categories = append(refund.Valuation.Categories, expenses.CategoryAmount{CategoryID: key, Amount: value})
			}
		case "unallocated":
			if basis == "native" {
				refund.Unallocated = append(refund.Unallocated, value)
			} else {
				refund.Valuation.Unallocated = append(refund.Valuation.Unallocated, value)
			}
		default:
			rows.Close()
			return nil, expenses.ErrInvalidRefund
		}
	}
	if err = rows.Err(); err != nil {
		rows.Close()
		return nil, err
	}
	rows.Close()
	requested := make(map[string]bool, len(operationIDs))
	for _, id := range operationIDs {
		requested[id] = true
	}
	for i := range values {
		if err = values[i].Validate(); err != nil {
			return nil, err
		}
		if requested[values[i].OperationID] {
			result[values[i].OperationID] = append(result[values[i].OperationID], values[i])
		}
		if requested[values[i].PurchaseID] {
			result[values[i].PurchaseID] = append(result[values[i].PurchaseID], values[i])
		}
	}
	return result, nil
}

func refundRevisionKey(id string, revision uint64) string {
	return id + ":" + strconv.FormatUint(revision, 10)
}
