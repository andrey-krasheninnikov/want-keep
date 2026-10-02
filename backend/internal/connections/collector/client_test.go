package collector

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/google/uuid"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
)

func TestClientUsesUnixSocketAndMarksOnlyProviderRead(t *testing.T) {
	var reads atomic.Int32
	socket := serveUnix(t, func(response http.ResponseWriter, request *http.Request) {
		switch request.URL.Path {
		case "/ready":
			_, _ = io.WriteString(response, "want-keep-browser-collector/1\n")
		case "/v1/capabilities":
			_, _ = io.WriteString(response, `{"provider":"bybit"}`)
		case "/v1/read":
			var body struct {
				Storage json.RawMessage `json:"storageState"`
			}
			if json.NewDecoder(request.Body).Decode(&body) != nil || !bytes.Contains(body.Storage, []byte("private-session")) {
				http.Error(response, "bad request", http.StatusUnprocessableEntity)
				return
			}
			_, _ = io.WriteString(response, `{"outcome":"page","page":{"complete":true}}`)
		}
	})
	client, err := NewClient(socket, testBinding(), 3, []byte(`{"cookies":[],"origins":[],"marker":"private-session"}`), func(context.Context) error {
		reads.Add(1)
		return nil
	})
	if !errors.Is(err, ErrSessionInvalid) {
		// Unknown storage-state fields must fail before any provider IO.
		t.Fatal("unknown session field did not require reauthentication", err)
	}
	client, err = NewClient(socket, testBinding(), 3, []byte(`{"cookies":[{"name":"session","value":"private-session"}],"origins":[]}`), func(context.Context) error {
		reads.Add(1)
		return nil
	})
	if err != nil || !client.Ready(context.Background()) {
		t.Fatal("collector unavailable", err)
	}
	defer client.Close()
	if _, err = client.CapabilityManifest(context.Background()); err != nil || reads.Load() != 0 || client.ExternalStarted() {
		t.Fatal("capability check marked provider IO", err, reads.Load())
	}
	result, err := client.Read(context.Background(), []byte(`{"jobId":"value"}`))
	if err != nil || reads.Load() != 1 || !client.ExternalStarted() || bytes.Contains(result, []byte("private-session")) {
		t.Fatal("provider read boundary failed", err, reads.Load())
	}
	page, complete, failure := client.Outcome()
	if !page || !complete || failure {
		t.Fatal("page outcome not retained")
	}
}

func TestBusyCollectorResponseIsKnownBeforeProviderIO(t *testing.T) {
	for _, responseBody := range []string{`{"code":"collector_busy"}`, `{"code":"other"}`} {
		t.Run(responseBody, func(t *testing.T) {
			socket := serveUnix(t, func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/v1/read" {
					response.WriteHeader(http.StatusConflict)
					_, _ = io.WriteString(response, responseBody)
				}
			})
			var starts atomic.Int32
			client, err := NewClient(socket, testBinding(), 3, []byte(`{"cookies":[],"origins":[]}`), func(context.Context) error {
				starts.Add(1)
				return nil
			})
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			_, err = client.Read(context.Background(), []byte(`{"jobId":"value"}`))
			want := ErrUnavailable
			if responseBody == `{"code":"collector_busy"}` {
				want = ErrBusy
			}
			if !errors.Is(err, want) || starts.Load() != 1 || !client.ExternalStarted() {
				t.Fatal("incorrect pre-provider rejection", err, starts.Load())
			}
		})
	}
}

func TestKnownPreReadRejectionIsDistinctFromUnknownProviderOutcome(t *testing.T) {
	for _, test := range []struct {
		status int
		body   string
		want   error
	}{
		{http.StatusUnprocessableEntity, `{"code":"collector_preflight_rejected"}`, ErrPreflightRejected},
		{http.StatusUnprocessableEntity, `{"code":"collector_session_invalid"}`, ErrSessionInvalid},
		{http.StatusServiceUnavailable, `{"code":"collector_before_io_unavailable"}`, ErrBeforeIOUnavailable},
	} {
		t.Run(test.body, func(t *testing.T) {
			socket := serveUnix(t, func(response http.ResponseWriter, request *http.Request) {
				if request.URL.Path == "/v1/read" {
					response.WriteHeader(test.status)
					_, _ = io.WriteString(response, test.body)
				}
			})
			client, err := NewClient(socket, testBinding(), 3, []byte(`{"cookies":[null],"origins":[]}`), func(context.Context) error { return nil })
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			_, err = client.Read(context.Background(), []byte(`{"jobId":"value"}`))
			if !errors.Is(err, test.want) || !client.ExternalStarted() {
				t.Fatal("preflight rejection lost its no-IO proof", err)
			}
		})
	}
}

func TestEvidenceStoreEncryptsRawBytesAndBindsAAD(t *testing.T) {
	path := filepath.Join(t.TempDir(), "keys")
	if err := cryptobox.Generate(path, "connections"); err != nil {
		t.Fatal(err)
	}
	keys, err := cryptobox.Load(path, "connections")
	if err != nil {
		t.Fatal(err)
	}
	repository := &evidenceRepository{}
	store, err := NewEvidenceStore(repository, keys)
	if err != nil {
		t.Fatal(err)
	}
	at, _ := calendar.ParseInstant("2026-09-14T10:00:00.123456789Z")
	raw := []byte(`{"balance":"5000"}`)
	digest := fmt.Sprintf("%x", sha256.Sum256(raw))
	batch := ingestion.EvidenceBatch{
		HouseholdID: "11111111-1111-4111-8111-111111111111", JobID: "22222222-2222-4222-8222-222222222222",
		PageReference: "evidence:page:1", FetchedAt: at, Disposition: ingestion.EvidenceStaged,
		Items: []ingestion.StoredEvidence{{Reference: "evidence:raw:1", Raw: ingestion.Evidence{ID: "raw-1", MediaType: "application/json", Digest: digest, Locator: "synthetic:page", Data: raw}}},
	}
	if err = store.Save(context.Background(), batch); err != nil {
		t.Fatal(err)
	}
	ciphertext := repository.saved.Items[0].Ciphertext
	if bytes.Contains(ciphertext, raw) {
		t.Fatal("plaintext evidence persisted")
	}
	aad, _ := evidenceAAD(batch.HouseholdID, batch.JobID, batch.PageReference, batch.Items[0].Reference)
	plain, err := keys.Open(ciphertext, aad)
	var decoded ingestion.Evidence
	if err == nil {
		err = json.Unmarshal(plain, &decoded)
	}
	if err != nil || !bytes.Equal(decoded.Data, raw) {
		t.Fatal("evidence did not round trip", err)
	}
	if _, err = keys.Open(ciphertext, append(aad, 'x')); err == nil {
		t.Fatal("foreign AAD accepted")
	}
}

type evidenceRepository struct {
	saved ingestion.EncryptedEvidenceBatch
}

func (r *evidenceRepository) SaveCollectorEvidence(_ context.Context, batch ingestion.EncryptedEvidenceBatch) error {
	r.saved = batch
	return nil
}
func (*evidenceRepository) SetCollectorEvidenceDisposition(context.Context, ingestion.EvidenceDisposition) error {
	return nil
}
func (*evidenceRepository) StagedCollectorEvidence(context.Context, string, int) ([]ingestion.StagedEvidence, error) {
	return nil, nil
}

func testBinding() connections.Binding {
	return connections.Binding{
		Provider: "bybit", Environment: "test", AdapterBuildDigest: "sha256:" + strings.Repeat("0", 64), CollectorImageDigest: "sha256:" + strings.Repeat("1", 64),
		ContractVersion: "10", AllowlistRevision: "allowlist-1", NonSecretConfigRevision: "config-1", OperatorPermissionRevision: "permission-1",
	}
}

func serveUnix(t *testing.T, handler http.HandlerFunc) string {
	t.Helper()
	socket := filepath.Join("/tmp", "wk-"+uuid.NewString()[:8]+".sock")
	_ = os.Remove(socket)
	listener, err := net.Listen("unix", socket)
	if err != nil {
		t.Fatal(err)
	}
	server := &http.Server{Handler: handler}
	go func() { _ = server.Serve(listener) }()
	t.Cleanup(func() {
		_ = server.Close()
		_ = os.Remove(socket)
	})
	return socket
}
