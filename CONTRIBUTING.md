# CONTRIBUTING

## mise

- Use mise to add and manage tools
- Don't use tools outside what's installed via mise. If you need a tool, add it with `mise use` first.

## Development

```sh
mise install          # install the pinned Go + golangci-lint
mise run pipeline     # fetch -> archive (data/) -> dist/calendar.ics
mise run test         # unit tests
mise run lint         # golangci-lint
mise run fmt          # format
```

Other tasks: `mise run build`, `mise run tidy`, `mise run verify-protocol`, `mise run inspect-har <file.har>`. Run `mise tasks` to list them.

## How it works

The pipeline is a small Go program, run on a schedule by GitHub Actions:

```
fetch /open/data → decode → normalize → merge archive (data/) → render .ics → upload to R2
```

- **Fetch** the live window (current month … +12 months) from freecalend's stateless JSON API (see [docs/protocol.md](docs/protocol.md)).
- **Archive** (`data/{YYYY}-{MM}.json`) — a durable record of every event ever seen, kept even after events roll out of the live window. Owner deletions inside the window are removed; past events are frozen.
- **`.ics`** mirrors only the live set (the current fetch window); all-day events, dates in `Asia/Tokyo`.

Design rationale and decisions are in [ROADMAP.md](ROADMAP.md).

### Layout

| Path                     | Purpose |
| ------------------------ | ------- |
| `cmd/calendar`           | pipeline entrypoint (`-backfill` mode) |
| `cmd/verify-protocol`    | protocol canary |
| `cmd/inspect-har`        | HAR data-request locator |
| `internal/config`        | centralized settings (member, name, window, …) |
| `internal/fetch`         | `/open/data` client |
| `internal/decode`        | response → raw events |
| `internal/event`         | normalize, split multi-event cells, all-day model |
| `internal/archive`       | per-month store + merge/delete rules |
| `internal/ics`           | RFC 5545 feed |
| `data/`                  | durable event archive |
| `docs/`                  | protocol spec & re-RE runbook |
| `.github/workflows/`     | publish + check workflows |

## Tests & fixtures

Tests use the standard `testing` package with golden fixtures under each package's `testdata/`. The decode fixture (`internal/decode/testdata/response_2026-08.json`) is a captured real `/open/data` response; regenerate it only when the protocol changes, and update [docs/protocol.md](docs/protocol.md) in the same commit.

## The archive (`data/`)

`data/{YYYY}-{MM}.json` is a durable, deterministic store maintained by the pipeline — don't hand-edit it. A normal run mutates it via the merge/delete rules; the one-time historical seed is `mise run pipeline -- -backfill`.

## When the source API changes

`/open/data` is undocumented and unversioned. If the canary (`mise run verify-protocol`) fails, follow [docs/reverse-engineering.md](docs/reverse-engineering.md).

## Commits

- Conventional commit messages.
- Small units, committed as work progresses (not one squashed commit).
- When merging a branch, use Git's default merge commit.
