package domain

import (
	"slices"
	"sort"
	"strconv"

	ledger "github.com/pchkauu/want-keep/backend/internal/ledger/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
)

type componentKey struct {
	account string
	role    ledger.Role
	asset   money.Asset
}
type assignedComponent struct {
	posting         ledger.Posting
	carrier         ledger.Revision
	position        int
	state           ledger.State
	protectedFields map[ledger.Field]ledger.Revision
}
type effectAssignment struct {
	group      Group
	components map[string]*assignedComponent
	buckets    map[componentKey][]string
	result     []ledger.Revision
}

func (a *effectAssignment) assign(facts []ledger.Revision, allowPartial bool) (Group, []ledger.Revision, error) {
	if len(facts) < 1 || len(facts) > 100 || a.group.Kind == Payment && len(facts) < 2 {
		return a.group, nil, ErrInvalid
	}
	seen := map[string]bool{}
	primary := false
	for _, r := range facts {
		v := r.Clone()
		v.Participation = ledger.Participation{}
		v, validationErr := v.RefreshAllocation()
		if seen[r.OperationID] || validationErr != nil || v.Validate() != nil || !slices.Contains([]ledger.Type{ledger.Income, ledger.Expense, ledger.Transfer, ledger.Exchange, ledger.Refund}, r.Type) || r.Type == ledger.Refund && a.group.Kind != Payment {
			return a.group, nil, ErrInvalid
		}
		seen[r.OperationID] = true
		primary = primary || r.OperationID == a.group.PrimaryID
		if a.group.Kind == Payment && r.Correspondence != nil {
			for _, other := range facts {
				if other.Correspondence != nil && r.Correspondence.DistinctPayment(*other.Correspondence) {
					return a.group, nil, ErrConflict
				}
			}
		}
	}
	if !primary {
		return a.group, nil, ErrInvalid
	}
	facts = slices.Clone(facts)
	sort.SliceStable(facts, func(i, j int) bool {
		x, y := facts[i], facts[j]
		// Preserve an existing carrier, then a fact with an existing effect. Waiting
		// evidence and display-primary selection cannot take ownership of its date.
		xr, yr := a.carrierRank(x), a.carrierRank(y)
		if xr != yr {
			return xr < yr
		}
		if (x.RecordedAt.String() == "") != (y.RecordedAt.String() == "") {
			return x.RecordedAt.String() != ""
		}
		if x.RecordedAt != y.RecordedAt {
			return x.RecordedAt.Time().Before(y.RecordedAt.Time())
		}
		return x.OperationID < y.OperationID
	})
	a.components = map[string]*assignedComponent{}
	a.buckets = map[componentKey][]string{}
	a.result = []ledger.Revision{}
	// Seed each retained component independently: owning the incoming component
	// does not let a record displace the existing outgoing carrier.
	for _, r := range facts {
		for _, part := range r.Participation.Parts {
			if part.CarrierID != r.OperationID || part.Position >= len(r.Postings) {
				continue
			}
			p := r.Postings[part.Position]
			if (part.Role == "fee") != (p.Role == ledger.Fee) {
				continue
			}
			id := r.OperationID + ":" + strconv.Itoa(part.Position)
			a.components[id] = &assignedComponent{posting: p, carrier: r, position: part.Position, state: r.State}
			key := componentKey{p.AccountID, p.Role, p.Money.Asset()}
			a.buckets[key] = append(a.buckets[key], id)
		}
	}
	for _, r := range facts {
		if err := a.observe(r); err != nil {
			return a.group, nil, err
		}
	}
	if err := a.validatePrincipal(allowPartial); err != nil {
		return a.group, nil, err
	}
	a.group.Members = nil
	a.group.Candidates = nil
	a.group.CandidatesComplete = true
	for i := range a.result {
		r := &a.result[i]
		for j := range r.Participation.Parts {
			p := &r.Participation.Parts[j]
			c := a.components[p.ComponentID]
			p.State = c.state
			p.At = c.carrier.OccurredAt
		}
		refreshed, err := r.RefreshAllocation()
		if err != nil {
			return a.group, nil, ErrConflict
		}
		if _, protected := r.Protections[ledger.AllocationField]; protected && !r.FieldEqual(refreshed, ledger.AllocationField) {
			return a.group, nil, ErrConflict
		}
		*r = refreshed
		if r.Validate() != nil {
			return a.group, nil, ErrInvalid
		}
		a.group.Members = append(a.group.Members, Member{OperationID: r.OperationID, Revision: r.Revision})
	}
	return a.group, a.result, nil
}
func (a *effectAssignment) carrierRank(r ledger.Revision) int {
	for _, p := range r.Participation.Parts {
		if p.CarrierID == r.OperationID {
			return 0
		}
	}
	if r.Participation.GroupID == "" || r.Participation.State == "retained" {
		return 1
	}
	return 2
}
func (a *effectAssignment) observe(r ledger.Revision) error {
	if a.group.Kind == Payment && len(a.result) > 0 && (r.Type != a.result[0].Type || len(r.Postings) != len(a.result[0].Postings)) {
		return ErrConflict
	}
	next := r.Clone()
	next.Participation = ledger.Participation{GroupID: a.group.ID, Kind: ledger.ParticipationKind(a.group.Kind), State: "linked"}
	for i, p := range r.Postings {
		if !p.MovesMoney() || p.Role != ledger.Principal && p.Role != ledger.Fee {
			return ErrInvalid
		}
		key := componentKey{p.AccountID, p.Role, p.Money.Asset()}
		matches := []string{}
		for _, id := range a.buckets[key] {
			c := a.components[id]
			if p.Role == ledger.Fee && c.carrier.OperationID == r.OperationID && c.position != i {
				continue
			}
			if p.Role == ledger.Fee && c.posting.FeeID != "" && p.FeeID != "" && c.posting.FeeID != p.FeeID {
				continue
			}
			matches = append(matches, id)
		}
		if len(matches) > 1 {
			return ErrConflict
		}
		id := r.OperationID + ":" + strconv.Itoa(i)
		if len(matches) == 1 {
			id = matches[0]
			c := a.components[id]
			if !c.posting.SameMoney(p) || !c.posting.Funding.SameBasis(p.Funding) || c.carrier.OperationID == r.OperationID && c.position != i {
				return ErrConflict
			}

			if r.Origin == "source" {
				if r.State == ledger.Reversed || r.State == ledger.Cancelled {
					c.state = r.State
				} else if c.state != ledger.Reversed && c.state != ledger.Cancelled && r.State == ledger.Posted {
					c.state = ledger.Posted
				}
			}
		} else {
			if a.group.Kind == Payment && len(a.result) > 0 {
				return ErrConflict
			}
			a.components[id] = &assignedComponent{posting: p, carrier: r, position: i, state: r.State}
			a.buckets[key] = append(a.buckets[key], id)
		}
		role := ledger.ContributionRole("payment")
		if p.Role == ledger.Fee {
			role = "fee"
		} else if a.group.Kind != Payment {
			role = "incoming"
			if p.Money.Sign() < 0 {
				role = "outgoing"
			}
		}
		c := a.components[id]
		if err := c.observeProtectedFields(r); err != nil {
			return err
		}
		next.Participation.Parts = append(next.Participation.Parts, ledger.Contribution{ComponentID: id, Position: i, CarrierID: c.carrier.OperationID, CarrierPosition: c.position, Role: role})
	}
	a.result = append(a.result, next)
	return nil
}
func (a *effectAssignment) validatePrincipal(allowPartial bool) error {
	principal := []ledger.Posting{}
	for _, c := range a.components {
		if c.posting.Role == ledger.Principal {
			principal = append(principal, c.posting)
		}
	}
	a.group.State = Linked
	if a.group.Kind == Payment {
		if len(principal) != 1 {
			return ErrConflict
		}
		return nil
	}
	if len(principal) < 2 && allowPartial {
		a.group.State = WaitingSide
		return nil
	}
	if len(principal) != 2 || principal[0].AccountID == principal[1].AccountID || principal[0].Money.Sign()*principal[1].Money.Sign() != -1 {
		return ErrConflict
	}
	if a.group.Kind == Transfer {
		sum, err := principal[0].Money.Add(principal[1].Money)
		if err != nil || sum.Sign() != 0 {
			return ErrConflict
		}
	} else if a.group.Kind == Exchange {
		if principal[0].Money.Asset() == principal[1].Money.Asset() {
			return ErrConflict
		}
	} else {
		return ErrInvalid
	}
	return nil
}

func (c *assignedComponent) observeProtectedFields(r ledger.Revision) error {
	_, legacy := r.Protections[ledger.LegacyField]
	legacy = legacy || r.HumanOverride && len(r.Protections) == 0
	for _, field := range []ledger.Field{ledger.DateField, ledger.PayerField, ledger.MerchantField, ledger.NoteField, ledger.CategoryField, ledger.MerchantIDField, ledger.ReceiptItemsField, ledger.AllocationField} {
		if _, protected := r.Protections[field]; !protected && !legacy {
			continue
		}
		if previous, found := c.protectedFields[field]; found && !previous.FieldEqual(r, field) {
			return ErrConflict
		}
		if c.protectedFields == nil {
			c.protectedFields = map[ledger.Field]ledger.Revision{}
		}
		c.protectedFields[field] = r
	}
	return nil
}
