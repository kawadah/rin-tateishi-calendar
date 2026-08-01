// Package archive is the durable per-month event store under data/.
//
// It keeps every event observed and preserves records after they roll out of
// the fetch window. On a normal run it upserts the fetched events (edits
// overwrite) and hard-deletes archived events that are inside the current
// window (date >= 1st of the current month) but absent from the fetch — a
// genuine owner deletion. Records dated before the current month are outside
// the window, so their absence proves nothing and they are left untouched.
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
	"sort"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
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
		if e.IsDir() || filepath.Ext(e.Name()) != ".json" || wanted[e.Name()] {
			continue
		}
		if err := os.Remove(filepath.Join(s.Dir, e.Name())); err != nil {
			return err
		}
	}
	return nil
}

// Merge applies a normal run to archive (mutated in place): upsert the fetched
// events, then hard-delete in-window archived events absent from the fetch.
func Merge(archive map[string]Record, events []event.Event, now time.Time) {
	upsert(archive, events, dateOf(now))

	windowStart := event.Date{Year: now.Year(), Month: int(now.Month()), Day: 1}
	present := make(map[string]bool, len(events))
	for _, e := range events {
		present[e.UID] = true
	}
	for uid, rec := range archive {
		if !present[uid] && rec.Date.Compare(windowStart) >= 0 {
			delete(archive, uid)
		}
	}
}

// Backfill additively seeds historical events into archive (mutated in place)
// without deleting anything. Used for the one-time back-catalog seed.
func Backfill(archive map[string]Record, events []event.Event, now time.Time) {
	upsert(archive, events, dateOf(now))
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

func dateOf(t time.Time) event.Date {
	return event.Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}
}
