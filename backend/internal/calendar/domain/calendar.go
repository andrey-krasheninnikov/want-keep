package domain

import (
	"errors"
	"regexp"
	"strings"
	"time"
	_ "time/tzdata"
)

var ErrInvalidTime = errors.New("invalid time")
var instantSyntax = regexp.MustCompile(`^[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}(\.[0-9]{1,9})?Z$`)

type Instant struct {
	value time.Time
	valid bool
}
type Date struct{ value string }
type Month struct{ value string }
type Timezone struct{ value string }

func ParseInstant(value string) (Instant, error) {
	if !instantSyntax.MatchString(value) {
		return Instant{}, ErrInvalidTime
	}
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil || parsed.Year() < 1 {
		return Instant{}, ErrInvalidTime
	}
	return Instant{value: parsed.UTC(), valid: true}, nil
}

func (i Instant) String() string {
	if !i.valid {
		return ""
	}
	return i.value.Format(time.RFC3339Nano)
}
func (i Instant) Time() time.Time { return i.value }

func ParseDate(value string) (Date, error) {
	parsed, err := time.Parse(time.DateOnly, value)
	if err != nil || parsed.Year() < 1 || parsed.Format(time.DateOnly) != value {
		return Date{}, ErrInvalidTime
	}
	return Date{value: value}, nil
}
func (d Date) String() string { return d.value }

func ParseMonth(value string) (Month, error) {
	parsed, err := time.Parse("2006-01", value)
	if err != nil || parsed.Year() < 1 || parsed.Format("2006-01") != value {
		return Month{}, ErrInvalidTime
	}
	return Month{value: value}, nil
}
func (m Month) String() string { return m.value }

func ParseTimezone(value string) (Timezone, error) {
	if value != "UTC" && (!strings.Contains(value, "/") || strings.HasPrefix(value, "/") || strings.Contains(value, "..")) {
		return Timezone{}, ErrInvalidTime
	}
	if _, err := time.LoadLocation(value); err != nil {
		return Timezone{}, ErrInvalidTime
	}
	return Timezone{value: value}, nil
}
func (z Timezone) String() string { return z.value }

func (i Instant) DateIn(zone Timezone) (Date, error) {
	if !i.valid {
		return Date{}, ErrInvalidTime
	}
	validated, err := ParseTimezone(zone.value)
	if err != nil {
		return Date{}, err
	}
	location, err := time.LoadLocation(validated.value)
	if err != nil {
		return Date{}, ErrInvalidTime
	}
	return ParseDate(i.value.In(location).Format(time.DateOnly))
}

// StartIn returns the earliest existing instant of the local calendar date.
func (d Date) StartIn(timezone Timezone) (Instant, error) {
	validated, err := ParseTimezone(timezone.String())
	if err != nil {
		return Instant{}, err
	}
	zone, err := time.LoadLocation(validated.String())
	if err != nil {
		return Instant{}, ErrInvalidTime
	}
	date, err := time.Parse(time.DateOnly, d.String())
	if err != nil {
		return Instant{}, ErrInvalidTime
	}
	// Visit UTC offset intervals in order: a midnight gap begins at the transition,
	// while a repeated midnight uses its first occurrence. A skipped whole date has no candidate.
	end := date.Add(72 * time.Hour)
	for start := date.Add(-48 * time.Hour); start.Before(end); {
		local := start.In(zone)
		_, offset := local.Zone()
		_, bound := local.ZoneBounds()
		if bound.IsZero() || bound.After(end) {
			bound = end
		}
		candidate := date.Add(-time.Duration(offset) * time.Second)
		if candidate.Before(start) {
			candidate = start
		}
		if candidate.Before(bound) && candidate.In(zone).Format(time.DateOnly) == d.String() {
			return ParseInstant(candidate.UTC().Format(time.RFC3339Nano))
		}
		start = bound
	}
	return Instant{}, ErrInvalidTime
}
