//go:build integration

package aicommands_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	rule "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

func TestRuleReviewPreservesMixedReceipt(t *testing.T) {
	f := newFixture(t)
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Member B"}
	member := household.Membership{ID: household.MembershipID(uuid.NewString()), UserID: user.ID, HouseholdID: f.family.ID, Active: true}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.AddMember(ctx, user, member) }); err != nil {
		t.Fatal(err)
	}
	var sharedCategory, personalCategory string
	if err := f.admin.QueryRow(testContext, `SELECT min(id::text),max(id::text) FROM want_keep.categories WHERE household_id=$1`, f.family.ID).Scan(&sharedCategory, &personalCategory); err != nil {
		t.Fatal(err)
	}
	catalog := category.NewService(f.store, uuid.NewString)
	merchant := f.command("merchant.create", func(ctx context.Context) (command.Result, error) {
		return catalog.CreateMerchant(ctx, f.p, category.MerchantInput{Name: "Synthetic supermarket"})
	})
	if merchant.Status() != command.Succeeded {
		t.Fatal(merchant.ErrorCode())
	}
	merchantResult, _ := merchant.Result()
	f.step(jobs.Outbox, jobapp.OutboxHandler{Repository: f.store})
	allocations := allocation.NewService(f.store, func() calendar.Instant { return f.now }, uuid.NewString)
	inputs := []allocation.RuleInput{
		{Priority: 20, State: rule.Active, Condition: rule.Condition{MerchantID: merchantResult.ResourceID}, Shares: []rule.Share{{MemberID: f.membership.ID, Value: "50"}, {MemberID: member.ID, Value: "50"}}},
		{Priority: 10, State: rule.Active, Condition: rule.Condition{CategoryID: personalCategory}, Shares: []rule.Share{{MemberID: f.membership.ID, Value: "100"}}},
	}
	bases := []ledger.AllocationInput{}
	for _, input := range inputs {
		value := f.command("rules.create", func(ctx context.Context) (command.Result, error) { return allocations.CreateRule(ctx, f.p, input) })
		if value.Status() != command.Succeeded {
			t.Fatal(value.ErrorCode())
		}
		result, _ := value.Result()
		basis := ledger.AllocationInput{Mode: ledger.AllocationByShares, Purpose: ledger.AllocationShared, Origin: ledger.AllocationRule, Reason: "allocation_rule", RuleRefs: []ledger.AllocationRuleRef{{ID: result.ResourceID, Revision: 1}}}
		for _, share := range input.Shares {
			basis.Members = append(basis.Members, ledger.AllocationMemberInput{MemberID: share.MemberID, Share: share.Value})
		}
		if len(basis.Members) == 1 {
			basis.Purpose = ledger.AllocationPersonal
		}
		bases = append(bases, basis)
		f.step(jobs.Outbox, jobapp.OutboxHandler{Repository: f.store})
	}
	r := f.revision(uuid.NewString(), f.account("1000"))
	r.Postings[0].Money = mustMoney("-100")
	r.MerchantID = merchantResult.ResourceID
	r.ReceiptItems = []ledger.ReceiptItem{
		{ID: uuid.NewString(), Name: "Personal item", Quantity: "1", CategoryID: personalCategory, Gross: mustMoney("40"), Discount: mustMoney("0")},
		{ID: uuid.NewString(), Name: "Shared item", Quantity: "1", CategoryID: sharedCategory, Gross: mustMoney("60"), Discount: mustMoney("0")},
	}
	r, err := r.WithAllocation(bases[0], []ledger.ItemAllocationInput{{ItemID: r.ReceiptItems[0].ID, Allocation: bases[1]}, {ItemID: r.ReceiptItems[1].ID, Allocation: bases[0]}}, []household.MembershipID{f.membership.ID, member.ID})
	if err != nil {
		t.Fatal(err)
	}
	created := f.command("transaction.create", func(ctx context.Context) (command.Result, error) {
		return command.Result{ResourceType: "transaction", ResourceID: r.OperationID, Revision: 1}, f.writer.Append(ctx, f.p, r, 0)
	})
	if created.Status() != command.Succeeded {
		t.Fatal(created.ErrorCode())
	}
	f.step(jobs.Outbox, jobapp.OutboxHandler{Repository: f.store})
	mode := "rule"
	f.step(jobs.AI, aiapp.NewHandler(f.store, &gateway{output: output(ai.ReviewCommand{Kind: "distribution", Distribution: &mode, Evidence: []string{"ledger_revision"}, Reason: "Use saved receipt rules."})}, time.Now, uuid.NewString))
	f.step(jobs.AIValidation, f.service())
	if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		current, found, err := f.store.CurrentLedgerRevision(ctx, f.p, r.OperationID)
		if err != nil {
			return err
		}
		if !found || current.Revision != 1 || len(current.ReceiptItems) != 2 {
			t.Fatalf("unchanged item rules: %+v", current)
		}
		for _, amount := range current.Allocation.Members {
			want := "30"
			if amount.MemberID == f.membership.ID {
				want = "70"
			}
			if amount.Money.Amount() != want {
				t.Fatalf("item allocation flattened: %+v", current.Allocation)
			}
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	var validation string
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.review_validations`).Scan(&validation); err != nil || validation != "applied" {
		t.Fatalf("validation: %s %v", validation, err)
	}
}

func TestTerminalAnswerReturnsQuestionToOpen(t *testing.T) {
	for _, failure := range []string{"deadline", "cancellation", "charged", "validation_deadline"} {
		t.Run(failure, func(t *testing.T) {
			f := newFixture(t)
			f.enqueueReviewJobs(1)
			text := "Confirm purchase purpose."
			g := &gateway{output: output(ai.ReviewCommand{Kind: "clarify", Question: &text, Evidence: []string{"ledger_revision"}, Reason: "Purpose is unknown."})}
			f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
			service := f.service()
			f.step(jobs.AIValidation, service)
			var question ai.Clarification
			if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
				items, _, err := service.Clarifications(ctx, f.p, "", 50)
				if err == nil && len(items) == 1 {
					question = items[0]
				}
				return err
			}); err != nil || question.ID == "" {
				t.Fatalf("question: %v", err)
			}
			answer := f.command("question.answer", func(ctx context.Context) (command.Result, error) {
				return service.Answer(ctx, f.p, ai.ReviewAnswer{ClarificationID: question.ID, ExpectedRevision: 1, SubjectExpectedRevision: 1, Text: "Shared purchase"})
			})
			if answer.Status() != command.Succeeded {
				t.Fatal(answer.ErrorCode())
			}
			switch failure {
			case "deadline":
				if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET deadline=clock_timestamp()-INTERVAL '1 second' WHERE kind='ai_answer'`); err != nil {
					t.Fatal(err)
				}
			case "cancellation":
				if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET cancel_requested=true WHERE kind='ai_answer'`); err != nil {
					t.Fatal(err)
				}
			case "charged":
				g.failure = aiapp.GatewayFailure{Code: "provider_timeout", OutcomeUnknown: true}
			case "validation_deadline":
				g.output = output(ai.ReviewCommand{Kind: "no_change", Evidence: []string{"ledger_revision"}, Reason: "Confirmed purpose."})
			}
			f.step(jobs.AIAnswer, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
			if failure == "charged" {
				f.step(jobs.AIValidation, service)
				var state string
				if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.review_clarifications WHERE id=$1`, question.ID).Scan(&state); err != nil || state != "checking" {
					t.Fatalf("unresolved call reopened: %s %v", state, err)
				}
				var attempt string
				if err := f.admin.QueryRow(testContext, `SELECT a.id FROM want_keep.ai_attempts a JOIN want_keep.jobs j ON (j.household_id,j.id)=(a.household_id,a.job_id) WHERE j.kind='ai_answer'`).Scan(&attempt); err != nil {
					t.Fatal(err)
				}
				if _, err := f.maintenancePool().Exec(testContext, `SELECT want_keep.reconcile_ai_attempt($1,'charged',0.001,'provider-dashboard:synthetic-answer')`, attempt); err != nil {
					t.Fatal(err)
				}
			}
			if failure == "validation_deadline" {
				if _, err := f.admin.Exec(testContext, `UPDATE want_keep.jobs SET deadline=clock_timestamp()-INTERVAL '1 second' WHERE kind='ai_validation' AND state='ready'`); err != nil {
					t.Fatal(err)
				}
				f.step(jobs.AIValidation, service)
			}
			f.step(jobs.AIValidation, service)
			var state string
			var revision, history int
			if err := f.admin.QueryRow(testContext, `SELECT state,revision FROM want_keep.review_clarifications WHERE id=$1`, question.ID).Scan(&state, &revision); err != nil || state != "open" || revision != 3 {
				t.Fatalf("terminal answer: %s revision=%d %v", state, revision, err)
			}
			if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.review_state_history WHERE clarification_id=$1`, question.ID).Scan(&history); err != nil || history != 3 {
				t.Fatalf("question history: %d %v", history, err)
			}
			value := f.command("question.answer", func(ctx context.Context) (command.Result, error) {
				return service.Answer(ctx, f.p, ai.ReviewAnswer{ClarificationID: question.ID, ExpectedRevision: 3, SubjectExpectedRevision: 1, ChoiceID: "keep"})
			})
			if value.Status() != command.Succeeded {
				t.Fatalf("new answer blocked: %s", value.ErrorCode())
			}
			calls := int32(1)
			if failure == "charged" || failure == "validation_deadline" {
				calls = 2
			}
			if g.calls.Load() != calls {
				t.Fatalf("recovery repeated paid IO: %d", g.calls.Load())
			}
		})
	}
}
