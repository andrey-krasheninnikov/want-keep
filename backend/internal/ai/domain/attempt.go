package domain

import (
	"encoding/json"
	"errors"
	"unicode/utf8"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

var ErrInvalidAttempt = errors.New("invalid AI attempt")

type Purpose string

const (
	TransactionReview    Purpose = "transaction_review"
	ReceiptPage          Purpose = "receipt_page"
	ChatInsight          Purpose = "chat_insight"
	ComplexClarification Purpose = "complex_clarification"
)

func (p Purpose) Valid() bool {
	return p == TransactionReview || p == ReceiptPage || p == ChatInsight || p == ComplexClarification
}

func (p Purpose) Limits() (int64, int64) {
	switch p {
	case TransactionReview:
		return 8192, 2048
	case ReceiptPage, ChatInsight:
		return 16384, 4096
	case ComplexClarification:
		return 32768, 8192
	default:
		return 0, 0
	}
}

type Model string
type Qualification string

const (
	Terra      Model         = "gpt-5.6-terra"
	TerraXHigh Qualification = "terra_xhigh"
)

type RuntimeContract struct {
	Model                                                   Model
	Qualification                                           Qualification
	PromptFingerprint, SchemaFingerprint, ConfigFingerprint string
}

func (c RuntimeContract) Validate() error {
	request := Request{
		ID: "contract", JobID: "contract", ResourceID: "contract",
		HouseholdID: "contract", ActorID: "contract", ResourceRevision: 1,
		Purpose: TransactionReview, Model: c.Model, Qualification: c.Qualification,
		PromptFingerprint: c.PromptFingerprint, SchemaFingerprint: c.SchemaFingerprint,
		ConfigFingerprint: c.ConfigFingerprint, Input: json.RawMessage(`{}`), MaximumOutputTokens: 1,
	}
	return request.Validate()
}

type State string

const (
	Counting       State = "counting"
	Reserved       State = "reserved"
	Completed      State = "completed"
	Refused        State = "refused"
	Incomplete     State = "incomplete"
	SchemaError    State = "schema_error"
	KnownRejection State = "known_rejection"
	Unknown        State = "unknown"
)

func (s State) Valid() bool {
	switch s {
	case Counting, Reserved, Completed, Refused, Incomplete, SchemaError, KnownRejection, Unknown:
		return true
	default:
		return false
	}
}

func (s State) Active() bool   { return s == Counting || s == Reserved }
func (s State) Terminal() bool { return s.Valid() && !s.Active() }

type Request struct {
	ID, JobID, ResourceID                                   string
	HouseholdID                                             household.HouseholdID
	ActorID                                                 household.UserID
	ResourceRevision                                        uint64
	Purpose                                                 Purpose
	Model                                                   Model
	Qualification                                           Qualification
	PromptFingerprint, SchemaFingerprint, ConfigFingerprint string
	Input                                                   json.RawMessage
	Image                                                   []byte
	MaximumOutputTokens                                     int64
}

func (r Request) Validate() error {
	maximumInput, maximumOutput := r.Purpose.Limits()
	if len(r.Image) > 17*1024*1024 || ((len(r.Image) > 0) != (r.Purpose == ReceiptPage)) {
		return ErrInvalidAttempt
	}
	if r.ID == "" || r.JobID == "" || r.ResourceID == "" || r.HouseholdID == "" || r.ActorID == "" || r.ResourceRevision < 1 || !r.Purpose.Valid() || r.Model != Terra || r.Qualification != TerraXHigh || r.MaximumOutputTokens < 1 || r.MaximumOutputTokens > maximumOutput || maximumInput == 0 || !json.Valid(r.Input) || len(r.Input) == 0 || len(r.Input) > 1<<20 {
		return ErrInvalidAttempt
	}
	for _, fingerprint := range []string{r.PromptFingerprint, r.SchemaFingerprint, r.ConfigFingerprint} {
		if len(fingerprint) != 64 {
			return ErrInvalidAttempt
		}
		for _, c := range fingerprint {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
				return ErrInvalidAttempt
			}
		}
	}
	return nil
}

type Result struct {
	ProviderID    string
	ProviderModel string
	State         State
	Output        json.RawMessage
	Usage         Usage
	Code          string
}

func (r Result) Validate() error {
	if !r.State.Terminal() || utf8.RuneCountInString(r.ProviderID) > 200 || utf8.RuneCountInString(r.ProviderModel) > 200 || utf8.RuneCountInString(r.Code) > 100 || len(r.Output) > 1<<20 {
		return ErrInvalidAttempt
	}
	if r.State == Completed {
		if r.ProviderID == "" || r.ProviderModel == "" || !json.Valid(r.Output) || len(r.Output) == 0 {
			return ErrInvalidAttempt
		}
	} else if len(r.Output) > 0 && !json.Valid(r.Output) {
		return ErrInvalidAttempt
	}
	if r.State != KnownRejection && r.State != Unknown {
		if r.ProviderID == "" || Model(r.ProviderModel) != Terra {
			return ErrInvalidAttempt
		}
		return r.Usage.Validate()
	}
	return nil
}
