// Package ics renders the published event set as an RFC 5545 iCalendar feed.
//
// All events are all-day (DATE values), so no VTIMEZONE is needed — the dates
// are already JST calendar days. DTSTAMP is pinned to a fixed epoch so that an
// unchanged event set serializes to byte-identical output (a real "now" would
// change every run and defeat skip-on-unchanged).
package ics

import (
	"time"

	goics "github.com/arran4/golang-ical"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
)

// Metadata is the calendar-level information for the feed.
type Metadata struct {
	Name        string
	Description string
	ProductID   string
	Timezone    string
	RefreshTTL  string
}

// dtStamp is a fixed timestamp for DTSTAMP, keeping output deterministic.
var dtStamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// Build renders the given events as an iCalendar document. Events are
// all-day and expected already sorted (see event.Sort).
func Build(events []event.Event, meta Metadata) string {
	cal := goics.NewCalendar()
	cal.SetMethod(goics.MethodPublish)
	cal.SetProductId(meta.ProductID)
	cal.SetName(meta.Name)
	cal.SetXWRCalName(meta.Name)
	cal.SetDescription(meta.Description)
	cal.SetXWRCalDesc(meta.Description)
	cal.SetXWRTimezone(meta.Timezone)
	cal.SetRefreshInterval(meta.RefreshTTL)
	cal.SetXPublishedTTL(meta.RefreshTTL)

	for _, e := range events {
		start := time.Date(e.Date.Year, time.Month(e.Date.Month), e.Date.Day, 0, 0, 0, 0, time.UTC)
		ve := cal.AddEvent(e.UID)
		ve.SetDtStampTime(dtStamp)
		ve.SetAllDayStartAt(start)
		ve.SetAllDayEndAt(start.AddDate(0, 0, 1)) // DTEND is exclusive
		ve.SetSummary(e.Title)
	}
	return cal.Serialize()
}
