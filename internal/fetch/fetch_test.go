package fetch

import (
	"encoding/json"
	"testing"
	"time"
)

func TestMonthAdd(t *testing.T) {
	tests := []struct {
		start Month
		n     int
		want  Month
	}{
		{Month{2026, 8}, 0, Month{2026, 8}},
		{Month{2026, 8}, 12, Month{2027, 8}},
		{Month{2026, 12}, 1, Month{2027, 1}},
		{Month{2026, 1}, -1, Month{2025, 12}},
	}
	for _, tt := range tests {
		if got := tt.start.Add(tt.n); got != tt.want {
			t.Errorf("%v.Add(%d) = %v, want %v", tt.start, tt.n, got, tt.want)
		}
	}
}

func TestWindow(t *testing.T) {
	now := time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC)
	from, to := Window(now, 12)
	if from != (Month{2026, 8}) {
		t.Errorf("from = %v, want 2026-08", from)
	}
	if to != (Month{2027, 8}) {
		t.Errorf("to = %v, want 2027-08", to)
	}
}

func TestMonthsBetween(t *testing.T) {
	ms := monthsBetween(Month{2026, 8}, Month{2027, 8})
	if len(ms) != 13 { // inclusive: Aug 2026 .. Aug 2027
		t.Fatalf("got %d months, want 13", len(ms))
	}
	if ms[0] != (Month{2026, 8}) || ms[12] != (Month{2027, 8}) {
		t.Errorf("bounds = %v..%v, want 2026-08..2027-08", ms[0], ms[12])
	}
}

func TestBuildKeys(t *testing.T) {
	keys, err := buildKeys(231613, []Month{{2026, 8}})
	if err != nil {
		t.Fatal(err)
	}
	var payload struct {
		Data [][]any `json:"data"`
	}
	if err := json.Unmarshal([]byte(keys), &payload); err != nil {
		t.Fatalf("keys is not valid JSON: %v", err)
	}
	// 1 month key + 31 day keys.
	if len(payload.Data) != 32 {
		t.Fatalf("got %d entries, want 32", len(payload.Data))
	}
	first := payload.Data[0]
	if got := first[0].(string); got != "cald-231613-2026-8" {
		t.Errorf("month key = %q, want cald-231613-2026-8", got)
	}
	if got := first[1].(string); got != staleTimestamp {
		t.Errorf("timestamp = %q, want %q", got, staleTimestamp)
	}
	if got := payload.Data[1][0].(string); got != "cald-231613-2026-8-1" {
		t.Errorf("first day key = %q, want cald-231613-2026-8-1", got)
	}
}
