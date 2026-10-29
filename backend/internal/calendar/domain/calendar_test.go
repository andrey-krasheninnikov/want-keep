package domain_test

import (
	calendar "github.com/pchkauu/want-keep/backend/internal/calendar/domain"
	"testing"
)

func TestCalendarAndUTCInstant(t *testing.T) {
	for _, input := range []string{"2026-02-30", "2026-1-01", "0000-01-01", "2026-01-01Z"} {
		if _, err := calendar.ParseDate(input); err == nil {
			t.Errorf("accepted date %s", input)
		}
	}
	for _, input := range []string{"2026-13", "2026-1", "0000-01"} {
		if _, err := calendar.ParseMonth(input); err == nil {
			t.Errorf("accepted month %s", input)
		}
	}
	for _, input := range []string{"2026-02-30T00:00:00Z", "2026-01-01T00:00:00+03:00", "2026-01-01T00:00:00.1234567891Z", "2026-01-01T00:00:00,1Z"} {
		if _, err := calendar.ParseInstant(input); err == nil {
			t.Errorf("accepted instant %s", input)
		}
	}
	for _, input := range []string{"Local", "", "../UTC", "/etc/localtime", "Mars/Olympus"} {
		if _, err := calendar.ParseTimezone(input); err == nil {
			t.Errorf("accepted zone %s", input)
		}
	}
	instant, err := calendar.ParseInstant("2026-08-31T22:00:00.123456789Z")
	if err != nil {
		t.Fatal(err)
	}
	zone, err := calendar.ParseTimezone("Europe/Moscow")
	if err != nil {
		t.Fatal(err)
	}
	date, err := instant.DateIn(zone)
	if err != nil || date.String() != "2026-09-01" {
		t.Fatalf("local date: %s %v", date.String(), err)
	}
	if _, err := calendar.ParseDate("2024-02-29"); err != nil {
		t.Fatal(err)
	}
	if _, err := (calendar.Instant{}).DateIn(zone); err == nil {
		t.Fatal("uninitialized instant accepted")
	}
	if _, err := date.StartIn(calendar.Timezone{}); err == nil {
		t.Fatal("uninitialized timezone accepted")
	}
	if _, err := (calendar.Date{}).StartIn(zone); err == nil {
		t.Fatal("uninitialized date accepted")
	}

}
