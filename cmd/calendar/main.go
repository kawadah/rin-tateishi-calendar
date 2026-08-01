// Command calendar runs the pipeline: fetch the source calendar from
// freecalend, update the durable archive under data/, regenerate the live
// .ics, and hand the result off for upload.
//
// Phase 1 status: fetches the current window and prints the decoded events.
package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
)

const (
	defaultMemberNo = 231613
	monthsAhead     = 12
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run() error {
	memberNo, err := memberNoFromEnv()
	if err != nil {
		return err
	}

	client := fetch.New(memberNo)
	from, to := fetch.Window(time.Now(), monthsAhead)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	body, err := client.Fetch(ctx, from, to)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	events, err := decode.Events(body)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}

	fmt.Printf("fetched %d events (%04d-%02d .. %04d-%02d)\n",
		len(events), from.Year, from.Month, to.Year, to.Month)
	for _, e := range events {
		fmt.Printf("  %04d-%02d-%02d  %s\n", e.Year, e.Month, e.Day, e.Title)
	}
	return nil
}

func memberNoFromEnv() (int, error) {
	v := os.Getenv("SOURCE_MEMBER_NO")
	if v == "" {
		return defaultMemberNo, nil
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0, fmt.Errorf("invalid SOURCE_MEMBER_NO %q: %w", v, err)
	}
	return n, nil
}
