package credentials

import (
	"bytes"
	"encoding/json"
	"io"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

// DecodeBrowserSession separates borrowed credential headers from Playwright storage state.
func DecodeBrowserSession(provider string, plain []byte) (json.RawMessage, map[string]string, error) {
	if len(plain) < 2 || len(plain) > 1024*1024 {
		return nil, nil, connections.ErrSecretAccess
	}
	var state struct {
		Cookies []json.RawMessage `json:"cookies"`
		Origins []json.RawMessage `json:"origins"`
		Headers map[string]string `json:"headers,omitempty"`
	}
	d := json.NewDecoder(bytes.NewReader(plain))
	d.DisallowUnknownFields()
	if d.Decode(&state) != nil || d.Decode(new(any)) != io.EOF || state.Cookies == nil || state.Origins == nil || len(state.Cookies) > 256 || len(state.Origins) > 8 {
		return nil, nil, connections.ErrSecretAccess
	}
	if provider == "alfa" {
		if len(state.Headers) != 5 {
			return nil, nil, connections.ErrSecretAccess
		}
		for _, name := range []string{"x-xsrf-token", "x-security-request", "device-app-id", "ao-origin-application-id", "zone-offset"} {
			value := state.Headers[name]
			if len(value) < 1 || len(value) > 2048 {
				return nil, nil, connections.ErrSecretAccess
			}
			for _, r := range value {
				if r < 32 || r > 126 {
					return nil, nil, connections.ErrSecretAccess
				}
			}
		}
	} else if state.Headers != nil {
		return nil, nil, connections.ErrSecretAccess
	}
	headers := state.Headers
	state.Headers = nil
	clean, err := json.Marshal(state)
	if err != nil {
		return nil, nil, connections.ErrSecretAccess
	}
	return clean, headers, nil
}
func (v *Vault) ValidateBrowserSession(provider string, plain []byte) error {
	state, headers, err := DecodeBrowserSession(provider, plain)
	clear(state)
	clear(headers)
	return err
}
