package raiffeisen

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"sort"
	"time"

	"github.com/pchkauu/want-keep/backend/internal/connections/admission"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ingestion "github.com/pchkauu/want-keep/backend/internal/integrations/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Reports interface {
	ClaimRBOReport(context.Context, household.Principal, jobs.Job, string) (connections.RBOReport, bool, error)
	SaveRBOReport(context.Context, household.Principal, jobs.Job, connections.RBOReport, string, string) error
}
type Gateway struct {
	Client     *Client
	Tokens     Tokens
	Gate       *admission.Service
	Reports    Reports
	Principal  household.Principal
	Job        jobs.Job
	Connection connections.ConnectionRecord
	Now        func() time.Time
	BeforeIO   func(context.Context) error
	Complete   bool
}

func (g *Gateway) Binding() connections.Binding { return g.Job.Binding }
func (g *Gateway) Manifest(context.Context) (ingestion.Manifest, error) {
	return ingestion.Manifest{Provider: "raiffeisen", Version: "10", Actions: []ingestion.ReadAction{ingestion.ReadAccounts, ingestion.ReadBalances, ingestion.ReadTransactions, ingestion.ReadHistory}, Products: []string{"current"}, Logs: []ingestion.CapabilityLog{{Product: "current", Namespace: camtLog, RecordKinds: []ingestion.RecordKind{ingestion.AccountRecordKind, ingestion.BalanceRecordKind, ingestion.TransactionRecordKind}}}, Paginated: true}, nil
}
func (g *Gateway) io(ctx context.Context) error {
	if err := g.Gate.BeforeRead(ctx, g.Principal, g.Job); err != nil {
		return err
	}
	return g.BeforeIO(ctx)
}

type cursor struct{ Day, AccountID string }

func (g *Gateway) Read(ctx context.Context, t ingestion.JobToken) (ingestion.Result, error) {
	client := g.Client.withPermit(g.io)
	accounts, raw, err := client.Accounts(ctx, g.Tokens)
	if err != nil {
		return g.failure(t, err, raw, "accounts"), nil
	}
	sort.Slice(accounts, func(i, j int) bool { return accounts[i].ID < accounts[j].ID })
	zone, _ := time.LoadLocation("Europe/Moscow")
	now := g.Now()
	today := now.In(zone).Format(time.DateOnly)
	day := g.Connection.HistoryFrom.String()
	if day == "" {
		return g.failure(t, ErrResponse, raw, "history"), nil
	}
	if !g.Connection.LastSuccess.IsZero() {
		overlap := g.Connection.LastSuccess.In(zone).AddDate(0, 0, -7).Format(time.DateOnly)
		if overlap > day {
			day = overlap
		}
	}
	end := today
	if !t.ReplayFrom.IsZero() {
		day = t.ReplayFrom.In(zone).Format(time.DateOnly)
		end = t.ReplayTo.Add(-time.Nanosecond).In(zone).Format(time.DateOnly)
	}
	index := 0
	if t.Cursor != "" {
		var c cursor
		if json.Unmarshal([]byte(t.Cursor), &c) != nil {
			return ingestion.Result{}, ErrResponse
		}
		day = c.Day
		found := false
		for i, a := range accounts {
			if a.ID == c.AccountID {
				index = i
				found = true
				break
			}
		}
		if !found {
			return g.failure(t, ErrResponse, raw, "account_identity"), nil
		}
	}
	date, err := time.ParseInLocation(time.DateOnly, day, zone)
	if err != nil || day > end {
		return ingestion.Result{}, ErrResponse
	}
	a := accounts[index]
	keyBytes, _ := json.Marshal([]string{g.Job.ID, a.ID, day})
	key := string(keyBytes)
	var report connections.RBOReport
	var claimed bool
	err = g.Gate.WithReadPermit(ctx, g.Principal, g.Job, func(ctx context.Context) error {
		var err error
		report, claimed, err = g.Reports.ClaimRBOReport(ctx, g.Principal, g.Job, key)
		return err
	})
	if err != nil {
		return ingestion.Result{}, err
	}
	if claimed {
		id, response, createErr := client.CreateReport(ctx, g.Tokens, a, date, date, now)
		phase := "requested"
		if errors.Is(createErr, ErrNoStatements) {
			phase = "no_statements"
		} else if createErr != nil {
			phase = "unknown"
		}
		err = g.Gate.WithReadPermit(ctx, g.Principal, g.Job, func(ctx context.Context) error {
			return g.Reports.SaveRBOReport(ctx, g.Principal, g.Job, report, phase, id)
		})
		if err != nil {
			return ingestion.Result{}, err
		}
		report.Phase, report.ReportID = phase, id
		if createErr != nil && !errors.Is(createErr, ErrNoStatements) {
			return g.failure(t, createErr, response, "report_create"), nil
		}
	}
	if report.Phase == "claimed" || report.Phase == "unknown" {
		return g.failure(t, ErrUnknown, raw, "report_unknown"), nil
	}
	evidence := []ingestion.Evidence{rawEvidence(raw, "application/json", "accounts")}
	records := []ingestion.Record{}
	for _, account := range accounts {
		ref := ingestion.AccountReference{ExternalAccountID: account.ID, Product: "current", AssetCode: currency(account.Currency)}
		descriptor := ingestion.AccountRecord{Reference: ref, Name: account.Name, LogNamespace: camtLog, EvidenceID: evidence[0].ID, OpeningDate: g.Connection.HistoryFrom}
		canonical, _ := json.Marshal(account)
		records = append(records, ingestion.Record{Kind: ingestion.AccountRecordKind, Account: &descriptor, CanonicalPayload: canonical})
	}
	gaps := []string{}
	if report.Phase == "no_statements" {
		gaps = append(gaps, "intraday_unavailable")
	} else {
		var file, status []byte
		for attempt := 0; attempt < 10; attempt++ {
			file, status, err = client.Report(ctx, g.Tokens, report.ReportID)
			if !errors.Is(err, ErrReportPending) {
				break
			}
			timer := time.NewTimer(time.Second)
			select {
			case <-ctx.Done():
				timer.Stop()
				return ingestion.Result{}, ctx.Err()
			case <-timer.C:
			}
		}
		if errors.Is(err, ErrNoStatements) {
			gaps = append(gaps, "intraday_unavailable")
			evidence = append(evidence, rawEvidence(status, "application/json", "report-status"))
		} else if err != nil {
			return g.failure(t, err, status, "report_read"), nil
		} else {
			statement, err := Normalize(file, a, report.ReportID)
			if err != nil {
				return g.failure(t, err, file, "report_file"), nil
			}
			if statement.From.In(zone).Format(time.DateOnly) != day || statement.To.In(zone).Format(time.DateOnly) != day {
				return g.failure(t, ErrResponse, file, "report_range"), nil
			}
			evidence = append(evidence, statement.Evidence)
			gaps = append(gaps, statement.Gaps...)
			if statement.Closing != nil {
				canonical, _ := json.Marshal(struct{ Amount, At string }{statement.Closing.Owned.Value, statement.Closing.SourceAsOf.String()})
				records = append(records, ingestion.Record{Kind: ingestion.BalanceRecordKind, Balance: statement.Closing, CanonicalPayload: canonical})
			} else {
				gaps = appendUnique(gaps, "balance_unavailable")
			}
			for _, fact := range statement.Facts {
				records = append(records, fact.Record)
			}
		}
	}
	if len(records) > ingestion.MaxRecordsPerPage {
		return g.failure(t, ErrResponse, raw, "page_size"), nil
	}
	nextDay, nextIndex := day, index+1
	if nextIndex == len(accounts) {
		nextIndex = 0
		nextDay = date.AddDate(0, 0, 1).Format(time.DateOnly)
	}
	complete := nextDay > end
	next := ""
	if !complete {
		data, _ := json.Marshal(cursor{Day: nextDay, AccountID: accounts[nextIndex].ID})
		next = string(data)
	}
	state := reporting.Complete
	if len(gaps) > 0 {
		state = reporting.Partial
	}
	coverage, _ := reporting.NewCoverage(state, gaps)
	g.Complete = complete
	return ingestion.Result{Page: &ingestion.Page{Token: t, NextCursor: next, Complete: complete, Coverage: coverage, Evidence: evidence, Records: records}}, nil
}
func rawEvidence(data []byte, media, kind string) ingestion.Evidence {
	if len(data) == 0 {
		data = []byte(`{"status":"unavailable"}`)
	}
	if media == "application/json" && !json.Valid(data) {
		data, _ = json.Marshal(struct{ Encoding, Body string }{"base64", base64.StdEncoding.EncodeToString(data)})
	}
	hash := sha256.Sum256(data)
	digest := hex.EncodeToString(hash[:])
	return ingestion.Evidence{ID: kind + "-" + digest, MediaType: media, Digest: digest, Locator: "raiffeisen:" + kind, Data: data}
}
func (g *Gateway) failure(t ingestion.JobToken, err error, raw []byte, kind string) ingestion.Result {
	k := ingestion.ContractViolation
	retry := false
	seconds := 0
	switch {
	case errors.Is(err, ErrReauth):
		k = ingestion.ReauthenticationRequired
	case errors.Is(err, ErrUnavailable) || errors.Is(err, ErrReportPending):
		k = ingestion.TemporaryFailure
		retry = true
		seconds = 30
	case errors.Is(err, ErrUnknown):
		k = ingestion.PermanentFailure
	}
	return ingestion.Result{Failure: &ingestion.ProviderFailure{Token: t, Kind: k, Retryable: retry, RetryAfterSeconds: seconds, SafeMessage: "Bank data is incomplete; inspect connection status.", Evidence: []ingestion.Evidence{rawEvidence(raw, "application/json", kind)}}}
}
