package domain

import (
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	household "github.com/pchkauu/want-keep/backend/internal/household/domain"
	money "github.com/pchkauu/want-keep/backend/internal/money/domain"
	reporting "github.com/pchkauu/want-keep/backend/internal/reporting/domain"
)

type Amounts struct{ Owned, Available, Locked, Debt reporting.Amount }

func (v Amounts) Fields() []reporting.Amount {
	return []reporting.Amount{v.Owned, v.Available, v.Locked, v.Debt}
}
func (v Amounts) Validate(asset money.Asset) error {
	for _, a := range v.Fields() {
		if err := a.Validate(); err != nil {
			return err
		}
		if m, ok := a.Value(); ok && m.Asset() != asset {
			return money.ErrAssetMismatch
		}
	}
	return nil
}
func CashAmounts(m money.Money) (Amounts, error) {
	if err := m.Validate(); err != nil {
		return Amounts{}, err
	}
	if m.Sign() < 0 {
		return Amounts{}, ErrInvalidAccount
	}
	value, err := reporting.KnownAmount(m)
	if err != nil {
		return Amounts{}, err
	}
	zero, _ := money.NewMoney("0", m.Asset())
	z, _ := reporting.KnownAmount(zero)
	return Amounts{value, value, z, z}, nil
}
func UnknownAmounts(reason string) Amounts {
	a, _ := reporting.MissingAmount(reporting.Unknown, reason)
	return Amounts{a, a, a, a}
}

type Opening struct {
	AccountID, OperationID string
	Revision               uint64
	Date                   calendar.Date
	Timezone               calendar.Timezone
	Confirmed              bool
	Amounts                Amounts
	ActorID                household.UserID
	Reason                 string
	At                     calendar.Instant
}

func (o Opening) Validate(asset money.Asset) error {
	if o.AccountID == "" || o.Revision < 1 || o.Revision > 9007199254740991 || o.Date.String() == "" || o.ActorID == "" || len(o.Reason) < 1 || len(o.Reason) > 2000 || o.At.String() == "" {
		return ErrInvalidAccount
	}
	if _, err := calendar.ParseTimezone(o.Timezone.String()); err != nil {
		return err
	}
	start, err := o.Instant()
	if err != nil {
		return err
	}
	if start.Time().After(o.At.Time()) {
		return calendar.ErrInvalidTime
	}
	if err := o.Amounts.Validate(asset); err != nil {
		return err
	}
	if o.Confirmed {
		if _, ok := o.Amounts.Owned.Value(); !ok {
			return ErrInvalidAccount
		}
		if o.OperationID == "" {
			return ErrInvalidAccount
		}
	}
	for _, v := range []reporting.Amount{o.Amounts.Available, o.Amounts.Locked, o.Amounts.Debt} {
		if m, ok := v.Value(); ok && m.Sign() < 0 {
			return ErrInvalidAccount
		}
	}
	return nil
}
func (o Opening) Instant() (calendar.Instant, error) {
	return o.Date.StartIn(o.Timezone)
}

type Effect struct {
	OperationID string
	Revision    uint64
	At          calendar.Instant
	Amount      money.Money
	Changes     *Amounts
}

// Projection uses current monetary effects, including pending holds, without source observations.
func (o Opening) Project(asset money.Asset, effects []Effect) (Amounts, error) {
	if err := o.Validate(asset); err != nil {
		return Amounts{}, err
	}
	if !o.Confirmed {
		return UnknownAmounts("opening_unconfirmed"), nil
	}
	start, err := o.Instant()
	if err != nil {
		return Amounts{}, err
	}
	result := o.Amounts
	for _, e := range effects {
		if e.At.String() == "" {
			return Amounts{}, ErrInvalidAccount
		}
		if e.At.Time().Before(start.Time()) {
			continue
		}
		changes := e.Changes
		if changes == nil {
			if err := e.Amount.Validate(); err != nil {
				return Amounts{}, err
			}
			value, _ := reporting.KnownAmount(e.Amount)
			zero, _ := money.NewMoney("0", asset)
			z, _ := reporting.KnownAmount(zero)
			changes = &Amounts{value, value, z, z}
		}
		if err := changes.Validate(asset); err != nil {
			return Amounts{}, err
		}
		for i, field := range []*reporting.Amount{&result.Owned, &result.Available, &result.Locked, &result.Debt} {
			delta := changes.Fields()[i]
			change, known := delta.Value()
			if !known {
				*field = delta
				continue
			}
			current, known := field.Value()
			if !known {
				continue
			}
			next, err := current.Add(change)
			if err != nil {
				return Amounts{}, err
			}
			*field, err = reporting.KnownAmount(next)
			if err != nil {
				return Amounts{}, err
			}
		}
	}
	// A source contradiction is preserved as uncertainty, never a negative credit debt or hold.
	for _, field := range []*reporting.Amount{&result.Locked, &result.Debt} {
		if value, known := field.Value(); known && value.Sign() < 0 {
			*field, _ = reporting.MissingAmount(reporting.Unknown, "balance_effect_conflict")
		}
	}
	return result, nil
}
