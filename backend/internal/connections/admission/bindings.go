package admission

import (
	"encoding/json"
	"errors"
	"io"
	"os"

	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
)

// LoadBindings reads deployment-owned metadata; it cannot admit a provider.
func LoadBindings(path, environment string) ([]connections.Binding, error) {
	if path == "" {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	decoder := json.NewDecoder(io.LimitReader(file, 1024*1024))
	decoder.DisallowUnknownFields()
	var bindings []connections.Binding
	if err = decoder.Decode(&bindings); err != nil {
		return nil, err
	}
	if decoder.Decode(new(any)) != io.EOF {
		return nil, errors.New("invalid binding file")
	}
	seen := map[string]bool{}
	for _, b := range bindings {
		if b.Environment != environment || b.Validate() != nil || seen[b.Provider] {
			return nil, errors.New("binding environment mismatch or duplicate provider")
		}
		seen[b.Provider] = true
	}
	return bindings, nil
}
