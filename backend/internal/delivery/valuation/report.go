package valuation

import (
	"context"
	"fmt"
	"net/http"
	"time"

	account "github.com/pchkauu/want-keep/backend/internal/accounts/domain"
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/contract"
	"github.com/pchkauu/want-keep/backend/internal/delivery/http/generated"
	ledgerapp "github.com/pchkauu/want-keep/backend/internal/ledger/application"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
	valuation "github.com/pchkauu/want-keep/backend/internal/valuation"
	rates "github.com/pchkauu/want-keep/backend/internal/valuation/application"
)

type component struct {
	operationID string
	revision    uint64
	index       int
	kind        string
	native      money.Money
	snapshot    *valuation.Snapshot
}

type currentTotal struct {
	sum     money.Money
	known   bool
	reasons []string
	stale   bool
	inputs  []generated.CalculationInput
}

func (s *Server) report(w http.ResponseWriter, r *http.Request) {
	access, err := s.guard.Authorize(r, s.sessions, false)
	if err != nil {
		s.problem(w, err)
		return
	}
	values := r.URL.Query()
	for key, entries := range values {
		if len(entries) != 1 {
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
		switch key {
		case "view", "asset", "from", "to", "memberId":
		default:
			s.problem(w, contract.ErrInvalidRequest)
			return
		}
	}
	if values.Get("view") != "" && values.Get("view") != "household" || values.Get("memberId") != "" {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	target, err := money.ParseAsset(values.Get("asset"))
	if err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	from, err := calendar.ParseDate(values.Get("from"))
	if err != nil {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	to, err := calendar.ParseDate(values.Get("to"))
	if err != nil || from.String() > to.String() {
		s.problem(w, contract.ErrInvalidRequest)
		return
	}
	now := s.now().UTC()
	var totals []account.AssetTotal
	var components []component
	var timezone calendar.Timezone
	err = s.reads.WithinFinancialRead(r.Context(), access.Principal, func(ctx context.Context) error {
		var readErr error
		timezone, readErr = s.reads.AccountTimezone(ctx, access.Principal)
		if readErr != nil {
			return readErr
		}
		future, dateErr := reportEndIsFuture(now, timezone, to)
		if dateErr != nil {
			return dateErr
		}
		if future {
			return contract.ErrInvalidRequest
		}
		totals, readErr = s.accounts.NativeTotals(ctx, access.Principal)
		if readErr != nil {
			return readErr
		}
		cursor := ledgerapp.Cursor{}
		for {
			page, next, pageErr := s.ledger.List(ctx, access.Principal, ledgerapp.Filter{From: from, To: to}, cursor, 100)
			if pageErr != nil {
				return pageErr
			}
			for _, item := range page {
				facts, componentErr := item.Revision.Components()
				if componentErr != nil {
					return componentErr
				}
				for index, fact := range facts {
					part := component{operationID: item.Revision.OperationID, revision: item.Revision.Revision, index: index, kind: fact.Kind, native: fact.Money}
					snapshot, found, snapshotErr := s.reads.ValuationSnapshot(ctx, access.Principal, part.operationID, part.revision, index, target)
					if snapshotErr != nil {
						return snapshotErr
					}
					if found {
						part.snapshot = &snapshot
					}
					components = append(components, part)
				}
			}
			if len(components) > 10000 {
				return contract.ErrInvalidRequest
			}
			if next == nil {
				return nil
			}
			cursor = *next
		}
	})
	if err != nil {
		s.problem(w, err)
		return
	}
	contextDTO := generated.ReportContext{View: "household", ReportingAsset: generated.Asset(target), From: from.String(), To: to.String(), Timezone: timezone.String()}
	out := generated.Report{Context: contextDTO, Amounts: []generated.ExplainableAmount{}}
	reasons := []string{}
	stale := false
	currentDate, _ := calendar.ParseDate(now.Format(time.DateOnly))
	currentTotals := map[string]*currentTotal{}
	for _, total := range totals {
		for _, entry := range []struct {
			name   string
			amount account.AggregateAmount
		}{{"current_owned", total.Owned}, {"current_available", total.Available}, {"current_debt", total.Debt}} {
			value, rate, amountReasons, amountStale, buildErr := s.currentAmount(r.Context(), entry.name, entry.amount, total.Asset, target, currentDate)
			if buildErr != nil {
				s.problem(w, buildErr)
				return
			}
			out.Amounts = append(out.Amounts, value)
			reasons = append(reasons, amountReasons...)
			stale = stale || amountStale
			aggregate := currentTotals[entry.name]
			if aggregate == nil {
				zero, _ := money.NewMoney("0", target)
				aggregate = &currentTotal{sum: zero, known: true}
				currentTotals[entry.name] = aggregate
			}
			aggregate.reasons = append(aggregate.reasons, amountReasons...)
			aggregate.stale = aggregate.stale || amountStale
			aggregate.inputs = append(aggregate.inputs, generated.CalculationInput{Name: value.Name, Amount: value.Reporting, Resources: []generated.ResourceReference{}})
			if known, knownErr := value.Reporting.AsKnownAmount(); knownErr == nil {
				part, parseErr := money.NewMoney(known.Value.Amount, target)
				if parseErr != nil {
					s.problem(w, parseErr)
					return
				}
				aggregate.sum, err = aggregate.sum.Add(part)
				if err != nil {
					s.problem(w, err)
					return
				}
			} else {
				aggregate.known = false
			}
			if entry.name == "current_owned" && len(entry.amount.MissingAccountIDs) == 0 && total.Asset != target && rate.Reason == "" {
				historical, historicalErr := s.rates.Reference(r.Context(), total.Asset, target, from, false)
				if historicalErr != nil {
					s.problem(w, historicalErr)
					return
				}
				delta, deltaReasons, deltaErr := s.revaluation(entry.amount.KnownSubtotal, target, rate, historical)
				if deltaErr != nil {
					s.problem(w, deltaErr)
					return
				}
				out.Amounts = append(out.Amounts, delta)
				reasons = append(reasons, deltaReasons...)
			}
		}
	}
	for _, name := range []string{"current_owned", "current_available", "current_debt"} {
		if aggregate := currentTotals[name]; aggregate != nil {
			amount, totalErr := s.currentTotalAmount(name+"_total", *aggregate)
			if totalErr != nil {
				s.problem(w, totalErr)
				return
			}
			out.Amounts = append(out.Amounts, amount)
		}
	}
	for _, part := range components {
		value, partReasons, buildErr := s.historicalAmount(part, target)
		if buildErr != nil {
			s.problem(w, buildErr)
			return
		}
		out.Amounts = append(out.Amounts, value)
		reasons = append(reasons, partReasons...)
		stale = stale || part.snapshot != nil && part.snapshot.Freshness == reporting.Stale
	}
	reasons = unique(reasons)
	state := reporting.Complete
	if len(reasons) != 0 {
		state = reporting.Partial
	}
	coverage, _ := reporting.NewCoverage(state, reasons)
	freshness := reporting.Fresh
	if stale {
		freshness = reporting.Stale
	}
	out.Quality, err = s.quality(coverage, freshness, nil)
	if err != nil {
		s.problem(w, err)
		return
	}
	s.write(w, http.StatusOK, out)
}

func reportEndIsFuture(now time.Time, zone calendar.Timezone, end calendar.Date) (bool, error) {
	instant, err := calendar.ParseInstant(now.UTC().Format(time.RFC3339Nano))
	if err != nil {
		return false, err
	}
	today, err := instant.DateIn(zone)
	if err != nil {
		return false, err
	}
	return end.String() > today.String(), nil
}

func (s *Server) currentTotalAmount(name string, total currentTotal) (generated.ExplainableAmount, error) {
	reasons := unique(total.reasons)
	value, err := s.amountValue(total.sum, total.known, reasons)
	if err != nil {
		return generated.ExplainableAmount{}, err
	}
	state := reporting.Complete
	if len(reasons) != 0 {
		state = reporting.Partial
	}
	coverage, err := reporting.NewCoverage(state, reasons)
	if err != nil {
		return generated.ExplainableAmount{}, err
	}
	freshness := reporting.Fresh
	if total.stale {
		freshness = reporting.Stale
	}
	quality, err := s.quality(coverage, freshness, nil)
	if err != nil {
		return generated.ExplainableAmount{}, err
	}
	return generated.ExplainableAmount{Name: name, Kind: "actual", Native: []generated.Money{}, Reporting: value, Quality: quality, Method: "sum of current per-asset equivalents; incomplete input makes total unknown", Inputs: total.inputs, Resources: []generated.ResourceReference{}}, nil
}

func (s *Server) currentAmount(ctx context.Context, name string, amount account.AggregateAmount, asset, target money.Asset, date calendar.Date) (generated.ExplainableAmount, applicationReference, []string, bool, error) {
	native, err := (contract.MoneyConverter{}).ToDTO(amount.KnownSubtotal)
	if err != nil {
		return generated.ExplainableAmount{}, applicationReference{}, nil, false, err
	}
	known := len(amount.MissingAccountIDs) == 0
	reasons := []string{}
	if !known {
		reasons = append(reasons, "incomplete_accounts")
	}
	reference := applicationReference{}
	converted := amount.KnownSubtotal
	if asset != target {
		reference, err = s.rates.Reference(ctx, asset, target, date, true)
		if err != nil {
			return generated.ExplainableAmount{}, reference, nil, false, err
		}
		if reference.Reason != "" {
			known = false
			reasons = append(reasons, reference.Reason)
		} else {
			converted, _, _, err = valuation.Convert(amount.KnownSubtotal, target, reference.Legs)
			if err != nil {
				return generated.ExplainableAmount{}, reference, nil, false, err
			}
			reasons = append(reasons, reference.Coverage.Reasons()...)
		}
	}
	report, err := s.amountValue(converted, known, reasons)
	if err != nil {
		return generated.ExplainableAmount{}, reference, nil, false, err
	}
	state := reporting.Complete
	if len(reasons) != 0 {
		state = reporting.Partial
	}
	coverage, _ := reporting.NewCoverage(state, unique(reasons))
	freshness := reference.Freshness
	if asset == target {
		freshness = reporting.UnknownFreshness
	}
	quality, err := s.quality(coverage, freshness, reference.Legs)
	if err != nil {
		return generated.ExplainableAmount{}, reference, nil, false, err
	}
	inputAmount, _ := reporting.KnownAmount(amount.KnownSubtotal)
	input, err := s.boundary.AmountToDTO(inputAmount)
	if err != nil {
		return generated.ExplainableAmount{}, reference, nil, false, err
	}
	out := generated.ExplainableAmount{Name: name + "_" + string(asset), Kind: "actual", Native: []generated.Money{native}, Reporting: report, Quality: quality, Method: "reference valuation of current native amount; incomplete accounts remain unknown", Inputs: []generated.CalculationInput{{Name: "known_native_subtotal", Amount: input, Resources: []generated.ResourceReference{}}}, Resources: []generated.ResourceReference{}}
	return out, reference, reasons, reference.Freshness == reporting.Stale, nil
}

// applicationReference keeps the rate selection result at the delivery boundary.
type applicationReference = rates.Reference

func (s *Server) revaluation(native money.Money, target money.Asset, current, historical applicationReference) (generated.ExplainableAmount, []string, error) {
	reasons := []string{}
	if historical.Reason != "" {
		reasons = append(reasons, historical.Reason)
	}
	var difference money.Money
	if len(reasons) == 0 {
		now, _, _, err := valuation.Convert(native, target, current.Legs)
		if err != nil {
			return generated.ExplainableAmount{}, nil, err
		}
		before, _, _, err := valuation.Convert(native, target, historical.Legs)
		if err != nil {
			return generated.ExplainableAmount{}, nil, err
		}
		difference, err = now.Subtract(before)
		if err != nil {
			return generated.ExplainableAmount{}, nil, err
		}
	}
	report, err := s.amountValue(difference, len(reasons) == 0, reasons)
	if err != nil {
		return generated.ExplainableAmount{}, nil, err
	}
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	if len(reasons) != 0 {
		coverage, _ = reporting.NewCoverage(reporting.Partial, reasons)
	}
	legs := append(append([]valuation.Observation(nil), current.Legs...), historical.Legs...)
	quality, err := s.quality(coverage, current.Freshness, legs)
	if err != nil {
		return generated.ExplainableAmount{}, nil, err
	}
	nativeDTO, _ := (contract.MoneyConverter{}).ToDTO(native)
	out := generated.ExplainableAmount{Name: "reference_revaluation_" + string(native.Asset()), Kind: "revaluation", Native: []generated.Money{nativeDTO}, Reporting: report, Quality: quality, Method: "change in reference value of current native holdings between requested start date and current date; not realized profit", Inputs: []generated.CalculationInput{}, Resources: []generated.ResourceReference{}}
	return out, reasons, nil
}

func (s *Server) historicalAmount(part component, target money.Asset) (generated.ExplainableAmount, []string, error) {
	nativeDTO, err := (contract.MoneyConverter{}).ToDTO(part.native)
	if err != nil {
		return generated.ExplainableAmount{}, nil, err
	}
	reasons := []string{}
	var converted money.Money
	var legs []valuation.Observation
	known := false
	freshness := reporting.UnknownFreshness
	if part.snapshot == nil {
		reasons = append(reasons, "valuation_pending")
	} else if part.snapshot.Reporting == nil {
		reasons = append(reasons, part.snapshot.Reason)
		legs = part.snapshot.Legs
		freshness = part.snapshot.Freshness
	} else {
		converted = *part.snapshot.Reporting
		legs = part.snapshot.Legs
		reasons = append(reasons, part.snapshot.CoverageReasons...)
		freshness = part.snapshot.Freshness
		known = true
	}
	report, err := s.amountValue(converted, known, reasons)
	if err != nil {
		return generated.ExplainableAmount{}, nil, err
	}
	coverage, _ := reporting.NewCoverage(reporting.Complete, nil)
	if len(reasons) != 0 {
		coverage, _ = reporting.NewCoverage(reporting.Partial, reasons)
	}
	quality, err := s.quality(coverage, freshness, legs)
	if err != nil {
		return generated.ExplainableAmount{}, nil, err
	}
	revision := generated.Revision(part.revision)
	resource := generated.ResourceReference{Type: "transaction", Id: part.operationID, Revision: &revision}
	out := generated.ExplainableAmount{Name: fmt.Sprintf("%s_%s_%d", part.kind, part.operationID, part.index), Kind: "actual", Native: []generated.Money{nativeDTO}, Reporting: report, Quality: quality, Method: "pinned transaction-date reference valuation", Inputs: []generated.CalculationInput{}, Resources: []generated.ResourceReference{resource}}
	if part.kind == "unrealized_pnl" {
		out.Kind = "revaluation"
	}
	return out, reasons, nil
}

func (s *Server) amountValue(value money.Money, known bool, reasons []string) (generated.AmountValue, error) {
	if known {
		amount, err := reporting.KnownAmount(value)
		if err != nil {
			return generated.AmountValue{}, err
		}
		return s.boundary.AmountToDTO(amount)
	}
	reason := "valuation_unavailable"
	if len(reasons) != 0 {
		reason = reasons[0]
	}
	amount, err := reporting.MissingAmount(reporting.Unknown, reason)
	if err != nil {
		return generated.AmountValue{}, err
	}
	return s.boundary.AmountToDTO(amount)
}

func unique(values []string) []string {
	seen := map[string]bool{}
	out := []string{}
	for _, value := range values {
		if value != "" && !seen[value] {
			out = append(out, value)
			seen[value] = true
		}
	}
	return out
}
