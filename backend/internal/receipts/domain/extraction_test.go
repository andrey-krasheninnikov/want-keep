package domain_test

import (
	"errors"
	"testing"

	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

func text(s string) *string { return &s }
func paid(b bool) *bool     { return &b }
func page() receipt.Extraction {
	return receipt.Extraction{Document: "receipt", Kind: "expense", Paid: paid(true), Date: text("2026-08-31"), Time: text("14:30:00"), Asset: text("RUB"), Total: text("900"), Discount: text("100"), Merchant: text("Synthetic shop"), Items: []receipt.Item{{Name: "Food", Quantity: "1", Gross: "600"}, {Name: "Home", Quantity: "1", Gross: "400"}}}
}
func TestArithmeticAndUnknownFacts(t *testing.T) {
	for _, asset := range []string{"RUB", "USD", "USDT", "USDC", "BTC", "ETH"} {
		e := page()
		e.Asset = text(asset)
		facts, err := receipt.Build([]receipt.Extraction{e}, asset, "stable")
		if err != nil || facts.Total.Amount() != "900" || len(facts.Items) != 2 {
			t.Fatalf("%s: %+v %v", asset, facts, err)
		}
		sum, err := facts.Items[0].Net()
		if err != nil {
			t.Fatal(err)
		}
		n, _ := facts.Items[1].Net()
		sum, _ = sum.Add(n)
		if sum.Amount() != "900" || facts.Items[0].Discount.Amount() != "60" {
			t.Fatal("discount changed payment")
		}
	}
	for _, change := range []func(*receipt.Extraction){
		func(e *receipt.Extraction) { e.Total = text("901") },
		func(e *receipt.Extraction) { e.Paid = nil },
		func(e *receipt.Extraction) { e.Asset = text("USDC.E") },
		func(e *receipt.Extraction) { e.Date = text("2026-02-30") },
		func(e *receipt.Extraction) { e.Items[0].Discount = text("1") },
		func(e *receipt.Extraction) { e.Document = "mixed" },
	} {
		e := page()
		change(&e)
		if _, err := receipt.Build([]receipt.Extraction{e}, "RUB", "stable"); !errors.Is(err, receipt.ErrClarification) {
			t.Fatalf("unconfirmed facts accepted: %v", err)
		}
	}
}
func TestPagesAndStableItemIdentity(t *testing.T) {
	first := page()
	second := page()
	first.Total = nil
	first.Discount = nil
	first.Items = first.Items[:1]
	second.Items = second.Items[1:]
	a, err := receipt.Build([]receipt.Extraction{first, second}, "RUB", "stable")
	if err != nil {
		t.Fatal(err)
	}
	b, err := receipt.Build([]receipt.Extraction{first, second}, "RUB", "stable")
	if err != nil || a.Items[0].ID != b.Items[0].ID {
		t.Fatal("unstable item IDs")
	}
	second.Total = text("901")
	first.Total = text("900")
	if _, err = receipt.Build([]receipt.Extraction{first, second}, "RUB", "stable"); !errors.Is(err, receipt.ErrClarification) {
		t.Fatal("conflicting page totals accepted")
	}
	e := receipt.Extraction{Document: "irrelevant", Reason: "Menu, not proof of payment."}
	if _, err = receipt.Build([]receipt.Extraction{e}, "RUB", "stable"); !errors.Is(err, receipt.ErrSkipped) {
		t.Fatal(err)
	}
}
func TestExactFractionsAndInputBounds(t *testing.T) {
	for _, asset := range []string{"USDC", "BTC", "ETH"} {
		e := page()
		e.Asset = text(asset)
		e.Total = text("0.000000000000000001")
		e.Discount = text("0")
		e.Items = []receipt.Item{{Name: "Fraction", Quantity: "0.001", Gross: "0.000000000000000001", Discount: text("0")}}
		f, err := receipt.Build([]receipt.Extraction{e}, asset, "stable")
		if err != nil || f.Total.Amount() != *e.Total {
			t.Fatalf("precision lost: %v", err)
		}
	}
	e := page()
	e.Total = text("1e3")
	if _, err := receipt.Build([]receipt.Extraction{e}, "RUB", "stable"); !errors.Is(err, receipt.ErrClarification) {
		t.Fatal("exponent accepted")
	}
}

func TestContinuationWithoutPaymentFacts(t *testing.T) {
	first, last := page(), page()
	first.Kind = "unknown"
	first.Total = nil
	first.Paid = nil
	first.Discount = nil
	first.Items = first.Items[:1]
	last.Items = last.Items[1:]
	f, err := receipt.Build([]receipt.Extraction{first, last}, "RUB", "continued")
	if err != nil || f.Total.Amount() != "900" || len(f.Items) != 2 {
		t.Fatal("continuation became a separate or incomplete payment", err)
	}
}

func TestItemDiscountsWithoutPrintedGlobalDiscount(t *testing.T) {
	e := page()
	e.Discount = nil
	e.Items[0].Discount = text("60")
	e.Items[1].Discount = text("40")
	f, err := receipt.Build([]receipt.Extraction{e}, "RUB", "item-discounts")
	if err != nil || len(f.Items) != 2 {
		t.Fatal("known per-item discounts lost", err)
	}
	first, last := page(), page()
	first.Items = nil
	first.Total = text("900.00")
	first.Reference = text("receipt-123")
	last.Reference = text("receipt-123")
	last.Items = nil
	if _, err = receipt.Build([]receipt.Extraction{first, last}, "RUB", "same-total"); err != nil {
		t.Fatal("equal monetary values treated as conflict", err)
	}
}

func TestCompletePaymentsNeedSharedIdentity(t *testing.T) {
	for _, refs := range [][2]string{{"", ""}, {"receipt-1", ""}, {"receipt-1", "receipt-2"}, {"receipt-1", "receipt-1"}} {
		t.Run(refs[0]+"/"+refs[1], func(t *testing.T) {
			first, last := page(), page()
			for index, item := range []*receipt.Extraction{&first, &last} {
				item.Items = nil
				item.Time = nil
				item.Total = text("100")
				item.Discount = text("0")
				if refs[index] != "" {
					item.Reference = text(refs[index])
				}
			}
			_, err := receipt.Build([]receipt.Extraction{first, last}, "RUB", "multiple-payments")
			if refs[0] != "" && refs[0] == refs[1] {
				if err != nil {
					t.Fatal("common receipt identity rejected", err)
				}
			} else if !errors.Is(err, receipt.ErrClarification) {
				t.Fatal("independent or ambiguous payments merged", err)
			}
		})
	}
}
