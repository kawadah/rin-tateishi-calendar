// Command inspect-har locates the calendar data request inside a browser HAR
// export. Given a HAR file, it ranks the captured requests (calendar-looking
// responses first, then by response size) and prints each candidate's method,
// URL, request-parameter shape, and a response preview — turning the "find the
// data request in DevTools" step of re-reverse-engineering into a repeatable
// command. See docs/reverse-engineering.md.
package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"sort"
	"strings"
)

type har struct {
	Log struct {
		Entries []entry `json:"entries"`
	} `json:"log"`
}

type entry struct {
	Request struct {
		Method   string `json:"method"`
		URL      string `json:"url"`
		PostData struct {
			Text   string `json:"text"`
			Params []struct {
				Name  string `json:"name"`
				Value string `json:"value"`
			} `json:"params"`
		} `json:"postData"`
	} `json:"request"`
	Response struct {
		Status  int `json:"status"`
		Content struct {
			Size     int    `json:"size"`
			MimeType string `json:"mimeType"`
			Text     string `json:"text"`
			Encoding string `json:"encoding"`
		} `json:"content"`
	} `json:"response"`
}

type paramInfo struct {
	Name string
	Size int
}

type candidate struct {
	Method   string
	URL      string
	Status   int
	RespSize int
	RespHead string
	Params   []paramInfo
	Calendar bool
}

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: inspect-har <file.har>")
		os.Exit(2)
	}
	data, err := os.ReadFile(os.Args[1])
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	cands, err := analyze(data)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Print(report(cands, 10))
}

func analyze(data []byte) ([]candidate, error) {
	var h har
	if err := json.Unmarshal(data, &h); err != nil {
		return nil, fmt.Errorf("parse HAR: %w", err)
	}
	cands := make([]candidate, 0, len(h.Log.Entries))
	for _, e := range h.Log.Entries {
		respText := decodeContent(e.Response.Content.Text, e.Response.Content.Encoding)
		size := e.Response.Content.Size
		if size <= 0 {
			size = len(respText)
		}
		cands = append(cands, candidate{
			Method:   e.Request.Method,
			URL:      e.Request.URL,
			Status:   e.Response.Status,
			RespSize: size,
			RespHead: head(respText, 160),
			Params:   paramsOf(e),
			Calendar: looksLikeCalendar(respText),
		})
	}
	sort.SliceStable(cands, func(i, j int) bool {
		if cands[i].Calendar != cands[j].Calendar {
			return cands[i].Calendar
		}
		return cands[i].RespSize > cands[j].RespSize
	})
	return cands, nil
}

func decodeContent(text, encoding string) string {
	if encoding == "base64" {
		if b, err := base64.StdEncoding.DecodeString(text); err == nil {
			return string(b)
		}
	}
	return text
}

// looksLikeCalendar flags responses carrying the calendar-data markers.
func looksLikeCalendar(resp string) bool {
	return strings.Contains(resp, "cald-") ||
		strings.Contains(resp, `"set"`) ||
		strings.Contains(resp, `"hda"`)
}

func paramsOf(e entry) []paramInfo {
	if p := e.Request.PostData.Params; len(p) > 0 {
		out := make([]paramInfo, len(p))
		for i, param := range p {
			out[i] = paramInfo{Name: param.Name, Size: len(param.Value)}
		}
		return out
	}
	if text := e.Request.PostData.Text; text != "" {
		if vals, err := url.ParseQuery(text); err == nil {
			out := make([]paramInfo, 0, len(vals))
			for k, vs := range vals {
				out = append(out, paramInfo{Name: k, Size: len(strings.Join(vs, ""))})
			}
			sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
			return out
		}
	}
	return nil
}

func head(s string, n int) string {
	s = strings.ReplaceAll(s, "\n", " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n]) + "…"
	}
	return s
}

func report(cands []candidate, limit int) string {
	lines := []string{fmt.Sprintf("%d requests in HAR; top candidates for the data endpoint:\n", len(cands))}
	for i, c := range cands {
		if i >= limit {
			break
		}
		tag := ""
		if c.Calendar {
			tag = " [CALENDAR?]"
		}
		lines = append(lines,
			fmt.Sprintf("#%d%s %s %s", i+1, tag, c.Method, c.URL),
			fmt.Sprintf("    status=%d respSize=%d", c.Status, c.RespSize),
		)
		if len(c.Params) > 0 {
			parts := make([]string, len(c.Params))
			for j, p := range c.Params {
				parts[j] = fmt.Sprintf("%s(%d)", p.Name, p.Size)
			}
			lines = append(lines, "    req params: "+strings.Join(parts, ", "))
		}
		if c.RespHead != "" {
			lines = append(lines, "    resp head: "+c.RespHead)
		}
		lines = append(lines, "")
	}
	return strings.Join(lines, "\n")
}
