# Re-reverse-engineering runbook

We depend on freecalend's **undocumented, unversioned** `/open/data` API. When it
changes, this is the repeatable procedure to re-derive the protocol. The
canonical current spec is [protocol.md](protocol.md).

## Detecting a break

- **Contract canary:** `mise run verify-protocol` fetches a historically dense
  window and asserts the response still decodes into ≥30 events. It runs on a
  schedule in CI (a separate job) and fails loudly on drift, writing the raw
  response to `verify-protocol-failure.json`.
- **Pipeline zero-event guard:** a normal run that decodes zero events fails
  *before* touching the archive.

A failure of either is the signal to run this runbook.

## Runbook

1. **Confirm & capture.** Run `mise run verify-protocol` locally (or download the
   `verify-protocol-failure.json` artifact from the failed CI run). Keep the raw
   response — it's your evidence of what changed.

2. **Check the page still isn't static.** `curl -s https://freecalend.com/open/mem231613 | grep -c cald-`
   — if the events are now inline in the HTML, the whole approach changed; skip to step 5.

3. **Capture live traffic as a HAR.** Open `https://freecalend.com/open/mem231613`
   in a browser with DevTools → Network open, reload, then **Export HAR**
   (right-click the request list → "Save all as HAR") to `.tmp/capture.har`.
   Browser automation (e.g. Claude in Chrome) can do this too, but it's optional —
   a manual HAR export needs no tooling and works in any browser.

4. **Locate the data request.** `mise run inspect-har .tmp/capture.har`. It ranks
   requests with calendar-looking responses first and prints each candidate's
   method, URL, request-parameter shape, and a response preview. The top
   `[CALENDAR?]` row is almost certainly the new data endpoint.

5. **Diff against [protocol.md](protocol.md).** Compare the captured request/response
   to the documented spec. Identify exactly what moved:
   - endpoint path or method,
   - request parameter names/values (especially the `keys` blob shape and the
     `version`/`fversion`/`dokisuru` constants),
   - response encoding (the op array, the `set` escaped-JSON payload, the
     `payload[1]` title position).

6. **Reconstruct & verify.** Update the Go code to match:
   - request shape → [`internal/fetch`](../internal/fetch),
   - response decode → [`internal/decode`](../internal/decode).
   Confirm a cold run reproduces the browser's data: `mise run verify-protocol`
   should pass, and `mise run pipeline` should decode the expected events.

7. **Update the fixtures & spec.** Refresh the decode fixture
   (`internal/decode/testdata/response_2026-08.json`) if the response shape
   changed, update [protocol.md](protocol.md), and bump its **Protocol version**.

8. **Verify green.** `mise run test`, `mise run lint`, `mise run verify-protocol`,
   `mise run pipeline`. Commit the code, fixture, and spec together so they stay
   in lockstep.

## Tools

| Tool | Command | Purpose |
| ---- | ------- | ------- |
| Canary | `mise run verify-protocol` | detect drift; dump raw response |
| HAR locator | `mise run inspect-har <file.har>` | find the data request in a capture |
| Pipeline | `mise run pipeline` | end-to-end sanity check |
