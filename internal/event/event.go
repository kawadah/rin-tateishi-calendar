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

var (
	spaceRun  = regexp.MustCompile(`[ \t]+`)
	blankLine = regexp.MustCompile("\n[ \t　]*\n+")
)

// splitTitle splits a day cell's text into one event per blank-line-separated
// block. A source day cell can hold several events the owner separated with a
// blank line; a lone line break within a block is treated as a soft wrap and
// collapsed to a space. Full-width (U+3000) spaces inside text are preserved;
// blank/whitespace-only blocks are dropped.
func splitTitle(s string) []string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	var out []string
	for _, block := range blankLine.Split(s, -1) {
		block = strings.ReplaceAll(block, "\n", " ")
		block = spaceRun.ReplaceAllString(block, " ")
		if block = strings.TrimSpace(block); block != "" {
			out = append(out, block)
		}
	}
	return out
}

// Normalize converts raw decoded events into normalized, all-day events,
// sorted by date then UID. A day cell with multiple blank-line-separated
// events yields one Event per block, its UID suffixed "#<index>".
func Normalize(raw []decode.RawEvent) []Event {
	var events []Event
	for _, r := range raw {
		date := Date{Year: r.Year, Month: r.Month, Day: r.Day}
		for i, title := range splitTitle(r.Title) {
			events = append(events, Event{
				UID:    fmt.Sprintf("%s#%d@freecalend.com", r.Key, i),
				Date:   date,
				Title:  title,
				AllDay: true,
			})
		}
	}
	sort.Slice(events, func(i, j int) bool {
		if a, b := events[i].Date.ordinal(), events[j].Date.ordinal(); a != b {
			return a < b
		}
		return events[i].UID < events[j].UID
	})
	return events
}
