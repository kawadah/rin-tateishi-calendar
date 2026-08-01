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
- [ ] Implement the fetcher: build the `keys` blob for a configurable month window, POST, decode `set` ops into raw events.
- [ ] Capture a raw fixture (`test/fixtures/`) of a real API response for offline testing.
- [ ] **Exit criteria:** a documented, repeatable way to get raw event data on demand. _(Approach proven end-to-end via curl; implementation pending Phase 0.)_

## Phase 2 — Parsing & normalization

- [ ] Define an internal `CalendarEvent` type (uid, title, start, end, allDay, description, location, url, lastModified).
- [ ] Parse raw source data into `CalendarEvent[]`.
- [ ] Timezone handling: source is JST (`Asia/Tokyo`). Emit **all-day** events (`DATE` values) keyed to the JST calendar day from the API `dateMeta`.
- [ ] Stable `UID` generation (deterministic per event so subscribers get updates, not duplicates).
- [ ] Defensive parsing: skip/log malformed entries rather than failing the whole run.
- [ ] Unit tests against the Phase 1 fixture.

## Phase 3 — ICS generation

- [ ] Emit RFC 5545 `.ics` via `ical-generator` with proper `VTIMEZONE` (Asia/Tokyo).
- [ ] Set calendar metadata: name, description, `X-WR-CALNAME`, refresh interval hint (`REFRESH-INTERVAL`, `X-PUBLISHED-TTL`).
- [ ] Deterministic output ordering so unchanged data produces a byte-identical file (avoids noisy diffs / needless uploads).
- [ ] Validate output against an ICS validator and by importing into Google Calendar + Apple Calendar.

## Phase 4 — Publishing

- [ ] GitHub Actions workflow on `schedule` (cron) + `workflow_dispatch` for manual runs.
- [ ] Steps: checkout → install → run pipeline → produce `dist/calendar.ics`.
- [ ] **Upload to Cloudflare R2** (`aws s3` S3-compatible API or `@aws-sdk/client-s3`), credentials from GitHub Secrets.
- [ ] Set correct `Content-Type: text/calendar; charset=utf-8` and sensible `Cache-Control`.
- [ ] Decide public access: R2 public bucket URL vs. Cloudflare Worker/custom domain in front. _(Open decision.)_
- [ ] Skip upload when the file is unchanged (compare hash) to minimize churn.
- [ ] **Alternative considered:** commit `.ics` to repo + GitHub Pages. Simpler, but R2 keeps the repo clean and decouples hosting.

## Phase 5 — Reliability & observability

- [ ] Fail the CI job loudly on zero events or parse errors (guards against silent breakage when the source site changes).
- [ ] Notify on failure (GitHub Actions failure → email, or a Slack/Discord webhook).
- [ ] Log a run summary (events found, date range, output size).
- [ ] Retry/backoff on transient network errors.
- [ ] Document how to detect and recover from a source-site layout change.

## Phase 6 — Polish & docs

- [ ] README: what it is, the subscription URL, how to subscribe (Google/Apple/Outlook), update cadence.
- [ ] `CONTRIBUTING.md`: local dev, running the pipeline, updating fixtures.
- [ ] License and attribution to the source.
- [ ] Optional: a tiny landing page with the "Add to calendar" link.

---

## Open decisions

- Final hosting target: R2 public bucket vs. R2 + Worker/custom domain.
- Refresh cadence (e.g. every 6h vs. daily) — balance freshness against source load.
- Rolling window size (how many months ahead/behind to include).
- ~~Event modeling~~ → **Resolved: all-day events** (inline showtimes kept in the title text, not modeled as timed events).
- ~~Whether a headless browser is required~~ → **Resolved: no. Direct stateless API fetch.**
