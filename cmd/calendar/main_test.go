package main

import (
	"strings"
	"testing"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/config"
	"github.com/kawadah/rin-tateishi-calendar/internal/event"
)

// live is a stand-in for a fetch result: two events, deliberately unsorted so
// the tests can tell composition from ordering.
func live() []event.Event {
	return []event.Event{
		{UID: "cald-231613-2026-9-1#0@freecalend.com", Date: event.Date{Year: 2026, Month: 9, Day: 1}, Title: "B", AllDay: true},
		{UID: "cald-231613-2026-8-1#0@freecalend.com", Date: event.Date{Year: 2026, Month: 8, Day: 1}, Title: "A", AllDay: true},
	}
}

func jst(t *testing.T) *time.Location {
	t.Helper()
	loc, err := time.LoadLocation("Asia/Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	return loc
}

func birthdayUIDs(events []event.Event) []string {
	var uids []string
	for _, e := range events {
		if strings.HasPrefix(e.UID, "birthday-") {
			uids = append(uids, e.UID)
		}
	}
	return uids
}

func TestFeedEventsAddsBothYears(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, jst(t))
	feed := feedEvents(config.Default(), live(), now)

	got := birthdayUIDs(feed)
	want := []string{"birthday-2026@rin-tateishi-calendar", "birthday-2027@rin-tateishi-calendar"}
	if len(got) != len(want) {
		t.Fatalf("got %d birthdays %v, want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("birthday %d: got %q, want %q", i, got[i], want[i])
		}
	}
	if len(feed) != len(live())+len(want) {
		t.Errorf("feed holds %d events, want %d live + %d birthdays", len(feed), len(live()), len(want))
	}
}

// The archive is written from the live slice before the feed is composed, so
// composing must not touch it. If this fails, synthetic records reach data/,
// where the delete rule then removes them again on every subsequent run.
func TestFeedEventsLeavesLiveUntouched(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, jst(t))
	events := live()
	feedEvents(config.Default(), events, now)

	if uids := birthdayUIDs(events); uids != nil {
		t.Errorf("live set gained synthetic events %v — these would be archived", uids)
	}
	for i, want := range live() {
		if events[i] != want {
			t.Errorf("live event %d was modified:\n got %+v\nwant %+v", i, events[i], want)
		}
	}
}

func TestFeedEventsSorted(t *testing.T) {
	now := time.Date(2026, 8, 18, 12, 0, 0, 0, jst(t))
	feed := feedEvents(config.Default(), live(), now)
	for i := 1; i < len(feed); i++ {
		if feed[i-1].Date.Compare(feed[i].Date) > 0 {
			t.Errorf("feed not sorted: %s before %s", feed[i-1].Date, feed[i].Date)
		}
	}
}

// The year pair comes from the JST clock, not the UTC runner clock: a run in
// the last hours of a UTC year is already in the next JST year.
func TestFeedEventsUsesJSTYear(t *testing.T) {
	utcNewYearsEve := time.Date(2026, 12, 31, 23, 0, 0, 0, time.UTC)
	now := utcNewYearsEve.In(jst(t)) // 2027-01-01 08:00 JST
	if now.Year() != 2027 {
		t.Fatalf("test setup: JST year is %d, want 2027", now.Year())
	}

	got := birthdayUIDs(feedEvents(config.Default(), live(), now))
	want := []string{"birthday-2027@rin-tateishi-calendar", "birthday-2028@rin-tateishi-calendar"}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("birthday %d: got %q, want %q", i, got[i], want[i])
		}
	}
}
