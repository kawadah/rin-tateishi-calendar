package event

import (
	"testing"

	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
)

func TestCleanTitle(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trailing space", "🎤LuckyFes'26 ", "🎤LuckyFes'26"},
		{"embedded newline", "ねお・りーでぃんぐ\n「スライム」", "ねお・りーでぃんぐ 「スライム」"},
		{"crlf", "a\r\nb", "a b"},
		{"collapse ascii spaces", "a   b", "a b"},
		{"preserve fullwidth space", "a　b", "a　b"},
		{"only whitespace", "  \n\t ", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := cleanTitle(tt.in); got != tt.want {
				t.Errorf("cleanTitle(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNormalize(t *testing.T) {
	raw := []decode.RawEvent{
		{Year: 2026, Month: 8, Day: 22, Title: "後の予定", Key: "cald-231613-2026-8-22"},
		{Year: 2026, Month: 8, Day: 9, Title: "🎤LuckyFes'26 ", Key: "cald-231613-2026-8-9"},
		{Year: 2026, Month: 8, Day: 9, Title: "   ", Key: "cald-231613-2026-8-9-empty"}, // skipped
	}
	got := Normalize(raw)

	if len(got) != 2 {
		t.Fatalf("got %d events, want 2 (empty title skipped)", len(got))
	}
	// Sorted by date: day 9 before day 22.
	if got[0].Date.Day != 9 || got[1].Date.Day != 22 {
		t.Fatalf("not sorted by date: %v, %v", got[0].Date, got[1].Date)
	}
	first := got[0]
	if first.Title != "🎤LuckyFes'26" {
		t.Errorf("title = %q, want cleaned 🎤LuckyFes'26", first.Title)
	}
	if first.UID != "cald-231613-2026-8-9@freecalend.com" {
		t.Errorf("uid = %q", first.UID)
	}
	if !first.AllDay {
		t.Error("AllDay should be true")
	}
	if first.Date.String() != "2026-08-09" {
		t.Errorf("date = %q, want 2026-08-09", first.Date.String())
	}
}
