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

	"github.com/kawadah/rin-tateishi-calendar/internal/config"
	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
	"github.com/kawadah/rin-tateishi-calendar/internal/event"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
)

const failureDump = "verify-protocol-failure.json"

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "PROTOCOL CHECK FAILED:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}

	client := fetch.New(cfg.MemberNo)
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body, err := client.Fetch(ctx, cfg.CanaryFrom, cfg.CanaryTo)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	count, err := check(body, cfg.CanaryMinEvents)
	if err != nil {
		if werr := os.WriteFile(failureDump, body, 0o644); werr == nil {
			fmt.Fprintf(os.Stderr, "wrote raw response to %s\n", failureDump)
		}
		return err
	}
	fmt.Printf("protocol OK: decoded %d events for %04d-%02d..%04d-%02d (>= %d)\n",
		count, cfg.CanaryFrom.Year, cfg.CanaryFrom.Month, cfg.CanaryTo.Year, cfg.CanaryTo.Month, cfg.CanaryMinEvents)
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
