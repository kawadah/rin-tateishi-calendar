// Command verify-protocol is a contract canary for freecalend's /open/data API.
//
// It fetches a fixed, historically dense window and asserts the response still
// decodes into a healthy number of events. A hard drop (parse failure or far
// too few events) means the protocol or decoder has drifted — see
// docs/reverse-engineering.md. On failure it writes the raw response to
// failureDump for inspection and exits non-zero.
package main

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
	"github.com/kawadah/rin-tateishi-calendar/internal/event"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
)

const (
	memberNo    = 231613
	minEvents   = 30 // this window historically holds ~150; wide margin vs. deletions
	failureDump = "verify-protocol-failure.json"
)

// A past window that should stay densely populated.
var (
	fromMonth = fetch.Month{Year: 2024, Month: 7}
	toMonth   = fetch.Month{Year: 2025, Month: 6}
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "PROTOCOL CHECK FAILED:", err)
		os.Exit(1)
	}
}

func run() error {
	client := fetch.New(memberNo)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body, err := client.Fetch(ctx, fromMonth, toMonth)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	count, err := check(body, minEvents)
	if err != nil {
		if werr := os.WriteFile(failureDump, body, 0o644); werr == nil {
			fmt.Fprintf(os.Stderr, "wrote raw response to %s\n", failureDump)
		}
		return err
	}
	fmt.Printf("protocol OK: decoded %d events for %04d-%02d..%04d-%02d (>= %d)\n",
		count, fromMonth.Year, fromMonth.Month, toMonth.Year, toMonth.Month, minEvents)
	return nil
}

// check decodes and normalizes the response and verifies the event count meets
// the floor. It returns the decoded event count.
func check(body []byte, floor int) (int, error) {
	raw, err := decode.Events(body)
	if err != nil {
		return 0, fmt.Errorf("response structure changed: %w", err)
	}
	events := event.Normalize(raw)
	if len(events) < floor {
		return len(events), fmt.Errorf("decoded only %d events, expected >= %d — protocol may have changed", len(events), floor)
	}
	return len(events), nil
}
