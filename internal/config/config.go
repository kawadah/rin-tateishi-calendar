// Package config centralizes the project's tunable settings so they live in
// one place instead of scattered constants. Protocol facts (the endpoint URL,
// the stale-timestamp sentinel) stay in internal/fetch; this is the "what you'd
// change to retarget or tune" layer.
package config

import (
	"fmt"
	"os"
	"strconv"

	"github.com/kawadah/rin-tateishi-calendar/internal/event"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
)

// Config holds all project settings.
type Config struct {
	// Source
	MemberNo int

	// Published calendar metadata
	CalendarName string
	CalendarDesc string
	ProductID    string
	Timezone     string
	RefreshTTL   string

	// Birthday events (synthetic; added to the feed, never archived)
	BirthdayName string     // name used in the event title
	Birthday     event.Date // birth date; the age in the title is derived from it

	// Pipeline
	MonthsAhead int    // live window = [current month, +MonthsAhead]
	DataDir     string // archive root
	ICSPath     string // generated feed

	// Protocol canary
	CanaryFrom      fetch.Month
	CanaryTo        fetch.Month
	CanaryMinEvents int
	// A stable past event that must decode with the expected title. Catches a
	// title-position shift that the bare count check would miss.
	CanaryPinnedDate        event.Date
	CanaryPinnedTitleSubstr string
}

// Default returns the built-in configuration (member 231613, 立石凛).
func Default() Config {
	return Config{
		MemberNo: 231613,

		CalendarName: "立石凛スケジュール",
		CalendarDesc: "立石凛の公式スケジュール（フリカレ）を照会カレンダーとしてミラーしたもの",
		ProductID:    "-//rin-tateishi-calendar//freecalend mirror//JA",
		Timezone:     "Asia/Tokyo",
		RefreshTTL:   "PT6H",

		BirthdayName: "立石凛",
		Birthday:     event.Date{Year: 2001, Month: 7, Day: 10},

		MonthsAhead: 12,
		DataDir:     "data",
		ICSPath:     "dist/calendar.ics",

		CanaryFrom:              fetch.Month{Year: 2024, Month: 7},
		CanaryTo:                fetch.Month{Year: 2025, Month: 6},
		CanaryMinEvents:         30,
		CanaryPinnedDate:        event.Date{Year: 2024, Month: 7, Day: 6},
		CanaryPinnedTitleSubstr: "BCF2024",
	}
}

// FromEnv returns Default with environment overrides applied. Currently only
// SOURCE_MEMBER_NO is overridable; it applies consistently to every command.
func FromEnv() (Config, error) {
	c := Default()
	if v := os.Getenv("SOURCE_MEMBER_NO"); v != "" {
		n, err := strconv.Atoi(v)
		if err != nil {
			return c, fmt.Errorf("invalid SOURCE_MEMBER_NO %q: %w", v, err)
		}
		c.MemberNo = n
	}
	return c, nil
}
