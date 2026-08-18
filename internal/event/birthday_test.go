package event

import "testing"

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
		got := Birthdays(testBirth, "立石凛", tt.year)
		if len(got) != 1 {
			t.Fatalf("year %d: got %d events, want 1", tt.year, len(got))
		}
		if got[0].Title != tt.want {
			t.Errorf("year %d: got %q, want %q", tt.year, got[0].Title, tt.want)
		}
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
	if got := Birthdays(testBirth, "立石凛"); got != nil {
		t.Errorf("got %v, want nil for no years", got)
	}
}

// A birthday UID must never look like a fetched event's UID, or the two
// namespaces could collide in the feed.
func TestBirthdayUIDNamespace(t *testing.T) {
	got := Birthdays(testBirth, "立石凛", 2026)[0].UID
	if want := "birthday-2026@rin-tateishi-calendar"; got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}
