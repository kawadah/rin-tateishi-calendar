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
		{UID: "birthday-2027@rin-tateishi-calendar", Date: event.Date{Year: 2027, Month: 7, Day: 10}, Title: "🎂 立石凛の26歳の誕生日", AllDay: true},
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
		// A synthetic birthday renders like any other all-day event; the emoji
		// plus full-width text must survive unfolded.
		"UID:birthday-2027@rin-tateishi-calendar",
		"DTSTART;VALUE=DATE:20270710",
		"DTEND;VALUE=DATE:20270711",
		"SUMMARY:🎂 立石凛の26歳の誕生日",
		"END:VCALENDAR",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q", want)
		}
	}
	if n := strings.Count(out, "BEGIN:VEVENT"); n != len(sample()) {
		t.Errorf("got %d VEVENTs, want %d", n, len(sample()))
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
	if n := len(cal.Events()); n != len(sample()) {
		t.Fatalf("re-parsed %d events, want %d", n, len(sample()))
	}
}
