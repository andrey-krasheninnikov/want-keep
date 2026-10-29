package domain

import (
	"time"

	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
)

type State string

const (
	Uploaded      State = "uploaded"
	Validating    State = "validating"
	Processing    State = "processing"
	Clarification State = "clarification"
	Skipped       State = "skipped"
	Linked        State = "linked"
	Recorded      State = "recorded"
	Failed        State = "failed"
	WaitingAI     State = "waiting_ai"
)

type Receipt struct {
	ID, AccountID, AttachmentID, Hash string
	HouseholdID                       household.HouseholdID
	ActorID                           household.UserID
	Revision                          uint64
	State                             State
	Reason                            string
	Pages                             []Page
	Facts                             *Facts
	Confirmed                         *Extraction
	TransactionID                     string
	TransactionRevision               uint64
	Candidates                        []Candidate
	CreatedAt                         time.Time
}
type Page struct {
	ID, Hash   string
	Number     int
	Extraction *Extraction
	AttemptID  string
}
type Candidate struct {
	ID       string
	Revision uint64
}
type Answer struct {
	ExpectedRevision uint64
	Facts            *Extraction
	Match            *Candidate
	Distinct         bool
	Purchase         *Candidate
	RefundItems      []RefundItem
}

type RefundItem struct{ ID, Amount string }
