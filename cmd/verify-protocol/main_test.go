package main

import (
	"testing"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
)

// two valid single-event day cells
const goodBody = `[
	["set","[\"hda\",\"on\",\"cald-231613-2026-8-8\",[[2026,8,8],\"Event A\"]]"],
	["set","[\"hda\",\"on\",\"cald-231613-2026-8-9\",[[2026,8,9],\"Event B\"]]"]
]`

var pinnedDate = event.Date{Year: 2026, Month: 8, Day: 8}

func TestCheckPasses(t *testing.T) {
	n, err := check([]byte(goodBody), 2, pinnedDate, "Event A")
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}

func TestCheckBelowFloor(t *testing.T) {
	if _, err := check([]byte(goodBody), 5, pinnedDate, "Event A"); err == nil {
		t.Fatal("expected error when below floor")
	}
}

func TestCheckPinnedEventMissing(t *testing.T) {
	// Right count, but the pinned event's title doesn't match — the title-shift
	// case a bare count check would miss.
	if _, err := check([]byte(goodBody), 2, pinnedDate, "Different Title"); err == nil {
		t.Fatal("expected error when pinned event title doesn't match")
	}
}

func TestCheckStructureChanged(t *testing.T) {
	if _, err := check([]byte(`{"not":"an array"}`), 1, pinnedDate, "Event A"); err == nil {
		t.Fatal("expected error on malformed response")
	}
}
