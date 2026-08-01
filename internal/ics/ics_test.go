package ics

import (
	"strings"
	"testing"

	goics "github.com/arran4/golang-ical"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
)

func sample() []event.Event {
	return []event.Event{
		{UID: "cald-231613-2026-8-9#0@freecalend.com", Date: event.Date{Year: 2026, Month: 8, Day: 9}, Title: "🎤LuckyFes'26", AllDay: true},
		{UID: "cald-231613-2026-8-22#0@freecalend.com", Date: event.Date{Year: 2026, Month: 8, Day: 22}, Title: "『リアルモニタ凛グルーム #02』", AllDay: true},
	}
}

var sampleMeta = Metadata{
	Name:        "立石凛 スケジュール",
	Description: "test",
	ProductID:   "-//test//test//JA",
	Timezone:    "Asia/Tokyo",
	RefreshTTL:  "PT6H",
}

func TestBuildStructure(t *testing.T) {
	out := Build(sample(), sampleMeta)
	for _, want := range []string{
		"BEGIN:VCALENDAR",
		"METHOD:PUBLISH",
		"X-WR-CALNAME:立石凛 スケジュール",
		"X-WR-TIMEZONE:Asia/Tokyo",
		"REFRESH-INTERVAL",
		"UID:cald-231613-2026-8-9#0@freecalend.com",
		"DTSTART;VALUE=DATE:20260809",
		"DTEND;VALUE=DATE:20260810", // exclusive end
		"SUMMARY:🎤LuckyFes'26",
		"END:VCALENDAR",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if n := strings.Count(out, "BEGIN:VEVENT"); n != 2 {
		t.Errorf("got %d VEVENTs, want 2", n)
	}
}

func TestBuildDeterministic(t *testing.T) {
	first := Build(sample(), sampleMeta)
	second := Build(sample(), sampleMeta)
	if first != second {
		t.Error("Build is not deterministic across calls")
	}
}

func TestBuildParsesBack(t *testing.T) {
	out := Build(sample(), sampleMeta)
	cal, err := goics.ParseCalendar(strings.NewReader(out))
	if err != nil {
		t.Fatalf("output is not valid iCalendar: %v", err)
	}
	if n := len(cal.Events()); n != 2 {
		t.Fatalf("re-parsed %d events, want 2", n)
	}
}
