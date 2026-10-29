package application

import (
	"context"
	"encoding/json"
	"time"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	ai "github.com/pchkauu/want-keep/backend/internal/ai/domain"
	attachment "github.com/pchkauu/want-keep/backend/internal/attachments/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	category "github.com/pchkauu/want-keep/backend/internal/categories/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	jobs "github.com/pchkauu/want-keep/backend/internal/jobs/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	matching "github.com/pchkauu/want-keep/backend/internal/matching/domain"
	receipt "github.com/pchkauu/want-keep/backend/internal/receipts/domain"
)

type Validation struct {
	PageID, AttemptID, Code string
	State                   ai.State
	Output                  json.RawMessage
}
type CategoryReference struct {
	ID       string
	Revision uint64
}

type Repository interface {
	MatchingForOperation(context.Context, household.Principal, string) (matching.Group, bool, error)
	Category(context.Context, household.Principal, string) (category.Category, error)
	ReceiptPageReferences(context.Context, household.Principal, string) (map[string]CategoryReference, error)
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	Account(context.Context, household.Principal, string) (account.Account, error)
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	Attachment(context.Context, household.Principal, string) (attachment.Attachment, error)
	Receipt(context.Context, household.Principal, string) (receipt.Receipt, error)
	ReceiptProcessingState(context.Context, household.Principal, string) (receipt.State, string, error)
	ReceiptByHash(context.Context, household.Principal, string, string) (receipt.Receipt, bool, error)
	CreateReceipt(context.Context, household.Principal, receipt.Receipt) error
	AddReceiptEvidence(context.Context, household.Principal, string, attachment.Attachment) error
	SaveReceipt(context.Context, household.Principal, receipt.Receipt, uint64) error
	SaveReceiptAnswer(context.Context, household.Principal, receipt.Receipt, receipt.Answer) error
	ReceiptValidation(context.Context, household.Principal, jobs.Job) (Validation, error)
	SaveReceiptPageOutcome(context.Context, household.Principal, Validation, *receipt.Extraction) error
	ReceiptForPage(context.Context, household.Principal, string) (receipt.Receipt, int, error)
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	ReceiptCandidates(context.Context, household.Principal, receipt.Receipt, receipt.Facts, string) (receipt.CandidateSet, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}
type PageRepository interface {
	ReceiptPageProjection(context.Context, household.Principal, receipt.Receipt, int) (json.RawMessage, error)
	WithinHousehold(context.Context, household.Principal, func(context.Context) error) error
	FenceJob(context.Context, household.Principal, jobs.Job) (jobs.Job, error)
	ReceiptForPage(context.Context, household.Principal, string) (receipt.Receipt, int, error)
}
type Documents interface {
	Content(context.Context, household.Principal, string, int) (attachment.Attachment, []byte, error)
}
type Pages struct {
	Repository PageRepository
	Documents  Documents
}

func (s Pages) ReceiptPageInput(ctx context.Context, p household.Principal, job jobs.Job) (json.RawMessage, []byte, error) {
	var value receipt.Receipt
	var number int
	err := s.Repository.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, err := s.Repository.FenceJob(ctx, p, job); err != nil {
			return err
		}
		var err error
		value, number, err = s.Repository.ReceiptForPage(ctx, p, job.ResourceID)
		return err
	})
	if err != nil {
		return nil, nil, err
	}
	if value.State != receipt.Processing || job.ResourceRevision != 1 {
		return nil, nil, jobs.ErrStaleAttempt
	}
	a, data, err := s.Documents.Content(ctx, p, value.AttachmentID, number)
	if err != nil {
		return nil, nil, err
	}
	if number > len(a.Pages) || attachment.Hash(data) != a.Pages[number-1].Hash {
		clear(data)
		return nil, nil, receipt.ErrInvalid
	}
	var input json.RawMessage
	err = s.Repository.WithinHousehold(ctx, p, func(ctx context.Context) error {
		if _, e := s.Repository.FenceJob(ctx, p, job); e != nil {
			return e
		}
		var e error
		input, e = s.Repository.ReceiptPageProjection(ctx, p, value, number)
		return e
	})
	return input, data, err
}

type Clock func() time.Time
