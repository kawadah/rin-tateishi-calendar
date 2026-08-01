# Development Roadmap

Fetch and parse Rin Tateishi's official calendar and publish it as a subscribable `.ics` file.

- **Source:** [立石凛 ＠ フリカレ](https://freecalend.com/open/mem231613) (freecalend.com)
- **Stack:** TypeScript / Node (toolchain managed via [mise](https://mise.jdx.dev/))
- **Publishing:** GitHub Actions on a schedule → upload `.ics` to Cloudflare R2 _(not fully locked in)_

## Conventions (from repo)

- **Tooling:** manage every tool through mise; add with `mise use <tool>` before using it (project `mise.toml` has `auto_install = false`, `pin = true`, `minimum_release_age = "3d"`). Don't rely on ambient/global installs.
- **Git:** conventional commit messages; small units committed as work progresses (not one squashed commit); merges use Git's default merge commit, edited to add the commit footer.
- **Temp files:** write scratch to `.tmp/` inside the repo; anything outside the repo needs explicit permission.

## Key technical risk — RESOLVED (Phase 1 spike): direct API, no browser

Events are **not** in static HTML; the page loads them via `POST /open/data`. The spike proved this endpoint is a **clean, stateless JSON API** — **no cookies, no session token, no signing** (verified with `credentials:"omit"` and a cold `curl`). **Decision: fetch `/open/data` directly with `undici`/`fetch`. No headless browser / Playwright needed.** Full write-up + protocol in `.tmp/phase1-findings.md`.

### Protocol
- `POST https://freecalend.com/open/data`, `Content-Type: application/x-www-form-urlencoded; charset=UTF-8`.
- Body params: `target_mem_no=231613`, `version`, `mem_no=0`, `dokisuru`, `fversion`, and `keys` (URL-encoded JSON).
- `keys` = `{"data":[ ["cald-231613-{Y}-{M}", ts, ts, [["ok-231613-all-r-cald",true]], []], …one "cald-231613-{Y}-{M}-{D}" per day… ]}` where `ts="1602144014"` (stale sentinel → server returns full data). Extend the list across months to fetch a whole window in one request.
- Response: JSON array of ops. `["set", "<escaped-json>"]` entries carry events: decode → `["hda","on","cald-…-{Y}-{M}-{D}", [ dateMeta, title, … ]]`, where `dateMeta` includes a **Unix timestamp** and Y/M/D, and `title` is the event string. `["get", key, ts, "noexs"]` = empty day.

Consequences for later phases:
- Event **titles** are still unstructured Japanese free text: optional leading emoji, optional showtime(s) (`13:30、18:30` — multiple per day), title in 「」/『』, optional `@location`, and **no end times**. Phase 2 parses the title; the date/timestamp come structured from the API.
- **Event modeling decision:** **all-day events for now** (ignore inline showtimes for scheduling; keep the full original string as the title/description).

## Data model & storage — two stores, two lifecycles

The project maintains **two outputs with different lifecycles**:

1. **Durable archive (committed to the repo).** Accumulates events as they're observed and **preserves them after they roll past** (age out of the fetch window). Per-month JSON under `data/` (e.g. `data/2026-08.json`) so diffs stay small and readable. Each record carries `date`, `title` (raw source string), source `uid`, `first_seen`, `last_seen`. Git history is the ultimate backstop — anything removed from `data/` stays recoverable in commits.
2. **Published `.ics` (live set).** The events in the **current fetch window** (1st of current month → +12 months). Rolls forward monthly; past events leave the feed but remain in the archive.

### Fetch window
freecalend imposes no hard bound (API returns HTTP 200 for any month; verified 1990 & 2098 → empty), so we choose the window: **from the 1st of the current month through +12 months**. The fetch only ever requests dates ≥ 1st of the current month; events dated before that are never re-fetched. *(One-time exception: a backfill seeds the pre-existing back-catalog — see Phase 3.)*

### Merge / edit / delete rules
Per run, compare the fetch (dates ≥ 1st of current month) against the archive:
- **New / present:** upsert by `uid`; **edits overwrite** the existing record.
- **Absent, date ≥ 1st of current month:** inside the window, so absence = the owner deleted it → **hard-delete from `data/`** (git history preserves it).
- **Absent, date < 1st of current month:** outside the window, absence proves nothing → **keep** the record untouched (frozen history).

Each run: fetch → merge/edit/delete per the rules above → regenerate `.ics` from the current fetch window → commit `data/` changes.

---

## Phase 0 — Project scaffolding

- [ ] Pin the toolchain into project `mise.toml`: `mise use node@lts pnpm` (add other tools the same way as needed).
- [ ] `pnpm init`, TypeScript config, ESLint + Prettier, `tsconfig` targeting the pinned Node. **pnpm** is the package manager.
- [ ] Directory layout: `src/` (fetch, parse, ics, publish), `test/`, `dist/`.
- [ ] Add scripts: `build`, `test`, `lint`, `start` (run the full pipeline once).
- [ ] Choose libraries: `ical-generator` for ICS output, `undici`/`fetch` for HTTP. No headless browser needed (Phase 1 resolved to a direct API).
- [ ] `.env.example` for secrets (R2 credentials, source URL).

## Phase 1 — Data acquisition (the crux)

Goal: obtain structured events (start/end datetime, title, description, location) as JSON.

- [x] **Reverse-engineer acquisition.** Done — see `.tmp/phase1-findings.md`. `POST /open/data` is a stateless JSON API (no auth); protocol documented above.
- [x] **Decision: direct `fetch` of `/open/data`.** No headless browser.
- [ ] Implement the fetcher: build the `keys` blob for the window (1st of current month → +12 months), POST, decode `set` ops into raw events.
- [ ] Capture a raw fixture (`test/fixtures/`) of a real API response for offline testing.
- [ ] **Exit criteria:** a documented, repeatable way to get raw event data on demand. _(Approach proven end-to-end via curl; implementation pending Phase 0.)_

## Phase 2 — Parsing & normalization

- [ ] Define an internal `CalendarEvent` type (uid, title, start, end, allDay, description, location, url, lastModified).
- [ ] Parse raw source data into `CalendarEvent[]`.
- [ ] Timezone handling: source is JST (`Asia/Tokyo`). Emit **all-day** events (`DATE` values) keyed to the JST calendar day from the API `dateMeta`.
- [ ] Stable `UID` generation (deterministic per event so subscribers get updates, not duplicates).
- [ ] Defensive parsing: skip/log malformed entries rather than failing the whole run.
- [ ] Unit tests against the Phase 1 fixture.

## Phase 3 — Persistence & archive (durable store)

- [ ] Define the on-disk archive format: per-month JSON under `data/` (`data/{YYYY}-{MM}.json`), records sorted deterministically.
- [ ] Merge logic: upsert current events by `uid`; set `first_seen` on new, bump `last_seen` on seen. **Edits overwrite** the existing record.
- [ ] Delete logic: archived event absent from the fetch **and** dated ≥ 1st of current month → owner deleted it → **hard-remove from `data/`**. Absent **and** dated < 1st of current month → keep untouched. (No tombstone flag — git history is the record.)
- [ ] **One-time backfill:** a separate seeding run fetches the pre-existing back-catalog (2024-06 → last month) once to populate the archive, then normal runs use the current-month+12 window.
- [ ] Deterministic serialization (stable key order, sorted records) so unchanged data yields no diff.
- [ ] Tests for merge/edit/delete (in-window vs. out-of-window) against fixtures.

## Phase 4 — ICS generation (live set)

- [ ] Generate the `.ics` from the **current fetch window** (events dated 1st of current month → +12 months present in the latest fetch).
- [ ] Emit RFC 5545 `.ics` via `ical-generator` with proper `VTIMEZONE` (Asia/Tokyo).
- [ ] Set calendar metadata: name, description, `X-WR-CALNAME`, refresh interval hint (`REFRESH-INTERVAL`, `X-PUBLISHED-TTL`).
- [ ] Deterministic output ordering so unchanged data produces a byte-identical file (avoids noisy diffs / needless uploads).
- [ ] Validate output against an ICS validator and by importing into Google Calendar + Apple Calendar.

## Phase 5 — Publishing

- [ ] GitHub Actions workflow on `schedule` (cron) + `workflow_dispatch` for manual runs.
- [ ] Steps: checkout → install → run pipeline → produce `dist/calendar.ics` **and updated `data/` archive**.
- [ ] **Commit the archive** back to the repo when it changes (the run mutates `data/`; commit via the workflow, e.g. a bot commit).
- [ ] **Upload the `.ics` to Cloudflare R2** (`aws s3` S3-compatible API or `@aws-sdk/client-s3`), credentials from GitHub Secrets.
- [ ] Set correct `Content-Type: text/calendar; charset=utf-8` and sensible `Cache-Control`.
- [ ] Decide public access: R2 public bucket URL vs. Cloudflare Worker/custom domain in front. _(Open decision.)_
- [ ] Skip upload when the file is unchanged (compare hash) to minimize churn.
- [ ] **Alternative considered:** commit `.ics` to repo + GitHub Pages. Simpler, but R2 keeps the repo clean and decouples hosting.

## Phase 6 — Reliability & observability

- [ ] Fail the CI job loudly on zero events or parse errors (guards against silent breakage when the source site changes).
- [ ] Notify on failure (GitHub Actions failure → email, or a Slack/Discord webhook).
- [ ] Log a run summary (events found, date range, output size).
- [ ] Retry/backoff on transient network errors.
- [ ] Document how to detect and recover from a source-site layout change.

## Phase 7 — Polish & docs

- [ ] README: what it is, the subscription URL, how to subscribe (Google/Apple/Outlook), update cadence.
- [ ] `CONTRIBUTING.md`: local dev, running the pipeline, updating fixtures.
- [ ] License and attribution to the source.
- [ ] Optional: a tiny landing page with the "Add to calendar" link.

---

## Open decisions

- Final hosting target: R2 public bucket vs. R2 + Worker/custom domain.
- Refresh cadence (e.g. every 6h vs. daily) — balance freshness against source load.
- ~~Fetch window~~ → **Resolved: 1st of current month → +12 months.** Fetch only dates ≥ 1st of current month; a one-time backfill seeds the pre-existing back-catalog (2024-06 →).
- ~~Published scope~~ → **Resolved: live set** = current fetch window; archive preserves past events after they roll out; owner-deletions inside the window are hard-removed from `data/` (git keeps history).
- ~~Edit handling~~ → **Resolved: overwrite** the existing record.
- ~~Event modeling~~ → **Resolved: all-day events** (inline showtimes kept in the title text, not modeled as timed events).
- ~~Whether a headless browser is required~~ → **Resolved: no. Direct stateless API fetch.**
