package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

func TestClientUsesQualifiedRequestShapeAndNoSDKRetry(t *testing.T) {
	var countBody, generationBody map[string]any
	countCalls, generationCalls := 0, 0
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Header.Get("Authorization") != "Bearer synthetic-key" || r.Header.Get("OpenAI-Project") != "synthetic-project" {
			t.Error("gateway omitted provider authentication scope")
		}
		switch r.URL.Path {
		case "/v1/responses/input_tokens":
			countCalls++
			decodeRequest(t, r, &countBody)
			return jsonResponse(t, http.StatusOK, map[string]any{"object": "response.input_tokens", "input_tokens": 100}), nil
		case "/v1/responses":
			generationCalls++
			decodeRequest(t, r, &generationBody)
			proposal := `{"version":"transaction_review_v1","caseId":"case-1","commands":[{"kind":"no_change","category":null,"merchant":null,"distribution":null,"member":null,"candidate":null,"question":null,"evidence":["ledger_revision"],"reason":"Confirmed existing transaction"}]}`
			return jsonResponse(t, http.StatusOK, map[string]any{
				"id": "resp_synthetic", "model": "gpt-5.6-terra", "status": "completed",
				"output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": proposal}}}},
				"usage": map[string]any{
					"input_tokens": 100, "input_tokens_details": map[string]any{"cached_tokens": 20, "cache_write_tokens": 10},
					"output_tokens": 50, "output_tokens_details": map[string]any{"reasoning_tokens": 12}, "total_tokens": 150,
				},
			}), nil
		default:
			return jsonResponse(t, http.StatusNotFound, map[string]any{"error": map[string]any{"message": "missing"}}), nil
		}
	})}

	client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
	request := validRequest(client.Contract())
	if count, err := client.Count(context.Background(), request); err != nil || count != 100 {
		t.Fatalf("count = %d, %v", count, err)
	}
	result, err := client.Generate(context.Background(), request)
	if err != nil {
		t.Fatal(err)
	}
	if result.State != ai.Completed || result.ProviderID != "resp_synthetic" || result.Usage.InputTokens != 100 || result.Usage.CachedTokens != 20 || result.Usage.CacheWriteTokens == nil || *result.Usage.CacheWriteTokens != 10 || result.Usage.OutputTokens != 50 || result.Usage.ReasoningTokens != 12 {
		t.Fatalf("unexpected result: %+v", result)
	}
	if countCalls != 1 || generationCalls != 1 {
		t.Fatalf("unexpected requests: count=%d generation=%d", countCalls, generationCalls)
	}
	assertCountShape(t, countBody)
	assertGenerationShape(t, generationBody)

	failureCalls := 0
	failureClient := &http.Client{Transport: roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		failureCalls++
		return jsonResponse(t, http.StatusInternalServerError, map[string]any{"error": map[string]any{"message": "temporary"}}), nil
	})}
	failing := newTestClientWithHTTP(t, "http://127.0.0.1/v1", failureClient)
	_, err = failing.Count(context.Background(), validRequest(failing.Contract()))
	var failure aiapp.GatewayFailure
	if !errors.As(err, &failure) || !failure.Retryable || failure.OutcomeUnknown || failureCalls != 1 {
		t.Fatalf("SDK retry or classification mismatch: calls=%d err=%v", failureCalls, err)
	}
}

func TestProductionRejectsRuntimeSchemaPendingQualification(t *testing.T) {
	_, err := New(Config{Environment: "production", APIKeyFile: "/does/not/exist"})
	if err == nil || !strings.Contains(err.Error(), "not admitted for production") {
		t.Fatalf("unqualified runtime contract admitted in production: %v", err)
	}
}

func TestGenerationTimeoutIsUnknown(t *testing.T) {
	httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		return nil, context.DeadlineExceeded
	})}
	client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
	_, err := client.Generate(context.Background(), validRequest(client.Contract()))
	var failure aiapp.GatewayFailure
	if !errors.As(err, &failure) || !failure.OutcomeUnknown || failure.Retryable {
		t.Fatalf("generation timeout was not unknown: %v", err)
	}
}

func TestCountRejectsMissingOrZeroInputTokens(t *testing.T) {
	for _, body := range []map[string]any{{"object": "response.input_tokens"}, {"object": "response.input_tokens", "input_tokens": 0}} {
		httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			return jsonResponse(t, http.StatusOK, body), nil
		})}
		client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
		if _, err := client.Count(context.Background(), validRequest(client.Contract())); err == nil {
			t.Fatalf("invalid count accepted: %#v", body)
		}
	}
}

func TestGenerationKeepsUnauditableResponseBehindReconciliation(t *testing.T) {
	proposal := `{"version":"transaction_review_v1","caseId":"case-1","commands":[{"kind":"no_change","category":null,"merchant":null,"distribution":null,"member":null,"candidate":null,"question":null,"evidence":["ledger_revision"],"reason":"Confirmed existing transaction"}]}`
	response := func() map[string]any {
		return map[string]any{
			"id": "resp_synthetic", "model": "gpt-5.6-terra", "status": "completed",
			"output": []any{map[string]any{"type": "message", "content": []any{map[string]any{"type": "output_text", "text": proposal}}}},
			"usage": map[string]any{
				"input_tokens": 100, "input_tokens_details": map[string]any{"cached_tokens": 20},
				"output_tokens": 50, "output_tokens_details": map[string]any{"reasoning_tokens": 12}, "total_tokens": 150,
			},
		}
	}
	tests := []struct {
		name   string
		mutate func(map[string]any)
	}{
		{"missing usage", func(body map[string]any) { delete(body, "usage") }},
		{"contradictory total", func(body map[string]any) { body["usage"].(map[string]any)["total_tokens"] = 149 }},
		{"model mismatch", func(body map[string]any) { body["model"] = "gpt-5.6-terra-preview" }},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body := response()
			test.mutate(body)
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(t, http.StatusOK, body), nil
			})}
			client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
			_, err := client.Generate(context.Background(), validRequest(client.Contract()))
			var failure aiapp.GatewayFailure
			if !errors.As(err, &failure) || !failure.OutcomeUnknown || failure.Observation.ID != "resp_synthetic" {
				t.Fatalf("response was not preserved for reconciliation: %+v, %v", failure, err)
			}
			if test.name == "contradictory total" {
				usage := failure.Observation.Usage
				if usage == nil || usage.InputTokens == nil || *usage.InputTokens != 100 || usage.OutputTokens == nil || *usage.OutputTokens != 50 || usage.TotalTokens == nil || *usage.TotalTokens != 149 {
					t.Fatalf("contradictory usage evidence was discarded: %+v", usage)
				}
			}
		})
	}
}

func TestProviderHTTPFailuresPreserveChargeUncertainty(t *testing.T) {
	testCases := []struct {
		status    int
		retryable bool
	}{{http.StatusUnauthorized, false}, {http.StatusForbidden, false}, {http.StatusRequestTimeout, true}, {http.StatusTooManyRequests, true}, {http.StatusInternalServerError, true}}
	calls := []struct {
		name string
		run  func(*Client, ai.Request) error
	}{
		{"count", func(client *Client, request ai.Request) error {
			_, err := client.Count(context.Background(), request)
			return err
		}},
		{"generation", func(client *Client, request ai.Request) error {
			_, err := client.Generate(context.Background(), request)
			return err
		}},
	}
	for _, call := range calls {
		for _, testCase := range testCases {
			t.Run(call.name+"/"+http.StatusText(testCase.status), func(t *testing.T) {
				requests := 0
				httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
					requests++
					return jsonResponse(t, testCase.status, map[string]any{"error": map[string]any{"message": "synthetic"}}), nil
				})}
				client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
				err := call.run(client, validRequest(client.Contract()))
				var failure aiapp.GatewayFailure
				wantUnknown := call.name == "generation" && (testCase.status == http.StatusRequestTimeout || testCase.status == http.StatusTooManyRequests || testCase.status >= 500)
				wantRetryable := testCase.retryable && !wantUnknown
				if testCase.status == http.StatusTooManyRequests {
					wantRetryable = false
				}
				wantConfirmedNoCharge := false
				if !errors.As(err, &failure) || failure.Retryable != wantRetryable || failure.OutcomeUnknown != wantUnknown || failure.ConfirmedNoCharge != wantConfirmedNoCharge || requests != 1 {
					t.Fatalf("failure classification: calls=%d failure=%+v err=%v", requests, failure, err)
				}
				if call.name == "count" && (testCase.status == http.StatusUnauthorized || testCase.status == http.StatusForbidden) && failure.Code != "provider_configuration_invalid" {
					t.Fatalf("count authentication code = %q", failure.Code)
				}
			})
		}
	}
}

func TestProvider429ClassificationUsesStructuredCodeAndBoundedRetry(t *testing.T) {
	tests := []struct {
		name, code, kind, wantCode string
		retryable, confirmed       bool
		wantDelay                  time.Duration
	}{
		{"request rate", "rate_limit_exceeded", "requests", "provider_rate_limited", true, true, 2 * time.Minute},
		{"quota", "insufficient_quota", "insufficient_quota", "provider_quota_exhausted", false, true, 0},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				response := jsonResponse(t, http.StatusTooManyRequests, map[string]any{"error": map[string]any{
					"message": "synthetic", "type": test.kind, "code": test.code, "param": nil,
				}})
				response.Header.Set("Retry-After", "120")
				return response, nil
			})}
			client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
			_, err := client.Generate(context.Background(), validRequest(client.Contract()))
			var failure aiapp.GatewayFailure
			if !errors.As(err, &failure) || failure.Code != test.wantCode || failure.Retryable != test.retryable || failure.ConfirmedNoCharge != test.confirmed || failure.OutcomeUnknown || failure.RetryAfter != test.wantDelay {
				t.Fatalf("429 classification mismatch: %+v, %v", failure, err)
			}
		})
	}

	response := &http.Response{Header: http.Header{"Retry-After": []string{"999999"}}}
	if delay := boundedRetryAfter(response); delay != 5*time.Minute {
		t.Fatalf("unbounded retry delay: %v", delay)
	}
}

func TestGenerationPreservesRefusalIncompleteAndSchemaError(t *testing.T) {
	validProposal := "{\"results\":[{\"id\":\"case-1\",\"action\":\"skip\",\"kind\":null,\"amount\":null,\"fee\":null,\"asset\":null,\"target\":null,\"month\":null,\"shares\":[],\"items\":[],\"evidence\":[\"ledger_revision\"],\"explanation\":\"ok\"}]}"
	testCases := []struct {
		name, status string
		content      []any
		want         ai.State
	}{
		{"refusal", "completed", []any{map[string]any{"type": "refusal", "refusal": "synthetic refusal"}}, ai.Refused},
		{"incomplete", "incomplete", []any{map[string]any{"type": "output_text", "text": validProposal}}, ai.Incomplete},
		{"schema", "completed", []any{map[string]any{"type": "output_text", "text": "{}"}}, ai.SchemaError},
	}
	for _, testCase := range testCases {
		t.Run(testCase.name, func(t *testing.T) {
			httpClient := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
				return jsonResponse(t, http.StatusOK, map[string]any{
					"id": "resp_synthetic", "model": "gpt-5.6-terra", "status": testCase.status,
					"output": []any{map[string]any{"type": "message", "content": testCase.content}},
					"usage": map[string]any{
						"input_tokens": 10, "input_tokens_details": map[string]any{"cached_tokens": 0, "cache_write_tokens": 0},
						"output_tokens": 5, "output_tokens_details": map[string]any{"reasoning_tokens": 1}, "total_tokens": 15,
					},
				}), nil
			})}
			client := newTestClientWithHTTP(t, "http://127.0.0.1/v1", httpClient)
			result, err := client.Generate(context.Background(), validRequest(client.Contract()))
			if err != nil || result.State != testCase.want || len(result.Output) != 0 {
				t.Fatalf("result = %+v, %v", result, err)
			}
		})
	}
}

func TestProposalValidationRejectsUnknownOrTrailingData(t *testing.T) {
	valid := []byte(`{"version":"transaction_review_v1","caseId":"case-1","commands":[{"kind":"no_change","category":null,"merchant":null,"distribution":null,"member":null,"candidate":null,"question":null,"evidence":["ledger_revision"],"reason":"Confirmed existing transaction"}]}`)
	expected := proposalInputCase{ID: "case-1", Source: "ledger_revision"}
	if err := validateProposal(valid, expected); err != nil {
		t.Fatal(err)
	}
	for _, data := range [][]byte{
		append(append([]byte(nil), valid...), []byte(` {}`)...),
		[]byte(`{"results":[],"unknown":true}`),
		[]byte(`{"results":[{"id":"1","action":"skip","kind":null,"amount":"1e3","fee":null,"asset":null,"target":null,"month":null,"shares":[],"items":[],"evidence":[],"explanation":"bad"}]}`),
	} {
		if err := validateProposal(data, expected); err == nil {
			t.Fatalf("invalid proposal accepted: %s", data)
		}
	}
	for _, field := range []string{"kind", "category", "merchant", "distribution", "member", "candidate", "question"} {
		var envelope struct {
			Commands []map[string]any `json:"commands"`
		}
		if err := json.Unmarshal(valid, &envelope); err != nil {
			t.Fatal(err)
		}
		delete(envelope.Commands[0], field)
		data, err := json.Marshal(envelope)
		if err != nil {
			t.Fatal(err)
		}
		if err = validateProposal(data, expected); err == nil {
			t.Fatalf("proposal without required %s accepted", field)
		}
	}
}

func TestProposalValidationBindsEvidenceAndRejectsMoney(t *testing.T) {
	expected := proposalInputCase{ID: "case-1", Source: "ledger_revision"}
	valid := []byte(`{"version":"transaction_review_v1","caseId":"case-1","commands":[{"kind":"no_change","category":null,"merchant":null,"distribution":null,"member":null,"candidate":null,"question":null,"evidence":["ledger_revision"],"reason":"Confirmed existing transaction"}]}`)
	if err := validateProposal(valid, expected); err != nil {
		t.Fatal(err)
	}
	missingSource := bytes.Replace(valid, []byte(`"ledger_revision"`), []byte(`"other"`), 1)
	if err := validateProposal(missingSource, expected); err == nil {
		t.Fatal("proposal without the bound source was accepted")
	}
}

func newTestClientWithHTTP(t *testing.T, baseURL string, httpClient *http.Client) *Client {
	t.Helper()
	path := filepath.Join(t.TempDir(), "openai-key")
	if err := os.WriteFile(path, []byte("synthetic-key\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	client, err := New(Config{APIKeyFile: path, ProjectID: "synthetic-project", BaseURL: baseURL, Environment: "test", HTTPClient: httpClient})
	if err != nil {
		t.Fatal(err)
	}
	return client
}

func validRequest(contract ai.RuntimeContract) ai.Request {
	return ai.Request{
		ID: "attempt", JobID: "job", ResourceID: "resource",
		HouseholdID: household.HouseholdID("household"), ActorID: household.UserID("actor"), ResourceRevision: 1,
		Purpose: ai.TransactionReview, Model: contract.Model, Qualification: contract.Qualification,
		PromptFingerprint: contract.PromptFingerprint, SchemaFingerprint: contract.SchemaFingerprint,
		ConfigFingerprint: contract.ConfigFingerprint, Input: json.RawMessage(`[{"id":"case-1","source":"ledger_revision","members":["actor"],"actor_id":"actor","text":"synthetic"}]`), MaximumOutputTokens: 2048,
	}
}

func decodeRequest(t *testing.T, r *http.Request, target *map[string]any) {
	t.Helper()
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.UseNumber()
	if err := decoder.Decode(target); err != nil {
		t.Fatal(err)
	}
}

func jsonResponse(t *testing.T, status int, value any) *http.Response {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return &http.Response{StatusCode: status, Header: http.Header{"Content-Type": []string{"application/json"}}, Body: io.NopCloser(strings.NewReader(string(data)))}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }

func assertCountShape(t *testing.T, body map[string]any) {
	t.Helper()
	assertKeys(t, body, "input", "instructions", "model", "parallel_tool_calls", "reasoning", "text")
	if body["model"] != "gpt-5.6-terra" || body["parallel_tool_calls"] != false {
		t.Fatalf("count contract mismatch: %#v", body)
	}
	assertReasoningAndSchema(t, body)
}

func assertGenerationShape(t *testing.T, body map[string]any) {
	t.Helper()
	assertKeys(t, body, "background", "input", "instructions", "max_output_tokens", "model", "parallel_tool_calls", "prompt_cache_options", "reasoning", "service_tier", "store", "text", "truncation")
	if body["model"] != "gpt-5.6-terra" || body["background"] != false || body["store"] != false || body["parallel_tool_calls"] != false || body["service_tier"] != "default" || body["truncation"] != "disabled" || body["max_output_tokens"] != json.Number("2048") {
		t.Fatalf("generation contract mismatch: %#v", body)
	}
	cache, _ := body["prompt_cache_options"].(map[string]any)
	if cache["mode"] != "explicit" {
		t.Fatalf("cache mode mismatch: %#v", cache)
	}
	input, ok := body["input"].(string)
	if !ok {
		t.Fatalf("runtime input is not the qualified case list: %#v", body["input"])
	}
	var cases []map[string]any
	if err := json.Unmarshal([]byte(input), &cases); err != nil || len(cases) != 1 || cases[0]["id"] != "case-1" || cases[0]["source"] != "ledger_revision" || cases[0]["actor_id"] != "actor" {
		t.Fatalf("runtime case binding mismatch: %#v, %v", cases, err)
	}
	assertReasoningAndSchema(t, body)
}

func assertReasoningAndSchema(t *testing.T, body map[string]any) {
	t.Helper()
	reasoning, _ := body["reasoning"].(map[string]any)
	text, _ := body["text"].(map[string]any)
	format, _ := text["format"].(map[string]any)
	if reasoning["effort"] != "xhigh" || reasoning["mode"] != "standard" || format["type"] != "json_schema" || format["strict"] != true || format["name"] != "want_keep_proposal" || format["schema"] == nil {
		t.Fatalf("reasoning/schema mismatch: %#v %#v", reasoning, format)
	}
}

func assertKeys(t *testing.T, body map[string]any, expected ...string) {
	t.Helper()
	if len(body) != len(expected) {
		t.Fatalf("unexpected fields: %#v", body)
	}
	for _, key := range expected {
		if _, ok := body[key]; !ok {
			t.Fatalf("missing field %q: %#v", key, body)
		}
	}
}
