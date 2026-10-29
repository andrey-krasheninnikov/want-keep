package contract

import (
	"bytes"
	"encoding/json"
	"io"
	"strings"
	"time"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

type Item struct {
	Name     string  `json:"name"`
	Quantity string  `json:"quantity"`
	Gross    string  `json:"gross"`
	Discount *string `json:"discount"`
	Category *string `json:"category"`
}
type Extraction struct {
	Document  string  `json:"document"`
	Kind      string  `json:"kind"`
	Paid      *bool   `json:"paid"`
	Date      *string `json:"date"`
	Time      *string `json:"time"`
	Asset     *string `json:"asset"`
	Total     *string `json:"total"`
	Discount  *string `json:"discount"`
	Merchant  *string `json:"merchant"`
	Reference *string `json:"reference"`
	Items     []Item  `json:"items"`
	Reason    string  `json:"reason"`
}

func Decode(data []byte) (receipt.Extraction, error) {
	var dto Extraction
	if len(data) == 0 || len(data) > 1<<20 || !utf8.Valid(data) || !uniqueKeys(json.NewDecoder(bytes.NewReader(data)), 0) {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	var fields map[string]json.RawMessage
	if json.Unmarshal(data, &fields) != nil || len(fields) != 12 {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	for _, name := range []string{"document", "kind", "paid", "date", "time", "asset", "total", "discount", "merchant", "reference", "items", "reason"} {
		if _, ok := fields[name]; !ok {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
	}
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&dto) != nil || decoder.Decode(new(any)) != io.EOF || dto.Items == nil || len(dto.Items) > 1000 || len(dto.Reason) > 8000 {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	if dto.Document != "receipt" && dto.Document != "irrelevant" && dto.Document != "uncertain" && dto.Document != "mixed" {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	if dto.Kind != "expense" && dto.Kind != "refund" && dto.Kind != "unknown" {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	if bytes.Equal(bytes.TrimSpace(fields["reason"]), []byte("null")) || utf8.RuneCountInString(dto.Reason) > 2000 {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	if dto.Date != nil {
		if _, err := calendar.ParseDate(*dto.Date); err != nil {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
	}
	if dto.Time != nil {
		value, err := time.Parse("15:04:05", *dto.Time)
		if err != nil || value.Format("15:04:05") != *dto.Time {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
	}
	if dto.Asset != nil {
		if _, err := money.NewMoney("0", money.Asset(*dto.Asset)); err != nil {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
	}
	for _, value := range []*string{dto.Total, dto.Discount} {
		if value != nil {
			amount, err := money.NewMoney(*value, money.RUB)
			if err != nil || strings.HasPrefix(*value, "-") || amount.Sign() < 0 {
				return receipt.Extraction{}, receipt.ErrInvalid
			}
		}
	}
	for _, value := range []*string{dto.Merchant, dto.Reference} {
		if value != nil && utf8.RuneCountInString(*value) > 2000 {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
	}
	if dto.Reference != nil && utf8.RuneCountInString(*dto.Reference) > 200 {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	var rawItems []map[string]json.RawMessage
	if json.Unmarshal(fields["items"], &rawItems) != nil {
		return receipt.Extraction{}, receipt.ErrInvalid
	}
	for index, item := range dto.Items {
		if len(rawItems[index]) != 5 {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
		for _, field := range []string{"name", "quantity", "gross", "discount", "category"} {
			if _, exists := rawItems[index][field]; !exists {
				return receipt.Extraction{}, receipt.ErrInvalid
			}
		}
		if utf8.RuneCountInString(item.Name) > 2000 || item.Name == "" {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
		for _, value := range []string{item.Quantity, item.Gross} {
			amount, err := money.NewMoney(value, money.RUB)
			if err != nil || strings.HasPrefix(value, "-") || amount.Sign() <= 0 {
				return receipt.Extraction{}, receipt.ErrInvalid
			}
		}
		if item.Discount != nil {
			amount, err := money.NewMoney(*item.Discount, money.RUB)
			if err != nil || strings.HasPrefix(*item.Discount, "-") || amount.Sign() < 0 {
				return receipt.Extraction{}, receipt.ErrInvalid
			}
		}
		if item.Category != nil && len(*item.Category) > 100 {
			return receipt.Extraction{}, receipt.ErrInvalid
		}
	}
	out := receipt.Extraction{Document: dto.Document, Kind: dto.Kind, Paid: dto.Paid, Date: dto.Date, Time: dto.Time, Asset: dto.Asset, Total: dto.Total, Discount: dto.Discount, Merchant: dto.Merchant, Reference: dto.Reference, Reason: dto.Reason, Items: []receipt.Item{}}
	for _, i := range dto.Items {
		out.Items = append(out.Items, receipt.Item{Name: i.Name, Quantity: i.Quantity, Gross: i.Gross, Discount: i.Discount, CategoryAlias: i.Category})
	}
	return out, nil
}

// Duplicate money fields must not acquire meaning from JSON parser ordering.
func uniqueKeys(decoder *json.Decoder, depth int) bool {
	if depth > 8 {
		return false
	}
	token, err := decoder.Token()
	if err != nil {
		return false
	}
	delimiter, container := token.(json.Delim)
	if !container {
		return true
	}
	switch delimiter {
	case '{':
		seen := map[string]bool{}
		for decoder.More() {
			token, err := decoder.Token()
			if err != nil {
				return false
			}
			key, ok := token.(string)
			if !ok || seen[key] {
				return false
			}
			seen[key] = true
			if !uniqueKeys(decoder, depth+1) {
				return false
			}
		}
	case '[':
		for decoder.More() {
			if !uniqueKeys(decoder, depth+1) {
				return false
			}
		}
	default:
		return false
	}
	_, err = decoder.Token()
	return err == nil
}
