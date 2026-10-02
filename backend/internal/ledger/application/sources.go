package application

import (
	"context"
	"errors"
	"fmt"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
)

type SourceRepository interface {
	AccountTimezone(context.Context, household.Principal) (calendar.Timezone, error)
	SaveSourceFact(context.Context, ledger.SourceRecord, ledger.Revision, string) error
	Source(context.Context, household.Principal, ledger.SourceKey) (ledger.SourceRecord, bool, error)
	HistoricalSourceRevision(context.Context, household.Principal, ledger.SourceRecord, string) (uint64, bool, error)
	SaveSource(context.Context, ledger.SourceRecord, ledger.SourceInput) (ledger.SourceRecord, error)
	RecordProvenance(context.Context, ledger.SourceRecord, ledger.SourceInput) error
	RecordSourceAmbiguity(context.Context, ledger.SourceInput) error
	RecordUnresolvedTransaction(context.Context, ledger.SourceInput) error
	CurrentLedgerRevision(context.Context, household.Principal, string) (ledger.Revision, bool, error)
	EmitEvent(context.Context, string, string, uint64, string) error
}
type Sources struct {
	repository  SourceRepository
	writer      JournalWriter
	allocations AllocationResolver
}

type SourceIdentityRepository interface {
	CheckSourceIdentity(context.Context, household.Principal, ledger.SourceInput) (string, error)
}

func NewSources(r SourceRepository, w JournalWriter, allocations AllocationResolver) *Sources {
	return &Sources{repository: r, writer: w, allocations: allocations}
}

// Apply belongs inside CommitPage's transaction, after the deployment and lease fences.
func (s *Sources) Apply(ctx context.Context, p household.Principal, input ledger.SourceInput) (ledger.SourceOutcome, error) {
	if err := input.Validate(); err != nil {
		return ledger.SourceOutcome{}, err
	}
	if err := p.RequireHousehold(input.Key.HouseholdID); err != nil {
		return ledger.SourceOutcome{}, err
	}
	if input.SourceAsOf.String() != "" || len(input.Aliases) > 0 {
		repository, ok := s.repository.(SourceIdentityRepository)
		if !ok {
			return ledger.SourceOutcome{}, ledger.ErrInvalidSource
		}
		var err error
		input.Classification, err = repository.CheckSourceIdentity(ctx, p, input)
		if err != nil {
			return ledger.SourceOutcome{}, err
		}
	}
	if input.Operation != nil {
		raw := input.Operation.Clone()
		raw.Revision = 1
		raw.ActorID = p.UserID()
		raw.HumanOverride = false
		raw.Participation = ledger.Participation{}
		raw.Protections = map[ledger.Field]ledger.Protection{}
		raw.FieldVersions = map[ledger.Field]uint64{}
		raw.AccountingState = ledger.IncludedInAccounting
		raw.CategoryID, raw.MerchantID, raw.ReceiptItems = "", "", nil
		raw.DecisionID, raw.ReviewState = "", ""
		raw.Allocation = ledger.AllocationSnapshot{}
		raw.AllocationReason = ""
		input.Operation = &raw
	}
	if input.Operation != nil && input.Operation.Validate() != nil {
		input.Operation = nil
		input.UnresolvedReason = "invalid_transaction"
	}
	if input.Operation == nil && input.UnresolvedReason == "" {
		input.UnresolvedReason = "transaction_not_normalized"
	}
	current, exists, err := s.repository.Source(ctx, p, input.Key)
	if errors.Is(err, ledger.ErrSourceAmbiguous) {
		return ledger.SourceOutcome{Record: ledger.SourceRecord{Key: input.Key, Ambiguous: true}}, s.recordAmbiguity(ctx, input)
	}
	if err != nil {
		return ledger.SourceOutcome{}, err
	}
	var duplicate bool
	provenance := current
	if exists {
		confirmedCorrection := input.Classification == "correction" && input.ExpectedRevision == current.Revision
		if current.Ambiguous && !confirmedCorrection && current.PayloadHash != input.PayloadHash {
			var revision uint64
			revision, duplicate, err = s.repository.HistoricalSourceRevision(ctx, p, current, input.PayloadHash)
			if err != nil {
				return ledger.SourceOutcome{}, err
			}
			if duplicate {
				provenance.Revision = revision
				provenance.PayloadHash = input.PayloadHash
			}
		}
		if !duplicate {
			current, duplicate, err = current.Next(input)
			if err != nil {
				return ledger.SourceOutcome{}, err
			}
			provenance = current
		}
	} else {
		current = ledger.SourceRecord{Key: input.Key, Revision: 1, PayloadHash: input.PayloadHash, Ambiguous: input.Classification != "new"}
		if input.Operation != nil && !current.Ambiguous {
			current.OperationID = input.Operation.OperationID
		}
		provenance = current
	}
	if !duplicate && input.Operation != nil && current.OperationID != "" && current.OperationID != input.Operation.OperationID {
		current.Ambiguous = true
	}
	if !duplicate && !current.Ambiguous && current.OperationID == "" && input.Operation != nil {
		current.OperationID = input.Operation.OperationID
	}
	if !duplicate {
		current, err = s.repository.SaveSource(ctx, current, input)
		if errors.Is(err, ledger.ErrSourceAmbiguous) {
			return ledger.SourceOutcome{Record: ledger.SourceRecord{Key: input.Key, Ambiguous: true}}, s.recordAmbiguity(ctx, input)
		}
		if err != nil {
			return ledger.SourceOutcome{}, err
		}
		provenance = current
	}
	if err = s.repository.RecordProvenance(ctx, provenance, input); err != nil {
		if errors.Is(err, ledger.ErrSourceAmbiguous) {
			return ledger.SourceOutcome{Record: ledger.SourceRecord{Key: input.Key, Ambiguous: true}}, s.recordAmbiguity(ctx, input)
		}
		return ledger.SourceOutcome{}, err
	}
	result := ledger.SourceOutcome{Record: current, Duplicate: duplicate}
	if current.Ambiguous {
		return result, s.recordAmbiguity(ctx, input)
	}
	if input.UnresolvedReason != "" {
		if err = s.repository.RecordUnresolvedTransaction(ctx, input); err != nil {
			return result, err
		}
		return result, nil
	}
	if duplicate {
		return result, nil
	}
	if input.Operation != nil {
		if current.OperationID != input.Operation.OperationID {
			return result, ledger.ErrSourceAmbiguous
		}
		previous, found, err := s.repository.CurrentLedgerRevision(ctx, p, current.OperationID)
		if err != nil {
			return result, err
		}
		expected := uint64(0)
		raw := input.Operation.Clone()
		raw.Origin = "source"
		raw.ActorID = p.UserID()
		zone, e := s.repository.AccountTimezone(ctx, p)
		if e != nil {
			return result, e
		}
		raw, e = raw.InTimezone(zone)
		if e != nil {
			return result, e
		}
		for i, posting := range raw.Postings {
			a, e := s.writer.Account(ctx, p, posting.AccountID)
			if e != nil {
				return result, e
			}
			if posting.Funding == "" {
				raw.Postings[i].Funding = ledger.OwnFunds
				if a.Product == "credit_card" {
					raw.Postings[i].Funding = ledger.UnknownFunds
				}
			}
		}
		if !found {
			raw, e = raw.WithAllocation(ledger.AllocationInput{}, nil, nil)
			if e != nil {
				return result, fmt.Errorf("initialize source allocation: %w", e)
			}
			if raw.Allocation.State != ledger.AllocationNotApplicable && s.allocations != nil {
				allocation, matched, resolveErr := s.allocations.ResolveSource(ctx, p, raw.Merchant)
				if resolveErr != nil {
					return result, fmt.Errorf("resolve source allocation: %w", resolveErr)
				}
				if matched {
					var members []household.MembershipID
					if allocation.Mode != ledger.AllocationUnknown {
						members, resolveErr = s.allocations.ActiveMemberIDs(ctx, p)
						if resolveErr != nil {
							return result, fmt.Errorf("load source allocation members: %w", resolveErr)
						}
					}
					raw, resolveErr = raw.WithAllocation(allocation, nil, members)
					if resolveErr != nil {
						return result, fmt.Errorf("apply source allocation: %w", resolveErr)
					}
				}
			}
		}
		r := raw.Clone()
		conflict := ""
		if found {
			expected = previous.Revision
			var differs bool
			r, differs, err = previous.MergeSource(raw)
			if err != nil {
				return result, err
			}
			result.PreservedOverride = len(previous.Protections) > 0 || previous.HumanOverride
			if differs {
				conflict = "protected_fields"
			}
			if _, legacy := previous.Protections[ledger.LegacyField]; legacy || previous.HumanOverride && len(previous.Protections) == 0 {
				return result, s.repository.SaveSourceFact(ctx, current, raw, "legacy_protection")
			}
		}
		r.Revision = expected + 1
		r.ActorID = p.UserID()
		r.DecisionID = ""
		// The storage boundary records commit-time observation separately from provider fetch time.
		r.RecordedAt = calendar.Instant{}
		r, err = r.InTimezone(zone)
		if err != nil {
			return result, err
		}
		if r.FieldVersions == nil {
			r.FieldVersions = map[ledger.Field]uint64{}
		}
		for _, f := range []ledger.Field{ledger.PrincipalField, ledger.FeesField, ledger.DateField, ledger.PayerField, ledger.MerchantField, ledger.NoteField} {
			if !found || !previous.FieldEqual(r, f) {
				r.FieldVersions[f] = r.Revision
			}
		}
		var prior *ledger.Revision
		if found {
			prior = &previous
		}
		validation := r.Clone()
		validation.Participation = ledger.Participation{}
		validationErr := validation.CheckSuccessor(prior)
		if found && errors.Is(validationErr, ledger.ErrInvalidAllocation) {
			refreshed, refreshErr := validation.RefreshAllocation()
			_, allocationProtected := previous.Protections[ledger.AllocationField]
			removesProtectedAllocation := allocationProtected && previous.Allocation.State != ledger.AllocationNotApplicable && refreshed.Allocation.State == ledger.AllocationNotApplicable
			if refreshErr == nil && !removesProtectedAllocation {
				if copyErr := r.CopyField(refreshed, ledger.AllocationField); copyErr != nil {
					return result, copyErr
				}
				if !previous.FieldEqual(r, ledger.AllocationField) {
					r.FieldVersions[ledger.AllocationField] = r.Revision
				}
				validation = refreshed
				validationErr = validation.CheckSuccessor(prior)
			}
		}
		if validationErr != nil {
			if !found {
				return result, s.repository.RecordUnresolvedTransaction(ctx, input)
			}
			return result, s.repository.SaveSourceFact(ctx, current, raw, "invalid_merge")
		}
		if found && previous.Participation.GroupID == "" && previous.SameFacts(r) {
			return result, s.repository.SaveSourceFact(ctx, current, raw, conflict)
		}
		if err = s.writer.AppendSource(ctx, p, r, expected, ledger.Evidence{Kind: "source", ID: current.ID, Revision: current.Revision}); err != nil {
			if errors.Is(err, ledger.ErrMatchingConflict) {
				return result, s.repository.SaveSourceFact(ctx, current, raw, "matching_conflict")
			}
			return result, err
		}
		if err = s.repository.SaveSourceFact(ctx, current, raw, conflict); err != nil {
			return result, err
		}
	}
	return result, nil
}

func (s *Sources) recordAmbiguity(ctx context.Context, input ledger.SourceInput) error {
	if err := s.repository.RecordSourceAmbiguity(ctx, input); err != nil {
		return err
	}
	if input.UnresolvedReason == "" {
		return nil
	}
	return s.repository.RecordUnresolvedTransaction(ctx, input)
}
