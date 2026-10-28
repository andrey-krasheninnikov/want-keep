package credentials

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestBrowserSessionClosedHeaderPolicy(t *testing.T) {
	valid := `{"cookies":[],"origins":[],"headers":{"x-xsrf-token":"synthetic-xsrf","x-security-request":"synthetic-security","device-app-id":"synthetic-device","ao-origin-application-id":"synthetic-app","zone-offset":"180"}}`
	state, headers, err := DecodeBrowserSession("alfa", []byte(valid))
	if err != nil || !json.Valid(state) || strings.Contains(string(state), "synthetic-xsrf") || len(headers) != 5 {
		t.Fatal("session/header separation failed", err)
	}
	for _, invalid := range []string{strings.Replace(valid, `"zone-offset":"180"`, `"authorization":"secret"`, 1), strings.Replace(valid, `synthetic-xsrf`, `a\r\nb`, 1), `{"cookies":[],"origins":[]}`, strings.Replace(valid, `"cookies":[]`, `"cookies":null`, 1), valid + " {}"} {
		if _, _, err := DecodeBrowserSession("alfa", []byte(invalid)); err == nil {
			t.Fatal("invalid secret accepted")
		}
	}
	if _, _, err := DecodeBrowserSession("bybit", []byte(valid)); err == nil {
		t.Fatal("Alfa credential headers accepted for another provider")
	}
}
