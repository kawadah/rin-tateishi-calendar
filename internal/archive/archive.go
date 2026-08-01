// Package archive is the durable per-month event store under data/.
//
// It keeps every event observed and preserves records after they roll out of
// the fetch window. On a normal run it upserts the fetched events (edits
// overwrite) and hard-deletes archived events that are inside the fetched
// window [from, to] but absent from the fetch — a genuine owner deletion.
// Records dated outside that window (before or after) have their absence
// treated as inconclusive and are left untouched.
//
// Serialization is deterministic (records sorted by date then UID, stable
// field order, 2-space indent) so an unchanged archive yields no diff. Git
// history is the ultimate backstop for anything removed.
package archive

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
)

// Record is one archived event.
type Record struct {
	UID       string     `json:"uid"`
	Date      event.Date `json:"date"`
	Title     string     `json:"title"`
	FirstSeen event.Date `json:"first_seen"`
}

// Store reads and writes the archive rooted at Dir.
type Store struct {
	Dir string
}

// NewStore returns a Store rooted at dir (e.g. "data").
func NewStore(dir string) *Store { return &Store{Dir: dir} }

// Load reads every data/*.json file into a map keyed by UID. A missing
// directory yields an empty archive.
func (s *Store) Load() (map[string]Record, error) {
	archive := make(map[string]Record)
	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return archive, nil
		}
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" {
			continue
		}
		b, err := os.ReadFile(filepath.Join(s.Dir, e.Name()))
		if err != nil {
			return nil, err
		}
		var recs []Record
		if err := json.Unmarshal(b, &recs); err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		for _, r := range recs {
			archive[r.UID] = r
		}
	}
	return archive, nil
}

// Save writes the archive as one sorted JSON file per month and removes any
// month file that no longer has records.
func (s *Store) Save(archive map[string]Record) error {
	byMonth := make(map[string][]Record)
	for _, r := range archive {
		key := monthKey(r.Date)
		byMonth[key] = append(byMonth[key], r)
	}

	if len(byMonth) > 0 {
		if err := os.MkdirAll(s.Dir, 0o755); err != nil {
			return err
		}
	}

	wanted := make(map[string]bool, len(byMonth))
	for month, recs := range byMonth {
		sortRecords(recs)
		b, err := json.MarshalIndent(recs, "", "  ")
		if err != nil {
			return err
		}
		b = append(b, '\n')
		name := month + ".json"
		if err := os.WriteFile(filepath.Join(s.Dir, name), b, 0o644); err != nil {
			return err
		}
		wanted[name] = true
	}

	entries, err := os.ReadDir(s.Dir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	for _, e := range entries {
		// Only sweep our own month files; never touch hand-added files.
		if e.IsDir() || !monthFilePattern.MatchString(e.Name()) || wanted[e.Name()] {
			continue
		}
		if err := os.Remove(filepath.Join(s.Dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// monthFilePattern matches the archive's own file names (YYYY-MM.json).
var monthFilePattern = regexp.MustCompile(`^\d{4}-\d{2}\.json$`)

// Merge applies a normal run to archive (mutated in place): upsert the fetched
// events (edits overwrite), then hard-delete archived records inside the fetch
// window [from, to] that are absent from the fetch (owner deletions). Records
// outside the window are left untouched — their absence proves nothing. The
// window is passed explicitly (not derived from a clock) so it matches exactly
// what was fetched, and today (JST) is used for FirstSeen.
func Merge(archive map[string]Record, events []event.Event, from, to fetch.Month, today event.Date) {
	upsert(archive, events, today)

	windowStart := event.Date{Year: from.Year, Month: from.Month, Day: 1}
	afterTo := to.Add(1)
	windowEnd := event.Date{Year: afterTo.Year, Month: afterTo.Month, Day: 1} // exclusive
	present := make(map[string]bool, len(events))
	for _, e := range events {
		present[e.UID] = true
	}
	for uid, rec := range archive {
		if present[uid] {
			continue
		}
		if rec.Date.Compare(windowStart) >= 0 && rec.Date.Compare(windowEnd) < 0 {
			delete(archive, uid)
		}
	}
}

// Backfill additively seeds historical events into archive (mutated in place)
// without deleting anything. Used for the one-time back-catalog seed.
func Backfill(archive map[string]Record, events []event.Event, today event.Date) {
	upsert(archive, events, today)
}

// upsert inserts new events and overwrites the title/date of existing ones,
// preserving FirstSeen. New records get FirstSeen = today.
func upsert(archive map[string]Record, events []event.Event, today event.Date) {
	for _, e := range events {
		if rec, ok := archive[e.UID]; ok {
			rec.Title = e.Title
			rec.Date = e.Date
			archive[e.UID] = rec
		} else {
			archive[e.UID] = Record{UID: e.UID, Date: e.Date, Title: e.Title, FirstSeen: today}
		}
	}
}

func sortRecords(recs []Record) {
	sort.Slice(recs, func(i, j int) bool {
		if c := recs[i].Date.Compare(recs[j].Date); c != 0 {
			return c < 0
		}
		return recs[i].UID < recs[j].UID
	})
}

func monthKey(d event.Date) string { return fmt.Sprintf("%04d-%02d", d.Year, d.Month) }
