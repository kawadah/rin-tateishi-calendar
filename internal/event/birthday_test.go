package event

import (
	"strconv"
	"strings"
	"testing"

	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
)

var testBirth = Date{Year: 2001, Month: 7, Day: 10}

func TestBirthdays(t *testing.T) {
	got := Birthdays(testBirth, "立石凛", 2026, 2027)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2", len(got))
	}
	want := []Event{
		{
			UID:    "birthday-2026@rin-tateishi-calendar",
			Date:   Date{Year: 2026, Month: 7, Day: 10},
			Title:  "🎂 立石凛の25歳の誕生日",
			AllDay: true,
		},
		{
			UID:    "birthday-2027@rin-tateishi-calendar",
			Date:   Date{Year: 2027, Month: 7, Day: 10},
			Title:  "🎂 立石凛の26歳の誕生日",
			AllDay: true,
		},
	}
	for i, w := range want {
		if got[i] != w {
			t.Errorf("event %d:\n got %+v\nwant %+v", i, got[i], w)
		}
	}
}

func TestBirthdaysAge(t *testing.T) {
	tests := []struct {
		year int
		want string
	}{
		{2001, "🎂 立石凛の0歳の誕生日"}, // the birth year itself
		{2026, "🎂 立石凛の25歳の誕生日"},
		{2100, "🎂 立石凛の99歳の誕生日"},
	}
	for _, tt := range tests {
		t.Run(strconv.Itoa(tt.year), func(t *testing.T) {
			got := Birthdays(testBirth, "立石凛", tt.year)
			if len(got) != 1 {
				t.Fatalf("got %d events, want 1", len(got))
			}
			if got[0].Title != tt.want {
				t.Errorf("got %q, want %q", got[0].Title, tt.want)
			}
		})
	}
}

func TestBirthdaysSkipsYearsBeforeBirth(t *testing.T) {
	got := Birthdays(testBirth, "立石凛", 1999, 2000, 2001)
	if len(got) != 1 {
		t.Fatalf("got %d events, want 1 (only the birth year)", len(got))
	}
	if got[0].Date.Year != 2001 {
		t.Errorf("got year %d, want 2001", got[0].Date.Year)
	}
}

func TestBirthdaysSorted(t *testing.T) {
	got := Birthdays(testBirth, "立石凛", 2028, 2026, 2027)
	for i, want := range []int{2026, 2027, 2028} {
		if got[i].Date.Year != want {
			t.Errorf("position %d: got year %d, want %d", i, got[i].Date.Year, want)
		}
	}
}

func TestBirthdaysNone(t *testing.T) {
	if got := Birthdays(testBirth, "立石凛"); len(got) != 0 {
		t.Errorf("got %v, want no events for no years", got)
	}
}

// A birthday UID must never collide with a fetched event's, so it is compared
// against a UID the source path actually produces rather than a literal.
func TestBirthdayUIDNamespace(t *testing.T) {
	fetched := Normalize([]decode.RawEvent{
		{Key: "cald-231613-2026-7-10", Year: 2026, Month: 7, Day: 10, Title: "イベント"},
	})
	if len(fetched) != 1 {
		t.Fatalf("test setup: normalized %d events, want 1", len(fetched))
	}
	synthetic := Birthdays(testBirth, "立石凛", 2026)[0]

	if synthetic.UID == fetched[0].UID {
		t.Fatalf("synthetic UID %q collides with a fetched one", synthetic.UID)
	}
	if strings.HasSuffix(synthetic.UID, "@freecalend.com") {
		t.Errorf("synthetic UID %q is in the source namespace", synthetic.UID)
	}
	// Same date: the two must still be distinguishable by UID alone.
	if synthetic.Date != fetched[0].Date {
		t.Fatalf("test setup: dates differ (%v vs %v)", synthetic.Date, fetched[0].Date)
	}
}
