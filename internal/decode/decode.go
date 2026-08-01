// Package decode turns a raw /open/data response body into raw events.
//
// The response is a JSON array of ops. Each op is a heterogeneous array:
//
//	["get", key, ts, "noexs"]          — an empty day, ignored
//	["set", "<escaped-json-string>"]   — a day with an event
//
// A "set" payload is itself a JSON string that decodes to:
//
//	["hda", "on", key, [dateMeta, title, ...]]
//
// where key is "cald-{memno}-{Y}-{M}-{D}" and title is the event text.
package decode

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

// RawEvent is a single event as decoded from the response, before
// normalization. The date comes from the source day-key.
type RawEvent struct {
	Year  int
	Month int
	Day   int
	Title string
	Key   string // source key, e.g. "cald-231613-2026-8-8"
}

// Events decodes a /open/data response body into raw events. "get" ops (empty
// days) and any "set" op that is not a per-day event are skipped.
func Events(body []byte) ([]RawEvent, error) {
	var ops []json.RawMessage
	if err := json.Unmarshal(body, &ops); err != nil {
		return nil, fmt.Errorf("response is not a JSON array: %w", err)
	}
	var events []RawEvent
	for i, raw := range ops {
		var op []json.RawMessage
		if err := json.Unmarshal(raw, &op); err != nil {
			return nil, fmt.Errorf("op %d is not an array: %w", i, err)
		}
		if len(op) < 2 {
			continue
		}
		var kind string
		if err := json.Unmarshal(op[0], &kind); err != nil || kind != "set" {
			continue
		}
		ev, ok, err := decodeSet(op[1])
		if err != nil {
			return nil, fmt.Errorf("op %d: %w", i, err)
		}
		if ok {
			events = append(events, ev)
		}
	}
	return events, nil
}

// decodeSet parses a "set" op payload. ok is false (without error) when the
// payload is a well-formed "set" that is not a per-day event (e.g. a
// month-level aggregate), which is skipped rather than treated as breakage.
func decodeSet(payload json.RawMessage) (ev RawEvent, ok bool, err error) {
	var innerStr string
	if err := json.Unmarshal(payload, &innerStr); err != nil {
		return RawEvent{}, false, fmt.Errorf("set payload is not a string: %w", err)
	}
	var inner []json.RawMessage
	if err := json.Unmarshal([]byte(innerStr), &inner); err != nil {
		return RawEvent{}, false, fmt.Errorf("set inner is not an array: %w", err)
	}
	if len(inner) < 4 {
		return RawEvent{}, false, fmt.Errorf("set inner too short (%d elements)", len(inner))
	}
	var key string
	if err := json.Unmarshal(inner[2], &key); err != nil {
		return RawEvent{}, false, fmt.Errorf("key is not a string: %w", err)
	}
	year, month, day, isDay := parseDayKey(key)
	if !isDay {
		return RawEvent{}, false, nil // month-level or other non-day key: skip
	}
	var fields []json.RawMessage
	if err := json.Unmarshal(inner[3], &fields); err != nil {
		return RawEvent{}, false, fmt.Errorf("payload for %s is not an array: %w", key, err)
	}
	if len(fields) < 2 {
		return RawEvent{}, false, fmt.Errorf("payload for %s too short (%d elements)", key, len(fields))
	}
	var title string
	if err := json.Unmarshal(fields[1], &title); err != nil {
		return RawEvent{}, false, fmt.Errorf("title for %s is not a string: %w", key, err)
	}
	return RawEvent{Year: year, Month: month, Day: day, Title: title, Key: key}, true, nil
}

// parseDayKey extracts Y, M, D from a day key "cald-{memno}-{Y}-{M}-{D}".
// isDay is false for keys that are not 5-part day keys (e.g. month keys).
func parseDayKey(key string) (year, month, day int, isDay bool) {
	parts := strings.Split(key, "-")
	if len(parts) != 5 {
		return 0, 0, 0, false
	}
	y, err1 := strconv.Atoi(parts[2])
	m, err2 := strconv.Atoi(parts[3])
	d, err3 := strconv.Atoi(parts[4])
	if err1 != nil || err2 != nil || err3 != nil {
		return 0, 0, 0, false
	}
	return y, m, d, true
}
