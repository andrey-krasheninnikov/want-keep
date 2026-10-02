//go:build integration

package refunds_test

import (
	"context"
	"strings"
	"sync"
	"testing"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	expensedomain "github.com/pchkauu/want-keep/backend/internal/expenses/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestConcurrentRefundsCannotExceedPurchase(t *testing.T) {
	f := newFixture(t)
	client := f.client(f.p)
	accountID := f.account(money.RUB, "5000")
	purchase := createExpense(t, client, accountID, money.RUB, "1000", "500")
	service := expenses.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	start := make(chan struct{})
	type outcome struct {
		command command.Command
		err     error
	}
	results := make(chan outcome, 2)
	var wait sync.WaitGroup
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			<-start
			request := commands.Request{ID: uuid.NewString(), Kind: "transactions.refund", PayloadHash: strings.Repeat("a", 64)}
			result, err := f.executor.Execute(context.Background(), f.p, request, func(ctx context.Context) (command.Result, error) {
				return service.Create(ctx, f.p, expenses.CreateInput{PurchaseID: purchase.Result.Id, PurchaseExpectedRevision: 1, AccountID: accountID, At: f.now, Amount: cash("600", money.RUB), Items: []expensedomain.ItemPortion{}, Reason: "Concurrent return"})
			})
			results <- outcome{command: result, err: err}
		}()
	}
	close(start)
	wait.Wait()
	close(results)
	succeeded, rejected := 0, 0
	for result := range results {
		if result.err != nil {
			t.Fatalf("unexpected infrastructure error: %v", result.err)
		}
		switch result.command.Status() {
		case command.Succeeded:
			succeeded++
		case command.Failed:
			if result.command.ErrorCode() != "refund_exceeds_purchase" {
				t.Fatalf("unexpected rejection: %s", result.command.ErrorCode())
			}
			rejected++
		default:
			t.Fatalf("unfinished command: %s", result.command.Status())
		}
	}
	if succeeded != 1 || rejected != 1 || f.available(accountID, f.p) != "4600" {
		t.Fatalf("succeeded=%d rejected=%d balance=%s", succeeded, rejected, f.available(accountID, f.p))
	}
}
