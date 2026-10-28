// Package alfa maps the closed cabinet read workflow into ingestion records.
package alfa

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"regexp"
	"strings"

	accountcatalog "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

const Namespace = "alfa-current-v1"

type Reader interface {
	Binding() connections.Binding
	Manifest(context.Context) (ingestion.Manifest, error)
	Read(context.Context, ingestion.JobToken) (ingestion.Result, error)
}
type Gateway struct {
	Reader      Reader
	HistoryFrom calendar.Date
}

func (g Gateway) Binding() connections.Binding { return g.Reader.Binding() }
func (g Gateway) Manifest(ctx context.Context) (ingestion.Manifest, error) {
	return g.Reader.Manifest(ctx)
}
func (g Gateway) Read(ctx context.Context, t ingestion.JobToken) (ingestion.Result, error) {
	if t.Binding.Provider != "alfa" || t.Binding.AllowlistRevision != "alfa-read-v1" || t.Binding != g.Binding() || g.HistoryFrom.String() == "" {
		return ingestion.Result{}, ingestion.ErrInvalidContract
	}
	result, err := g.Reader.Read(ctx, t)
	if err != nil || result.Failure != nil {
		return result, err
	}
	if result.Page == nil || !ingestion.SameToken(t, result.Page.Token) || len(result.Page.Evidence) != 1 || len(result.Page.Records) != 0 {
		return ingestion.Result{}, ingestion.ErrInvalidContract
	}
	p, err := Normalize(*result.Page, g.HistoryFrom)
	if err != nil {
		data := []byte(`{"code":"source_schema_changed"}`)
		hash := sha256.Sum256(data)
		return ingestion.Result{Failure: &ingestion.ProviderFailure{Token: t, Kind: ingestion.ContractViolation, SafeMessage: "Source contract requires verification", Evidence: []ingestion.Evidence{{ID: "alfa-source-schema", MediaType: "application/json", Locator: "alfa:read-status", Digest: hex.EncodeToString(hash[:]), Data: data}}}}, nil
	}
	return ingestion.Result{Page: &p}, nil
}

type batch struct {
	Version                              int `json:"version"`
	Accounts, Cards, Operations, Loyalty string
	Summaries                            []string
	Account                              string
}
type sourceAccount struct {
	Number, Type, OpenDate string
	Properties             struct{ CreditCardAccount *bool }
	Amount                 json.RawMessage
}
type sourceCard struct {
	ID, Number         string
	IsCredit, IsClosed *bool
	Account            struct{ Number string }
}

var safeID = regexp.MustCompile(`^[A-Za-z0-9_-]{1,128}$`)
var maskedCard = regexp.MustCompile(`[*xX•]`)
var lastFour = regexp.MustCompile(`[0-9]{4}$`)

// Normalize retains exact numeric tokens and never promotes unqualified source data to money.
func Normalize(page ingestion.Page, historyFrom calendar.Date) (ingestion.Page, error) {
	invalid := func() (ingestion.Page, error) { return ingestion.Page{}, ingestion.ErrInvalidContract }
	if page.Token.Binding.Provider != "alfa" || page.Validate() != nil || len(page.Evidence) != 1 || page.Evidence[0].Locator != "alfa:structured-read:v1" || historyFrom.String() == "" {
		return invalid()
	}
	var b batch
	d := json.NewDecoder(bytes.NewReader(page.Evidence[0].Data))
	d.DisallowUnknownFields()
	if d.Decode(&b) != nil || d.Decode(new(any)) != io.EOF || b.Version != 1 {
		return invalid()
	}
	var accounts struct{ Accounts []json.RawMessage }
	var cards struct{ Cards []sourceCard }
	if json.Unmarshal([]byte(b.Accounts), &accounts) != nil || accounts.Accounts == nil || len(accounts.Accounts) > 100 || json.Unmarshal([]byte(b.Cards), &cards) != nil || cards.Cards == nil || len(cards.Cards) > 500 {
		return invalid()
	}
	evidence := map[string]any{"version": 1, "account": b.Account}
	cleanAccounts := []any{}
	cleanCards := []any{}
	records := []ingestion.Record{}
	known := map[string]ingestion.AccountReference{}
	unsupportedAsset := false
	for _, raw := range accounts.Accounts {
		var a sourceAccount
		if json.Unmarshal(raw, &a) != nil {
			return invalid()
		}
		if a.Type != "EE" || a.Properties.CreditCardAccount == nil || *a.Properties.CreditCardAccount {
			continue
		}
		var amount struct {
			Currency   string
			Value      json.Number
			MinorUnits json.Number
		}
		if json.Unmarshal(a.Amount, &amount) != nil || !safeID.MatchString(a.Number) || amount.Currency == "" || len(amount.Currency) > 64 {
			return invalid()
		}
		if _, ok := known[a.Number]; ok {
			return invalid()
		}
		if _, err := accountcatalog.CanonicalSourceAsset("alfa", amount.Currency); err != nil {
			unsupportedAsset = true
		}
		ref := ingestion.AccountReference{ExternalAccountID: a.Number, Product: "current", AssetCode: amount.Currency}
		known[a.Number] = ref
		record := ingestion.AccountRecord{Reference: ref, Name: "Alfa " + amount.Currency, LogNamespace: Namespace, OpeningDate: historyFrom, EvidenceID: "alfa-page"}
		for _, c := range cards.Cards {
			if c.Account.Number != a.Number || c.IsCredit == nil || *c.IsCredit || c.IsClosed == nil || *c.IsClosed {
				continue
			}
			suffix := lastFour.FindString(strings.TrimSpace(c.Number))
			if !safeID.MatchString(c.ID) || !maskedCard.MatchString(c.Number) || suffix == "" {
				continue
			}
			record.Aliases = append(record.Aliases, ingestion.CardAlias{ID: c.ID, Label: "Debit card •••• " + suffix, LastFour: suffix})
			cleanCards = append(cleanCards, map[string]any{"id": c.ID, "lastFour": suffix, "account": a.Number})
		}
		canonical, _ := json.Marshal(record)
		records = append(records, ingestion.Record{Kind: ingestion.AccountRecordKind, Account: &record, CanonicalPayload: canonical})
		clean, err := selectFields(raw, "number", "type", "openDate", "amount", "total", "holds", "currentBalance", "properties")
		if err != nil {
			return invalid()
		}
		cleanAccounts = append(cleanAccounts, clean)
	}
	var operations struct {
		Operations []json.RawMessage
		PagesInfo  json.RawMessage
	}
	if json.Unmarshal([]byte(b.Operations), &operations) != nil || operations.Operations == nil || len(operations.Operations) > 20 {
		return invalid()
	}
	cleanOperations := []any{}
	seen := map[string]bool{}
	if b.Account != "" {
		if _, ok := known[b.Account]; !ok {
			return invalid()
		}
	}
	for _, raw := range operations.Operations {
		var op struct{ ID string }
		if json.Unmarshal(raw, &op) != nil || b.Account == "" || !safeID.MatchString(op.ID) || seen[op.ID] {
			return invalid()
		}
		seen[op.ID] = true
		clean, err := selectFields(raw, "id", "operationId", "smartVistaFrontReference", "dateTime", "title", "amount", "direction", "status", "loyaltyDetails")
		if err != nil {
			return invalid()
		}
		canonical, _ := json.Marshal(clean)
		tr := ingestion.TransactionRecord{ExternalAccountID: b.Account, Product: "current", LogNamespace: Namespace, ProviderRecordID: op.ID, Classification: "new", ProviderState: "unknown", EconomicType: "unknown", FeeKnowledge: "unknown", EvidenceID: "alfa-page", SemanticIdentity: true}
		records = append(records, ingestion.Record{Kind: ingestion.TransactionRecordKind, Transaction: &tr, CanonicalPayload: canonical})
		cleanOperations = append(cleanOperations, clean)
	}
	cleanLoyalty, err := selectFields([]byte(b.Loyalty), "accounts")
	if err != nil || len(b.Summaries) > 8 {
		return invalid()
	}
	summaries := []any{}
	for _, raw := range b.Summaries {
		clean, e := selectFields([]byte(raw), "account", "forecast", "statistics", "overallCashback")
		if e != nil {
			return invalid()
		}
		summaries = append(summaries, clean)
	}
	evidence["accounts"], evidence["cards"], evidence["operations"], evidence["loyalty"], evidence["summaries"] = cleanAccounts, cleanCards, cleanOperations, cleanLoyalty, summaries
	data, err := json.Marshal(evidence)
	if err != nil {
		return invalid()
	}
	hash := sha256.Sum256(data)
	page.Evidence = []ingestion.Evidence{{ID: "alfa-page", MediaType: "application/json", Digest: hex.EncodeToString(hash[:]), Locator: "alfa:normalized-source:v1", Data: data}}
	page.Records = records
	gaps := []string{"source_partial", "financial_mapping_unverified", "balance_as_of_unverified", "cashback_payout_unverified"}
	if unsupportedAsset {
		gaps = append(gaps, "unsupported_asset")
	}
	page.Coverage, _ = reporting.NewCoverage(reporting.Partial, gaps)
	return page, page.Validate()
}
func selectFields(raw []byte, keys ...string) (map[string]any, error) {
	d := json.NewDecoder(bytes.NewReader(raw))
	d.UseNumber()
	var object map[string]any
	if d.Decode(&object) != nil || object == nil || d.Decode(new(any)) != io.EOF {
		return nil, ingestion.ErrInvalidContract
	}
	result := map[string]any{}
	for _, key := range keys {
		if value, ok := object[key]; ok {
			result[key] = value
		}
	}
	return cleanEvidence(result).(map[string]any), nil
}

func cleanEvidence(value any) any {
	switch v := value.(type) {
	case map[string]any:
		result := map[string]any{}
		for k, x := range v {
			switch k {
			case "accounts", "account", "id", "number", "type", "openDate", "amount", "currency", "minorUnits", "value", "total", "holds", "currentBalance", "properties", "creditCardAccount", "closed", "blocked", "inactive", "operationId", "smartVistaFrontReference", "dateTime", "title", "direction", "status", "loyaltyDetails", "balance", "forecast", "totalActivePeriodAmount", "statistics", "charges", "month", "overallCashback":
				result[k] = cleanEvidence(x)
			}
		}
		return result
	case []any:
		result := make([]any, len(v))
		for i, x := range v {
			result[i] = cleanEvidence(x)
		}
		return result
	default:
		return v
	}
}
