// Package event defines the normalized calendar-event model and the
// transformation from raw decoded events into it.
//
// Events are all-day (see the roadmap's modeling decision): the source's
// inline showtimes are kept verbatim in the title rather than modeled as
// timed events. Dates are Asia/Tokyo calendar days, taken straight from the
// source day-key, so no timezone conversion is needed for all-day values.
package event

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
)

// Date is a calendar date (no time), interpreted in Asia/Tokyo. It serializes
// to and from a "YYYY-MM-DD" JSON string.
type Date struct {
	Year  int
	Month int
	Day   int
}

// String renders the date as YYYY-MM-DD.
func (d Date) String() string { return fmt.Sprintf("%04d-%02d-%02d", d.Year, d.Month, d.Day) }

// ordinal is a sortable key for a date.
func (d Date) ordinal() int { return d.Year*10000 + d.Month*100 + d.Day }

// Compare returns -1, 0, or +1 as d is before, equal to, or after o.
func (d Date) Compare(o Date) int {
	switch a, b := d.ordinal(), o.ordinal(); {
	case a < b:
		return -1
	case a > b:
		return 1
	default:
		return 0
	}
}

// MarshalJSON encodes the date as a "YYYY-MM-DD" string.
func (d Date) MarshalJSON() ([]byte, error) { return json.Marshal(d.String()) }

// UnmarshalJSON decodes a "YYYY-MM-DD" string into the date.
func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}
	parsed, err := ParseDate(s)
	if err != nil {
		return err
	}
	*d = parsed
	return nil
}

// ParseDate parses a "YYYY-MM-DD" string.
func ParseDate(s string) (Date, error) {
	t, err := time.Parse("2006-01-02", s)
	if err != nil {
		return Date{}, fmt.Errorf("invalid date %q: %w", s, err)
	}
	return Date{Year: t.Year(), Month: int(t.Month()), Day: t.Day()}, nil
}

// Event is a normalized, all-day calendar event.
type Event struct {
	UID    string
	Date   Date
	Title  string
	AllDay bool
}

var spaceRun = regexp.MustCompile(`[ \t]+`)

// cleanTitle normalizes source title text: line breaks become spaces, runs of
// ASCII spaces/tabs collapse to one, and surrounding whitespace is trimmed.
// Full-width (U+3000) spaces inside the text are preserved as-is.
func cleanTitle(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	s = strings.ReplaceAll(s, "\n", " ")
	s = spaceRun.ReplaceAllString(s, " ")
	return strings.TrimSpace(s)
}

// Normalize converts raw decoded events into normalized events, sorted by date
// then UID for deterministic output. Entries whose title is empty after
// cleaning are skipped.
func Normalize(raw []decode.RawEvent) []Event {
	events := make([]Event, 0, len(raw))
	for _, r := range raw {
		title := cleanTitle(r.Title)
		if title == "" {
			continue
		}
		events = append(events, Event{
			// The source day-key is stable per event, so keying the UID off it
			// means an edited title updates in place instead of duplicating.
			UID:    r.Key + "@freecalend.com",
			Date:   Date{Year: r.Year, Month: r.Month, Day: r.Day},
			Title:  title,
			AllDay: true,
		})
	}
	sort.Slice(events, func(i, j int) bool {
		if a, b := events[i].Date.ordinal(), events[j].Date.ordinal(); a != b {
			return a < b
		}
		return events[i].UID < events[j].UID
	})
	return events
}
