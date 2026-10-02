package contract_test

import (
	"encoding/json"
	"strings"
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	command "github.com/pchkauu/want-keep/backend/internal/commands/domain"
	connection "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
)

func TestAdmissionReadBoundaryAndUntrustedCommands(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	binding := connection.Binding{Provider: "bybit", Environment: "production", AdapterBuildDigest: "sha256:" + strings.Repeat("a", 64), CollectorImageDigest: "sha256:" + strings.Repeat("b", 64), ContractVersion: "10", AllowlistRevision: "1", NonSecretConfigRevision: "1", OperatorPermissionRevision: "1"}
	pending, _ := connection.NewAdmission(binding)
	at, _ := calendar.ParseInstant("2026-09-07T00:00:00Z")
	provider, _ := pending.RecordCheck(connection.Check{Kind: connection.ProviderCheck, Binding: binding, Result: connection.CheckPassed, At: at})
	admitted, _ := provider.RecordCheck(connection.Check{Kind: connection.HostCheck, Binding: binding, Result: connection.CheckPassed, At: at})
	blocked, _ := pending.RecordCheck(connection.Check{Kind: connection.HostCheck, Binding: binding, Result: connection.CheckFailed, At: at})
	for _, admission := range []connection.Admission{{}, pending, provider, admitted, blocked} {
		dto, err := b.AdmissionToDTO(admission)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(dto)
		var result struct {
			Status            string
			Binding           *generated.DeploymentBinding
			AdmissionRevision *int64
		}
		if err := json.Unmarshal(data, &result); err != nil || result.Status != string(admission.Status()) {
			t.Fatal("admission state lost")
		}
		if admission.Binding().Provider != "" && (result.Binding == nil || result.Binding.OperatorPermissionRevision != "1" || result.Binding.AdapterBuildDigest != binding.AdapterBuildDigest) {
			t.Fatal("binding lost")
		}
		if admission.Revision() == 0 {
			if result.AdmissionRevision != nil || result.Binding != nil {
				t.Fatal("missing admission was assigned a revision or binding")
			}
		} else if result.AdmissionRevision == nil || *result.AdmissionRevision != admission.Revision() {
			t.Fatal("admission revision lost")
		}
		var fields map[string]json.RawMessage
		if err := json.Unmarshal(data, &fields); err != nil {
			t.Fatal(err)
		}
		if admission.Revision() == 0 {
			fields["checkedAt"] = json.RawMessage(`"2026-09-07T00:00:00Z"`)
			invalid, _ := json.Marshal(fields)
			var raw json.RawMessage
			if b.Decode("DeploymentGate", invalid, &raw) == nil {
				t.Fatal("check time without admission accepted")
			}
			continue
		}
		for _, revision := range []string{"", "0", "-1", "1.5", "9007199254740992", `"3"`, "null"} {
			if revision == "" {
				delete(fields, "admissionRevision")
			} else {
				fields["admissionRevision"] = json.RawMessage(revision)
			}
			invalid, _ := json.Marshal(fields)
			var raw json.RawMessage
			if b.Decode("DeploymentGate", invalid, &raw) == nil {
				t.Fatal("invalid admission revision accepted", revision)
			}
		}
		fields["admissionRevision"] = json.RawMessage("9007199254740991")
		maximum, _ := json.Marshal(fields)
		var raw json.RawMessage
		if b.Decode("DeploymentGate", maximum, &raw) != nil {
			t.Fatal("maximum exact revision rejected")
		}
		delete(fields, "binding")
		invalid, _ := json.Marshal(fields)
		if b.Decode("DeploymentGate", invalid, &raw) == nil {
			t.Fatal("revision without binding accepted")
		}
	}
	for _, input := range []struct{ schema, data string }{
		{"DeploymentGate", `{"status":"admitted","reasons":[]}`},
		{"DeploymentGate", `{"status":"pending","reasons":[]}`},
		{"DeploymentGate", `{"status":"blocked","reasons":["secret details"]}`},
		{"ConnectionCreate", `{"provider":"bybit","externalAccountOwnerId":"10000000-0000-4000-8000-000000000001","historyFrom":"2026-09-07","products":["funding"],"deploymentGate":{"status":"admitted"}}`},
		{"ConnectionAction", `{"expectedRevision":1,"deploymentGate":{"status":"admitted"}}`},
		{"ConnectionAction", `{"expectedRevision":1,"admissionRevision":3}`},
		{"ConnectionCreate", `{"provider":"bybit","externalAccountOwnerId":"10000000-0000-4000-8000-000000000001","historyFrom":"2026-09-07","products":["funding"],"admissionRevision":3}`},
	} {
		var value json.RawMessage
		if b.Decode(input.schema, []byte(input.data), &value) == nil {
			t.Fatalf("invalid %s accepted", input.schema)
		}
	}
	response := (contract.ErrorConverter{}).ToResponse(connection.ErrProviderNotAdmitted, "10000000-0000-4000-8000-000000000001")
	if response.Status != 409 || response.Body.Code != "provider_not_admitted" || response.Body.Retryable {
		t.Fatal("unsafe admission error")
	}
}

func TestExpiredOutcomeAndEveryMutationResponse(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const id = "10000000-0000-4000-8000-000000000001"
	for _, outcome := range []*command.Outcome{nil, {CommandID: id, Status: command.Succeeded, Result: command.Result{ResourceType: "transaction", ResourceID: id, Revision: 2}}, {CommandID: id, Status: command.Failed, FailureCode: "version_conflict"}} {
		dto, err := b.ExpiredCommandToDTO(id, outcome)
		if err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(dto)
		var roundtrip generated.ExpiredCommand
		if err := b.Decode("ExpiredCommand", data, &roundtrip); err != nil {
			t.Fatal(err)
		}
		if (roundtrip.Outcome == nil) != (outcome == nil) {
			t.Fatal("outcome visibility lost")
		}
	}
	for _, status := range []command.Status{command.Pending, "invalid"} {
		if _, err := b.OutcomeToDTO(command.Outcome{CommandID: id, Status: status}); err == nil {
			t.Fatal("nonterminal outcome accepted")
		}
	}
	problem := (contract.ErrorConverter{}).ToResponse(command.ErrCommandExpired, id)
	data, _ := json.Marshal(problem.Body)
	var dto generated.ExpiredCommand
	if problem.Status != 410 || b.Decode("ExpiredCommand", data, &dto) != nil {
		t.Fatal("generic expiration error differs from HTTP schema")
	}
	for _, code := range []string{"source_ambiguous", "quote_unavailable", "command_expired", "provider_not_admitted"} {
		var v generated.ErrorCode
		data, _ := json.Marshal(code)
		if b.Decode("ErrorCode", data, &v) != nil {
			t.Fatal("missing safe error", code)
		}
	}
	schema, _ := generated.GetSwagger()
	if schema.Paths.Find("/commands/{commandId}").Get.Responses.Status(410) == nil {
		t.Fatal("missing detail expiration")
	}
	for path, item := range schema.Paths.Map() {
		for _, op := range item.Operations() {
			if _, ok := op.Extensions["x-command-type"]; ok && op.Responses.Status(410) == nil {
				t.Error("missing replay expiration", path)
			}
		}
	}
}

func TestRatesExplainUnavailableAndQuoteCoverage(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const quality = `"quality":{"coverage":{"state":"complete","reasons":[]},"freshness":"fresh"}`
	const quote = `{"method":"platform_quote","provider":"bybit","direction":"sell_base","base":"USDT","quote":"USD","applicableAmount":{"asset":"USDT","amount":"1.000000000001"},"rate":{"base":"USDT","quote":"USD","value":"0.9999"},"observedAt":"2026-09-07T00:00:00Z","fees":[],"feeCoverage":"included","spreadCoverage":"included",` + quality + `}`
	for _, data := range []string{quote, `{"method":"unavailable","base":"BTC","quote":"USD","requestedDate":"2020-01-01","reason":"valuation_unavailable","detail":"history_out_of_range",` + quality + `}`} {
		var dto generated.RateObservation
		if err := b.Decode("RateObservation", []byte(data), &dto); err != nil {
			t.Fatal(err)
		}
		encoded, _ := json.Marshal(dto)
		var raw json.RawMessage
		if b.Decode("RateObservation", encoded, &raw) != nil {
			t.Fatal("rate roundtrip failed")
		}
	}
	for _, invalid := range []string{strings.Replace(quote, `,"feeCoverage":"included"`, "", 1), strings.Replace(quote, `"spreadCoverage":"included"`, `"spreadCoverage":"unknown"`, 1), strings.Replace(quote, `"amount":"1.000000000001"`, `"amount":1`, 1)} {
		var raw json.RawMessage
		if b.Decode("RateObservation", []byte(invalid), &raw) == nil {
			t.Fatal("incomplete/numeric quote accepted")
		}
	}
	for _, reason := range []string{"no_bracket", "numeric_error_unbounded"} {
		var raw json.RawMessage
		if err := b.Decode("ReturnValue", []byte(`{"state":"unavailable","reason":"`+reason+`"}`), &raw); err != nil {
			t.Fatal(err)
		}
	}
}

func TestValuationRequiresSourceLegs(t *testing.T) {
	b, err := contract.NewBoundary()
	if err != nil {
		t.Fatal(err)
	}
	const source = `{"id":"10000000-0000-4000-8000-000000000001","revision":1,"rate":{"base":"BTC","quote":"RUB","value":"5000000.000000000001"},"requestedDate":"2026-09-07","observedAt":"2026-09-07T00:00:00Z","fetchedAt":"2026-09-07T01:00:00Z","source":"synthetic-cross","method":"reference","granularity":"daily","quality":{"coverage":{"state":"complete","reasons":[]},"freshness":"fresh"}}`
	const legs = `[{"providerAssetId":"bitcoin","source":"CoinGecko","attributionUrl":"https://www.coingecko.com/","transport":"demo-api","requestedDate":"2026-09-07","effectiveAt":"2026-09-07T00:00:00Z","fetchedAt":"2026-09-07T01:00:00Z","granularity":"daily","revision":2,"rate":{"base":"BTC","quote":"USD","value":"50000.00000000000001"}},{"providerAssetId":"USD","source":"CBR","attributionUrl":"https://www.cbr.ru/","transport":"daily-xml","requestedDate":"2026-09-07","effectiveAt":"2026-09-05T00:00:00Z","fetchedAt":"2026-09-07T01:00:00Z","granularity":"daily","revision":3,"rate":{"base":"USD","quote":"RUB","value":"100"}}]`
	var fields map[string]json.RawMessage
	if err := json.Unmarshal([]byte(source), &fields); err != nil {
		t.Fatal(err)
	}
	fields["legs"] = json.RawMessage(legs)
	data, _ := json.Marshal(fields)
	var dto generated.ValuationObservation
	if err := b.Decode("ValuationObservation", data, &dto); err != nil {
		t.Fatal(err)
	}
	encoded, err := json.Marshal(dto)
	if err != nil {
		t.Fatal(err)
	}
	var roundtrip generated.ValuationObservation
	if err := b.Decode("ValuationObservation", encoded, &roundtrip); err != nil {
		t.Fatal(err)
	}
	if len(roundtrip.Legs) != 2 || roundtrip.Legs[0].ProviderAssetId != "bitcoin" || roundtrip.Legs[0].Rate.Value != "50000.00000000000001" || roundtrip.Legs[1].EffectiveAt != "2026-09-05T00:00:00Z" || roundtrip.Legs[1].Revision != 3 {
		t.Fatal("source leg provenance or precision lost")
	}
	for _, method := range []string{"reference", "executed"} {
		fields["method"], _ = json.Marshal(method)
		for _, invalidLegs := range []string{"", "[]", "null", `[{}]`} {
			if invalidLegs == "" {
				delete(fields, "legs")
			} else {
				fields["legs"] = json.RawMessage(invalidLegs)
			}
			invalid, _ := json.Marshal(fields)
			for _, schema := range []string{"ValuationObservation", "RateObservation"} {
				var raw json.RawMessage
				if b.Decode(schema, invalid, &raw) == nil {
					t.Fatal("valuation with missing/incomplete source legs accepted", schema, method, invalidLegs)
				}
			}
		}
	}
}
