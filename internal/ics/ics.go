// Package ics renders the live event set as an RFC 5545 iCalendar feed.
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

const (
	calName    = "立石凛 スケジュール"
	calDesc    = "立石凛さんの公式カレンダー（freecalend）の非公式ミラー"
	productID  = "-//rin-tateishi-calendar//freecalend mirror//JA"
	timezone   = "Asia/Tokyo"
	refreshTTL = "PT6H"
)

// dtStamp is a fixed timestamp for DTSTAMP, keeping output deterministic.
var dtStamp = time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)

// Build renders the given live events as an iCalendar document. Events are
// all-day and expected already sorted (event.Normalize output).
func Build(events []event.Event) string {
	cal := goics.NewCalendar()
	cal.SetMethod(goics.MethodPublish)
	cal.SetProductId(productID)
	cal.SetName(calName)
	cal.SetXWRCalName(calName)
	cal.SetDescription(calDesc)
	cal.SetXWRCalDesc(calDesc)
	cal.SetXWRTimezone(timezone)
	cal.SetRefreshInterval(refreshTTL)
	cal.SetXPublishedTTL(refreshTTL)

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
