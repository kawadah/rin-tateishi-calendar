package archive

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
)

// Fixed fetch window and run date for merge tests: window 2026-08 .. 2027-08
// (windowStart 2026-08-01, windowEnd exclusive 2027-09-01); today 2026-08-15.
var (
	winFrom = fetch.Month{Year: 2026, Month: 8}
	winTo   = fetch.Month{Year: 2027, Month: 8}
	today   = event.Date{Year: 2026, Month: 8, Day: 15}
)

func ev(y, m, d int, uid, title string) event.Event {
	return event.Event{UID: uid, Date: event.Date{Year: y, Month: m, Day: d}, Title: title, AllDay: true}
}

func TestMergeUpsertAndEdit(t *testing.T) {
	arc := map[string]Record{
		"u9": {UID: "u9", Date: event.Date{Year: 2026, Month: 8, Day: 9}, Title: "old title", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Merge(arc, []event.Event{ev(2026, 8, 9, "u9", "new title")}, winFrom, winTo, today)

	got, ok := arc["u9"]
	if !ok {
		t.Fatal("u9 missing after merge")
	}
	if got.Title != "new title" {
		t.Errorf("title = %q, want overwritten 'new title'", got.Title)
	}
	if got.FirstSeen != (event.Date{Year: 2026, Month: 8, Day: 1}) {
		t.Errorf("first_seen = %v, want preserved 2026-08-01", got.FirstSeen)
	}
}

func TestMergeAddsNewWithFirstSeenToday(t *testing.T) {
	arc := map[string]Record{}
	Merge(arc, []event.Event{ev(2026, 9, 2, "u", "e")}, winFrom, winTo, today)
	if arc["u"].FirstSeen != (event.Date{Year: 2026, Month: 8, Day: 15}) {
		t.Errorf("first_seen = %v, want today 2026-08-15", arc["u"].FirstSeen)
	}
}

func TestMergeDeletesInWindowAbsentee(t *testing.T) {
	// Dated 2026-09-01 (>= window start 2026-08-01) and absent from the fetch.
	arc := map[string]Record{
		"gone": {UID: "gone", Date: event.Date{Year: 2026, Month: 9, Day: 1}, Title: "cancelled", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Merge(arc, nil, winFrom, winTo, today)
	if _, ok := arc["gone"]; ok {
		t.Error("in-window absentee should be hard-deleted")
	}
}

func TestMergeKeepsOutOfWindowAbsentee(t *testing.T) {
	// Dated 2026-07-31 (< window start 2026-08-01): absence proves nothing.
	arc := map[string]Record{
		"past": {UID: "past", Date: event.Date{Year: 2026, Month: 7, Day: 31}, Title: "happened", FirstSeen: event.Date{Year: 2026, Month: 7, Day: 1}},
	}
	Merge(arc, nil, winFrom, winTo, today)
	if _, ok := arc["past"]; !ok {
		t.Error("out-of-window absentee should be kept frozen")
	}
}

func TestBackfillNeverDeletes(t *testing.T) {
	// An in-window record absent from the backfill fetch must survive.
	arc := map[string]Record{
		"live": {UID: "live", Date: event.Date{Year: 2026, Month: 9, Day: 1}, Title: "upcoming", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Backfill(arc, []event.Event{ev(2024, 6, 1, "old", "back-catalog")}, today)
	if _, ok := arc["live"]; !ok {
		t.Error("backfill must not delete existing records")
	}
	if _, ok := arc["old"]; !ok {
		t.Error("backfill should add the historical event")
	}
}

func TestMergeKeepsAfterWindowAbsentee(t *testing.T) {
	// Dated 2027-10-01, past the window end (2027-09-01 exclusive): absence must
	// not be treated as a deletion (guards a shortened window / clock skew).
	arc := map[string]Record{
		"future": {UID: "future", Date: event.Date{Year: 2027, Month: 10, Day: 1}, Title: "far ahead", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Merge(arc, nil, winFrom, winTo, today)
	if _, ok := arc["future"]; !ok {
		t.Error("after-window absentee should be kept, not deleted")
	}
}

func TestMergeUpsertAndDeleteInSameCall(t *testing.T) {
	// One in-window record survives (present in fetch, edited) while another
	// in-window record is deleted (absent) — both in a single Merge.
	arc := map[string]Record{
		"keep": {UID: "keep", Date: event.Date{Year: 2026, Month: 9, Day: 1}, Title: "old", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
		"gone": {UID: "gone", Date: event.Date{Year: 2026, Month: 9, Day: 2}, Title: "cancelled", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Merge(arc, []event.Event{ev(2026, 9, 1, "keep", "edited")}, winFrom, winTo, today)

	if got, ok := arc["keep"]; !ok || got.Title != "edited" {
		t.Errorf("keep = %+v, ok=%v; want title 'edited'", got, ok)
	}
	if _, ok := arc["gone"]; ok {
		t.Error("gone should be deleted")
	}
}

func TestStoreRoundTripAndDeterminism(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)

	arc := map[string]Record{
		"a": {UID: "a", Date: event.Date{Year: 2026, Month: 8, Day: 22}, Title: "later", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
		"b": {UID: "b", Date: event.Date{Year: 2026, Month: 8, Day: 9}, Title: "earlier", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
		"c": {UID: "c", Date: event.Date{Year: 2026, Month: 9, Day: 2}, Title: "next month", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	if err := store.Save(arc); err != nil {
		t.Fatal(err)
	}

	// Two month files: 2026-08 (a,b) and 2026-09 (c).
	if _, err := os.Stat(filepath.Join(dir, "2026-08.json")); err != nil {
		t.Errorf("2026-08.json missing: %v", err)
	}
	aug1, err := os.ReadFile(filepath.Join(dir, "2026-08.json"))
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load()
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded) != 3 {
		t.Fatalf("loaded %d records, want 3", len(loaded))
	}
	if loaded["b"].Title != "earlier" {
		t.Errorf("round-trip lost data: %+v", loaded["b"])
	}

	// Re-saving identical data yields byte-identical files (deterministic).
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	aug2, err := os.ReadFile(filepath.Join(dir, "2026-08.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(aug1) != string(aug2) {
		t.Error("serialization is not deterministic across saves")
	}
}

func TestSaveRemovesEmptyMonthFile(t *testing.T) {
	dir := t.TempDir()
	store := NewStore(dir)
	arc := map[string]Record{
		"a": {UID: "a", Date: event.Date{Year: 2026, Month: 8, Day: 9}, Title: "x", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	if err := store.Save(arc); err != nil {
		t.Fatal(err)
	}
	// Remove the only record and re-save: the month file should disappear.
	delete(arc, "a")
	if err := store.Save(arc); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "2026-08.json")); !os.IsNotExist(err) {
		t.Error("empty month file should be removed")
	}
}
