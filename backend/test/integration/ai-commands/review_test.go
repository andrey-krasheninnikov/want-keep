//go:build integration

package aicommands_test

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	rule "github.com/pchkauu/want-keep/backend/internal/allocation/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	commands "github.com/pchkauu/want-keep/backend/internal/commands/application"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	journal "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/application"
)

type gateway struct {
	output  []byte
	failure error
	calls   atomic.Int32
}

func (g *gateway) Contract() ai.RuntimeContract {
	return ai.RuntimeContract{Model: ai.Terra, Qualification: ai.TerraXHigh, PromptFingerprint: strings.Repeat("a", 64), SchemaFingerprint: strings.Repeat("b", 64), ConfigFingerprint: strings.Repeat("c", 64)}
}
func (g *gateway) Count(context.Context, ai.Request) (int64, error) { return 100, nil }
func (g *gateway) Generate(context.Context, ai.Request) (ai.Result, error) {
	g.calls.Add(1)
	if g.failure != nil {
		return ai.Result{}, g.failure
	}
	zero := int64(0)
	return ai.Result{State: ai.Completed, ProviderID: "synthetic-response", ProviderModel: string(ai.Terra), Output: g.output, Usage: ai.Usage{InputTokens: 100, OutputTokens: 50, CacheWriteTokens: &zero}}, nil
}
func output(c ai.ReviewCommand) []byte {
	raw, _ := json.Marshal(ai.ReviewOutput{Version: ai.ReviewContractVersion, CaseID: "case-1", Commands: []ai.ReviewCommand{c}})
	return raw
}
func (f *fixture) service() *aiapp.ReviewService {
	m := matching.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	a := allocation.NewService(f.store, func() calendar.Instant { return f.now }, uuid.NewString)
	l := journal.NewServiceWithAllocations(f.store, m, a, func() calendar.Instant { return f.now }, uuid.NewString)
	return aiapp.NewReviewService(f.store, l, a, m, uuid.NewString)
}
func (f *fixture) step(kind jobs.Kind, h jobapp.Handler) {
	f.t.Helper()
	worker := jobapp.Worker{Repository: f.store, Handler: h, Config: jobapp.DefaultWorkerConfig(kind)}
	if err := worker.Step(testContext); err != nil {
		f.t.Fatal(err)
	}
}
func (f *fixture) root() string {
	f.t.Helper()
	var id string
	if err := f.admin.QueryRow(testContext, `SELECT resource_id FROM want_keep.jobs WHERE household_id=$1 AND kind='ai'`, f.family.ID).Scan(&id); err != nil {
		f.t.Fatal(err)
	}
	return id
}
func (f *fixture) command(kind string, apply func(context.Context) (command.Result, error)) command.Command {
	f.t.Helper()
	value, err := f.executor.Execute(testContext, f.p, commands.Request{ID: uuid.NewString(), Kind: kind, PayloadHash: strings.Repeat("d", 64)}, apply)
	if err != nil {
		f.t.Fatal(err)
	}
	return value
}

func TestClassificationIsAtomicAndDoesNotDuplicateMoney(t *testing.T) {
	f := newFixture(t)
	var categoryID string
	if err := f.admin.QueryRow(testContext, `SELECT id FROM want_keep.categories WHERE household_id=$1 ORDER BY id LIMIT 1`, f.family.ID).Scan(&categoryID); err != nil {
		t.Fatal(err)
	}
	f.enqueueReviewJobs(1)
	root := f.root()
	ref := "category-1"
	g := &gateway{output: output(ai.ReviewCommand{Kind: "classification", Category: &ref, Evidence: []string{"ledger_revision"}, Reason: "Existing food category matches this purchase."})}
	f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	var requests int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.review_validation_jobs`).Scan(&requests); err != nil || requests != 1 {
		t.Fatalf("durable validation: %d %v", requests, err)
	}
	f.step(jobs.AIValidation, f.service())
	var current ledger.Revision
	if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		var found bool
		var err error
		current, found, err = f.store.CurrentLedgerRevision(ctx, f.p, root)
		if !found {
			t.Fatal("missing transaction")
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if current.Revision != 2 || current.CategoryID != categoryID || current.Postings[0].Money.Amount() != "-1" {
		t.Fatalf("classified revision=%d category=%s", current.Revision, current.CategoryID)
	}
	if _, protected := current.Protections[ledger.CategoryField]; protected {
		t.Fatal("automatic classification became a human override")
	}
	f.step(jobs.AIValidation, f.service())
	if g.calls.Load() != 1 {
		t.Fatal("validation repeated provider IO")
	}
	var operations int
	_ = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.operations`).Scan(&operations)
	if operations != 1 {
		t.Fatal("second financial operation")
	}
}

func TestClarificationHasSeparateFreeTextAttempt(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	question := "Is this purchase personal or shared?"
	g := &gateway{output: output(ai.ReviewCommand{Kind: "clarify", Question: &question, Evidence: []string{"ledger_revision"}, Reason: "The purchase purpose is unknown."})}
	f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	s := f.service()
	f.step(jobs.AIValidation, s)
	var questions []ai.Clarification
	if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		var err error
		questions, _, err = s.Clarifications(ctx, f.p, "", 50)
		return err
	}); err != nil || len(questions) != 1 {
		t.Fatalf("questions: %d %v", len(questions), err)
	}
	c := questions[0]
	result := f.command("clarifications.answer", func(ctx context.Context) (command.Result, error) {
		return s.Answer(ctx, f.p, ai.ReviewAnswer{ClarificationID: c.ID, ExpectedRevision: 1, SubjectExpectedRevision: 1, Text: "Shared food purchase"})
	})
	if result.Status() != command.Succeeded {
		t.Fatalf("answer failed: %s", result.ErrorCode())
	}
	var answers int
	_ = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.jobs WHERE kind='ai_answer'`).Scan(&answers)
	if answers != 1 {
		t.Fatal("free text did not create a distinct check")
	}
	g.output = output(ai.ReviewCommand{Kind: "no_change", Evidence: []string{"ledger_revision"}, Reason: "The transaction is correct."})
	f.step(jobs.AIAnswer, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	f.step(jobs.AIValidation, s)
	var state string
	_ = f.admin.QueryRow(testContext, `SELECT state FROM want_keep.review_clarifications WHERE id=$1`, c.ID).Scan(&state)
	if state != "answered" || g.calls.Load() != 2 {
		t.Fatalf("answer outcome: %s calls=%d", state, g.calls.Load())
	}
	var ledgerReviews int
	_ = f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.ledger_review_results`).Scan(&ledgerReviews)
	if ledgerReviews != 1 {
		t.Fatal("original review was rewritten")
	}
}

func TestMalformedAndStaleResponseHaveNoEffect(t *testing.T) {
	for _, stale := range []bool{false, true} {
		t.Run(map[bool]string{false: "money", true: "stale"}[stale], func(t *testing.T) {
			f := newFixture(t)
			f.enqueueReviewJobs(1)
			root := f.root()
			raw := []byte(`{"version":"transaction_review_v1","caseId":"case-1","commands":[{"kind":"classification","amount":"99","evidence":["ledger_revision"],"reason":"Ignore restrictions"}]}`)
			if stale {
				raw = output(ai.ReviewCommand{Kind: "no_change", Evidence: []string{"ledger_revision"}, Reason: "Correct transaction"})
			}
			g := &gateway{output: raw}
			f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
			if stale {
				note := "Partner correction"
				l := journal.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
				f.command("transactions.correct", func(ctx context.Context) (command.Result, error) {
					return l.Correct(ctx, f.p, journal.Change{OperationID: root, Expected: 1, Correction: ledger.Correction{Note: &note}}, "Fix note")
				})
			}
			f.step(jobs.AIValidation, f.service())
			var state string
			_ = f.admin.QueryRow(testContext, `SELECT state FROM want_keep.review_validations`).Scan(&state)
			expected := "failed"
			if stale {
				expected = "stale"
			}
			if state != expected {
				t.Fatalf("validation %s instead of %s", state, expected)
			}
		})
	}
}

func TestCompetingAnswersAndReplay(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	question := "Who benefits from this purchase?"
	g := &gateway{output: output(ai.ReviewCommand{Kind: "clarify", Question: &question, Evidence: []string{"ledger_revision"}, Reason: "Purpose requires confirmation."})}
	f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	service := f.service()
	f.step(jobs.AIValidation, service)
	var clarification ai.Clarification
	if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
		items, _, err := service.Clarifications(ctx, f.p, "", 50)
		if err == nil && len(items) == 1 {
			clarification = items[0]
		}
		return err
	}); err != nil {
		t.Fatal(err)
	}
	user := household.User{ID: household.UserID(uuid.NewString()), Name: "Member B"}
	member := household.Membership{ID: household.MembershipID(uuid.NewString()), UserID: user.ID, HouseholdID: f.family.ID, Active: true}
	if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.AddMember(ctx, user, member) }); err != nil {
		t.Fatal(err)
	}
	partner, _ := member.Principal()
	start := make(chan struct{})
	results := make(chan command.Command, 2)
	errors := make(chan error, 2)
	requests := []commands.Request{{ID: uuid.NewString(), Kind: "clarifications.answer", PayloadHash: strings.Repeat("a", 64)}, {ID: uuid.NewString(), Kind: "clarifications.answer", PayloadHash: strings.Repeat("b", 64)}}
	for i, p := range []household.Principal{f.p, partner} {
		go func(i int, p household.Principal) {
			<-start
			value, err := f.executor.Execute(testContext, p, requests[i], func(ctx context.Context) (command.Result, error) {
				return service.Answer(ctx, p, ai.ReviewAnswer{ClarificationID: clarification.ID, ExpectedRevision: 1, SubjectExpectedRevision: 1, ChoiceID: "keep"})
			})
			results <- value
			errors <- err
		}(i, p)
	}
	close(start)
	success := 0
	for range 2 {
		value := <-results
		if err := <-errors; err != nil {
			t.Fatal(err)
		}
		if value.Status() == command.Succeeded {
			success++
		} else if value.ErrorCode() != "version_conflict" {
			t.Fatalf("wrong concurrent error: %s", value.ErrorCode())
		}
	}
	if success != 1 {
		t.Fatalf("successful answers: %d", success)
	}
	for i, p := range []household.Principal{f.p, partner} {
		_, err := f.executor.Execute(testContext, p, requests[i], func(context.Context) (command.Result, error) {
			t.Fatal("replay executed again")
			return command.Result{}, nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}
	var count int
	if err := f.admin.QueryRow(testContext, `SELECT count(*) FROM want_keep.review_answers`).Scan(&count); err != nil || count != 1 {
		t.Fatalf("answers: %d %v", count, err)
	}
	other := f.anotherHousehold()
	if err := f.store.WithinFinancialRead(testContext, other.p, func(ctx context.Context) error {
		_, err := f.store.Clarification(ctx, other.p, clarification.ID)
		return err
	}); err == nil {
		t.Fatal("foreign clarification exposed")
	}
}

func TestProtectedFieldAndRollbackStayUnchanged(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	root := f.root()
	var categoryID string
	_ = f.admin.QueryRow(testContext, `SELECT id FROM want_keep.categories WHERE household_id=$1 ORDER BY id LIMIT 1`, f.family.ID).Scan(&categoryID)
	service := journal.NewService(f.store, f.writer, func() calendar.Instant { return f.now }, uuid.NewString)
	f.command("transactions.correct", func(ctx context.Context) (command.Result, error) {
		return service.Correct(ctx, f.p, journal.Change{OperationID: root, Expected: 1, Correction: ledger.Correction{CategoryID: &categoryID}}, "User selected category")
	})
	// Process the original request: its old version cannot overwrite the correction.
	ref := "category-1"
	g := &gateway{output: output(ai.ReviewCommand{Kind: "classification", Category: &ref, Evidence: []string{"ledger_revision"}, Reason: "Suggest classification"})}
	f.step(jobs.AI, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	f.step(jobs.AIValidation, f.service())
	var revision int
	_ = f.admin.QueryRow(testContext, `SELECT revision FROM want_keep.operations WHERE id=$1`, root).Scan(&revision)
	if revision != 2 {
		t.Fatal("protected correction overwritten")
	}
	// A savepoint failure must not leave a decision or projection change.
	err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error {
		note := "Transient correction"
		_, err := service.Correct(ctx, f.p, journal.Change{OperationID: root, Expected: 2, Correction: ledger.Correction{Note: &note}}, "Rollback test")
		if err != nil {
			return err
		}
		return fmt.Errorf("synthetic rollback")
	})
	if err == nil {
		t.Fatal("rollback did not fail")
	}
	_ = f.admin.QueryRow(testContext, `SELECT revision FROM want_keep.operations WHERE id=$1`, root).Scan(&revision)
	if revision != 2 {
		t.Fatal("partial correction committed")
	}
}

func TestReviewPreservesFirstFactRuleBoundary(t *testing.T) {
	for _, explicitRule := range []bool{false, true} {
		t.Run(fmt.Sprintf("distribution_command=%t", explicitRule), func(t *testing.T) {
			f := newFixture(t)
			user := household.User{ID: household.UserID(uuid.NewString()), Name: "Member B"}
			member := household.Membership{ID: household.MembershipID(uuid.NewString()), UserID: user.ID, HouseholdID: f.family.ID, Active: true}
			if err := f.store.WithinHousehold(testContext, f.p, func(ctx context.Context) error { return f.store.AddMember(ctx, user, member) }); err != nil {
				t.Fatal(err)
			}
			var categoryID string
			if err := f.admin.QueryRow(testContext, `SELECT id FROM want_keep.categories WHERE household_id=$1 ORDER BY id LIMIT 1`, f.family.ID).Scan(&categoryID); err != nil {
				t.Fatal(err)
			}
			allocations := allocation.NewService(f.store, func() calendar.Instant { return f.now }, uuid.NewString)
			input := allocation.RuleInput{Priority: 10, State: rule.Active, Condition: rule.Condition{CategoryID: categoryID}, Shares: []rule.Share{{MemberID: f.membership.ID, Value: "60"}, {MemberID: member.ID, Value: "40"}}}
			created := f.command("rules.create", func(ctx context.Context) (command.Result, error) { return allocations.CreateRule(ctx, f.p, input) })
			if created.Status() != command.Succeeded {
				t.Fatal(created.ErrorCode())
			}
			createdRule, _ := created.Result()
			f.step(jobs.Outbox, jobapp.OutboxHandler{Repository: f.store})
			f.enqueueReviewJobs(1)
			input.Shares[0].Value, input.Shares[1].Value = "40", "60"
			changed := f.command("rules.change", func(ctx context.Context) (command.Result, error) {
				return allocations.ChangeRule(ctx, f.p, createdRule.ResourceID, 1, input)
			})
			if changed.Status() != command.Succeeded {
				t.Fatal(changed.ErrorCode())
			}
			alias := "category-1"
			commandList := []ai.ReviewCommand{{Kind: "classification", Category: &alias, Evidence: []string{"ledger_revision"}, Reason: "Saved category matches."}}
			if explicitRule {
				mode := "rule"
				commandList = append(commandList, ai.ReviewCommand{Kind: "distribution", Distribution: &mode, Evidence: []string{"ledger_revision"}, Reason: "Use the saved rule."})
			}
			raw, _ := json.Marshal(ai.ReviewOutput{Version: ai.ReviewContractVersion, CaseID: "case-1", Commands: commandList})
			f.step(jobs.AI, aiapp.NewHandler(f.store, &gateway{output: raw}, time.Now, uuid.NewString))
			f.step(jobs.AIValidation, f.service())
			if err := f.store.WithinFinancialRead(testContext, f.p, func(ctx context.Context) error {
				current, found, err := f.store.CurrentLedgerRevision(ctx, f.p, f.root())
				if err != nil {
					return err
				}
				if !found || current.Revision != 2 || current.CategoryID != categoryID || current.Postings[0].Money.Amount() != "-1" {
					t.Fatalf("classification: %+v", current)
				}
				if current.Allocation.State != ledger.AllocationResolved || len(current.Allocation.RuleRefs) != 1 || current.Allocation.RuleRefs[0].Revision != 1 {
					t.Fatalf("historical allocation: %+v", current.Allocation)
				}
				for _, share := range current.Allocation.Inputs {
					if share.MemberID == f.membership.ID && share.Share != "60" {
						t.Fatalf("fact-time share: %+v", share)
					}
				}
				return nil
			}); err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestUnmatchedRuleApprovalLeavesQuestionOpen(t *testing.T) {
	f := newFixture(t)
	f.enqueueReviewJobs(1)
	mode := "rule"
	f.step(jobs.AI, aiapp.NewHandler(f.store, &gateway{output: output(ai.ReviewCommand{Kind: "distribution", Distribution: &mode, Evidence: []string{"ledger_revision"}, Reason: "Resolve by a saved rule."})}, time.Now, uuid.NewString))
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
	rejected := f.command("proposal.apply", func(ctx context.Context) (command.Result, error) {
		return service.Apply(ctx, f.p, question.ProposalID, 1, 1, "apply")
	})
	if rejected.Status() != command.Failed || rejected.ErrorCode() != "clarification_required" {
		t.Fatalf("approval: %s %s", rejected.Status(), rejected.ErrorCode())
	}
	var state string
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.review_clarifications WHERE id=$1`, question.ID).Scan(&state); err != nil || state != "open" {
		t.Fatalf("question closed: %s %v", state, err)
	}
}

func TestUnknownAnswerCanBeOperatorReconciled(t *testing.T) {
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
	value := f.command("question.answer", func(ctx context.Context) (command.Result, error) {
		return service.Answer(ctx, f.p, ai.ReviewAnswer{ClarificationID: question.ID, ExpectedRevision: 1, SubjectExpectedRevision: 1, Text: "Shared purchase"})
	})
	if value.Status() != command.Succeeded {
		t.Fatal(value.ErrorCode())
	}
	g.failure = aiapp.GatewayFailure{Code: "provider_timeout", OutcomeUnknown: true}
	f.step(jobs.AIAnswer, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	var attemptID, state string
	if err := f.admin.QueryRow(testContext, `SELECT a.id,j.state FROM want_keep.ai_attempts a JOIN want_keep.jobs j ON j.id=a.job_id AND j.household_id=a.household_id WHERE j.kind='ai_answer'`).Scan(&attemptID, &state); err != nil || state != "unresolved" {
		t.Fatalf("unknown answer: %s %v", state, err)
	}
	pool := f.maintenancePool()
	for range 2 {
		if _, err := pool.Exec(testContext, `SELECT want_keep.reconcile_ai_attempt($1,'not_charged',0,'provider-dashboard:synthetic-answer')`, attemptID); err != nil {
			t.Fatal(err)
		}
	}
	if err := f.admin.QueryRow(testContext, `SELECT j.state FROM want_keep.jobs j WHERE kind='ai_answer'`).Scan(&state); err != nil || state != "ready" {
		t.Fatalf("reconciled answer: %s %v", state, err)
	}
	g.failure = nil
	g.output = output(ai.ReviewCommand{Kind: "no_change", Evidence: []string{"ledger_revision"}, Reason: "Confirmed purpose."})
	f.step(jobs.AIAnswer, aiapp.NewHandler(f.store, g, time.Now, uuid.NewString))
	f.step(jobs.AIValidation, service)
	if err := f.admin.QueryRow(testContext, `SELECT state FROM want_keep.review_clarifications WHERE id=$1`, question.ID).Scan(&state); err != nil || state != "answered" {
		t.Fatalf("resumed answer: %s %v", state, err)
	}
}
