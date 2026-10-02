package domain

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	connections "github.com/pchkauu/want-keep/backend/internal/connections/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

const (
	ContractVersion           = "10"
	MaxRecordsPerPage         = 1000
	MaxEvidencePerPage        = 16
	MaxEvidenceBytes          = 10 * 1024 * 1024
	MaxEncryptedEvidenceBytes = 20 * 1024 * 1024
	MaxTextLength             = 2000
	MaxAdmissionRevision      = int64(9007199254740991)
)

var (
	ErrInvalidContract = errors.New("invalid ingestion contract")
	ErrEvidence        = errors.New("evidence storage failed")
	ErrResultRejected  = errors.New("ingestion result rejected")
	sha256Syntax       = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

type JobToken struct {
	JobID, LeaseToken, ConnectionID string
	Attempt                         int
	ConnectionGeneration            uint64
	Binding                         connections.Binding
	AdmissionRevision               int64
	Cursor                          string
	ReplayFrom, ReplayTo            time.Time
}

func (t JobToken) Validate() error {
	if t.JobID == "" || !validBoundedText(t.LeaseToken, true) || t.ConnectionID == "" || t.Attempt < 1 || t.Attempt > 100 || t.ConnectionGeneration < 1 || t.ConnectionGeneration > uint64(MaxAdmissionRevision) || t.AdmissionRevision < 1 || t.AdmissionRevision > MaxAdmissionRevision || !validBoundedText(t.Cursor, false) || t.Binding.ContractVersion != ContractVersion || t.Binding.Validate() != nil {
		return ErrInvalidContract
	}
	if t.ReplayFrom.IsZero() != t.ReplayTo.IsZero() || !t.ReplayFrom.IsZero() && (!t.ReplayFrom.Before(t.ReplayTo) || t.ReplayTo.Sub(t.ReplayFrom) > 90*24*time.Hour) {
		return ErrInvalidContract
	}
	return nil
}

type ReadAction string

const (
	ReadAccounts     ReadAction = "read_accounts"
	ReadBalances     ReadAction = "read_balances"
	ReadTransactions ReadAction = "read_transactions"
	ReadHistory      ReadAction = "read_history"
	ReadRates        ReadAction = "read_rates"
)

func (a ReadAction) Valid() bool {
	switch a {
	case ReadAccounts, ReadBalances, ReadTransactions, ReadHistory, ReadRates:
		return true
	default:
		return false
	}
}

type RecordKind string

const (
	AccountRecordKind     RecordKind = "account"
	BalanceRecordKind     RecordKind = "balance_snapshot"
	TransactionRecordKind RecordKind = "transaction"
)

type CapabilityLog struct {
	Product     string
	Namespace   string
	RecordKinds []RecordKind
}

type Manifest struct {
	Provider, Version   string
	Actions             []ReadAction
	Products            []string
	Logs                []CapabilityLog
	Paginated           bool
	MaximumLookbackDays *int
}

func (m Manifest) Validate(provider string) error {
	if m.Provider != provider || m.Version != ContractVersion || len(m.Actions) < 1 || len(m.Actions) > 5 || len(m.Products) < 1 || len(m.Products) > 32 || len(m.Logs) < 1 || len(m.Logs) > 64 || m.MaximumLookbackDays != nil && (*m.MaximumLookbackDays < 1 || *m.MaximumLookbackDays > 36500) {
		return ErrInvalidContract
	}
	actions := map[ReadAction]bool{}
	for _, action := range m.Actions {
		if !action.Valid() || actions[action] {
			return ErrInvalidContract
		}
		actions[action] = true
	}
	products := map[string]bool{}
	for _, product := range m.Products {
		if !ValidProduct(product) || products[product] {
			return ErrInvalidContract
		}
		products[product] = true
	}
	logs := map[string]bool{}
	for _, log := range m.Logs {
		if !products[log.Product] || !validText(log.Namespace) || logs[log.Product+"\x00"+log.Namespace] || len(log.RecordKinds) < 1 || len(log.RecordKinds) > 3 {
			return ErrInvalidContract
		}
		logs[log.Product+"\x00"+log.Namespace] = true
		kinds := map[RecordKind]bool{}
		for _, kind := range log.RecordKinds {
			if kind != AccountRecordKind && kind != BalanceRecordKind && kind != TransactionRecordKind || kinds[kind] {
				return ErrInvalidContract
			}
			kinds[kind] = true
		}
	}
	return nil
}

// RequirePage rejects results outside the connector capabilities reviewed for
// the exact admitted build. It is a consistency fence; D-43 remains the source
// of server-side deployment authorization.
func (m Manifest) RequirePage(page Page) error {
	if err := m.Validate(page.Token.Binding.Provider); err != nil {
		return err
	}
	if err := page.Validate(); err != nil {
		return err
	}
	actions := make(map[ReadAction]bool, len(m.Actions))
	for _, action := range m.Actions {
		actions[action] = true
	}
	logs := make(map[string]map[RecordKind]bool, len(m.Logs))
	for _, log := range m.Logs {
		key := log.Product + "\x00" + log.Namespace
		logs[key] = make(map[RecordKind]bool, len(log.RecordKinds))
		for _, kind := range log.RecordKinds {
			logs[key][kind] = true
		}
	}
	if !page.Token.ReplayFrom.IsZero() {
		if !actions[ReadHistory] || m.MaximumLookbackDays != nil && page.Token.ReplayTo.Sub(page.Token.ReplayFrom) > time.Duration(*m.MaximumLookbackDays)*24*time.Hour {
			return ErrInvalidContract
		}
	}
	if page.NextCursor != "" && !m.Paginated {
		return ErrInvalidContract
	}
	for _, record := range page.Records {
		var action ReadAction
		var product, namespace string
		switch record.Kind {
		case AccountRecordKind:
			action, product, namespace = ReadAccounts, record.Account.Reference.Product, record.Account.LogNamespace
		case BalanceRecordKind:
			action, product, namespace = ReadBalances, record.Balance.Reference.Product, record.Balance.LogNamespace
		case TransactionRecordKind:
			action, product, namespace = ReadTransactions, record.Transaction.Product, record.Transaction.LogNamespace
		default:
			return ErrInvalidContract
		}
		if !actions[action] || !logs[product+"\x00"+namespace][record.Kind] {
			return ErrInvalidContract
		}
	}
	return nil
}

func ValidProduct(product string) bool {
	switch product {
	case "current", "debit_card", "credit_card", "savings", "deposit", "wallet", "funding", "spot", "earn", "coinhold", "crypto_card", "futures", "mining":
		return true
	default:
		return false
	}
}

type Evidence struct {
	ID, MediaType, Digest, Locator string
	Data                           []byte
}

func (e Evidence) Validate() error {
	digest := sha256.Sum256(e.Data)
	if !validTextLimit(e.ID, 128) || !validText(e.Locator) || !sha256Syntax.MatchString(e.Digest) || len(e.Data) < 1 || len(e.Data) > MaxEvidenceBytes || e.Digest != hex.EncodeToString(digest[:]) {
		return ErrInvalidContract
	}
	switch e.MediaType {
	case "application/json", "application/xml", "text/csv", "text/html":
		return nil
	default:
		return ErrInvalidContract
	}
}

type StoredEvidence struct {
	Reference string
	Raw       Evidence
}

type EncryptedEvidenceItem struct {
	Reference  string
	Ciphertext []byte
}

type EncryptedEvidenceBatch struct {
	HouseholdID, JobID string
	PageReference      string
	FetchedAt          calendar.Instant
	Items              []EncryptedEvidenceItem
}

func (b EncryptedEvidenceBatch) Validate() error {
	if !validText(b.HouseholdID) || !validText(b.JobID) || !validText(b.PageReference) || b.FetchedAt.String() == "" || len(b.Items) < 1 || len(b.Items) > MaxEvidencePerPage {
		return ErrEvidence
	}
	seenReferences := map[string]bool{}
	total := 0
	for _, item := range b.Items {
		if !validText(item.Reference) || len(item.Ciphertext) < 1 || seenReferences[item.Reference] {
			return ErrEvidence
		}
		total += len(item.Ciphertext)
		if total > MaxEncryptedEvidenceBytes {
			return ErrEvidence
		}
		seenReferences[item.Reference] = true
	}
	return nil
}

type EvidenceDispositionState string

const (
	EvidenceStaged          EvidenceDispositionState = "staged"
	EvidenceApplied         EvidenceDispositionState = "applied"
	EvidenceRejected        EvidenceDispositionState = "rejected_result"
	EvidenceStale           EvidenceDispositionState = "stale_result"
	EvidenceProviderOutcome EvidenceDispositionState = "provider_outcome"
)

type EvidenceBatch struct {
	HouseholdID, JobID string
	PageReference      string
	FetchedAt          calendar.Instant
	Disposition        EvidenceDispositionState
	Items              []StoredEvidence
}

func (b EvidenceBatch) Validate() error {
	if !validText(b.HouseholdID) || !validText(b.JobID) || !validText(b.PageReference) || b.FetchedAt.String() == "" || b.Disposition != EvidenceStaged || len(b.Items) < 1 || len(b.Items) > MaxEvidencePerPage {
		return ErrEvidence
	}
	seenIDs, seenReferences := map[string]bool{}, map[string]bool{}
	total := 0
	for _, item := range b.Items {
		if item.Raw.Validate() != nil || !validText(item.Reference) || seenIDs[item.Raw.ID] || seenReferences[item.Reference] {
			return ErrEvidence
		}
		total += len(item.Raw.Data)
		if total > MaxEvidenceBytes {
			return ErrEvidence
		}
		seenIDs[item.Raw.ID], seenReferences[item.Reference] = true, true
	}
	return nil
}

type EvidenceDisposition struct {
	HouseholdID, JobID string
	PageReference      string
	State              EvidenceDispositionState
}

type StagedEvidence struct {
	HouseholdID, JobID string
	PageReference      string
}

func (e StagedEvidence) Validate() error {
	if !validText(e.HouseholdID) || !validText(e.JobID) || !validText(e.PageReference) {
		return ErrEvidence
	}
	return nil
}

func (d EvidenceDisposition) Validate() error {
	if !validText(d.HouseholdID) || !validText(d.JobID) || !validText(d.PageReference) {
		return ErrEvidence
	}
	switch d.State {
	case EvidenceApplied, EvidenceRejected, EvidenceStale, EvidenceProviderOutcome:
		return nil
	default:
		return ErrEvidence
	}
}

type Amount struct {
	State     reporting.Knowledge
	AssetCode string
	Value     string
	Reason    string
}

func (a Amount) Reporting(asset money.Asset) (reporting.Amount, error) {
	return a.ReportingAs(string(asset), asset)
}

func (a Amount) ReportingAs(externalAssetCode string, asset money.Asset) (reporting.Amount, error) {
	if a.AssetCode != externalAssetCode {
		return reporting.Amount{}, money.ErrAssetMismatch
	}
	switch a.State {
	case reporting.Known:
		if a.Reason != "" {
			return reporting.Amount{}, ErrInvalidContract
		}
		value, err := money.NewMoney(a.Value, asset)
		if err != nil {
			return reporting.Amount{}, err
		}
		return reporting.KnownAmount(value)
	case reporting.Unknown, reporting.Unavailable:
		if a.Value != "" || !validText(a.Reason) {
			return reporting.Amount{}, ErrInvalidContract
		}
		return reporting.MissingAmount(a.State, a.Reason)
	default:
		return reporting.Amount{}, ErrInvalidContract
	}
}

type AccountReference struct {
	ExternalAccountID, Product, Network, AssetCode string
}

func (r AccountReference) Validate() error {
	if !validText(r.ExternalAccountID) || !ValidProduct(r.Product) || !validBoundedText(r.Network, false) || len(r.AssetCode) < 1 || len(r.AssetCode) > 64 {
		return ErrInvalidContract
	}
	return nil
}

func (r AccountReference) Key() string {
	encoded, _ := json.Marshal([]string{r.ExternalAccountID, r.Product, r.Network, r.AssetCode})
	return string(encoded)
}

type CardAlias struct{ ID, Label, LastFour string }

type AccountRecord struct {
	Reference    AccountReference
	Name         string
	LogNamespace string
	OpeningDate  calendar.Date
	Aliases      []CardAlias
	EvidenceID   string
}

type BalanceSnapshot struct {
	Reference                      AccountReference
	LogNamespace                   string
	SourceAsOf                     calendar.Instant
	Owned, Available, Locked, Debt Amount
	CreditLimit                    Amount
	OwnAvailable                   bool
	Coverage                       reporting.Coverage
	Freshness                      reporting.Freshness
	EvidenceID                     string
}

type Posting struct {
	Reference     AccountReference
	Amount        string
	Role, Funding string
	Treatment     string
}

type TransactionRecord struct {
	ExternalAccountID, Product                     string
	LogNamespace, ProviderRecordID, Classification string
	ProviderState, EconomicType                    string
	OccurredAt, PostedAt                           calendar.Instant
	Merchant, Note, FeeKnowledge, PnLBasis         string
	EvidenceID                                     string
	Postings                                       []Posting
	// Native adapters retain record identity separately from the enclosing report evidence.
	SemanticIdentity bool
	SourceAsOf       calendar.Instant
	Aliases          []SourceAlias
}

type SourceAlias struct{ Kind, Value string }

type Record struct {
	Kind             RecordKind
	Account          *AccountRecord
	Balance          *BalanceSnapshot
	Transaction      *TransactionRecord
	CanonicalPayload []byte
}

type Page struct {
	Token      JobToken
	NextCursor string
	Complete   bool
	Coverage   reporting.Coverage
	Evidence   []Evidence
	Records    []Record
}

func (p Page) Validate() error {
	if err := p.Token.Validate(); err != nil || !validBoundedText(p.NextCursor, false) || len(p.Evidence) < 1 || len(p.Evidence) > MaxEvidencePerPage || len(p.Records) > MaxRecordsPerPage {
		return ErrInvalidContract
	}
	if _, err := reporting.NewCoverage(p.Coverage.State(), p.Coverage.Reasons()); err != nil || !ValidGapReasons(p.Coverage.Reasons()) {
		return ErrInvalidContract
	}
	if p.Complete && p.NextCursor != "" || !p.Complete && (p.NextCursor == "" || p.NextCursor == p.Token.Cursor) {
		return ErrInvalidContract
	}
	ids := map[string]bool{}
	total := 0
	for _, evidence := range p.Evidence {
		if evidence.Validate() != nil || ids[evidence.ID] {
			return ErrInvalidContract
		}
		ids[evidence.ID] = true
		total += len(evidence.Data)
	}
	if total > MaxEvidenceBytes {
		return ErrInvalidContract
	}
	totalPostings := 0
	descriptors, balances, references := map[string]bool{}, map[string]bool{}, map[string]bool{}
	for _, record := range p.Records {
		count := 0
		var evidenceID string
		switch record.Kind {
		case AccountRecordKind:
			if record.Account == nil {
				return ErrInvalidContract
			}
		case BalanceRecordKind:
			if record.Balance == nil {
				return ErrInvalidContract
			}
		case TransactionRecordKind:
			if record.Transaction == nil {
				return ErrInvalidContract
			}
		default:
			return ErrInvalidContract
		}
		if record.Account != nil {
			count++
			evidenceID = record.Account.EvidenceID
			if err := record.Account.Reference.Validate(); err != nil || !validText(record.Account.LogNamespace) || descriptors[record.Account.Reference.Key()] {
				return ErrInvalidContract
			}
			descriptors[record.Account.Reference.Key()] = true
		}
		if record.Balance != nil {
			count++
			evidenceID = record.Balance.EvidenceID
			key := record.Balance.Reference.Key()
			if err := record.Balance.Reference.Validate(); err != nil || !validText(record.Balance.LogNamespace) || balances[key] {
				return ErrInvalidContract
			}
			balances[key] = true
			references[key] = true
		}
		if record.Transaction != nil {
			count++
			evidenceID = record.Transaction.EvidenceID
		}
		if count != 1 || !ids[evidenceID] || len(record.CanonicalPayload) == 0 {
			return ErrInvalidContract
		}
		if record.Transaction != nil {
			totalPostings += len(record.Transaction.Postings)
			if totalPostings > MaxRecordsPerPage {
				return ErrInvalidContract
			}
			for _, posting := range record.Transaction.Postings {
				if err := posting.Reference.Validate(); err != nil {
					return ErrInvalidContract
				}
				references[posting.Reference.Key()] = true
			}
		}
	}
	for reference := range references {
		if !descriptors[reference] {
			return ErrInvalidContract
		}
	}
	return nil
}

type FailureKind string

const (
	ReauthenticationRequired FailureKind = "reauthentication_required"
	MFARequired              FailureKind = "mfa_required"
	CaptchaRequired          FailureKind = "captcha_required"
	RateLimited              FailureKind = "rate_limited"
	TemporaryFailure         FailureKind = "temporary_failure"
	PermanentFailure         FailureKind = "permanent_failure"
	UnsupportedCapability    FailureKind = "unsupported_capability"
	ContractViolation        FailureKind = "contract_violation"
)

type ProviderFailure struct {
	Token             JobToken
	Kind              FailureKind
	Retryable         bool
	RetryAfterSeconds int
	SafeMessage       string
	Evidence          []Evidence
}

func (f ProviderFailure) Validate() error {
	if err := f.Token.Validate(); err != nil || len(f.Evidence) < 1 || len(f.Evidence) > MaxEvidencePerPage || !validBoundedText(f.SafeMessage, false) || f.RetryAfterSeconds < 0 || f.RetryAfterSeconds > 86400 {
		return ErrInvalidContract
	}
	switch f.Kind {
	case ReauthenticationRequired, MFARequired, CaptchaRequired:
		if f.Retryable || f.RetryAfterSeconds != 0 {
			return ErrInvalidContract
		}
	case RateLimited:
		if !f.Retryable || f.RetryAfterSeconds < 1 {
			return ErrInvalidContract
		}
	case TemporaryFailure:
		if !f.Retryable {
			return ErrInvalidContract
		}
	case PermanentFailure, UnsupportedCapability, ContractViolation:
		if f.Retryable || f.RetryAfterSeconds != 0 {
			return ErrInvalidContract
		}
	default:
		return ErrInvalidContract
	}
	ids := map[string]bool{}
	total := 0
	for _, evidence := range f.Evidence {
		if evidence.Validate() != nil || ids[evidence.ID] {
			return ErrInvalidContract
		}
		ids[evidence.ID] = true
		total += len(evidence.Data)
	}
	if total > MaxEvidenceBytes {
		return ErrInvalidContract
	}
	return nil
}

type Result struct {
	Page    *Page
	Failure *ProviderFailure
}

func (r Result) Validate() error {
	if (r.Page == nil) == (r.Failure == nil) {
		return ErrInvalidContract
	}
	if r.Page != nil {
		return r.Page.Validate()
	}
	return r.Failure.Validate()
}

func SameToken(want, got JobToken) bool {
	return want.JobID == got.JobID && want.LeaseToken == got.LeaseToken && want.ConnectionID == got.ConnectionID && want.Attempt == got.Attempt && want.ConnectionGeneration == got.ConnectionGeneration && want.Binding == got.Binding && want.AdmissionRevision == got.AdmissionRevision && want.Cursor == got.Cursor && want.ReplayFrom.Equal(got.ReplayFrom) && want.ReplayTo.Equal(got.ReplayTo)
}

func CanonicalHash(canonical []byte, rawDigest string) ([32]byte, error) {
	if len(canonical) == 0 || !sha256Syntax.MatchString(rawDigest) {
		return [32]byte{}, ErrInvalidContract
	}
	parts := [][]byte{[]byte(ContractVersion), canonical, []byte(rawDigest)}
	return sha256Parts(parts), nil
}

func sha256Parts(parts [][]byte) [32]byte {
	// Length-prefixing prevents different concatenations from sharing an input.
	buffer := bytes.NewBuffer(nil)
	for _, part := range parts {
		buffer.WriteByte(byte(len(part) >> 24))
		buffer.WriteByte(byte(len(part) >> 16))
		buffer.WriteByte(byte(len(part) >> 8))
		buffer.WriteByte(byte(len(part)))
		buffer.Write(part)
	}
	return sha256.Sum256(buffer.Bytes())
}

func SortedGapReasons(coverage reporting.Coverage, extra ...string) []string {
	values := append(coverage.Reasons(), extra...)
	seen := map[string]bool{}
	result := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	sort.Strings(result)
	return result
}

func ValidGapReasons(reasons []string) bool {
	if len(reasons) > 100 {
		return false
	}
	seen := make(map[string]bool, len(reasons))
	for _, reason := range reasons {
		if !validTextLimit(reason, MaxTextLength) || seen[reason] {
			return false
		}
		seen[reason] = true
	}
	return true
}

func validText(value string) bool {
	return validTextLimit(value, MaxTextLength) && strings.TrimSpace(value) != ""
}

func validTextLimit(value string, maximum int) bool {
	return value != "" && !strings.ContainsRune(value, 0) && utf8.ValidString(value) && utf8.RuneCountInString(value) <= maximum
}

func validBoundedText(value string, required bool) bool {
	if value == "" {
		return !required
	}
	return !strings.ContainsRune(value, 0) && utf8.ValidString(value) && utf8.RuneCountInString(value) <= MaxTextLength
}
