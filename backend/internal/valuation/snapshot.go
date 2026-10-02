package valuation

import (
	"errors"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

var ErrInvalidSnapshot = errors.New("invalid valuation snapshot")

// Snapshot pins the source observations used for one historical economic component.
type Snapshot struct {
	OperationID       string
	OperationRevision uint64
	ComponentIndex    int
	ComponentKind     string
	ValuationRevision uint64
	Native            money.Money
	Reporting         *money.Money
	Target            money.Asset
	RequestedDate     calendar.Date
	Reason            string
	CoverageReasons   []string
	Freshness         reporting.Freshness
	Legs              []Observation
	RecordedAt        calendar.Instant
}

func (s Snapshot) Validate() error {
	if s.OperationID == "" || s.OperationRevision < 1 || s.ComponentIndex < 0 || s.ComponentIndex > 1000 || s.ComponentKind == "" || len(s.ComponentKind) > 100 || s.ValuationRevision < 1 || s.RequestedDate.String() == "" || s.RecordedAt.String() == "" || s.Native.Validate() != nil || len(s.Legs) > 2 {
		return ErrInvalidSnapshot
	}
	if _, err := reporting.ParseFreshness(string(s.Freshness)); err != nil || len(s.CoverageReasons) > 20 {
		return ErrInvalidSnapshot
	}
	for _, reason := range s.CoverageReasons {
		if reason == "" || len(reason) > 2000 {
			return ErrInvalidSnapshot
		}
	}
	if _, err := money.ParseAsset(string(s.Target)); err != nil {
		return ErrInvalidSnapshot
	}
	if (s.Reporting == nil) == (s.Reason == "") || s.Reporting == nil && len(s.CoverageReasons) != 0 {
		return ErrInvalidSnapshot
	}
	if s.Reporting != nil && (s.Reporting.Validate() != nil || s.Reporting.Asset() != s.Target) {
		return ErrInvalidSnapshot
	}
	if s.Reporting != nil && s.Native.Asset() == s.Target && (s.Native.Amount() != s.Reporting.Amount() || len(s.Legs) != 0 || s.Freshness != reporting.UnknownFreshness) {
		return ErrInvalidSnapshot
	}
	if s.Reporting != nil && s.Native.Asset() != s.Target {
		converted, _, used, err := Convert(s.Native, s.Target, s.Legs)
		if err != nil || len(used) != len(s.Legs) || converted.Amount() != s.Reporting.Amount() {
			return ErrInvalidSnapshot
		}
	}
	for _, leg := range s.Legs {
		if leg.Validate() != nil || leg.RequestedDate != s.RequestedDate {
			return ErrInvalidSnapshot
		}
	}
	return nil
}
