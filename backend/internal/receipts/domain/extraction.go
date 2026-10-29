package domain

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

var (
	ErrClarification = errors.New("receipt needs clarification")
	ErrSkipped       = errors.New("document is not a receipt")
	ErrNotFound      = errors.New("receipt not found")
	ErrInvalid       = errors.New("invalid receipt")
)

type Item struct {
	Name, Quantity, Gross string
	CategoryAlias         *string
	CategoryID            string
	Discount              *string
}
type Extraction struct {
	Document, Kind, Reason                                  string
	Paid                                                    *bool
	Date, Time, Asset, Total, Discount, Merchant, Reference *string
	Items                                                   []Item
}
type Facts struct {
	Kind                      ledger.Type
	Date                      calendar.Date
	Time, Merchant, Reference string
	Total                     money.Money
	Items                     []ledger.ReceiptItem
}

// Build merges pages before validating one payment; page subtotals must not masquerade as the final total.
func Build(pages []Extraction, asset, identity string) (Facts, error) {
	var out Facts
	if len(pages) < 1 || len(pages) > 10 || identity == "" {
		return out, ErrInvalid
	}
	var date, clock, currency, total, discount, merchant, reference, kind *string
	var paid *bool
	inputs := []ledger.ReceiptItemInput{}
	irrelevant := 0
	for page, e := range pages {
		if e.Document == "irrelevant" {
			irrelevant++
			continue
		}
		if e.Document != "receipt" || e.Kind != "expense" && e.Kind != "refund" && e.Kind != "unknown" || len(e.Items) > 1000 || len(e.Reason) > 8000 {
			return out, ErrClarification
		}
		if e.Kind != "unknown" {
			if kind != nil && *kind != e.Kind {
				return out, ErrClarification
			}
			value := e.Kind
			kind = &value
		}
		for _, pair := range []struct {
			dest    **string
			source  *string
			decimal bool
		}{{&date, e.Date, false}, {&clock, e.Time, false}, {&currency, e.Asset, false}, {&total, e.Total, true}, {&discount, e.Discount, true}, {&merchant, e.Merchant, false}, {&reference, e.Reference, false}} {
			if pair.source == nil {
				continue
			}
			if *pair.dest != nil && **pair.dest != *pair.source {
				if !pair.decimal {
					return out, ErrClarification
				}
				left, e1 := money.NewMoney(**pair.dest, money.Asset(asset))
				right, e2 := money.NewMoney(*pair.source, money.Asset(asset))
				if e1 != nil || e2 != nil {
					return out, ErrClarification
				}
				if comparison, e := left.Compare(right); e != nil || comparison != 0 {
					return out, ErrClarification
				}
			}
			v := *pair.source
			*pair.dest = &v
		}
		if e.Paid != nil {
			if paid != nil && *paid != *e.Paid {
				return out, ErrClarification
			}
			v := *e.Paid
			paid = &v
		}
		for index, item := range e.Items {
			if !utf8.ValidString(item.Name) || strings.TrimSpace(item.Name) == "" || utf8.RuneCountInString(item.Name) > 2000 {
				return out, ErrClarification
			}
			gross, err := money.NewMoney(item.Gross, money.Asset(asset))
			if err != nil {
				return out, ErrClarification
			}
			var reduction *money.Money
			if item.Discount != nil {
				m, err := money.NewMoney(*item.Discount, money.Asset(asset))
				if err != nil {
					return out, ErrClarification
				}
				reduction = &m
			}
			sum := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d", identity, page+1, index+1)))
			inputs = append(inputs, ledger.ReceiptItemInput{ID: fmt.Sprintf("%x-%x-%x-%x-%x", sum[:4], sum[4:6], sum[6:8], sum[8:10], sum[10:16]), Name: item.Name, Quantity: item.Quantity, Gross: gross, Discount: reduction, CategoryID: item.CategoryID})
		}
	}
	if irrelevant == len(pages) {
		return out, ErrSkipped
	}
	if irrelevant > 0 {
		return out, ErrClarification
	}
	if kind == nil || paid == nil || !*paid || date == nil || currency == nil || *currency != asset || total == nil || len(inputs) > 1000 {
		return out, ErrClarification
	}
	var err error
	out.Kind = ledger.Type(*kind)
	out.Date, err = calendar.ParseDate(*date)
	if err != nil {
		return Facts{}, ErrClarification
	}
	if clock != nil {
		out.Time = *clock
	}
	if merchant != nil {
		if !utf8.ValidString(*merchant) || utf8.RuneCountInString(*merchant) > 2000 {
			return Facts{}, ErrClarification
		}
		out.Merchant = *merchant
	}
	if reference != nil {
		if !utf8.ValidString(*reference) || utf8.RuneCountInString(*reference) > 200 {
			return Facts{}, ErrClarification
		}
		out.Reference = *reference
	}
	out.Total, err = money.NewMoney(*total, money.Asset(asset))
	if err != nil || out.Total.Sign() <= 0 {
		return Facts{}, ErrClarification
	}
	if len(inputs) > 0 {
		d := "0"
		if discount != nil {
			d = *discount
		}
		reduction, err := money.NewMoney(d, money.Asset(asset))
		if err != nil {
			return Facts{}, ErrClarification
		}
		if discount == nil {
			allKnown := true
			for _, item := range inputs {
				if item.Discount == nil {
					allKnown = false
					break
				}
			}
			if allKnown {
				for _, item := range inputs {
					reduction, err = reduction.Add(*item.Discount)
					if err != nil {
						return Facts{}, ErrClarification
					}
				}
			}
		}
		out.Items, err = ledger.BuildReceiptItems(inputs, reduction, out.Total)
		if err != nil {
			return Facts{}, ErrClarification
		}
	}
	return out, nil
}
