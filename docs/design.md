# Design & decisions

Why this project is built the way it is. This is reference material, not a plan — the build is complete. For the API itself see [protocol.md](protocol.md); for recovering when the API changes see [reverse-engineering.md](reverse-engineering.md).

## Architecture

A small Go program, run every 6 hours by GitHub Actions:

```
fetch /open/data → decode → normalize → merge archive (data/) → render .ics → upload to R2
```

The source (`freecalend.com/open/mem231613`) is a ~2.7 MB jQuery SPA; the events come from a stateless JSON API (`POST /open/data`) with no auth, so the pipeline is pure HTTP + JSON — no headless browser or JS runtime.

## Two stores, two lifecycles

The project maintains two outputs with different lifecycles:

1. **Durable archive** (`data/{YYYY}-{MM}.json`, committed to the repo) — a record of every event ever observed, preserved even after events roll out of the live window. Deterministic serialization (records sorted by date then UID, stable field order, 2-space indent) means an unchanged archive yields no diff. Git history is the ultimate backstop for anything removed.
2. **Published `.ics`** (uploaded to R2) — mirrors only the **live set**: the current fetch window. It rolls forward monthly; past events leave the feed but remain in the archive.

### Fetch window

freecalend imposes no hard viewable bound (the API returns HTTP 200 for any month and is simply empty outside the real event span; verified 1990 and 2098 → empty). We choose the window: **current month through +12 months**. "Now" is computed in `Asia/Tokyo` (not the UTC runner clock) so the window and month rollover match the JST-scheduled runs. A one-time backfill seeds the pre-existing back-catalog (earliest event 2024-07).

### Merge / edit / delete rules

Each run compares the fetch against the archive, keyed by `uid`:

- **Present:** upsert; **edits overwrite** the existing record (`first_seen` is set once and preserved).
- **Absent, inside the fetched window `[from, to]`:** the owner deleted it → **hard-remove from `data/`** (git history preserves it). The window is passed explicitly (not re-derived from a clock) so it matches exactly what was fetched, and the delete is bounded on both ends — a record dated *after* the window is never treated as a deletion.
- **Absent, outside the window:** absence proves nothing → keep the record untouched.

Records carry only `uid`, `date`, `title`, and `first_seen`. `last_seen` was dropped: it would churn every run (defeating the no-diff goal) and git history already records liveness.

## Key decisions

- **Go, not TypeScript.** The pipeline is pure HTTP + JSON with no browser need, so a JS runtime was an unnecessary constraint. The browser was only a one-time reverse-engineering aid.
- **Direct API fetch, no headless browser.** `/open/data` is a clean stateless JSON API (verified with `credentials:"omit"` and a cold `curl`), so nothing renders the SPA in production.
- **All-day events.** Inline showtimes (e.g. `13:30、18:30`) are kept in the title text rather than modeled as timed events. Dates are JST calendar days, so no timezone conversion and no `VTIMEZONE` is needed.
- **No title parsing.** The source payload exposes only one free-text field per day (fields after `title` are null/empty/render metadata); location and times exist only as substrings of that text. Parsing them into separate fields is brittle, so `Event` keeps the full string in `title`.
- **Multi-event days: split on blank lines.** A day cell can hold several events separated by a blank line; `Normalize` splits blank-line blocks into separate all-day events (`{day-key}#{index}` UIDs). A lone `\n` is treated as a soft wrap within one event (accepted limit: events separated by only a single `\n` stay one entry).
- **UID stability trade-off (accepted).** The `#index` is positional, so it's stable under title *edits* (the common case) but **not** under reordering/insertion within a multi-event day, where indices shift and `first_seen` is misattributed. Chosen deliberately: edits are far more common than intra-day reordering, and multi-event days are rare. A content-hash UID would trade this for edit-instability instead.
- **Publishing: R2 bucket + custom domain, uploaded via `wrangler`; every 6h (JST) + manual dispatch.** The archive is committed back to the repo; the `.ics` goes to R2 and is skipped when byte-identical to the published object. (Alternative considered: commit the `.ics` to the repo + GitHub Pages — simpler, but R2 keeps the repo clean and decouples hosting.)

## Protocol drift & recovery

We depend on an **undocumented, unversioned** third-party API that can change without notice, so drift must be detectable early and recovery reproducible.

- **Zero-event guard:** a normal run that decodes zero events fails *before* the merge, so a broken fetch can never wipe the archive.
- **Contract canary** (`cmd/verify-protocol`): live-fetches a fixed dense past window and asserts (a) ≥30 events decode and (b) a **pinned known event** (2024-07-06, title containing "BCF2024") decodes with the expected title. The pinned check catches a title-position shift that a bare count would miss (you'd otherwise archive garbage titles, green). It runs as an independent scheduled job and dumps the raw response on failure.
- **Recovery:** when the canary fails, [reverse-engineering.md](reverse-engineering.md) is a guided procedure (HAR export → `cmd/inspect-har` → diff against [protocol.md](protocol.md) → update the decoder/fixture/pinned event).

## Deployment (one-time setup)

Done in the Cloudflare and GitHub consoles (credentials never live in the repo):

- Create the R2 bucket and attach the **custom domain**; subscribers point at `https://<domain>/<object-key>`.
- Create a scoped **API token** (R2 read+write) and set GitHub Actions secrets `CLOUDFLARE_API_TOKEN` and `CLOUDFLARE_ACCOUNT_ID`, plus variables `R2_BUCKET` and `R2_OBJECT_KEY` (the object key includes any path prefix in the subscribe URL).
