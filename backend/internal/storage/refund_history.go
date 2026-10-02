package storage

import (
	"context"
	"encoding/json"
	"time"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type refundRevisionPoint struct {
	OperationID string    `json:"operation_id"`
	Revision    uint64    `json:"operation_revision"`
	RecordedAt  time.Time `json:"recorded_at"`
	RecordedNS  int16     `json:"recorded_ns"`
}

type refundItemValue struct {
	ItemID, Amount, Asset string
}

type refundEffectValue struct {
	Basis, Dimension, Key, Amount, Asset string
}

func (s *Store) RefundsForRevisions(ctx context.Context, p household.Principal, revisions []ledger.Revision) (map[uint64][]expenses.Refund, error) {
	q, err := s.reader(ctx, p)
	if err != nil {
		return nil, err
	}
	result := make(map[uint64][]expenses.Refund, len(revisions))
	if len(revisions) == 0 {
		return result, nil
	}
	if len(revisions) > 200 {
		return nil, expenses.ErrInvalidRefund
	}
	operationID := revisions[0].OperationID
	points := make([]refundRevisionPoint, 0, len(revisions))
	seen := make(map[uint64]bool, len(revisions))
	for _, revision := range revisions {
		if revision.OperationID != operationID || revision.Revision < 1 || revision.RecordedAt.String() == "" {
			return nil, expenses.ErrInvalidRefund
		}
		if seen[revision.Revision] {
			continue
		}
		seen[revision.Revision] = true
		at, ns := splitInstant(revision.RecordedAt)
		points = append(points, refundRevisionPoint{operationID, revision.Revision, at, ns})
	}
	encoded, err := json.Marshal(points)
	if err != nil {
		return nil, err
	}
	rows, err := q.Query(ctx, `WITH points AS (SELECT * FROM jsonb_to_recordset($2::jsonb) AS p(operation_id uuid,operation_revision bigint,recorded_at timestamptz,recorded_ns smallint)) SELECT points.operation_revision,r.refund_operation_id,r.purchase_operation_id,rr.revision,rr.purchase_revision,rr.refund_revision,rr.actor_id,rr.reason,rr.state,rr.expense_month,rr.cash_date,rr.amount::text,rr.remaining::text,rr.asset,rr.valuation_basis_native_amount::text,rr.valuation_basis_native_asset,rr.valuation_basis_reporting_amount::text,rr.valuation_basis_reporting_asset,rr.valuation_basis_ref,rr.valuation_amount::text,rr.valuation_asset,rr.recorded_at,rr.recorded_ns,COALESCE(items.values,'[]'::jsonb),COALESCE(effects.values,'[]'::jsonb) FROM points JOIN want_keep.refunds r ON r.household_id=$1 AND (r.refund_operation_id=points.operation_id OR r.purchase_operation_id=points.operation_id) JOIN LATERAL (SELECT * FROM want_keep.refund_revisions candidate WHERE (candidate.household_id,candidate.refund_operation_id)=(r.household_id,r.refund_operation_id) AND (candidate.recorded_at,candidate.recorded_ns)<=(points.recorded_at,points.recorded_ns) AND ((r.refund_operation_id=points.operation_id AND candidate.refund_revision<=points.operation_revision) OR (r.purchase_operation_id=points.operation_id AND candidate.purchase_revision<=points.operation_revision)) ORDER BY candidate.recorded_at DESC,candidate.recorded_ns DESC,candidate.revision DESC LIMIT 1) rr ON true LEFT JOIN LATERAL (SELECT jsonb_agg(jsonb_build_object('ItemID',i.item_id,'Amount',i.amount::text,'Asset',i.asset) ORDER BY i.position) AS values FROM want_keep.refund_item_portions i WHERE (i.household_id,i.refund_operation_id,i.revision)=(rr.household_id,rr.refund_operation_id,rr.revision)) items ON true LEFT JOIN LATERAL (SELECT jsonb_agg(jsonb_build_object('Basis',e.basis,'Dimension',e.dimension,'Key',COALESCE(e.member_id::text,e.category_id::text,''),'Amount',e.amount::text,'Asset',e.asset) ORDER BY e.basis,e.dimension,e.position) AS values FROM want_keep.refund_effects e WHERE (e.household_id,e.refund_operation_id,e.revision)=(rr.household_id,rr.refund_operation_id,rr.revision)) effects ON true ORDER BY points.operation_revision DESC,r.refund_operation_id`, p.HouseholdID(), encoded)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var revision uint64
		var value expenses.Refund
		var month, cashDate, recordedAt time.Time
		var recordedNS int16
		var amount, remaining, asset string
		var basisNativeAmount, basisNativeAsset, basisValueAmount, basisValueAsset, basisRef, valueAmount, valueAsset *string
		var itemJSON, effectJSON []byte
		if err = rows.Scan(&revision, &value.OperationID, &value.PurchaseID, &value.Revision, &value.PurchaseRevision, &value.RefundRevision, &value.ActorID, &value.Reason, &value.State, &month, &cashDate, &amount, &remaining, &asset, &basisNativeAmount, &basisNativeAsset, &basisValueAmount, &basisValueAsset, &basisRef, &valueAmount, &valueAsset, &recordedAt, &recordedNS, &itemJSON, &effectJSON); err != nil {
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
			return nil, err
		}
		if basisNativeAmount != nil && basisNativeAsset != nil && basisValueAmount != nil && basisValueAsset != nil && basisRef != nil && valueAmount != nil && valueAsset != nil {
			purchase, parseErr := money.NewMoney(*basisNativeAmount, money.Asset(*basisNativeAsset))
			if parseErr != nil {
				return nil, parseErr
			}
			reporting, parseErr := money.NewMoney(*basisValueAmount, money.Asset(*basisValueAsset))
			if parseErr != nil {
				return nil, parseErr
			}
			actual, parseErr := money.NewMoney(*valueAmount, money.Asset(*valueAsset))
			if parseErr != nil {
				return nil, parseErr
			}
			basis := expenses.ValuationBasis{Purchase: purchase, Value: reporting, Ref: *basisRef}
			value.Valuation = &expenses.Valuation{Value: actual, Ref: *basisRef, Basis: &basis}
		}
		var items []refundItemValue
		if err = json.Unmarshal(itemJSON, &items); err != nil {
			return nil, err
		}
		for _, item := range items {
			valueAmount, parseErr := money.NewMoney(item.Amount, money.Asset(item.Asset))
			if parseErr != nil {
				return nil, parseErr
			}
			value.Items = append(value.Items, expenses.ItemPortion{ItemID: item.ItemID, Amount: valueAmount})
		}
		var effects []refundEffectValue
		if err = json.Unmarshal(effectJSON, &effects); err != nil {
			return nil, err
		}
		for _, effect := range effects {
			valueAmount, parseErr := money.NewMoney(effect.Amount, money.Asset(effect.Asset))
			if parseErr != nil {
				return nil, parseErr
			}
			target := &value
			if effect.Basis == "valuation" {
				if value.Valuation == nil {
					return nil, expenses.ErrInvalidRefund
				}
				target = nil
			}
			switch effect.Dimension {
			case "member":
				member := expenses.MemberAmount{MemberID: household.MembershipID(effect.Key), Amount: valueAmount}
				if target == nil {
					value.Valuation.Members = append(value.Valuation.Members, member)
				} else {
					value.Members = append(value.Members, member)
				}
			case "category":
				category := expenses.CategoryAmount{CategoryID: effect.Key, Amount: valueAmount}
				if target == nil {
					value.Valuation.Categories = append(value.Valuation.Categories, category)
				} else {
					value.Categories = append(value.Categories, category)
				}
			case "unallocated":
				if target == nil {
					value.Valuation.Unallocated = append(value.Valuation.Unallocated, valueAmount)
				} else {
					value.Unallocated = append(value.Unallocated, valueAmount)
				}
			default:
				return nil, expenses.ErrInvalidRefund
			}
		}
		if err = value.Validate(); err != nil {
			return nil, err
		}
		result[revision] = append(result[revision], value)
	}
	return result, rows.Err()
}
