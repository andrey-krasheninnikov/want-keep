//go:build integration

package receipts_test

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"sync"
	"syscall"
	"testing"
	"time"

	"github.com/google/uuid"
	aiapp "github.com/pchkauu/want-keep/backend/internal/ai/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	attachments "github.com/pchkauu/want-keep/backend/internal/attachments/application"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	"github.com/pchkauu/want-keep/backend/internal/attachments/processor"
	"github.com/pchkauu/want-keep/backend/internal/gateways/openai"
	jobapp "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	receiptapp "github.com/pchkauu/want-keep/backend/internal/receipts/application"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

// The journal survives test restarts. An unresolved entry stops further paid calls.
type liveBudget struct {
	aiapp.Gateway
	path   string
	counts sync.Map
}
type liveEntry struct{ ID, Reserved, Actual, State string }

func (g *liveBudget) Count(ctx context.Context, r ai.Request) (int64, error) {
	n, err := g.Gateway.Count(ctx, r)
	if err == nil {
		g.counts.Store(r.ID, n)
	}
	return n, err
}
func (g *liveBudget) update(change func(*[]liveEntry) error) error {
	lock, err := os.OpenFile(g.path+".lock", os.O_CREATE|os.O_RDWR, 0600)
	if err != nil {
		return err
	}
	defer lock.Close()
	if info, e := lock.Stat(); e != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 {
		return errors.New("invalid private live budget lock")
	}
	if err = syscall.Flock(int(lock.Fd()), syscall.LOCK_EX); err != nil {
		return err
	}
	defer syscall.Flock(int(lock.Fd()), syscall.LOCK_UN)
	info, err := os.Lstat(g.path)
	if err != nil || !info.Mode().IsRegular() || info.Mode().Perm() != 0600 || info.Size() < 1 || info.Size() > 1<<20 {
		return errors.New("invalid private live budget journal")
	}
	data, err := os.ReadFile(g.path)
	if err != nil {
		return err
	}
	var entries []liveEntry
	if json.Unmarshal(data, &entries) != nil || entries == nil {
		return errors.New("live budget journal requires reconciliation")
	}
	if err = change(&entries); err != nil {
		return err
	}
	data, err = json.Marshal(entries)
	if err != nil {
		return err
	}
	temporary, err := os.CreateTemp(filepath.Dir(g.path), ".receipt-budget-*")
	if err != nil {
		return err
	}
	defer temporary.Close()
	defer os.Remove(temporary.Name())
	if _, err = temporary.Write(data); err != nil {
		return err
	}
	if err = temporary.Sync(); err != nil {
		return err
	}
	if err = temporary.Close(); err != nil {
		return err
	}
	if err = os.Rename(temporary.Name(), g.path); err != nil {
		return err
	}
	directory, err := os.Open(filepath.Dir(g.path))
	if err != nil {
		return err
	}
	defer directory.Close()
	return directory.Sync()
}

func (g *liveBudget) Generate(ctx context.Context, r ai.Request) (ai.Result, error) {
	value, exists := g.counts.LoadAndDelete(r.ID)
	if !exists {
		return ai.Result{}, errors.New("missing live token count")
	}
	reserve, err := ai.TerraPricing().Reservation(value.(int64), r.MaximumOutputTokens)
	if err != nil {
		return ai.Result{}, err
	}
	err = g.update(func(entries *[]liveEntry) error {
		total := ai.MustCost("0")
		for _, entry := range *entries {
			if entry.State != "settled" {
				return errors.New("live outcome requires reconciliation")
			}
			cost, e := ai.NewCost(entry.Actual)
			if e != nil {
				return e
			}
			total, e = total.Add(cost)
			if e != nil {
				return e
			}
		}
		total, e := total.Add(reserve)
		if e != nil {
			return e
		}
		comparison, e := total.Compare(ai.MustCost("2"))
		if e != nil {
			return e
		}
		if comparison > 0 {
			return errors.New("live USD 2 cap reached")
		}
		*entries = append(*entries, liveEntry{ID: r.ID, Reserved: reserve.String(), State: "unknown"})
		return nil
	})
	if err != nil {
		return ai.Result{}, err
	}
	result, callErr := g.Gateway.Generate(ctx, r)
	if callErr != nil {
		return result, callErr
	}
	actual, _, err := ai.TerraPricing().Actual(result.Usage)
	if err != nil {
		return ai.Result{}, err
	}
	if comparison, e := actual.Compare(reserve); e != nil || comparison > 0 {
		return ai.Result{}, errors.New("live cost exceeds reserved amount")
	}
	err = g.update(func(entries *[]liveEntry) error {
		for i := range *entries {
			if (*entries)[i].ID == r.ID {
				(*entries)[i].Actual = actual.String()
				(*entries)[i].State = "settled"
				return nil
			}
		}
		return errors.New("missing live reservation")
	})
	return result, err
}

func TestLiveReceipts(t *testing.T) {
	if os.Getenv("WANT_KEEP_RECEIPTS_LIVE") != "1" {
		t.Skip("Live provider evaluation requires explicit key, journal and isolated processor.")
	}
	key, journal, socket := os.Getenv("WANT_KEEP_OPENAI_API_KEY_FILE"), os.Getenv("WANT_KEEP_RECEIPTS_LIVE_JOURNAL"), os.Getenv("WANT_KEEP_TEST_PROCESSOR_SOCKET")
	if !filepath.IsAbs(key) || !filepath.IsAbs(journal) || !filepath.IsAbs(socket) {
		t.Fatal("Live key, journal and processor paths must be explicit.")
	}
	client, err := openai.New(openai.Config{APIKeyFile: key, Environment: "test", HTTPClient: nil})
	if err != nil {
		t.Fatal("Live gateway configuration failed.")
	}
	pages, err := client.ReceiptPages()
	if err != nil {
		t.Fatal(err)
	}
	gateway := &liveBudget{Gateway: pages, path: journal}
	process, err := processor.NewClient(socket)
	if err != nil || !process.Ready(ctx) {
		t.Fatal("Isolated document processor unavailable.")
	}
	for _, tc := range []struct {
		name, asset, total string
		state              receipt.State
		pages              int
	}{
		{"ru-paid.png", "RUB", "900", receipt.Recorded, 1},
		{"en-fractional.png", "USDC", "0.000001", receipt.Recorded, 1},
		{"two-pages.pdf", "RUB", "900", receipt.Recorded, 2},
		{"irrelevant.png", "RUB", "", receipt.Skipped, 1},
		{"injection.png", "RUB", "100", receipt.Recorded, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newFixture(t)
			f.documents = attachments.NewService(f.store, f.blobs, process)
			if tc.asset != "RUB" {
				f.account = f.createAccount(tc.asset)
			}
			data, err := os.ReadFile(filepath.Join("testdata", tc.name))
			if err != nil {
				t.Fatal(err)
			}
			media := "image/png"
			if filepath.Ext(tc.name) == ".pdf" {
				media = "application/pdf"
			}
			id := uuid.NewString()
			_, err = f.documents.Upload(ctx, f.a, attachment.Upload{ID: id, AccountID: f.account, Name: tc.name, MediaType: media, Size: int64(len(data)), Hash: attachment.Hash(data)}, data)
			if err != nil {
				t.Fatal(err)
			}
			if err = f.documents.ProcessNext(ctx); err != nil {
				t.Fatal(err)
			}
			receiptID := f.start(id, f.ta)
			handler := aiapp.NewPageHandler(f.store, gateway, receiptapp.Pages{Repository: f.store, Documents: f.documents}, time.Now, uuid.NewString)
			for range tc.pages {
				worker := jobapp.Worker{Repository: f.store, Handler: handler, Config: jobapp.DefaultWorkerConfig(jobs.ReceiptExtract)}
				if err = worker.Step(ctx); err != nil {
					t.Fatal("Receipt extraction failed.")
				}
			}
			for range tc.pages {
				worker := jobapp.Worker{Repository: f.store, Handler: f.service, Config: jobapp.DefaultWorkerConfig(jobs.ReceiptValidation)}
				if err = worker.Step(ctx); err != nil {
					t.Fatal(err)
				}
			}
			r := f.read(receiptID)
			if r.State != tc.state {
				for _, page := range r.Pages {
					value, _ := json.Marshal(page.Extraction)
					t.Logf("Synthetic extracted page: %s", value)
				}
				t.Fatalf("receipt state %s, expected %s (%s)", r.State, tc.state, r.Reason)
			}
			if tc.total != "" {
				expected, _ := money.NewMoney(tc.total, money.Asset(tc.asset))
				if r.Facts == nil {
					t.Fatal("Live extracted total is unknown.")
				}
				if comparison, e := r.Facts.Total.Compare(expected); e != nil || comparison != 0 {
					t.Fatal("Live extracted total did not match the visible payment.")
				}
			}
		})
	}
}

func TestLiveJournalRejectsEmptyAndKeepsConcurrentHistory(t *testing.T) {
	path := filepath.Join(t.TempDir(), "budget.json")
	if err := os.WriteFile(path, nil, 0600); err != nil {
		t.Fatal(err)
	}
	g := liveBudget{path: path}
	if err := g.update(func(*[]liveEntry) error { return nil }); err == nil {
		t.Fatal("empty journal reset spending history")
	}
	if err := os.WriteFile(path, []byte("[]"), 0600); err != nil {
		t.Fatal(err)
	}
	var group sync.WaitGroup
	for range 4 {
		group.Add(1)
		go func() {
			defer group.Done()
			for range 10 {
				if err := g.update(func(v *[]liveEntry) error {
					*v = append(*v, liveEntry{ID: uuid.NewString(), Actual: "0.01", State: "settled"})
					return nil
				}); err != nil {
					t.Error(err)
					return
				}
			}
		}()
	}
	group.Wait()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var entries []liveEntry
	if err = json.Unmarshal(data, &entries); err != nil || len(entries) != 40 {
		t.Fatal("concurrent journal history lost", err)
	}
	sentinel := errors.New("interrupted update")
	if err = g.update(func(*[]liveEntry) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Fatal("failed update accepted")
	}
	after, err := os.ReadFile(path)
	if err != nil || string(after) != string(data) {
		t.Fatal("failed update damaged journal")
	}
}
