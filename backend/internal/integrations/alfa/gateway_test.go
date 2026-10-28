package alfa_test

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	"github.com/pchkauu/want-keep/backend/internal/integrations/alfa"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func fixture(t *testing.T, name string) string {
	t.Helper()
	_, file, _, _ := runtime.Caller(0)
	data, err := os.ReadFile(filepath.Join(filepath.Dir(file), "../../../../collector/contracts/alfa/fixtures", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
func page(t *testing.T, mutate func(map[string]any)) ingestion.Page {
	t.Helper()
	b := map[string]any{"version": 1, "accounts": fixture(t, "accounts"), "cards": fixture(t, "cards"), "operations": fixture(t, "history"), "loyalty": fixture(t, "loyalty"), "summaries": []string{fixture(t, "summary")}, "account": "synthetic-current-a"}
	if mutate != nil {
		mutate(b)
	}
	data, _ := json.Marshal(b)
	hash := sha256.Sum256(data)
	binding := connections.Binding{Provider: "alfa", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("0", 64), CollectorImageDigest: "sha256:" + strings.Repeat("1", 64), ContractVersion: "10", AllowlistRevision: "alfa-read-v1", NonSecretConfigRevision: "test-v1", OperatorPermissionRevision: "synthetic-v1"}
	coverage, _ := reporting.NewCoverage(reporting.Partial, []string{"source_partial"})
	return ingestion.Page{Token: ingestion.JobToken{JobID: "11111111-1111-4111-8111-111111111111", ConnectionID: "22222222-2222-4222-8222-222222222222", Attempt: 1, LeaseToken: "synthetic-lease", ConnectionGeneration: 1, Binding: binding, AdmissionRevision: 1}, Complete: true, Coverage: coverage, Evidence: []ingestion.Evidence{{ID: "alfa-page", MediaType: "application/json", Digest: hex.EncodeToString(hash[:]), Locator: "alfa:structured-read:v1", Data: data}}}
}
func TestNormalizePreservesExactEvidenceAndWithholdsUnverifiedMoney(t *testing.T) {
	from, _ := calendar.ParseDate("2026-08-01")
	p, err := alfa.Normalize(page(t, nil), from)
	if err != nil {
		t.Fatal(err)
	}
	if len(p.Records) != 2 || p.Records[0].Account.Reference.AssetCode != "RUR" || p.Records[0].Account.OpeningDate != from {
		t.Fatal("account scope or opening changed")
	}
	a := p.Records[0].Account
	if len(a.Aliases) != 1 || a.Aliases[0].LastFour != "1234" {
		t.Fatal("card alias duplicated balance or retained credit product")
	}
	tr := p.Records[1].Transaction
	if tr.ProviderState != "unknown" || len(tr.Postings) != 0 || !tr.SemanticIdentity {
		t.Fatal("unqualified fact became a posting")
	}
	raw := string(p.Evidence[0].Data)
	if !strings.Contains(raw, "900719925474099312345") || strings.Contains(raw, "synthetic-secret") || strings.Contains(raw, "**** 1234") || strings.Contains(raw, "synthetic-credit") {
		t.Fatal("evidence lost exact number or retained excluded fields")
	}
	if p.Coverage.State() != reporting.Partial {
		t.Fatal("incomplete history became complete")
	}
}
func TestNormalizeRejectsAmbiguousOrMalformedPage(t *testing.T) {
	from, _ := calendar.ParseDate("2026-08-01")
	for name, mutate := range map[string]func(map[string]any){
		"duplicate": func(b map[string]any) {
			b["accounts"] = `{"accounts":[{"number":"x","type":"EE","amount":{"currency":"RUR"},"properties":{"creditCardAccount":false}},{"number":"x","type":"EE","amount":{"currency":"RUR"},"properties":{"creditCardAccount":false}}]}`
		},
		"foreign-account":     func(b map[string]any) { b["account"] = "foreign" },
		"invalid-json":        func(b map[string]any) { b["operations"] = "{" },
		"duplicate-operation": func(b map[string]any) { b["operations"] = `{"operations":[{"id":"x"},{"id":"x"}]}` },
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := alfa.Normalize(page(t, mutate), from); err == nil {
				t.Fatal("invalid page accepted")
			}
		})
	}
}
func TestUnknownStatusAndUnsupportedAssetAreNotGuessed(t *testing.T) {
	from, _ := calendar.ParseDate("2026-08-01")
	p, err := alfa.Normalize(page(t, func(b map[string]any) {
		b["accounts"] = strings.ReplaceAll(b["accounts"].(string), "RUR", "USDC.E")
		b["operations"] = strings.ReplaceAll(b["operations"].(string), `"SUCCESS"`, `null`)
	}), from)
	if err != nil || p.Records[0].Account.Reference.AssetCode != "USDC.E" || p.Records[1].Transaction.ProviderState != "unknown" {
		t.Fatal("unknown source data guessed", err)
	}
}

type reader struct {
	p     ingestion.Page
	reads int
}

func (r *reader) Binding() connections.Binding { return r.p.Token.Binding }
func (r *reader) Manifest(context.Context) (ingestion.Manifest, error) {
	return ingestion.Manifest{}, nil
}
func (r *reader) Read(context.Context, ingestion.JobToken) (ingestion.Result, error) {
	r.reads++
	return ingestion.Result{Page: &r.p}, nil
}
func TestGatewayChecksBindingBeforeRead(t *testing.T) {
	from, _ := calendar.ParseDate("2026-08-01")
	r := &reader{p: page(t, nil)}
	g := alfa.Gateway{Reader: r, HistoryFrom: from}
	token := r.p.Token
	token.Binding.AllowlistRevision = "other"
	if _, err := g.Read(context.Background(), token); err == nil || r.reads != 0 {
		t.Fatal("misbound request reached reader")
	}
	if _, err := g.Read(context.Background(), r.p.Token); err != nil {
		t.Fatal(err)
	}
}
