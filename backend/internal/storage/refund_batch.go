package storage

import (
	"context"
	"encoding/json"
	"time"

	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type refundPointerRow struct {
	OperationID string `json:"refund_operation_id"`
	PurchaseID  string `json:"purchase_operation_id"`
	Revision    uint64 `json:"revision"`
	Expected    uint64 `json:"expected"`
}

type refundRevisionRow struct {
	OperationID, PurchaseID, ActorID, Reason, State, ExpenseMonth, CashDate, Amount, Remaining, Asset string
	Revision, PurchaseRevision, RefundRevision                                                        uint64
	BasisNativeAmount, BasisNativeAsset, BasisValueAmount, BasisValueAsset, BasisRef                  *string
	ValueAmount, ValueAsset                                                                           *string
	RecordedAt                                                                                        time.Time
	RecordedNS                                                                                        int16
}

type refundItemRow struct {
	OperationID, PurchaseID, ItemID, Amount, Asset string
	Revision, PurchaseRevision                     uint64
	Position                                       int
}

type refundEffectRow struct {
	OperationID, Basis, Dimension, Key, Amount, Asset string
	Revision                                          uint64
	Position                                          int
}

type refundEventRow struct {
	OperationID, JobID, ActorID string
	Revision                    uint64
	RecordedAt                  time.Time
}

// SaveRefunds persists all recalculated links and their events in the caller's transaction.
func (s *Store) SaveRefunds(ctx context.Context, p household.Principal, refunds []expenses.Refund, expectedRevisions map[string]uint64) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	if len(refunds) == 0 || len(refunds) != len(expectedRevisions) {
		return expenses.ErrInvalidRefund
	}
	pointers := make([]refundPointerRow, 0, len(refunds))
	revisions := make([]refundRevisionRow, 0, len(refunds))
	items := []refundItemRow{}
	effects := []refundEffectRow{}
	events := make([]refundEventRow, 0, len(refunds))
	seen := make(map[string]bool, len(refunds))
	newCount, updateCount := int64(0), int64(0)
	for _, refund := range refunds {
		expected, ok := expectedRevisions[refund.OperationID]
		if !ok || seen[refund.OperationID] || refund.Validate() != nil || refund.ActorID != p.UserID() || expected >= command.MaxRevision || refund.Revision != expected+1 {
			return expenses.ErrInvalidRefund
		}
		seen[refund.OperationID] = true
		if expected == 0 {
			newCount++
		} else {
			updateCount++
		}
		pointers = append(pointers, refundPointerRow{refund.OperationID, refund.PurchaseID, refund.Revision, expected})
		at, ns := splitInstant(refund.RecordedAt)
		row := refundRevisionRow{OperationID: refund.OperationID, PurchaseID: refund.PurchaseID, ActorID: string(refund.ActorID), Reason: refund.Reason, State: string(refund.State), ExpenseMonth: refund.ExpenseMonth.String() + "-01", CashDate: refund.CashDate.String(), Amount: refund.Amount.Amount(), Remaining: refund.Remaining.Amount(), Asset: string(refund.Amount.Asset()), Revision: refund.Revision, PurchaseRevision: refund.PurchaseRevision, RefundRevision: refund.RefundRevision, RecordedAt: at, RecordedNS: ns}
		if refund.Valuation != nil {
			basis := refund.Valuation.Basis
			if basis == nil || basis.Ref != refund.Valuation.Ref {
				return expenses.ErrInvalidRefund
			}
			row.BasisNativeAmount = stringPointer(basis.Purchase.Amount())
			row.BasisNativeAsset = stringPointer(string(basis.Purchase.Asset()))
			row.BasisValueAmount = stringPointer(basis.Value.Amount())
			row.BasisValueAsset = stringPointer(string(basis.Value.Asset()))
			row.BasisRef = stringPointer(basis.Ref)
			row.ValueAmount = stringPointer(refund.Valuation.Value.Amount())
			row.ValueAsset = stringPointer(string(refund.Valuation.Value.Asset()))
		}
		revisions = append(revisions, row)
		for position, item := range refund.Items {
			items = append(items, refundItemRow{refund.OperationID, refund.PurchaseID, item.ItemID, item.Amount.Amount(), string(item.Amount.Asset()), refund.Revision, refund.PurchaseRevision, position})
		}
		effects = appendRefundEffects(effects, refund.OperationID, refund.Revision, "native", refund.Members, refund.Categories, refund.Unallocated)
		if refund.Valuation != nil {
			effects = appendRefundEffects(effects, refund.OperationID, refund.Revision, "valuation", refund.Valuation.Members, refund.Valuation.Categories, refund.Valuation.Unallocated)
		}
		events = append(events, refundEventRow{refund.OperationID, newID(), string(p.UserID()), refund.Revision, at})
	}
	family := p.HouseholdID()
	pointerJSON, err := json.Marshal(pointers)
	if err != nil {
		return err
	}
	if newCount > 0 {
		var inserted int64
		err = scope.tx.QueryRow(ctx, `WITH input AS (SELECT * FROM jsonb_to_recordset($2::jsonb) AS x(refund_operation_id uuid,purchase_operation_id uuid,revision bigint,expected bigint)), inserted AS (INSERT INTO want_keep.refunds(household_id,refund_operation_id,purchase_operation_id,revision) SELECT $1,refund_operation_id,purchase_operation_id,revision FROM input WHERE expected=0 ON CONFLICT DO NOTHING RETURNING 1) SELECT count(*) FROM inserted`, family, pointerJSON).Scan(&inserted)
		if err != nil {
			return err
		}
		if inserted != newCount {
			return command.ErrVersionConflict
		}
	}
	if updateCount > 0 {
		tag, updateErr := scope.tx.Exec(ctx, `WITH input AS (SELECT * FROM jsonb_to_recordset($2::jsonb) AS x(refund_operation_id uuid,purchase_operation_id uuid,revision bigint,expected bigint)) UPDATE want_keep.refunds r SET revision=input.revision FROM input WHERE r.household_id=$1 AND (r.refund_operation_id,r.purchase_operation_id,r.revision)=(input.refund_operation_id,input.purchase_operation_id,input.expected) AND input.expected>0`, family, pointerJSON)
		if updateErr != nil {
			return updateErr
		}
		if tag.RowsAffected() != updateCount {
			return command.ErrVersionConflict
		}
	}
	revisionJSON, err := json.Marshal(revisions)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_revisions(household_id,refund_operation_id,revision,purchase_operation_id,purchase_revision,refund_revision,actor_id,reason,state,expense_month,cash_date,amount,remaining,asset,valuation_basis_native_amount,valuation_basis_native_asset,valuation_basis_reporting_amount,valuation_basis_reporting_asset,valuation_basis_ref,valuation_amount,valuation_asset,recorded_at,recorded_ns) SELECT $1,x."OperationID",x."Revision",x."PurchaseID",x."PurchaseRevision",x."RefundRevision",x."ActorID",x."Reason",x."State",x."ExpenseMonth"::date,x."CashDate"::date,x."Amount"::numeric,x."Remaining"::numeric,x."Asset",x."BasisNativeAmount"::numeric,x."BasisNativeAsset",x."BasisValueAmount"::numeric,x."BasisValueAsset",x."BasisRef",x."ValueAmount"::numeric,x."ValueAsset",x."RecordedAt",x."RecordedNS" FROM jsonb_to_recordset($2::jsonb) AS x("OperationID" uuid,"PurchaseID" uuid,"ActorID" uuid,"Reason" text,"State" text,"ExpenseMonth" text,"CashDate" text,"Amount" text,"Remaining" text,"Asset" text,"Revision" bigint,"PurchaseRevision" bigint,"RefundRevision" bigint,"BasisNativeAmount" text,"BasisNativeAsset" text,"BasisValueAmount" text,"BasisValueAsset" text,"BasisRef" text,"ValueAmount" text,"ValueAsset" text,"RecordedAt" timestamptz,"RecordedNS" smallint)`, family, revisionJSON)
	if err != nil {
		return err
	}
	if len(items) > 0 {
		itemJSON, marshalErr := json.Marshal(items)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_item_portions(household_id,refund_operation_id,revision,purchase_operation_id,purchase_revision,position,item_id,amount,asset) SELECT $1,x."OperationID",x."Revision",x."PurchaseID",x."PurchaseRevision",x."Position",x."ItemID",x."Amount"::numeric,x."Asset" FROM jsonb_to_recordset($2::jsonb) AS x("OperationID" uuid,"PurchaseID" uuid,"ItemID" uuid,"Amount" text,"Asset" text,"Revision" bigint,"PurchaseRevision" bigint,"Position" integer)`, family, itemJSON)
		if err != nil {
			return err
		}
	}
	if len(effects) > 0 {
		effectJSON, marshalErr := json.Marshal(effects)
		if marshalErr != nil {
			return marshalErr
		}
		_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_effects(household_id,refund_operation_id,revision,basis,dimension,position,member_id,category_id,amount,asset) SELECT $1,x."OperationID",x."Revision",x."Basis",x."Dimension",x."Position",CASE WHEN x."Dimension"='member' THEN x."Key"::uuid END,CASE WHEN x."Dimension"='category' AND x."Key"<>'' THEN x."Key"::uuid END,x."Amount"::numeric,x."Asset" FROM jsonb_to_recordset($2::jsonb) AS x("OperationID" uuid,"Basis" text,"Dimension" text,"Key" text,"Amount" text,"Asset" text,"Revision" bigint,"Position" integer)`, family, effectJSON)
		if err != nil {
			return err
		}
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.refund_review_requests(household_id,refund_operation_id,revision,requested_at,requested_ns) SELECT $1,x."OperationID",x."Revision",x."RecordedAt",x."RecordedNS" FROM jsonb_to_recordset($2::jsonb) AS x("OperationID" uuid,"Revision" bigint,"RecordedAt" timestamptz,"RecordedNS" smallint)`, family, revisionJSON)
	if err != nil {
		return err
	}
	eventJSON, err := json.Marshal(events)
	if err != nil {
		return err
	}
	_, err = scope.tx.Exec(ctx, `WITH input AS (SELECT * FROM jsonb_to_recordset($2::jsonb) AS x("OperationID" uuid,"JobID" uuid,"ActorID" uuid,"Revision" bigint,"RecordedAt" timestamptz)), queued AS (INSERT INTO want_keep.jobs(household_id,id,actor_id,kind,state,max_attempts,available_at,deadline) SELECT $1,"JobID","ActorID",'outbox','ready',5,"RecordedAt","RecordedAt"+INTERVAL '24 hours' FROM input RETURNING id) INSERT INTO want_keep.outbox(household_id,id,actor_id,resource_type,resource_id,revision,event_type) SELECT $1,"JobID","ActorID",'refund',"OperationID","Revision",'refund.changed' FROM input JOIN queued ON queued.id=input."JobID"`, family, eventJSON)
	return err
}

func appendRefundEffects(rows []refundEffectRow, operationID string, revision uint64, basis string, members []expenses.MemberAmount, categories []expenses.CategoryAmount, unallocated []money.Money) []refundEffectRow {
	for position, effect := range members {
		rows = append(rows, refundEffectRow{operationID, basis, "member", string(effect.MemberID), effect.Amount.Amount(), string(effect.Amount.Asset()), revision, position})
	}
	for position, effect := range categories {
		rows = append(rows, refundEffectRow{operationID, basis, "category", effect.CategoryID, effect.Amount.Amount(), string(effect.Amount.Asset()), revision, position})
	}
	for position, effect := range unallocated {
		rows = append(rows, refundEffectRow{operationID, basis, "unallocated", "", effect.Amount(), string(effect.Asset()), revision, position})
	}
	return rows
}

func stringPointer(value string) *string { return &value }
