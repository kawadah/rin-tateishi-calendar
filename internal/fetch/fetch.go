// Package fetch retrieves raw event data from freecalend's /open/data endpoint.
//
// The endpoint is a stateless JSON API (no cookies/auth). A request lists the
// cache keys the client wants — a month key plus one per-day key for every day
// in the window — each carrying a stale sentinel timestamp that forces the
// server to return full data. See docs/protocol.md for the full protocol.
package fetch

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// DefaultEndpoint is freecalend's calendar-data endpoint.
const DefaultEndpoint = "https://freecalend.com/open/data"

// staleTimestamp is a sentinel "last cached" time far in the past; it tells the
// server the client has nothing cached, so it returns the full event data.
const staleTimestamp = "1602144014"

// Retry policy for transient failures (transport errors and 5xx).
const (
	maxAttempts  = 3
	retryBackoff = 500 * time.Millisecond
)

// Month identifies a calendar month.
type Month struct {
	Year  int
	Month int
}

// index returns a monotonic month index for comparison/arithmetic.
func (m Month) index() int { return m.Year*12 + (m.Month - 1) }

// Add returns the month n months after m (n may be negative).
func (m Month) Add(n int) Month {
	i := m.index() + n
	return Month{Year: i / 12, Month: i%12 + 1}
}

// Window returns the fetch window [current month, current month + aheadMonths].
func Window(now time.Time, aheadMonths int) (from, to Month) {
	from = Month{Year: now.Year(), Month: int(now.Month())}
	return from, from.Add(aheadMonths)
}

// monthsBetween returns every month in [from, to] inclusive.
func monthsBetween(from, to Month) []Month {
	var out []Month
	for m := from; m.index() <= to.index(); m = m.Add(1) {
		out = append(out, m)
	}
	return out
}

// Client fetches raw event data for a single freecalend member.
type Client struct {
	HTTP     *http.Client
	Endpoint string
	MemberNo int
}

// New returns a Client for the given member number with sensible defaults.
func New(memberNo int) *Client {
	return &Client{
		HTTP:     &http.Client{Timeout: 30 * time.Second},
		Endpoint: DefaultEndpoint,
		MemberNo: memberNo,
	}
}

// buildKeys constructs the `keys` form value: a JSON blob listing a month key
// plus one per-day key for every day in the window.
func buildKeys(memberNo int, ms []Month) (string, error) {
	perm := fmt.Sprintf("ok-%d-all-r-cald", memberNo)
	permBlock := []any{[]any{perm, true}}
	entry := func(key string) []any {
		return []any{key, staleTimestamp, staleTimestamp, permBlock, []any{}}
	}
	data := make([]any, 0)
	for _, m := range ms {
		data = append(data, entry(fmt.Sprintf("cald-%d-%d-%d", memberNo, m.Year, m.Month)))
		for d := 1; d <= 31; d++ {
			data = append(data, entry(fmt.Sprintf("cald-%d-%d-%d-%d", memberNo, m.Year, m.Month, d)))
		}
	}
	b, err := json.Marshal(map[string]any{"data": data})
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// Fetch requests all events in [from, to] and returns the raw response body.
// Transient failures (transport errors and 5xx) are retried with backoff.
func (c *Client) Fetch(ctx context.Context, from, to Month) ([]byte, error) {
	keys, err := buildKeys(c.MemberNo, monthsBetween(from, to))
	if err != nil {
		return nil, fmt.Errorf("build keys: %w", err)
	}
	encoded := url.Values{
		"target_mem_no": {strconv.Itoa(c.MemberNo)},
		"version":       {"2"},
		"mem_no":        {"0"},
		"keys":          {keys},
		"dokisuru":      {"sisi"},
		"fversion":      {"20"},
	}.Encode()

	var lastErr error
	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if attempt > 1 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(retryBackoff * time.Duration(1<<(attempt-2))):
			}
		}
		body, status, err := c.do(ctx, encoded)
		switch {
		case err != nil:
			lastErr = err
		case status == http.StatusOK:
			return body, nil
		case status >= 500:
			lastErr = fmt.Errorf("status %d: %s", status, truncate(body, 200))
		default: // 4xx and other non-retryable statuses
			return nil, fmt.Errorf("status %d: %s", status, truncate(body, 200))
		}
	}
	return nil, fmt.Errorf("after %d attempts: %w", maxAttempts, lastErr)
}

// do performs a single request and returns the body and status code.
func (c *Client) do(ctx context.Context, encodedForm string) (body []byte, status int, err error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.Endpoint, strings.NewReader(encodedForm))
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded; charset=UTF-8")

	resp, err := c.HTTP.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer func() { _ = resp.Body.Close() }()

	body, err = io.ReadAll(resp.Body)
	if err != nil {
		return nil, 0, err
	}
	return body, resp.StatusCode, nil
}

func truncate(b []byte, n int) string {
	if len(b) > n {
		return string(b[:n]) + "…"
	}
	return string(b)
}
