package event

import (
	"testing"

	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
)

func TestSplitTitle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want []string
	}{
		{"trailing space", "🎤LuckyFes'26 ", []string{"🎤LuckyFes'26"}},
		{"soft wrap stays one", "ねお・りーでぃんぐ\n「スライム」", []string{"ねお・りーでぃんぐ 「スライム」"}},
		{"blank line splits", "イベントA\n\n▶️配信B", []string{"イベントA", "▶️配信B"}},
		{"multiple blank lines", "A\n\n\nB", []string{"A", "B"}},
		{"blank line with spaces", "A\n  \nB", []string{"A", "B"}},
		{"crlf", "a\r\nb", []string{"a b"}},
		{"collapse ascii spaces", "a   b", []string{"a b"}},
		{"preserve fullwidth space", "a　b", []string{"a　b"}},
		{"only whitespace", "  \n\t ", nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := splitTitle(tt.in)
			if len(got) != len(tt.want) {
				t.Fatalf("splitTitle(%q) = %#v, want %#v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("block %d = %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	raw := []decode.RawEvent{
		{Year: 2026, Month: 8, Day: 22, Title: "後の予定", Key: "cald-231613-2026-8-22"},
		{Year: 2026, Month: 8, Day: 9, Title: "🎤LuckyFes'26 ", Key: "cald-231613-2026-8-9"},
		{Year: 2026, Month: 8, Day: 9, Title: "   ", Key: "cald-231613-2026-8-9x"}, // no events
	}
	got := Normalize(raw)

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 (empty cell skipped)", len(got))
	}
	// Sorted by date: day 9 before day 22.
	if got[0].Date.Day != 9 || got[1].Date.Day != 22 {
		t.Fatalf("not sorted by date: %v, %v", got[0].Date, got[1].Date)
	}
	first := got[0]
	if first.Title != "🎤LuckyFes'26" {
		t.Errorf("title = %q, want cleaned 🎤LuckyFes'26", first.Title)
	}
	if first.UID != "cald-231613-2026-8-9#0@freecalend.com" {
		t.Errorf("uid = %q, want ...#0@freecalend.com", first.UID)
	}
	if !first.AllDay || first.Date.String() != "2026-08-09" {
		t.Errorf("unexpected %+v", first)
	}
}

func TestNormalizeMultiEvent(t *testing.T) {
	raw := []decode.RawEvent{
		{Year: 2024, Month: 10, Day: 3, Title: "📕Maiden発売\n\n▶️22:00 TVLIVE", Key: "cald-231613-2024-10-3"},
	}
	got := Normalize(raw)
	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 from one multi-event cell", len(got))
	}
	if got[0].UID != "cald-231613-2024-10-3#0@freecalend.com" {
		t.Errorf("first uid = %q", got[0].UID)
	}
	if got[1].UID != "cald-231613-2024-10-3#1@freecalend.com" {
		t.Errorf("second uid = %q", got[1].UID)
	}
	if got[0].Title != "📕Maiden発売" || got[1].Title != "▶️22:00 TVLIVE" {
		t.Errorf("titles = %q, %q", got[0].Title, got[1].Title)
	}
}

func TestSort(t *testing.T) {
	events := []Event{
		{UID: "b@x", Date: Date{Year: 2026, Month: 7, Day: 10}},
		{UID: "z@x", Date: Date{Year: 2025, Month: 12, Day: 31}},
		{UID: "a@x", Date: Date{Year: 2026, Month: 7, Day: 10}},
	}
	got := Sort(events)
	want := []string{"z@x", "a@x", "b@x"} // date first, then UID within a date
	for i, uid := range want {
		if got[i].UID != uid {
			t.Errorf("position %d: got %q, want %q", i, got[i].UID, uid)
		}
	}
}
