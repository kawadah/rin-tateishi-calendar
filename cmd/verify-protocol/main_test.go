package main

import "testing"

// two valid single-event day cells
const goodBody = `[
	["set","[\"hda\",\"on\",\"cald-231613-2026-8-8\",[[2026,8,8],\"Event A\"]]"],
	["set","[\"hda\",\"on\",\"cald-231613-2026-8-9\",[[2026,8,9],\"Event B\"]]"]
]`

func TestCheckPasses(t *testing.T) {
	n, err := check([]byte(goodBody), 2)
	if err != nil {
		t.Fatalf("check: %v", err)
	}
	if n != 2 {
		t.Errorf("count = %d, want 2", n)
	}
}

func TestCheckBelowFloor(t *testing.T) {
	if _, err := check([]byte(goodBody), 5); err == nil {
		t.Fatal("expected error when below floor")
	}
}

func TestCheckStructureChanged(t *testing.T) {
	if _, err := check([]byte(`{"not":"an array"}`), 1); err == nil {
		t.Fatal("expected error on malformed response")
	}
}
