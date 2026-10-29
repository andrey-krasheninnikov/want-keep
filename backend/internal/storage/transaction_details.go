package storage

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func (s *Store) saveTransactionDetails(ctx context.Context, r ledger.Revision) error {
	scope, err := s.familyScope(ctx)
	if err != nil {
		return err
	}
	var at, ns any
	if r.PostedAt.String() != "" {
		at, ns = splitInstant(r.PostedAt)
	}
	_, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.transaction_details(household_id,operation_id,revision,posted_at,posted_ns,timezone,origin,fee_knowledge,pnl_basis,merchant,note,attachment_id,allocation_reason,category_id,merchant_id,occurred_precision) VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,NULLIF($12,'')::uuid,$13,NULLIF($14,'')::uuid,NULLIF($15,'')::uuid,$16)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, at, ns, r.Timezone.String(), r.Origin, r.FeeKnowledge, r.PnLBasis, r.Merchant, r.Note, r.AttachmentID, r.AllocationReason, r.CategoryID, r.MerchantID, r.OccurredPrecision)
	if err != nil {
		return err
	}
	for position, item := range r.ReceiptItems {
		if _, err = scope.tx.Exec(ctx, `INSERT INTO want_keep.receipt_items(household_id,operation_id,revision,position,id,name,quantity,gross,discount,asset,category_id) VALUES($1,$2,$3,$4,$5,$6,$7::numeric,$8::numeric,$9::numeric,$10,NULLIF($11,'')::uuid)`, scope.principal.HouseholdID(), r.OperationID, r.Revision, position, item.ID, item.Name, item.Quantity, item.Gross.Amount(), item.Discount.Amount(), item.Gross.Asset(), item.CategoryID); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadTransactionDetails(ctx context.Context, q reader, p household.Principal, r *ledger.Revision) error {
	var at *time.Time
	var ns *int16
	var timezone string
	err := q.QueryRow(ctx, `SELECT posted_at,posted_ns,timezone,origin,fee_knowledge,pnl_basis,merchant,note,COALESCE(attachment_id::text,''),allocation_reason,COALESCE(category_id::text,''),COALESCE(merchant_id::text,''),occurred_precision FROM want_keep.transaction_details WHERE household_id=$1 AND operation_id=$2 AND revision=$3`, p.HouseholdID(), r.OperationID, r.Revision).Scan(&at, &ns, &timezone, &r.Origin, &r.FeeKnowledge, &r.PnLBasis, &r.Merchant, &r.Note, &r.AttachmentID, &r.AllocationReason, &r.CategoryID, &r.MerchantID, &r.OccurredPrecision)
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
		r.PostedAt, err = restoreInstant(*at, *ns)
		if err != nil {
			return err
		}
	}
	if timezone != "" {
		r.Timezone, err = calendar.ParseTimezone(timezone)
	}
	if err != nil {
		return err
	}
	rows, err := q.Query(ctx, `SELECT id,name,quantity::text,gross::text,discount::text,asset,COALESCE(category_id::text,'') FROM want_keep.receipt_items WHERE household_id=$1 AND operation_id=$2 AND revision=$3 ORDER BY position`, p.HouseholdID(), r.OperationID, r.Revision)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var item ledger.ReceiptItem
		var gross, discount, asset string
		if err = rows.Scan(&item.ID, &item.Name, &item.Quantity, &gross, &discount, &asset, &item.CategoryID); err != nil {
			return err
		}
		item.Gross, err = money.NewMoney(gross, money.Asset(asset))
		if err != nil {
			return err
		}
		item.Discount, err = money.NewMoney(discount, money.Asset(asset))
		if err != nil {
			return err
		}
		r.ReceiptItems = append(r.ReceiptItems, item)
	}
	return rows.Err()
}
