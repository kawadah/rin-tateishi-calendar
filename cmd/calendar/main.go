// Command calendar runs the pipeline: fetch the source calendar from
// freecalend, update the durable archive under data/, and regenerate the live
// .ics for upload.
//
// Normal run: fetch [current month, +MonthsAhead] and merge into the archive.
// -backfill: one-time seed of the historical back-catalog (additive; no
// deletes), then exit.
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/kawadah/rin-tateishi-calendar/internal/archive"
	"github.com/kawadah/rin-tateishi-calendar/internal/config"
	"github.com/kawadah/rin-tateishi-calendar/internal/decode"
	"github.com/kawadah/rin-tateishi-calendar/internal/event"
	"github.com/kawadah/rin-tateishi-calendar/internal/fetch"
	"github.com/kawadah/rin-tateishi-calendar/internal/ics"
)

// backfillStart is the earliest month seeded by -backfill (before the first
// known event, 2024-06). Empty months cost nothing.
var backfillStart = fetch.Month{Year: 2024, Month: 1}

func main() {
	backfill := flag.Bool("backfill", false, "one-time seed of the historical back-catalog, then exit")
	flag.Parse()
	if err := run(*backfill); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

func run(backfill bool) error {
	cfg, err := config.FromEnv()
	if err != nil {
		return err
	}
	now := time.Now()
	from, to := fetchRange(cfg, now, backfill)

	client := fetch.New(cfg.MemberNo)
	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	body, err := client.Fetch(ctx, from, to)
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}
	raw, err := decode.Events(body)
	if err != nil {
		return fmt.Errorf("decode: %w", err)
	}
	events := event.Normalize(raw)

	// Guard against silent breakage: a normal run over the live window should
	// always find events. Zero means the source or protocol likely changed, so
	// fail loudly *before* the merge — otherwise the delete rule would wipe every
	// in-window record. (Backfill may legitimately hit sparse ranges.)
	if !backfill && len(events) == 0 {
		return fmt.Errorf("no events decoded for the live window (%04d-%02d..%04d-%02d) — source or protocol may have changed",
			from.Year, from.Month, to.Year, to.Month)
	}

	store := archive.NewStore(cfg.DataDir)
	records, err := store.Load()
	if err != nil {
		return fmt.Errorf("load archive: %w", err)
	}
	if backfill {
		archive.Backfill(records, events, now)
	} else {
		archive.Merge(records, events, now)
	}
	if err := store.Save(records); err != nil {
		return fmt.Errorf("save archive: %w", err)
	}

	// The .ics mirrors only the live set (current fetch window); backfill just
	// seeds history and does not regenerate the feed.
	if !backfill {
		if err := writeICS(cfg, events); err != nil {
			return fmt.Errorf("write ics: %w", err)
		}
	}

	mode := "run"
	if backfill {
		mode = "backfill"
	}
	fmt.Printf("%s: fetched %d events (%04d-%02d..%04d-%02d); archive holds %d records\n",
		mode, len(events), from.Year, from.Month, to.Year, to.Month, len(records))
	return nil
}

func writeICS(cfg config.Config, events []event.Event) error {
	if err := os.MkdirAll(filepath.Dir(cfg.ICSPath), 0o755); err != nil {
		return err
	}
	meta := ics.Metadata{
		Name:        cfg.CalendarName,
		Description: cfg.CalendarDesc,
		ProductID:   cfg.ProductID,
		Timezone:    cfg.Timezone,
		RefreshTTL:  cfg.RefreshTTL,
	}
	return os.WriteFile(cfg.ICSPath, []byte(ics.Build(events, meta)), 0o644)
}

// fetchRange returns the window to fetch: the live window, or
// [backfillStart, last month] for a one-time backfill.
func fetchRange(cfg config.Config, now time.Time, backfill bool) (from, to fetch.Month) {
	if backfill {
		lastMonth := fetch.Month{Year: now.Year(), Month: int(now.Month())}
		return backfillStart, lastMonth.Add(-1)
	}
	return fetch.Window(now, cfg.MonthsAhead)
}
