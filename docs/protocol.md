# freecalend `/open/data` protocol

Canonical spec of the undocumented API this project depends on. It is the diff baseline for recovery when freecalend changes — see [reverse-engineering.md](reverse-engineering.md).

- **Protocol version:** 1
- **Verified:** 2026-08 (member `231613`, 立石凛)
- **Go implementation:** [`internal/fetch`](../internal/fetch) builds the request; [`internal/decode`](../internal/decode) parses the response.

## Summary

The member page (`https://freecalend.com/open/mem<NO>`) is a ~2.7 MB jQuery SPA; static HTML contains only the member name. Events load at runtime from a **stateless JSON API**: `POST /open/data`. No cookies, session token, or signing is required — verified with `credentials:"omit"` in-browser and a cold `curl`.

## Request

```
POST https://freecalend.com/open/data
Content-Type: application/x-www-form-urlencoded; charset=UTF-8
```

Form parameters:

| param           | value                         | notes |
| --------------- | ----------------------------- | ----- |
| `target_mem_no` | `231613`                      | the member being read |
| `mem_no`        | `0`                           | the viewer (0 = anonymous) |
| `version`       | `2`                           | small constant¹ |
| `fversion`      | `20`                          | small constant¹ |
| `dokisuru`      | `sisi`                        | small constant¹ |
| `keys`          | URL-encoded JSON (see below)  | which day cells to fetch |

¹ The live page sends specific values (`version` 1 char, `fversion` 2 chars, `dokisuru` 4 chars); the made-up values above were accepted by the server. Treat them as a fragility point — if a future run 4xx/5xx's, capture the real values from a HAR (see the runbook).

### `keys`

A JSON object listing the cache keys the client wants. For each month in the window, include one month key **and** one per-day key for days 1–31:

```json
{"data":[
  ["cald-231613-2026-8",   "1602144014","1602144014",[["ok-231613-all-r-cald",true]],[]],
  ["cald-231613-2026-8-1", "1602144014","1602144014",[["ok-231613-all-r-cald",true]],[]],
  ["cald-231613-2026-8-2", "1602144014","1602144014",[["ok-231613-all-r-cald",true]],[]]
  // … through day 31, then the next month …
]}
```

- Key format: `cald-{memberNo}-{year}-{month}` and `cald-{memberNo}-{year}-{month}-{day}` (month/day **not** zero-padded).
- `1602144014` is a **stale sentinel timestamp**: it tells the server the client has nothing cached, forcing a full return. (Both timestamp slots use it.)
- `[["ok-{memberNo}-all-r-cald",true]]` is a constant read-permission block for a public calendar.

## Response

A JSON array of ops. Each op is a heterogeneous array:

```
["get", key, ts, "noexs"]          — empty day, no event
["set", "<escaped-json-string>"]   — a day WITH an event
```

A `set` payload (`op[1]`) is itself a **JSON string**; parse it to:

```
["hda", "on", key, [ dateMeta, title, …trailing… ]]
```

- `key` — the day key `cald-{memberNo}-{Y}-{M}-{D}`; the date is parsed from here.
- `dateMeta` — `[Y, M, D, …, unixTimestamp, weekday, …]` (index 7 is a Unix timestamp).
- `title` — the event text (the day cell's content).

### Payload shape notes

The payload array length varies (observed 8, 17, 18); the difference is only **trailing** metadata (a `" day_holi"` styling flag, a count, a next-day-key pointer) being present or truncated. In every shape, `payload[1]` is the single title string. **There is only one text field per day** — no separate location/time fields. Non-day (month-level) keys, if ever returned as `set`, are skipped by the decoder.

## Multi-event days

A day cell can hold several events the owner separated with a **blank line** inside the one title string. `internal/event` splits blank-line blocks into separate all-day events (`{day-key}#{index}` UIDs). A lone `\n` is treated as a soft wrap within a single event.

## Fetch window

freecalend imposes **no hard viewable bound** — the API returns HTTP 200 for any month and is simply empty outside the member's real event span (verified 1990 and 2098 → empty). The project fetches [current month, +12 months] for the live feed; a one-time backfill seeds the historical span.
