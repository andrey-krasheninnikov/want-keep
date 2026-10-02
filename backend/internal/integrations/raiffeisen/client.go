package raiffeisen

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
)

var (
	ErrUnavailable = errors.New("raiffeisen unavailable")
	ErrUnknown     = errors.New("raiffeisen request outcome unknown")
	ErrResponse    = errors.New("raiffeisen response invalid")
	ErrReauth      = errors.New("raiffeisen authorization required")
	uuidSyntax     = regexp.MustCompile(`^[0-9a-fA-F]{8}-(?:[0-9a-fA-F]{4}-){3}[0-9a-fA-F]{12}$`)
	accountSyntax  = regexp.MustCompile(`^[0-9]{20}$`)
)

const accountsURL = "https://api.openapi.raiffeisen.ru/api/v1/accounts?fields=Id,Number,Name,OrganizationName,Currency"
const reportsURL = "https://api.raiffeisen.ru/bank-statements/v1/reports/"
const tokenURL = "https://sso.rbo.raiffeisen.ru/token"
const authorizeURL = "https://sso.rbo.raiffeisen.ru/authorize"

type Tokens struct {
	Access  string `json:"access_token"`
	ID      string `json:"id_token"`
	Refresh string `json:"refresh_token"`
	Type    string `json:"token_type"`
}

func (t Tokens) Validate() error {
	for _, v := range []string{t.Access, t.ID, t.Refresh} {
		if v == "" || len(v) > 65536 || strings.ContainsAny(v, "\r\n\x00") {
			return ErrResponse
		}
	}
	if !strings.EqualFold(t.Type, "bearer") {
		return ErrResponse
	}
	return nil
}

type Account struct {
	ID               string `json:"id"`
	Number           string `json:"number"`
	Name             string `json:"name"`
	OrganizationName string `json:"organizationName"`
	Currency         string `json:"currency"`
}

// Client admits only the legacy RBO read/report and OAuth routes. Redirects are forbidden.
type Client struct {
	http          *http.Client
	beforeRequest func(context.Context) error
}

func (c *Client) withPermit(permit func(context.Context) error) *Client {
	return &Client{http: c.http, beforeRequest: permit}
}

func NewClient(transport http.RoundTripper) *Client {
	if transport == nil {
		transport = http.DefaultTransport
	}
	return &Client{http: &http.Client{Transport: transport, Timeout: 30 * time.Second, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}}
}

func permitted(method, endpoint string) bool {
	if method == "GET" && endpoint == accountsURL || method == "POST" && endpoint == tokenURL {
		return true
	}
	if method == "POST" && (endpoint == reportsURL+"camt-053?dryRun=false" || endpoint == reportsURL+"camt-052?dryRun=false") {
		return true
	}
	if method == "GET" && strings.HasPrefix(endpoint, reportsURL) {
		parts := strings.Split(strings.TrimPrefix(endpoint, reportsURL), "/")
		return len(parts) == 2 && uuidSyntax.MatchString(parts[0]) && (parts[1] == "status" || parts[1] == "file")
	}
	return false
}

func (c *Client) request(ctx context.Context, method, endpoint string, tokens Tokens, body []byte, basic []string) (int, []byte, error) {
	if !permitted(method, endpoint) {
		return 0, nil, ErrResponse
	}
	if c.beforeRequest != nil {
		if err := c.beforeRequest(ctx); err != nil {
			return 0, nil, err
		}
	}
	r, err := http.NewRequestWithContext(ctx, method, endpoint, bytes.NewReader(body))
	if err != nil {
		return 0, nil, ErrResponse
	}
	if len(basic) == 2 {
		r.SetBasicAuth(basic[0], basic[1])
		r.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	} else {
		if tokens.Validate() != nil {
			return 0, nil, ErrReauth
		}
		r.Header.Set("Authorization", "Bearer "+tokens.Access)
		r.Header.Set("Id-Token", tokens.ID)
		r.Header.Set("Content-Type", "application/json")
	}
	response, err := c.http.Do(r)
	if err != nil {
		return 0, nil, ErrUnknown
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, ingestion.MaxEvidenceBytes+1))
	if err != nil {
		return response.StatusCode, nil, ErrUnknown
	}
	if len(data) > ingestion.MaxEvidenceBytes {
		return response.StatusCode, nil, ErrResponse
	}
	if response.StatusCode == 401 || response.StatusCode == 403 {
		return response.StatusCode, data, ErrReauth
	}
	if response.StatusCode >= 300 && response.StatusCode < 400 {
		return response.StatusCode, data, ErrResponse
	}
	return response.StatusCode, data, nil
}

func (c *Client) Accounts(ctx context.Context, t Tokens) ([]Account, []byte, error) {
	status, data, err := c.request(ctx, "GET", accountsURL, t, nil, nil)
	if err != nil {
		return nil, data, err
	}
	if status != 200 {
		return nil, data, ErrUnavailable
	}
	var accounts []Account
	if json.Unmarshal(data, &accounts) != nil || len(accounts) == 0 || len(accounts) > 100 {
		return nil, data, ErrResponse
	}
	seen := map[string]bool{}
	numbers := map[string]bool{}
	for _, a := range accounts {
		if !uuidSyntax.MatchString(a.ID) || !accountSyntax.MatchString(a.Number) || a.Name == "" || len(a.Name) > 2000 || a.Currency == "" || len(a.Currency) > 64 || seen[a.ID] || numbers[a.Number] {
			return nil, data, ErrResponse
		}
		seen[a.ID] = true
		numbers[a.Number] = true
	}
	return accounts, data, nil
}

func (c *Client) CreateReport(ctx context.Context, t Tokens, a Account, from, to time.Time, now time.Time) (string, []byte, error) {
	zone, _ := time.LoadLocation("Europe/Moscow")
	today := now.In(zone).Format(time.DateOnly)
	start, end := from.In(zone).Format(time.DateOnly), to.In(zone).Format(time.DateOnly)
	if !accountSyntax.MatchString(a.Number) || start > end || end > today {
		return "", nil, ErrResponse
	}
	kind := "camt-053"
	if end == today {
		if start != today {
			return "", nil, ErrResponse
		}
		kind = "camt-052"
	}
	body, _ := json.Marshal(struct {
		Accounts []string `json:"accountKeys"`
		From     string   `json:"from"`
		To       string   `json:"to"`
		Zero     bool     `json:"zero"`
	}{[]string{a.Number}, start, end, true})
	status, data, err := c.request(ctx, "POST", reportsURL+kind+"?dryRun=false", t, body, nil)
	if err != nil {
		return "", data, err
	}
	if status == 404 && noStatements(data) {
		return "", data, ErrNoStatements
	}
	if status != 202 {
		return "", data, ErrUnavailable
	}
	var v struct {
		ID string `json:"reportId"`
	}
	if json.Unmarshal(data, &v) != nil || !uuidSyntax.MatchString(v.ID) {
		return "", data, ErrUnknown
	}
	return v.ID, data, nil
}

var ErrNoStatements = errors.New("raiffeisen no statements")
var ErrReportPending = errors.New("raiffeisen report pending")

func noStatements(data []byte) bool {
	var v struct {
		Code string `json:"code"`
	}
	return json.Unmarshal(data, &v) == nil && (v.Code == "NO_STATEMENTS" || v.Code == "no-statements")
}
func (c *Client) Report(ctx context.Context, t Tokens, id string) ([]byte, []byte, error) {
	status, data, err := c.request(ctx, "GET", reportsURL+id+"/status", t, nil, nil)
	if err != nil {
		return nil, data, err
	}
	if status == 404 && noStatements(data) {
		return nil, data, ErrNoStatements
	}
	if status != 200 {
		return nil, data, ErrUnavailable
	}
	var v struct {
		Status string `json:"status"`
		ID     string `json:"reportId"`
	}
	if json.Unmarshal(data, &v) != nil || (v.ID != "" && v.ID != id) {
		return nil, data, ErrResponse
	}
	switch v.Status {
	case "completed", "COMPLETED":
	case "CREATED", "STARTED", "RESTARTED", "created", "started", "restarted":
		return nil, data, ErrReportPending
	case "FAILED", "STOPPED", "CANCELLED", "failed", "stopped", "cancelled":
		return nil, data, ErrUnavailable
	default:
		return nil, data, ErrResponse
	}
	status, file, err := c.request(ctx, "GET", reportsURL+id+"/file", t, nil, nil)
	if err != nil {
		return nil, file, err
	}
	if status != 200 {
		return nil, file, ErrUnavailable
	}
	return file, data, nil
}

func (c *Client) Exchange(ctx context.Context, clientID, secret, code, verifier, redirect string) (Tokens, error) {
	return c.token(ctx, clientID, secret, url.Values{"grant_type": {"authorization_code"}, "client_id": {clientID}, "code": {code}, "code_verifier": {verifier}, "redirect_uri": {redirect}})
}
func (c *Client) Refresh(ctx context.Context, clientID, secret, refresh string) (Tokens, error) {
	return c.token(ctx, clientID, secret, url.Values{"grant_type": {"refresh_token"}, "client_id": {clientID}, "refresh_token": {refresh}})
}
func (c *Client) token(ctx context.Context, clientID, secret string, values url.Values) (Tokens, error) {
	if clientID == "" || secret == "" {
		return Tokens{}, ErrReauth
	}
	status, data, err := c.request(ctx, "POST", tokenURL, Tokens{}, []byte(values.Encode()), []string{clientID, secret})
	defer clear(data)
	if err != nil {
		return Tokens{}, err
	}
	if status != 200 {
		if status >= 500 {
			return Tokens{}, ErrUnknown
		}
		return Tokens{}, ErrReauth
	}
	var t Tokens
	if json.Unmarshal(data, &t) != nil || t.Validate() != nil {
		return Tokens{}, ErrUnknown
	}
	return t, nil
}
