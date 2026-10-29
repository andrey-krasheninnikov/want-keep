package receipts_test

import (
	"encoding/json"
	receiptcontract "github.com/pchkauu/want-keep/backend/internal/receipts/contract"
	"strings"
	"testing"
)

const valid = `{"document":"receipt","kind":"expense","paid":true,"date":"2026-08-31","time":"14:30:00","asset":"RUB","total":"900","discount":"100","merchant":"Synthetic shop","reference":null,"items":[{"name":"Food","quantity":"1","gross":"1000","discount":"100","category":null}],"reason":"Visible receipt."}`

func TestStrictExtractionContract(t *testing.T) {
	if _, err := receiptcontract.Decode([]byte(valid)); err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct {
		name string
		edit func(map[string]any)
	}{
		{"actor spoof", func(v map[string]any) { v["actor"] = "partner" }},
		{"tool injection", func(v map[string]any) { v["execute"] = "send payment" }},
		{"null reason", func(v map[string]any) { v["reason"] = nil }},
		{"negative zero", func(v map[string]any) { v["total"] = "-0" }},
		{"numeric amount", func(v map[string]any) { v["total"] = 900 }},
		{"missing nullable", func(v map[string]any) { delete(v, "reference") }},
		{"exponent", func(v map[string]any) { v["total"] = "9e2" }},
		{"length", func(v map[string]any) { v["total"] = strings.Repeat("1", 257) }},
		{"bad date", func(v map[string]any) { v["date"] = "2026-02-30" }},
		{"bad time", func(v map[string]any) { v["time"] = "25:00:00" }},
		{"missing item nullable", func(v map[string]any) { delete(v["items"].([]any)[0].(map[string]any), "discount") }},
		{"invented asset", func(v map[string]any) { v["asset"] = "USDC.E" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			var v map[string]any
			_ = json.Unmarshal([]byte(valid), &v)
			test.edit(v)
			data, _ := json.Marshal(v)
			if _, err := receiptcontract.Decode(data); err == nil {
				t.Fatal("untrusted extraction accepted")
			}
		})
	}
	if _, err := receiptcontract.Decode([]byte(valid + ` {}`)); err == nil {
		t.Fatal("trailing object accepted")
	}
}

func TestDuplicateExtractionKeysAreRejected(t *testing.T) {
	for _, data := range []string{
		strings.Replace(valid, `"total":"900"`, `"total":"1","total":"900"`, 1),
		strings.Replace(valid, `"gross":"1000"`, `"gross":"1","gross":"1000"`, 1),
	} {
		if _, err := receiptcontract.Decode([]byte(data)); err == nil {
			t.Fatal("duplicate financial key accepted")
		}
	}
}
