package event

import "fmt"

// birthdayTitleFormat renders a birthday title, e.g. "🎂 立石凛の25歳の誕生日".
const birthdayTitleFormat = "🎂 %sの%d歳の誕生日"

// Birthdays returns the all-day birthday events for the given calendar years,
// sorted. These are synthetic: the source calendar does not carry them, so they
// are generated for the published feed and never archived.
//
// Age is the count of birthdays reached in that year (year - birth year), so a
// 2001 birth date yields 25歳 in 2026. Years before the birth year are skipped
// rather than emitting a negative age.
func Birthdays(birth Date, name string, years ...int) []Event {
	var events []Event
	for _, year := range years {
		if year < birth.Year {
			continue
		}
		events = append(events, Event{
			UID:    birthdayUID(year),
			Date:   Date{Year: year, Month: birth.Month, Day: birth.Day},
			Title:  fmt.Sprintf(birthdayTitleFormat, name, year-birth.Year),
			AllDay: true,
		})
	}
	return Sort(events)
}

// birthdayUID is the stable UID for a year's birthday event. The suffix keeps
// it out of the source's "<key>@freecalend.com" namespace, so a synthetic event
// can never collide with a fetched one.
func birthdayUID(year int) string {
	return fmt.Sprintf("birthday-%04d@rin-tateishi-calendar", year)
}
