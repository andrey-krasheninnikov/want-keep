package reimbursements

import (
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func householdMembership(value string) household.MembershipID { return household.MembershipID(value) }

func reimbursementDTO(value ledger.Reimbursement) (generated.Reimbursement, error) {
	principal, err := (contract.MoneyConverter{}).ToDTO(value.Principal)
	if err != nil {
		return generated.Reimbursement{}, err
	}
	outstanding, err := (contract.MoneyConverter{}).ToDTO(value.Outstanding)
	if err != nil {
		return generated.Reimbursement{}, err
	}
	recordedAt, err := (contract.CalendarConverter{}).InstantToDTO(value.RecordedAt)
	if err != nil {
		return generated.Reimbursement{}, err
	}
	out := generated.Reimbursement{Id: value.ID, Revision: int64(value.Revision), DecisionId: value.DecisionID, State: generated.ReimbursementState(value.State), CreditorMemberId: string(value.CreditorMemberID), DebtorMemberId: string(value.DebtorMemberID), Principal: principal, Outstanding: outstanding, Reason: value.Reason, ActorId: string(value.ActorID), RecordedAt: recordedAt, Settlements: []generated.ReimbursementSettlement{}}
	if value.ExpenseID != "" {
		expenseID := generated.ID(value.ExpenseID)
		expenseRevision := generated.Revision(value.ExpenseRevision)
		out.ExpenseId, out.ExpenseRevision = &expenseID, &expenseRevision
	}
	if value.AttentionReason != "" {
		out.AttentionReason = &value.AttentionReason
	}
	for _, settlement := range value.Settlements {
		transfer, err := (contract.MoneyConverter{}).ToDTO(settlement.TransferAmount)
		if err != nil {
			return generated.Reimbursement{}, err
		}
		settled, err := (contract.MoneyConverter{}).ToDTO(settlement.SettledAmount)
		if err != nil {
			return generated.Reimbursement{}, err
		}
		at, err := (contract.CalendarConverter{}).InstantToDTO(settlement.RecordedAt)
		if err != nil {
			return generated.Reimbursement{}, err
		}
		item := generated.ReimbursementSettlement{Id: settlement.ID, State: generated.ReimbursementSettlementState(settlement.State), TransferId: settlement.TransferID, TransferRevision: int64(settlement.TransferRevision), TransferAmount: generated.PositiveMoney{Amount: transfer.Amount, Asset: transfer.Asset}, SettledAmount: generated.PositiveMoney{Amount: settled.Amount, Asset: settled.Asset}, ActorId: string(settlement.ActorID), RecordedAt: at, OperationIds: []generated.ID{}}
		for _, id := range settlement.OperationIDs {
			item.OperationIds = append(item.OperationIds, id)
		}
		out.Settlements = append(out.Settlements, item)
	}
	return out, nil
}
