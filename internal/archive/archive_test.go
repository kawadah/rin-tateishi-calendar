package archive

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
)

// runNow is a fixed "now" mid-month; window start is 2026-08-01.
var runNow = time.Date(2026, 8, 15, 12, 0, 0, 0, time.UTC)

func ev(y, m, d int, uid, title string) event.Event {
	return event.Event{UID: uid, Date: event.Date{Year: y, Month: m, Day: d}, Title: title, AllDay: true}
}

func TestMergeUpsertAndEdit(t *testing.T) {
	arc := map[string]Record{
		"u9": {UID: "u9", Date: event.Date{Year: 2026, Month: 8, Day: 9}, Title: "old title", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Merge(arc, []event.Event{ev(2026, 8, 9, "u9", "new title")}, runNow)

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
	Merge(arc, []event.Event{ev(2026, 9, 2, "u", "e")}, runNow)
	if arc["u"].FirstSeen != (event.Date{Year: 2026, Month: 8, Day: 15}) {
		t.Errorf("first_seen = %v, want today 2026-08-15", arc["u"].FirstSeen)
	}
}

func TestMergeDeletesInWindowAbsentee(t *testing.T) {
	// Dated 2026-09-01 (>= window start 2026-08-01) and absent from the fetch.
	arc := map[string]Record{
		"gone": {UID: "gone", Date: event.Date{Year: 2026, Month: 9, Day: 1}, Title: "cancelled", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Merge(arc, nil, runNow)
	if _, ok := arc["gone"]; ok {
		t.Error("in-window absentee should be hard-deleted")
	}
}

func TestMergeKeepsOutOfWindowAbsentee(t *testing.T) {
	// Dated 2026-07-31 (< window start 2026-08-01): absence proves nothing.
	arc := map[string]Record{
		"past": {UID: "past", Date: event.Date{Year: 2026, Month: 7, Day: 31}, Title: "happened", FirstSeen: event.Date{Year: 2026, Month: 7, Day: 1}},
	}
	Merge(arc, nil, runNow)
	if _, ok := arc["past"]; !ok {
		t.Error("out-of-window absentee should be kept frozen")
	}
}

func TestBackfillNeverDeletes(t *testing.T) {
	// An in-window record absent from the backfill fetch must survive.
	arc := map[string]Record{
		"live": {UID: "live", Date: event.Date{Year: 2026, Month: 9, Day: 1}, Title: "upcoming", FirstSeen: event.Date{Year: 2026, Month: 8, Day: 1}},
	}
	Backfill(arc, []event.Event{ev(2024, 6, 1, "old", "back-catalog")}, runNow)
	if _, ok := arc["live"]; !ok {
		t.Error("backfill must not delete existing records")
	}
	if _, ok := arc["old"]; !ok {
		t.Error("backfill should add the historical event")
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
