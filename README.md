# Rin Tateishi calendar

Fetches and parses Rin Tateishi's official calendar and publishes it as a
subscribable `.ics` feed, plus a durable per-month archive of every event.

Source: [立石凛 ＠ フリカレ](https://freecalend.com/open/mem231613) (freecalend.com)

## Subscribe

Add this URL as a calendar subscription:

```
https://<custom-domain>/<object-key>
```

_(Live once the R2 custom domain and GitHub secrets are configured — see the
[roadmap](ROADMAP.md), Phase 5.)_

- **Google Calendar:** Other calendars → From URL → paste the link.
- **Apple Calendar:** File → New Calendar Subscription → paste the link.
- **Outlook:** Add calendar → Subscribe from web → paste the link.

The feed refreshes roughly every 6 hours.

## How it works

Every 6 hours a GitHub Actions job runs the pipeline:

```
fetch /open/data  →  decode  →  normalize  →  merge archive (data/)  →  render .ics  →  upload to R2
```

- **Fetch** the live window (current month … +12 months) from freecalend's
  stateless JSON API. See [docs/protocol.md](docs/protocol.md).
- **Archive** (`data/{YYYY}-{MM}.json`) — a durable record of every event ever
  seen, kept even after events roll out of the live window. Owner deletions
  inside the window are removed; past events are frozen.
- **`.ics`** — mirrors only the **live set** (the current fetch window); all-day
  events, dates in `Asia/Tokyo`.

### Event modeling

Events are **all-day**. freecalend stores one free-text cell per day, so a
cell's full text (including inline showtimes and `@location`) is kept verbatim
as the title. When a cell holds several events separated by a blank line, they
are split into separate entries.

## Development

Tooling is managed with [mise](https://mise.jdx.dev/); the pipeline is a small
Go program.

```sh
mise install         # install pinned Go + golangci-lint
mise run pipeline    # fetch → archive → dist/calendar.ics
mise run test        # unit tests
mise run lint        # golangci-lint
```

One-time historical seed (already applied to `data/`):

```sh
mise run pipeline -- -backfill
```

### Layout

| Path                     | Purpose |
| ------------------------ | ------- |
| `cmd/calendar`           | pipeline entrypoint (`-backfill` mode) |
| `internal/fetch`         | `/open/data` client |
| `internal/decode`        | response → raw events |
| `internal/event`         | normalize, split multi-event cells, all-day model |
| `internal/archive`       | per-month store + merge/delete rules |
| `internal/ics`           | RFC 5545 feed |
| `data/`                  | durable event archive |
| `docs/`                  | protocol spec & re-RE runbook |
| `.github/workflows/`     | scheduled publish workflow |

## Documentation

- [ROADMAP.md](ROADMAP.md) — phased plan and design decisions.
- [docs/protocol.md](docs/protocol.md) — the freecalend API spec.

## Attribution

Calendar data belongs to its author and is sourced from the public
[freecalend](https://freecalend.com/open/mem231613) page. This project is an
unofficial mirror.

## License

Project code is [MIT licensed](LICENSE). The license covers the code only, not
the calendar data, which remains the property of its author.
