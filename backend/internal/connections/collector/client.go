package collector

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"path/filepath"
	"sync"
	"time"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
)

var ErrUnavailable = errors.New("browser collector unavailable")
var ErrBusy = errors.New("browser collector busy")
var ErrPreflightRejected = errors.New("browser collector rejected request before provider IO")
var ErrSessionInvalid = errors.New("browser collector session invalid")
var ErrBeforeIOUnavailable = errors.New("browser collector unavailable before provider IO")

const maxResponseBytes = 32 * 1024 * 1024

type Client struct {
	http              *http.Client
	binding           connections.Binding
	admissionRevision int64
	session           json.RawMessage
	beforeRead        func(context.Context) error
	mu                sync.Mutex
	last              readOutcome
	externalStarted   bool
}

type readOutcome struct {
	page, complete, failure bool
}

type bindingWire struct {
	Provider                   string `json:"provider"`
	Environment                string `json:"environment"`
	AdapterBuildDigest         string `json:"adapterBuildDigest"`
	CollectorImageDigest       string `json:"collectorImageDigest"`
	ContractVersion            string `json:"contractVersion"`
	AllowlistRevision          string `json:"allowlistRevision"`
	NonSecretConfigRevision    string `json:"nonSecretConfigRevision"`
	OperatorPermissionRevision string `json:"operatorPermissionRevision"`
}

func NewClient(socket string, binding connections.Binding, admissionRevision int64, session []byte, beforeRead func(context.Context) error) (*Client, error) {
	if !filepath.IsAbs(socket) || binding.Validate() != nil || admissionRevision < 1 || admissionRevision > ingestion.MaxAdmissionRevision || beforeRead == nil {
		return nil, ErrUnavailable
	}
	if len(session) < 2 || len(session) > 1024*1024 || !json.Valid(session) {
		return nil, ErrSessionInvalid
	}
	var state struct {
		Cookies []json.RawMessage `json:"cookies"`
		Origins []json.RawMessage `json:"origins"`
	}
	decoder := json.NewDecoder(bytes.NewReader(session))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&state) != nil || decoder.Decode(new(any)) != io.EOF || state.Cookies == nil || state.Origins == nil {
		return nil, ErrSessionInvalid
	}
	dialer := &net.Dialer{Timeout: time.Second}
	transport := &http.Transport{
		DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return dialer.DialContext(ctx, "unix", socket)
		},
		DisableKeepAlives:      true,
		MaxResponseHeaderBytes: 4096,
	}
	return &Client{
		http:    &http.Client{Transport: transport, Timeout: 2 * time.Minute, CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }},
		binding: binding, admissionRevision: admissionRevision, session: append(json.RawMessage(nil), session...), beforeRead: beforeRead,
	}, nil
}

func (c *Client) CapabilityManifest(ctx context.Context) ([]byte, error) {
	return c.call(ctx, "/v1/capabilities", struct {
		Version           int         `json:"version"`
		Binding           bindingWire `json:"binding"`
		AdmissionRevision int64       `json:"admissionRevision"`
	}{1, wireBinding(c.binding), c.admissionRevision})
}

func (c *Client) Read(ctx context.Context, request []byte) ([]byte, error) {
	if len(request) < 2 || len(request) > 64*1024 || !json.Valid(request) {
		return nil, ingestion.ErrInvalidContract
	}
	if err := c.beforeRead(ctx); err != nil {
		return nil, err
	}
	c.mu.Lock()
	c.externalStarted = true
	c.mu.Unlock()
	payload, err := c.call(ctx, "/v1/read", struct {
		Version      int             `json:"version"`
		SyncRequest  json.RawMessage `json:"syncRequest"`
		StorageState json.RawMessage `json:"storageState"`
	}{1, request, c.session})
	if err != nil {
		return nil, err
	}
	var outcome struct {
		Outcome string `json:"outcome"`
		Page    *struct {
			Complete bool `json:"complete"`
		} `json:"page"`
		Failure json.RawMessage `json:"failure"`
	}
	if json.Unmarshal(payload, &outcome) == nil {
		c.mu.Lock()
		c.last = readOutcome{page: outcome.Outcome == "page" && outcome.Page != nil, complete: outcome.Page != nil && outcome.Page.Complete, failure: outcome.Outcome == "failure" && len(outcome.Failure) > 0}
		c.mu.Unlock()
	}
	return payload, nil
}

func (c *Client) ExternalStarted() bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.externalStarted
}

func (c *Client) Outcome() (page, complete, failure bool) {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.last.page, c.last.complete, c.last.failure
}

func (c *Client) Close() {
	if c == nil {
		return
	}
	clear(c.session)
	c.session = nil
	c.http.CloseIdleConnections()
}

func (c *Client) Ready(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, time.Second)
	defer cancel()
	r, err := http.NewRequestWithContext(ctx, http.MethodGet, "http://collector/ready", nil)
	if err != nil {
		return false
	}
	response, err := c.http.Do(r)
	if err != nil {
		return false
	}
	defer response.Body.Close()
	data, err := io.ReadAll(io.LimitReader(response.Body, 64))
	return err == nil && response.StatusCode == http.StatusOK && string(data) == "want-keep-browser-collector/1\n"
}

func (c *Client) call(ctx context.Context, path string, value any) ([]byte, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return nil, ErrUnavailable
	}
	r, err := http.NewRequestWithContext(ctx, http.MethodPost, "http://collector"+path, bytes.NewReader(data))
	if err != nil {
		return nil, ErrUnavailable
	}
	r.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(r)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, ErrUnavailable
	}
	defer response.Body.Close()
	payload, err := io.ReadAll(io.LimitReader(response.Body, maxResponseBytes+1))
	if err == nil && path == "/v1/read" && response.StatusCode == http.StatusConflict && bytes.Equal(payload, []byte(`{"code":"collector_busy"}`)) {
		return nil, ErrBusy
	}
	if err == nil && path == "/v1/read" && response.StatusCode == http.StatusUnprocessableEntity && bytes.Equal(payload, []byte(`{"code":"collector_preflight_rejected"}`)) {
		return nil, ErrPreflightRejected
	}
	if err == nil && path == "/v1/read" && response.StatusCode == http.StatusUnprocessableEntity && bytes.Equal(payload, []byte(`{"code":"collector_session_invalid"}`)) {
		return nil, ErrSessionInvalid
	}
	if err == nil && path == "/v1/read" && response.StatusCode == http.StatusServiceUnavailable && bytes.Equal(payload, []byte(`{"code":"collector_before_io_unavailable"}`)) {
		return nil, ErrBeforeIOUnavailable
	}
	if err != nil || len(payload) > maxResponseBytes || response.StatusCode != http.StatusOK {
		return nil, ErrUnavailable
	}
	return payload, nil
}

func wireBinding(value connections.Binding) bindingWire {
	return bindingWire{
		Provider: value.Provider, Environment: value.Environment, AdapterBuildDigest: value.AdapterBuildDigest,
		CollectorImageDigest: value.CollectorImageDigest, ContractVersion: value.ContractVersion,
		AllowlistRevision: value.AllowlistRevision, NonSecretConfigRevision: value.NonSecretConfigRevision,
		OperatorPermissionRevision: value.OperatorPermissionRevision,
	}
}
