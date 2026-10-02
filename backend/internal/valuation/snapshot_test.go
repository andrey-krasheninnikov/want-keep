package valuation

import (
	"testing"

	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

func TestSnapshotKeepsPartialValueButRejectsChangedNativeAmount(t *testing.T) {
	date, _ := calendar.ParseDate("2026-09-01")
	at, _ := calendar.ParseInstant("2026-09-01T00:00:00Z")
	native, _ := money.NewMoney("10", money.USDC)
	value, _, legs, _ := Convert(native, money.USD, []Observation{observation(t, money.USDC, "0.997")})
	snapshot := Snapshot{OperationID: "synthetic", OperationRevision: 1, ComponentKind: "expense", ValuationRevision: 1, Native: native, Reporting: &value, Target: money.USD, RequestedDate: date, CoverageReasons: []string{"source_mismatch"}, Freshness: reporting.Stale, Legs: legs, RecordedAt: at}
	if err := snapshot.Validate(); err != nil {
		t.Fatalf("partial known snapshot: %v", err)
	}
	native, _ = money.NewMoney("10", money.USD)
	wrong, _ := money.NewMoney("11", money.USD)
	snapshot.Native, snapshot.Reporting, snapshot.Legs, snapshot.CoverageReasons, snapshot.Freshness = native, &wrong, nil, nil, reporting.UnknownFreshness
	if err := snapshot.Validate(); err == nil {
		t.Fatal("changed same-asset amount accepted")
	}
}
