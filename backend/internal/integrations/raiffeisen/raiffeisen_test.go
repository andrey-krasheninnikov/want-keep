package raiffeisen

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

func sample(t *testing.T) []byte {
	t.Helper()
	data, err := os.ReadFile("../../../../spec/001-want-keep-mvp/evidence/raiffeisen.camt053.sample.xml")
	if err != nil {
		t.Fatal(err)
	}
	// Stable structured party account references are independent of optional aliases.
	return []byte(strings.ReplaceAll(string(data), "<RmtInf>", "<RltdPties><DbtrAcct><Id><Othr><Id>SYNTHETIC-DEBTOR</Id></Othr></Id></DbtrAcct></RltdPties><RmtInf>"))
}
func sampleAccount() Account {
	return Account{ID: "00000000-0000-4000-8000-000000000001", Number: "00000000000000000000", Name: "Synthetic current account", Currency: "RUR"}
}

func TestCAMTIdentityAndAmounts(t *testing.T) {
	data := sample(t)
	first, err := Normalize(data, sampleAccount(), "synthetic-report-one")
	if err != nil {
		t.Fatal(err)
	}
	if len(first.Facts) != 3 || first.Closing == nil || first.Closing.Owned.Value != "990.00" || first.Closing.Available.Value != "" {
		t.Fatal("native records or unknown balance lost")
	}
	if first.Facts[2].Record.Transaction.ProviderState != "unknown" {
		t.Fatal("proprietary code invented a fee")
	}
	changed := strings.ReplaceAll(string(data), "<NtryRef>SYNTHETIC-ENTRY-1</NtryRef>", "")
	changed = strings.ReplaceAll(changed, "<EndToEndId>SYNTHETIC-END-1</EndToEndId>", "")
	changed = strings.ReplaceAll(changed, "1000.00", "1000.000000000000000123")
	changed = strings.ReplaceAll(changed, "2026-08-10T12:00:00+03:00", "2026-08-10T12:00:01+03:00")
	second, err := Normalize([]byte(changed), sampleAccount(), "synthetic-report-two")
	if err != nil {
		t.Fatal(err)
	}
	a, b := first.Facts[0].Record, second.Facts[0].Record
	if a.Transaction.ProviderRecordID != b.Transaction.ProviderRecordID || string(a.CanonicalPayload) == string(b.CanonicalPayload) || b.Transaction.Postings[0].Amount != "1000.000000000000000123" {
		t.Fatal("identity included optional aliases, amount or date")
	}
	for _, alias := range []string{"InstrId", "TxId"} {
		withAlias := strings.ReplaceAll(string(data), "<Refs>", "<Refs><"+alias+">SYNTHETIC-OPTIONAL</"+alias+">")
		aliased, err := Normalize([]byte(withAlias), sampleAccount(), "synthetic-alias")
		if err != nil || aliased.Facts[0].Record.Transaction.ProviderRecordID != a.Transaction.ProviderRecordID {
			t.Fatal("optional structured reference changed identity", err)
		}
	}
	withProprietary := strings.ReplaceAll(string(data), "<Refs>", "<Refs><Prtry><Ref>SYNTHETIC-OPTIONAL</Ref></Prtry>")
	aliased, err := Normalize([]byte(withProprietary), sampleAccount(), "synthetic-proprietary")
	if err != nil || aliased.Facts[0].Record.Transaction.ProviderRecordID != a.Transaction.ProviderRecordID {
		t.Fatal("optional proprietary reference changed identity", err)
	}
	for _, weak := range []string{
		strings.ReplaceAll(string(data), "<Ustrd>Синтетическое поступление</Ustrd>", "<Ustrd>  </Ustrd>"),
		strings.ReplaceAll(string(data), "SYNTHETIC-DEBTOR", "NOTPROVIDED"),
	} {
		statement, err := Normalize([]byte(weak), sampleAccount(), "synthetic-weak")
		if err != nil || statement.Facts[0].Record.Transaction.Classification != "ambiguous" {
			t.Fatal("insufficient identity posted", err)
		}
	}
	for _, bad := range []string{strings.ReplaceAll(string(data), "00000000000000000000", "11111111111111111111"), strings.ReplaceAll(string(data), "1000.00", "NaN"), strings.ReplaceAll(string(data), "camt.053.001.08", "camt.053.001.02"), "<!DOCTYPE x [<!ENTITY x SYSTEM 'file:///etc/passwd'>]>" + string(data)} {
		if _, err := Normalize([]byte(bad), sampleAccount(), "synthetic"); err == nil {
			t.Fatal("invalid statement accepted")
		}
	}
	entries := strings.Index(string(data), "<Ntry>")
	mismatched := string(data[:entries]) + strings.ReplaceAll(string(data[entries:]), `Ccy="RUB"`, `Ccy="USD"`)
	if _, err := Normalize([]byte(mismatched), sampleAccount(), "synthetic-currency"); err != ErrResponse {
		t.Fatal("entry currency reassigned to account", err)
	}
	placeholder := strings.ReplaceAll(string(data), "SYNTHETIC-ENTRY-1", "NOTPROVIDED")
	placeholder = strings.ReplaceAll(placeholder, "SYNTHETIC-END-1", "NOTPROVIDED")
	statement, err := Normalize([]byte(placeholder), sampleAccount(), "synthetic-placeholder")
	if err != nil || len(statement.Facts[0].Aliases) != 0 {
		t.Fatal("placeholder became correction evidence", err)
	}
	// Two details replace the entry principal; the entry itself is never posted.
	detail := `<TxDtls><Refs><InstrId>PART-B</InstrId></Refs><Amt Ccy="RUB">400.00</Amt><CdtDbtInd>CRDT</CdtDbtInd><RmtInf><Ustrd>Part B</Ustrd></RmtInf></TxDtls>`
	multi := strings.Replace(string(data), `<EndToEndId>SYNTHETIC-END-1</EndToEndId></Refs><Amt Ccy="RUB">1000.00</Amt>`, `<EndToEndId>SYNTHETIC-END-1</EndToEndId></Refs><Amt Ccy="RUB">600.00</Amt>`, 1)
	multi = strings.Replace(multi, "</TxDtls></NtryDtls>", "</TxDtls>"+detail+"</NtryDtls>", 1)
	got, err := Normalize([]byte(multi), sampleAccount(), "synthetic")
	if err != nil || len(got.Facts) != 4 || got.Facts[0].Record.Transaction.Postings[0].Amount != "600.00" || got.Facts[1].Record.Transaction.Postings[0].Amount != "400.00" {
		t.Fatalf("1:N normalization: %v", err)
	}
}

type transportFunc func(*http.Request) (*http.Response, error)

func (f transportFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestHTTPAllowlistAndUnknownTokenExchange(t *testing.T) {
	calls := 0
	client := NewClient(transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		if r.URL.String() != tokenURL {
			t.Fatal("wrong route")
		}
		username, password, ok := r.BasicAuth()
		if !ok || username != "synthetic-client" || password != "synthetic-secret" {
			t.Fatal("invalid basic auth")
		}
		data, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(data), "grant_type=refresh_token") {
			t.Fatal("wrong grant")
		}
		return &http.Response{StatusCode: 502, Body: io.NopCloser(strings.NewReader(`{"error":"synthetic"}`)), Header: http.Header{}}, nil
	}))
	if _, _, err := client.request(context.Background(), "POST", "https://api.raiffeisen.ru/payments", Tokens{}, nil, nil); err == nil || calls != 0 {
		t.Fatal("payment route admitted")
	}
	if _, err := client.Refresh(context.Background(), "synthetic-client", "synthetic-secret", "synthetic-refresh"); err != ErrUnknown || calls != 1 {
		t.Fatal("ambiguous rotation retried")
	}
	if !permitted("GET", reportsURL+"00000000-0000-4000-8000-000000000001/file") || permitted("GET", reportsURL+"../token/file") {
		t.Fatal("report allowlist")
	}
}

func TestReportRechecksPermit(t *testing.T) {
	allowed, calls := true, 0
	client := NewClient(transportFunc(func(r *http.Request) (*http.Response, error) {
		calls++
		allowed = false
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"status":"completed"}`)), Header: http.Header{}}, nil
	})).withPermit(func(context.Context) error {
		if !allowed {
			return connections.ErrProviderNotAdmitted
		}
		return nil
	})
	_, _, err := client.Report(context.Background(), Tokens{Access: "synthetic", ID: "synthetic", Refresh: "synthetic", Type: "bearer"}, sampleAccount().ID)
	if err != connections.ErrProviderNotAdmitted || calls != 1 {
		t.Fatal("file requested after revocation", err, calls)
	}
}

func TestOIDCSignatureAndBindings(t *testing.T) {
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 3, 10, 0, 0, 0, time.UTC)
	a := &Authorizer{Client: NewClient(nil), Config: OAuthConfig{ClientID: "synthetic-client", ClientSecret: "synthetic-secret", RedirectURI: "https://want-keep.tech/api/v1/connections/raiffeisen/callback", Issuer: "https://sso.rbo.raiffeisen.ru", Keys: map[string]*rsa.PublicKey{"synthetic": &key.PublicKey}}}
	claims := idClaims{RegisteredClaims: jwt.RegisteredClaims{Issuer: a.Config.Issuer, Audience: jwt.ClaimStrings{a.Config.ClientID}, Subject: "synthetic-owner", IssuedAt: jwt.NewNumericDate(now), ExpiresAt: jwt.NewNumericDate(now.Add(time.Hour))}, Nonce: "synthetic-nonce"}
	token := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
	token.Header["kid"] = "synthetic"
	encoded, err := token.SignedString(key)
	if err != nil {
		t.Fatal(err)
	}
	bundle := Tokens{Access: "synthetic-access", ID: encoded, Refresh: "synthetic-refresh", Type: "bearer"}
	if _, err := a.validate(bundle, claims.Nonce, "", now); err != nil {
		t.Fatal(err)
	}
	// The provider mints the token after the request starts. Validation uses response time.
	a.Now = func() time.Time { return now }
	a.Client = NewClient(transportFunc(func(*http.Request) (*http.Response, error) {
		now = now.Add(time.Second)
		claims.IssuedAt = jwt.NewNumericDate(now)
		responseToken := jwt.NewWithClaims(jwt.SigningMethodRS256, claims)
		responseToken.Header["kid"] = "synthetic"
		value, err := responseToken.SignedString(key)
		if err != nil {
			t.Fatal(err)
		}
		response, _ := json.Marshal(Tokens{Access: "synthetic-access", ID: value, Refresh: "synthetic-refresh", Type: "bearer"})
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(string(response))), Header: http.Header{}}, nil
	}))
	if _, err := a.Exchange(context.Background(), "synthetic-code", connections.OAuthSecrets{Nonce: claims.Nonce}); err != nil {
		t.Fatal("response-time exchange rejected", err)
	}
	if _, err := a.Refresh(context.Background(), connections.TokenSet{Refresh: "synthetic-refresh", Subject: claims.Subject}); err != nil {
		t.Fatal("response-time refresh rejected", err)
	}
	for _, input := range []struct {
		nonce, subject string
		at             time.Time
	}{{"other", "", now}, {claims.Nonce, "other", now}, {claims.Nonce, "", now.Add(2 * time.Hour)}} {
		if _, err := a.validate(bundle, input.nonce, input.subject, input.at); err == nil {
			t.Fatal("OIDC binding bypass")
		}
	}
	url, err := a.URL(connections.OAuthSecrets{State: strings.Repeat("s", 43), Nonce: strings.Repeat("n", 43), Verifier: strings.Repeat("v", 43)})
	if err != nil || !strings.Contains(url, "code_challenge_method=S256") {
		t.Fatal("PKCE missing")
	}
	var retained map[string]string
	if json.Unmarshal(rawEvidence([]byte("invalid-but-retained"), "application/json", "synthetic").Data, &retained) != nil || retained["Encoding"] != "base64" {
		t.Fatal("invalid provider bytes discarded")
	}
}
