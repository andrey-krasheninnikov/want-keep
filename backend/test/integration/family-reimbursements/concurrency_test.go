//go:build integration

package familyreimbursements_test

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestConcurrentSettlementsCannotOveruseTransfer(t *testing.T) {
	f := newFixture(t)
	from := f.personalAccount(1, money.RUB, "1000")
	to := f.personalAccount(0, money.RUB, "0")
	transferID := f.transfer(from, to, cash("300", money.RUB), cash("300", money.RUB))
	ids := make([]string, 2)
	for index := range ids {
		result := f.execute(f.p, "reimbursements.create", func(ctx context.Context) (command.Result, error) {
			return f.reimbursements.Create(ctx, f.p, journal.ReimbursementCreateInput{CreditorMemberID: f.members[0].ID, DebtorMemberID: f.members[1].ID, Amount: cash("200", money.RUB), Reason: "Concurrent debt"})
		})
		ids[index] = result.ResourceID
	}
	start := make(chan struct{})
	var succeeded atomic.Int32
	var group sync.WaitGroup
	for _, id := range ids {
		id := id
		group.Add(1)
		go func() {
			defer group.Done()
			<-start
			request := commands.Request{ID: uuid.NewString(), Kind: "reimbursements.settle", PayloadHash: "bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb"}
			value, _ := f.executor.Execute(testContext, f.p, request, func(ctx context.Context) (command.Result, error) {
				return f.reimbursements.Settle(ctx, f.p, id, journal.ReimbursementSettlementInput{ExpectedRevision: 1, TransferExpectedRevision: 1, TransferID: transferID, TransferAmount: cash("200", money.RUB), SettledAmount: cash("200", money.RUB)})
			})
			if value.Status() == command.Succeeded {
				succeeded.Add(1)
			}
		}()
	}
	close(start)
	group.Wait()
	if succeeded.Load() != 1 {
		t.Fatalf("successful settlements=%d want 1", succeeded.Load())
	}
	used, err := f.store.ActiveTransferUsage(testContext, f.p, "transaction:"+transferID, money.RUB)
	if err != nil || used.Amount() != "200" {
		t.Fatalf("used transfer amount=%s err=%v", used.Amount(), err)
	}
}
