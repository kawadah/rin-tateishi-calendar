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

Other tasks: `mise run build`, `mise run tidy`, `mise run verify-protocol`,
`mise run inspect-har <file.har>`. Run `mise tasks` to list them.

The code is a small Go program; see the layout table in the [README](README.md)
and the design rationale in [ROADMAP.md](ROADMAP.md).

## Tests & fixtures

Tests use the standard `testing` package with golden fixtures under each
package's `testdata/`. The decode fixture
(`internal/decode/testdata/response_2026-08.json`) is a captured real
`/open/data` response; regenerate it only when the protocol changes, and update
[docs/protocol.md](docs/protocol.md) in the same commit.

## The archive (`data/`)

`data/{YYYY}-{MM}.json` is a durable, deterministic store maintained by the
pipeline — don't hand-edit it. A normal run mutates it via the merge/delete
rules; the one-time historical seed is `mise run pipeline -- -backfill`.

## When the source API changes

`/open/data` is undocumented and unversioned. If the canary
(`mise run verify-protocol`) fails, follow
[docs/reverse-engineering.md](docs/reverse-engineering.md).

## Commits

- Conventional commit messages.
- Small units, committed as work progresses (not one squashed commit).
- When merging a branch, use Git's default merge commit.
