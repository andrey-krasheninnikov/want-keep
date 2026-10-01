package domain

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"slices"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type SettlementTransfer struct {
	Key, SelectedID, FromAccountID, ToAccountID string
	SelectedRevision                            uint64
	FromOwnerID, ToOwnerID                      household.UserID
	Sent, Received                              money.Money
	OperationIDs                                []string
}

func (t SettlementTransfer) Validate() error {
	if t.Key == "" || t.SelectedID == "" || t.SelectedRevision < 1 || t.SelectedRevision > MaxReimbursementRevision || t.FromAccountID == "" || t.ToAccountID == "" || t.FromAccountID == t.ToAccountID || t.FromOwnerID == "" || t.ToOwnerID == "" || t.FromOwnerID == t.ToOwnerID || t.Sent.Validate() != nil || t.Received.Validate() != nil || t.Sent.Sign() <= 0 || t.Received.Sign() <= 0 || len(t.OperationIDs) < 1 || len(t.OperationIDs) > 100 {
		return ErrInvalidReimbursement
	}
	seen := map[string]bool{}
	for _, id := range t.OperationIDs {
		if id == "" || seen[id] {
			return ErrInvalidReimbursement
		}
		seen[id] = true
	}
	return nil
}

func (t SettlementTransfer) Fingerprint() string {
	operations := slices.Clone(t.OperationIDs)
	slices.Sort(operations)
	data, _ := json.Marshal([]any{t.Key, t.FromAccountID, t.ToAccountID, t.FromOwnerID, t.ToOwnerID, t.Sent.Amount(), t.Sent.Asset(), t.Received.Amount(), t.Received.Asset(), operations})
	digest := sha256.Sum256(data)
	return hex.EncodeToString(digest[:])
}
