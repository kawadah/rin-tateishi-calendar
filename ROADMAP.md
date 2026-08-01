# Development Roadmap

Fetch and parse Rin Tateishi's official calendar and publish it as a subscribable `.ics` file.

- **Source:** [立石凛 ＠ フリカレ](https://freecalend.com/open/mem231613) (freecalend.com)
- **Stack:** Go (toolchain managed via [mise](https://mise.jdx.dev/)) — pipeline is pure HTTP + JSON, no browser/JS runtime needed
- **Publishing:** GitHub Actions on a schedule → upload `.ics` to Cloudflare R2 _(not fully locked in)_

## Conventions (from repo)

- **Tooling:** manage every tool through mise; add with `mise use <tool>` before using it (project `mise.toml` has `auto_install = false`, `pin = true`, `minimum_release_age = "3d"`). Don't rely on ambient/global installs.
- **Git:** conventional commit messages; small units committed as work progresses (not one squashed commit); merges use Git's default merge commit, edited to add the commit footer.
- **Temp files:** write scratch to `.tmp/` inside the repo; anything outside the repo needs explicit permission.

## Key technical risk — RESOLVED (Phase 1 spike): direct API, no browser

Events are **not** in static HTML; the page loads them via `POST /open/data`. The spike proved this endpoint is a **clean, stateless JSON API** — **no cookies, no session token, no signing** (verified with `credentials:"omit"` and a cold `curl`). **Decision: fetch `/open/data` directly with Go's `net/http`. No headless browser / JS runtime needed** — the browser was only a one-time reverse-engineering aid, never a production dependency (see *Protocol maintenance* below). Full write-up + protocol in `.tmp/phase1-findings.md`.

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

## Protocol maintenance & re-reverse-engineering

We depend on an **undocumented, unversioned** third-party API (`/open/data`) that can change without notice. This makes breakage **detectable early** and recovery **reproducible** — so re-deriving the protocol is a guided procedure, not archaeology. The one-time reverse-engineering that discovered the current protocol is captured here as a repeatable process.

### Detect — know when it broke
- **Contract canary** (scheduled, runs independently of the main fetch): replays the documented request and asserts the response still (a) is HTTP 200, (b) parses as the expected JSON array, (c) has ≥1 `set` op of the expected shape, and (d) contains a **pinned known event** (a fixed past date + title from the fixture). Any failure → alert "protocol may have changed."
- Main pipeline also fails loudly on zero events / decode errors (see Phase 6).
- On failure, CI uploads the **raw response as an artifact** for inspection.

### Reference baseline — what we diff against
- `docs/protocol.md` — the canonical spec (endpoint, method, params, `keys` structure, response encoding, decode steps). Single source of truth; carries a `PROTOCOL_VERSION`.
- `testdata/known-good.json` — a recorded real request + response including the pinned canary event. Drives decoder tests **and** the canary.

### Reproducible re-RE runbook — `docs/reverse-engineering.md`
When the canary fails:
1. **Confirm & capture.** Run the canary; download the failing raw-response artifact.
2. **Check static.** `curl` the member page; confirm data still isn't inline HTML.
3. **Capture live traffic.** Load the member page in a browser with DevTools → Network, reproduce a load, and **Export HAR** to `.tmp/`. (Browser automation such as Claude-in-Chrome is an *optional* accelerant — not a project dependency; a manual HAR export needs no code and works in any browser.)
4. **Locate the data request — scripted.** Run `inspect-har <file>`: lists all XHR/fetch POSTs ranked by response size / member-id presence, and dumps the top candidate's method, URL, header names, body param names+sizes, and response head. Turns "poke around DevTools" into a deterministic step.
5. **Diff vs. `docs/protocol.md`.** Identify what changed: endpoint path, param names, `keys` shape, or response encoding.
6. **Reconstruct & verify.** Rebuild the request from scratch; confirm a cold `curl`/Go client reproduces the browser response for the known event.
7. **Update & re-pin.** Update `docs/protocol.md`, `testdata/known-good.json`, the Go decoder, and the canary event; bump `PROTOCOL_VERSION`.
8. **Verify green.** Run the canary + full pipeline; confirm known events reappear.

### Reproducible tooling (checked into the repo)
- `cmd/inspect-har` — HAR (just JSON) → ranked candidate data requests + their shapes. Written in Go so there's no second toolchain.
- `cmd/verify-protocol` — replays the documented request and asserts the canary event/shape. Reused as both the scheduled canary and the runbook's step-6/8 verification.
- Fixtures + `docs/protocol.md` kept in lockstep with the decoder (a decoder change without a fixture/spec update should fail CI).

---

## Phase 0 — Project scaffolding

- [x] Pin the toolchain into project `mise.toml`: `go@1.26.4`, `golangci-lint@2.12.2`.
- [x] `go mod init github.com/kawadah/rin-tateishi-calendar`; layout: `cmd/` (`calendar`, `inspect-har`, `verify-protocol`), `internal/` (fetch, decode, archive, ics, upload), `testdata/`, `docs/`. _(internal/* dirs are populated per phase.)_
- [x] **Tools confirmed:** `golangci-lint` (v2 config, default `standard` + misspell/unconvert, gofmt/goimports), `golang-ical` for ICS, stdlib `testing` + `testdata`.
- [x] mise tasks: `build`, `test`, `lint`, `fmt`, `tidy`, `pipeline` (run via `mise run <task>`).
- [x] `.env.example` for secrets (R2 credentials, source member no).
- Module path assumes remote `github.com/kawadah/...` — adjust if the real remote differs.

## Phase 1 — Data acquisition (the crux)

Goal: obtain structured events (start/end datetime, title, description, location) as JSON.

- [x] **Reverse-engineer acquisition.** Done — see `.tmp/phase1-findings.md`. `POST /open/data` is a stateless JSON API (no auth); protocol documented above.
- [x] **Decision: direct `fetch` of `/open/data`.** No headless browser.
- [x] Implement the fetcher (`internal/fetch`): build the `keys` blob for the window (current month → +12 months), POST, return the raw body.
- [x] Decode `set` ops into raw events (`internal/decode`); `cmd/calendar` wires fetch→decode.
- [x] Capture a raw fixture (`internal/decode/testdata/response_2026-08.json`) for offline decode tests.
- [x] **Exit criteria met:** `mise run pipeline` fetches & decodes the live window (verified: 18 events, 2026-08 → 2027-08).

## Phase 2 — Parsing & normalization

- [x] Define the `event.Event` struct (uid, date, title, allDay). Leaner than first sketched: per the all-day decision the full source string stays in `title`, so description/location/url aren't split out.
- [x] `event.Normalize` parses `[]decode.RawEvent` → sorted `[]Event`.
- [x] Timezone: dates are JST (`Asia/Tokyo`) calendar days from the source day-key; all-day, so no conversion needed.
- [x] Stable `UID` (`{day-key}#{index}@freecalend.com`) so edited titles update in place; the `#index` distinguishes multiple events in one day cell (see multi-event decision below).
- [x] Defensive parsing: empty-after-clean titles skipped; malformed ops already skipped in decode.
- [x] Unit tests for `cleanTitle` and `Normalize` (whitespace, skip, sort, UID/date).

## Phase 3 — Persistence & archive (durable store)

- [x] On-disk format: per-month `data/{YYYY}-{MM}.json`, records sorted by date then UID, 2-space indent, `Date`/`first_seen` as `YYYY-MM-DD`.
- [x] Merge logic (`archive.Merge`): upsert by `uid`, **edits overwrite**, `first_seen` set once and preserved. (Dropped `last_seen` — it churns every run and git history already records liveness.)
- [x] Delete logic: archived event absent from the fetch **and** dated ≥ 1st of current month → **hard-removed from `data/`**; absent **and** dated < current month → kept frozen. No tombstone (git history is the record).
- [x] **One-time backfill** (`calendar -backfill`, additive/never-deletes): seeded 242 historical events; archive now spans 2024-07 → 2026-11 (260 records).
- [x] Deterministic serialization — verified: re-running produces byte-identical files (no diff on unchanged).
- [x] Tests for merge/edit/delete (in/out-of-window), backfill, round-trip, determinism, empty-month cleanup.

## Phase 4 — ICS generation (live set)

- [x] Generate `dist/calendar.ics` from the **live set** (current fetch window); backfill skips feed generation.
- [x] Emit RFC 5545 `.ics` via `golang-ical` — all-day `DATE` VEVENTs (exclusive `DTEND`). No `VTIMEZONE` needed: all-day dates are JST calendar days with no time component; `X-WR-TIMEZONE:Asia/Tokyo` set as a hint.
- [x] Calendar metadata: name/description, `X-WR-CALNAME`/`X-WR-CALDESC`, `METHOD:PUBLISH`, `REFRESH-INTERVAL`/`X-PUBLISHED-TTL` (PT6H).
- [x] Deterministic output — pinned `DTSTAMP` + sorted events; verified byte-identical across runs.
- [x] Validated: round-trips through the iCalendar parser in tests. _(Manual Google/Apple subscribe check deferred until there's a published URL — Phase 5.)_

## Phase 5 — Publishing

Decisions: **cadence = every 6h** (matches the feed's refresh hint) + `workflow_dispatch`; **access = custom domain** on the R2 bucket; **upload = wrangler**.

- [x] GitHub Actions workflow (`.github/workflows/publish.yml`): `schedule` (cron `0 */6 * * *`, `timezone: Asia/Tokyo` → 00/06/12/18 JST) + `workflow_dispatch`, serialized via `concurrency`.
- [x] Steps: checkout → mise-action → `mise run pipeline` → `dist/calendar.ics` **and updated `data/` archive**.
- [x] **Commit the archive** back when `data/` changes (bot commit, `--rebase --autostash` before push; no-op when unchanged).
- [x] **Upload `dist/calendar.ics` to R2 via `wrangler r2 object put`** with `Content-Type: text/calendar; charset=utf-8` and `Cache-Control: public, max-age=3600`.
- [x] Skip upload when the published object is byte-identical (compares via `wrangler r2 object get`).
- [x] Parameterized via secrets (`CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`) + vars (`R2_BUCKET`, `R2_OBJECT_KEY`).
- [ ] **Blocked on user:** create R2 bucket + custom domain, scoped API token, and set the secrets/vars (see below). Then verify a real run + subscribe in Google/Apple Calendar.

### Setup the user must do (Cloudflare + GitHub) — values I never handle
- Create the R2 bucket; attach the **custom domain** (Cloudflare dashboard → R2 → bucket → Settings → Custom Domains). Subscribers point at `https://<domain>/<key>`.
- Create a scoped **API token** (R2 write) and add GitHub secrets: `CLOUDFLARE_API_TOKEN`, `CLOUDFLARE_ACCOUNT_ID`; repo var `R2_BUCKET` (default `rin-tateishi-calendar`) and the object key.
- **Alternative considered:** commit `.ics` to repo + GitHub Pages. Simpler, but R2 keeps the repo clean and decouples hosting.

## Phase 6 — Reliability & observability

- [x] Fail loudly on zero events — the guard runs *before* the merge so a broken fetch can't wipe the archive.
- [x] **Protocol canary** (`cmd/verify-protocol`): checks a dense past window decodes ≥30 events; runs as an independent scheduled job, uploads the raw response artifact on failure.
- [x] Notify on failure — GitHub's default job-failure email (canary or publish); artifact attached for inspection.
- [x] Log a run summary (event count + date range).
- [x] Retry/backoff on transient fetch errors (transport/5xx); 4xx non-retryable.
- [x] Ship `docs/reverse-engineering.md` (re-RE runbook) and `cmd/inspect-har`.

## Phase 7 — Polish & docs

- [x] README: what it is, how to subscribe (Google/Apple/Outlook), cadence, layout, attribution. _(Subscribe URL is a placeholder until R2 is set up.)_
- [x] `CONTRIBUTING.md`: local dev, tasks, tests/fixtures, archive handling, re-RE pointer, commit conventions.
- [x] License (MIT, code only) and source attribution.
- [ ] Optional: a tiny landing page with an "Add to calendar" link. _(Not done — optional; revisit after the R2 URL exists.)_

---

## Open decisions

- ~~Hosting target~~ → **Resolved: R2 bucket + custom domain**, uploaded via `wrangler`.
- ~~Refresh cadence~~ → **Resolved: every 6h** (`0 */6 * * *`, `timezone: Asia/Tokyo` → 00/06/12/18 JST) + manual `workflow_dispatch`.
- ~~Language~~ → **Resolved: Go.** The pipeline is pure HTTP+JSON with no browser need, so JS was an unnecessary constraint; earlier JS sub-tool answers (Vitest/oxlint/ical-generator) are moot.
- ~~R2 upload mechanism~~ → **Resolved: wrangler/rclone as a CI step** (no S3 SDK in the Go binary).
- ~~Fetch window~~ → **Resolved: 1st of current month → +12 months.** Fetch only dates ≥ 1st of current month; a one-time backfill seeds the pre-existing back-catalog (2024-06 →).
- ~~Published scope~~ → **Resolved: live set** = current fetch window; archive preserves past events after they roll out; owner-deletions inside the window are hard-removed from `data/` (git keeps history).
- ~~Edit handling~~ → **Resolved: overwrite** the existing record.
- ~~Event modeling~~ → **Resolved: all-day events** (inline showtimes kept in the title text, not modeled as timed events).
- ~~Split location/time into separate fields~~ → **Resolved: no.** Verified the source payload has only one free-text field (fields after `title` are null/empty/render metadata); location & times exist only as title substrings. Parsing them out is brittle and not worth it. `Event` keeps the full string in `Title`.
- ~~Multiple events on one date~~ → **Resolved: split on blank lines.** The source stores one text cell per day, but owners pack multiple events into it separated by blank lines. `Normalize` splits blank-line blocks into separate all-day events (`{day-key}#{index}` UIDs); a lone `\n` stays a soft wrap within one event. Accepted heuristic limit: events separated by only a single `\n` remain one entry.
- ~~Whether a headless browser is required~~ → **Resolved: no. Direct stateless API fetch.**
