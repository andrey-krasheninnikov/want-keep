package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/google/uuid"
	accounts "github.com/pchkauu/want-keep/backend/internal/accounts/application"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/application"
	allocation "github.com/pchkauu/want-keep/backend/internal/allocation/application"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connectionaccess "github.com/pchkauu/want-keep/backend/internal/connections/access"
	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	collector "github.com/pchkauu/want-keep/backend/internal/connections/collector"
	"github.com/pchkauu/want-keep/backend/internal/connections/credentials"
	expenses "github.com/pchkauu/want-keep/backend/internal/expenses/application"
	openaigateway "github.com/pchkauu/want-keep/backend/internal/gateways/openai"
	integrations "github.com/pchkauu/want-keep/backend/internal/integrations/application"
	"github.com/pchkauu/want-keep/backend/internal/integrations/raiffeisen"
	ratesource "github.com/pchkauu/want-keep/backend/internal/integrations/rates"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/application"
	domain "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	"github.com/pchkauu/want-keep/backend/internal/privacy/cryptobox"
	reconciliation "github.com/pchkauu/want-keep/backend/internal/reconciliation/application"
	"github.com/pchkauu/want-keep/backend/internal/storage"
	valuationapp "github.com/pchkauu/want-keep/backend/internal/valuation/application"
)

func main() {
	if err := run(); err != nil && !errors.Is(err, context.Canceled) {
		fmt.Fprintln(os.Stderr, "Background process failed")
		os.Exit(1)
	}
}
func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	bindings, err := admission.LoadBindings(os.Getenv("WANT_KEEP_JOB_BINDINGS_FILE"), os.Getenv("WANT_KEEP_ENV"))
	if err != nil {
		return err
	}
	db, err := storage.Open(ctx, storage.Config{DSN: os.Getenv("WANT_KEEP_DATABASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"), MaxConnections: 8})
	if err != nil {
		return err
	}
	defer db.Close()
	report := func(d jobs.Diagnostic) {
		fmt.Fprintf(os.Stderr, "queue=%s job=%s source=%s transaction=%s stage=%s code=%s duration_ms=%d\n", d.Kind, d.JobID, d.ConnectionID, d.TransactionID, d.Stage, d.Code, d.Duration.Milliseconds())
	}
	admissionService := admission.NewService(db, db)
	scheduler := jobs.Scheduler{Repository: db, Admission: admissionService, Bindings: bindings, Report: report}
	now := func() calendar.Instant {
		at, err := calendar.ParseInstant(time.Now().UTC().Format(time.RFC3339Nano))
		if err != nil {
			panic(err)
		}
		return at
	}
	reconciliationService := reconciliation.NewService(db, db, ledger.NewWriter(db, db), admissionService, now, uuid.NewString)
	var syncHandler jobs.Handler
	stagedReconciler := &collector.StagedReconciler{
		Reconcile: db.ReconcileStagedCollectorEvidence,
		Report: func(error) {
			report(jobs.Diagnostic{Kind: domain.Sync, Stage: "evidence_reconciliation", Code: "collector_evidence_reconciliation_failed"})
		},
	}
	if socket := os.Getenv("WANT_KEEP_COLLECTOR_SOCKET"); socket != "" || os.Getenv("WANT_KEEP_CONNECTION_KEYRING") != "" {
		connectionKeys, keyErr := cryptobox.Load(os.Getenv("WANT_KEEP_CONNECTION_KEYRING"), "connections")
		evidence, evidenceErr := collector.NewEvidenceStore(db, connectionKeys)
		reimbursementService := ledger.NewReimbursementService(db, now, uuid.NewString)
		accountService := accounts.NewServiceWithOwnershipReconciliation(db, db, reconciliationService, reimbursementService, now, uuid.NewString)
		accountImporter, accountErr := integrations.NewAccountImporter(accountService, db)
		writer := ledger.NewWriterWithRefundsAndReimbursements(db, db, reconciliationService, expenses.NewProjector(db), reimbursementService)
		sources := ledger.NewSources(db, writer, allocation.NewService(db, now, uuid.NewString))
		sourceWriter, sourceErr := integrations.NewSourceWriter(sources, db)
		ingestionService, ingestionErr := integrations.NewService(admissionService, evidence, accountImporter, sourceWriter, now, uuid.NewString)
		if keyErr != nil || evidenceErr != nil || accountErr != nil || sourceErr != nil || ingestionErr != nil {
			report(jobs.Diagnostic{Kind: domain.Sync, Stage: "startup", Code: "collector_configuration_invalid"})
		} else {
			connectionAccess := connectionaccess.NewService(nil, db, admissionService)
			vault := credentials.New(connectionAccess, db, connectionKeys)
			var authorizer *raiffeisen.Authorizer
			if os.Getenv("WANT_KEEP_RAIF_CLIENT_ID_FILE") != "" {
				authorizer, _ = raiffeisen.LoadOAuth(os.Getenv("WANT_KEEP_RAIF_CLIENT_ID_FILE"), os.Getenv("WANT_KEEP_RAIF_CLIENT_SECRET_FILE"), os.Getenv("WANT_KEEP_RAIF_ISSUER"), os.Getenv("WANT_KEEP_RAIF_JWKS_FILE"), "https://want-keep.tech/api/v1/connections/raiffeisen/callback")
			}
			var collectorHandler jobs.Handler
			if socket != "" {
				collectorHandler = collector.Handler{Socket: socket, Vault: vault, Service: ingestionService}
			}
			syncHandler = raiffeisen.Dispatcher{Native: raiffeisen.Handler{Vault: vault, Service: ingestionService, Gate: admissionService, Repository: db, Authorizer: authorizer, Now: time.Now}, Collector: collectorHandler}
		}
	}
	rateService := valuationapp.Service{Repository: db, Sources: ratesource.New(&http.Client{Timeout: 3 * time.Second}, os.Getenv("WANT_KEEP_COINGECKO_KEY_FILE"), db), Now: time.Now}
	valuationService := valuationapp.SnapshotPreparer{Rates: rateService, Repository: db, Now: time.Now}
	var aiHandler jobs.Handler = ai.WaitingHandler{}
	var aiBudgetQueue *ai.BudgetQueue
	var aiGatewayQueue *ai.GatewayQueue
	if keyFile := os.Getenv("WANT_KEEP_OPENAI_API_KEY_FILE"); keyFile != "" {
		gateway, gatewayErr := openaigateway.New(openaigateway.Config{
			APIKeyFile: keyFile, ProjectID: os.Getenv("WANT_KEEP_OPENAI_PROJECT_ID"),
			BaseURL: os.Getenv("WANT_KEEP_OPENAI_BASE_URL"), Environment: os.Getenv("WANT_KEEP_ENV"),
			HTTPClient: &http.Client{Timeout: 2 * time.Minute},
		})
		if gatewayErr != nil {
			report(jobs.Diagnostic{Kind: domain.AI, Stage: "startup", Code: "gateway_configuration_invalid"})
		} else {
			aiHandler = ai.NewHandler(db, gateway, time.Now, uuid.NewString)
			aiBudgetQueue = ai.NewBudgetQueue(db, time.Now, time.Minute)
			aiGatewayQueue = ai.NewGatewayQueue(db, time.Minute)
		}
	}
	var group sync.WaitGroup
	group.Add(1)
	go func() { defer group.Done(); _ = scheduler.Run(ctx) }()
	group.Add(1)
	go func() { defer group.Done(); _ = stagedReconciler.Run(ctx) }()
	if aiBudgetQueue != nil {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := aiBudgetQueue.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				report(jobs.Diagnostic{Kind: domain.AI, Stage: "budget_queue", Code: "budget_queue_stopped"})
				stop()
			}
		}()
	}
	if aiGatewayQueue != nil {
		group.Add(1)
		go func() {
			defer group.Done()
			if err := aiGatewayQueue.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				report(jobs.Diagnostic{Kind: domain.AI, Stage: "gateway_queue", Code: "gateway_queue_stopped"})
				stop()
			}
		}()
	}
	for _, kind := range []domain.Kind{domain.Sync, domain.Outbox, domain.AI} {
		var handler jobs.Handler
		if kind == domain.Sync {
			handler = syncHandler
		} else if kind == domain.Outbox {
			handler = jobs.OutboxHandler{Repository: db, Reconciliation: reconciliationService, Valuation: valuationService}
		} else if kind == domain.AI {
			handler = aiHandler
		}
		worker := jobs.Worker{Admission: admissionService, Repository: db, Handler: handler, Config: jobs.DefaultWorkerConfig(kind), Report: report}
		group.Add(1)
		go func() {
			defer group.Done()
			if err := worker.Run(ctx); err != nil && !errors.Is(err, context.Canceled) {
				report(jobs.Diagnostic{Kind: worker.Config.Kind, Stage: "startup", Code: "worker_stopped"})
				stop()
			}
		}()
	}
	group.Wait()
	return ctx.Err()
}
