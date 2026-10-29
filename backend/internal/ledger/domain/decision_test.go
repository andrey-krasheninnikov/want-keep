package domain_test

import (
	"errors"
	"reflect"
	"testing"

	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

func TestCorrectionOwnsCopiesAndKeepsAccountingIndependent(t *testing.T) {
	e := examples{}
	r := e.expense()
	r.Protections = map[ledger.Field]ledger.Protection{ledger.MerchantField: {DecisionID: "previous", Revision: 1}}
	r.FieldVersions = map[ledger.Field]uint64{ledger.MerchantField: 1}
	before := r.Clone()
	posts := append([]ledger.Posting(nil), r.Postings...)
	posts[0].Money = e.money("-700", money.RUB)
	next, fields, err := r.Correct(ledger.Correction{Principal: &posts})
	if err != nil {
		t.Fatal(err)
	}
	d := ledger.Decision{ID: "correction", Kind: "correction", ActorID: "b", Reason: "Fixed amount", At: r.PostedAt}
	next = next.WithDecision(d, fields)
	next.Protections[ledger.MerchantField] = ledger.Protection{DecisionID: "changed", Revision: 2}
	if !reflect.DeepEqual(r, before) || posts[0].Money.Amount() != "-700" {
		t.Fatal("input changed")
	}
	next.AccountingState = ledger.ExcludedFromAccounting
	effects, err := next.BalanceEffects()
	if err != nil || len(effects) != 0 || next.State != ledger.Posted || len(next.Postings) != 1 {
		t.Fatal("exclusion destroyed facts", err)
	}
	next.AccountingState = "invalid"
	if next.Validate() == nil {
		t.Fatal("invalid disposition")
	}
}

func TestMoneyNoOpAndInvalidPrincipalIdentity(t *testing.T) {
	e := examples{}
	r := e.expense()
	posts := append([]ledger.Posting(nil), r.Postings...)
	posts[0].Money = e.money("-500.000", money.RUB)
	if _, _, err := r.Correct(ledger.Correction{Principal: &posts}); !errors.Is(err, ledger.ErrNoChange) {
		t.Fatal("scale became a correction", err)
	}
	posts[0].AccountID = "different"
	if _, _, err := r.Correct(ledger.Correction{Principal: &posts}); err == nil {
		t.Fatal("source identity replaced")
	}
	posts[0] = r.Postings[0]
	posts[0].Money = e.money("-700", money.USD)
	if _, _, err := r.Correct(ledger.Correction{Principal: &posts}); err == nil {
		t.Fatal("asset replaced")
	}
	r.Revision = 9007199254740991
	updated := r.WithDecision(ledger.Decision{ID: "overflow", ActorID: "b", Reason: "overflow", At: r.PostedAt}, []ledger.Field{ledger.NoteField})
	if updated.Validate() == nil || r.Revision != 9007199254740991 {
		t.Fatal("overflow or mutation")
	}
}

func TestSelectiveUndoChecksFieldOriginNotOnlyValue(t *testing.T) {
	e := examples{}
	initial := e.expense()
	name := "Market"
	changed, fields, err := initial.Correct(ledger.Correction{Merchant: &name})
	if err != nil {
		t.Fatal(err)
	}
	d := ledger.Decision{ID: "one", Kind: "correction", ActorID: "b", Reason: "merchant", At: initial.PostedAt}
	changed = changed.WithDecision(d, fields)
	entry := ledger.DecisionEntry{OperationID: initial.OperationID, Before: 1, After: 2, Fields: fields}
	later := changed.Clone()
	later.Revision = 4
	later.FieldVersions[ledger.MerchantField] = 4
	if _, err = later.UndoFields(entry, initial, nil); !errors.Is(err, ledger.ErrDecisionConflict) {
		t.Fatal("ABA accepted", err)
	}
	later = changed.Clone()
	later.Revision = 3
	later.Note = "Partner note"
	later.FieldVersions[ledger.NoteField] = 3
	undone, err := later.UndoFields(entry, initial, nil)
	if err != nil || undone.Merchant != "" || undone.Note != "Partner note" {
		t.Fatal("selective undo", err)
	}
	if later.Merchant != "Market" {
		t.Fatal("undo mutated input")
	}
}

func TestFeeNoOpInheritsExistingFunding(t *testing.T) {
	e := examples{}
	for _, funding := range []ledger.FundingKind{ledger.OwnFunds, ledger.CreditFunds, ledger.UnknownFunds} {
		r := e.expense()
		r.FeeKnowledge = ledger.KnownFees
		r.Postings = append(r.Postings, ledger.Posting{AccountID: "cash", Money: e.money("-10", money.RUB), Role: ledger.Fee, Funding: funding, Treatment: ledger.Movement})
		fees := []ledger.Posting{{AccountID: "cash", Money: e.money("-10.00", money.RUB), Role: ledger.Fee}}
		if _, _, err := r.Correct(ledger.Correction{Fees: &fees}); !errors.Is(err, ledger.ErrNoChange) {
			t.Fatalf("funding=%s: %v", funding, err)
		}
		if fees[0].Funding != "" {
			t.Fatal("input mutated")
		}
	}
}

func TestMatchingSourceDoesNotConflictWithFinancialProvenance(t *testing.T) {
	raw := (examples{}).expense()
	current := raw.Clone()
	current.Revision = 3
	current.FieldVersions[ledger.PrincipalField] = 3
	if current.ConflictsWithSource(&raw) {
		t.Fatal("matching source falsely conflicts with financial revision")
	}
	current.Protections[ledger.PrincipalField] = ledger.Protection{DecisionID: "decision", Revision: 2}
	if current.ConflictsWithSource(&raw) {
		t.Fatal("matching protected value falsely conflicts")
	}
	raw.Postings[0].Money = (examples{}).money("-700", money.RUB)
	if !current.ConflictsWithSource(&raw) {
		t.Fatal("contradicting source not detected")
	}
}

func TestDerivedContributionDoesNotProtectOrSupersedeAssociation(t *testing.T) {
	initial := (examples{}).expense()
	changed := initial.WithDecision(ledger.Decision{ID: "derived", Kind: "correction", ActorID: "b", Reason: "Recalculate date", At: initial.PostedAt}, []ledger.Field{ledger.ContributionField})
	if _, protected := changed.Protections[ledger.ContributionField]; protected || changed.Validate() != nil {
		t.Fatal("derived calculation became a protected choice")
	}
	later := changed.Clone()
	later.Revision = 4
	later.FieldVersions[ledger.ContributionField] = 4
	entry := ledger.DecisionEntry{OperationID: initial.OperationID, Before: 1, After: 2, Fields: []ledger.Field{ledger.ContributionField}}
	if _, err := later.UndoFields(entry, initial, nil); err != nil {
		t.Fatal("derived calculation blocked undo", err)
	}
	entry.Fields = []ledger.Field{ledger.MatchingField}
	later.FieldVersions[ledger.MatchingField] = 4
	if _, err := later.UndoFields(entry, initial, nil); !errors.Is(err, ledger.ErrDecisionConflict) {
		t.Fatal("association ABA accepted", err)
	}
	changed.Protections[ledger.ContributionField] = ledger.Protection{DecisionID: "invalid", Revision: 2}
	if changed.Validate() == nil {
		t.Fatal("derived field protection accepted")
	}
}

func TestFundingBasisPreservesUncertainty(t *testing.T) {
	for _, pair := range [][2]ledger.FundingKind{{"", ledger.OwnFunds}, {ledger.OwnFunds, ""}, {ledger.CreditFunds, ledger.CreditFunds}, {ledger.UnknownFunds, ledger.UnknownFunds}} {
		if !pair[0].SameBasis(pair[1]) {
			t.Fatal("equivalent basis rejected", pair)
		}
	}
	for _, pair := range [][2]ledger.FundingKind{{"", ledger.UnknownFunds}, {"", ledger.CreditFunds}, {ledger.OwnFunds, ledger.CreditFunds}, {ledger.UnknownFunds, ledger.OwnFunds}} {
		if pair[0].SameBasis(pair[1]) {
			t.Fatal("uncertain or credit funds treated as own", pair)
		}
	}
}

func TestReceiptDatePrecisionParticipatesInCorrectionAndUndo(t *testing.T) {
	r := (examples{}).expense()
	r.OccurredPrecision = "day"
	next, fields, err := r.Correct(ledger.Correction{OccurredAt: &r.OccurredAt})
	if err != nil || len(fields) != 1 || fields[0] != ledger.DateField || next.OccurredPrecision != "second" {
		t.Fatal("explicit time did not replace date-only anchor", err)
	}
	next.CopyField(r, ledger.DateField)
	if !next.FieldEqual(r, ledger.DateField) || next.OccurredPrecision != "day" {
		t.Fatal("date undo lost precision")
	}
	r.OccurredPrecision = ""
	if _, _, err = r.Correct(ledger.Correction{OccurredAt: &r.OccurredAt}); !errors.Is(err, ledger.ErrNoChange) {
		t.Fatal("legacy timestamp changed precision on replay", err)
	}
}
