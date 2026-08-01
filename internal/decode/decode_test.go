package decode

import (
	"os"
	"path/filepath"
	"testing"
)

func TestEventsFixture(t *testing.T) {
	body, err := os.ReadFile(filepath.Join("testdata", "response_2026-08.json"))
	if err != nil {
		t.Fatal(err)
	}
	events, err := Events(body)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("got %d events, want 3", len(events))
	}

	byDay := make(map[int]RawEvent, len(events))
	for _, e := range events {
		if e.Year != 2026 || e.Month != 8 {
			t.Errorf("unexpected month for %+v", e)
		}
		byDay[e.Day] = e
	}

	want := map[int]string{
		8:  "「迷子集会」出張版 第三弾 @鴻巣市文化センター 大ホール",
		9:  "🎤LuckyFes'26",
		22: "『リアルモニタ凛グルーム #02』@シアターマーキュリー新宿",
	}
	for day, title := range want {
		got, ok := byDay[day]
		if !ok {
			t.Errorf("missing event for day %d", day)
			continue
		}
		if got.Title != title {
			t.Errorf("day %d title = %q, want %q", day, got.Title, title)
		}
	}
}

func TestEventsIgnoresEmptyDays(t *testing.T) {
	// "get"/noexs ops and a non-day "set" must not produce events.
	body := []byte(`[
		["get","cald-231613-2026-8-1","1602144014","noexs"],
		["set","[\"hda\",\"on\",\"cald-231613-2026-8\",[[2026,8],\"month-agg\"]]"]
	]`)
	events, err := Events(body)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestEventsRejectsNonArray(t *testing.T) {
	if _, err := Events([]byte(`{"nope":true}`)); err == nil {
		t.Fatal("expected error for non-array response")
	}
}

func TestEventsSkipsShortMonthAggregate(t *testing.T) {
	// A 3-element month-level aggregate must be skipped, not errored.
	body := []byte(`[["set","[\"hda\",\"on\",\"cald-231613-2026-8\"]"]]`)
	events, err := Events(body)
	if err != nil {
		t.Fatalf("Events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("got %d events, want 0", len(events))
	}
}

func TestEventsErrorsOnMalformedDayEvent(t *testing.T) {
	// A day key with a non-string title is real structural breakage → error.
	body := []byte(`[["set","[\"hda\",\"on\",\"cald-231613-2026-8-8\",[[2026,8,8],123]]"]]`)
	if _, err := Events(body); err == nil {
		t.Fatal("expected error for malformed day-event title")
	}
}

func TestEventsErrorsOnDayEventMissingPayload(t *testing.T) {
	// A day key with fewer than 4 inner elements is a malformed day event.
	body := []byte(`[["set","[\"hda\",\"on\",\"cald-231613-2026-8-8\"]"]]`)
	if _, err := Events(body); err == nil {
		t.Fatal("expected error for day event missing payload")
	}
}
