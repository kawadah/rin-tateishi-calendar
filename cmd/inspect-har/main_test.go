package main

import "testing"

const sampleHAR = `{
  "log": {
    "entries": [
      {
        "request": {
          "method": "GET",
          "url": "https://freecalend.com/font/x.woff2",
          "postData": {}
        },
        "response": {"status": 200, "content": {"size": 5000, "mimeType": "font/woff2", "text": ""}}
      },
      {
        "request": {
          "method": "POST",
          "url": "https://freecalend.com/open/data",
          "postData": {
            "text": "target_mem_no=231613&keys=%7B%22data%22%3A%5B%5D%7D&version=2"
          }
        },
        "response": {
          "status": 200,
          "content": {"size": 1200, "mimeType": "application/json", "text": "[[\"set\",\"cald-231613-2026-8-8\"]]"}
        }
      }
    ]
  }
}`

func TestAnalyzeRanksCalendarFirst(t *testing.T) {
	cands, err := analyze([]byte(sampleHAR))
	if err != nil {
		t.Fatal(err)
	}
	if len(cands) != 2 {
		t.Fatalf("got %d candidates, want 2", len(cands))
	}
	top := cands[0]
	if !top.Calendar {
		t.Error("calendar-looking response should rank first")
	}
	if top.URL != "https://freecalend.com/open/data" {
		t.Errorf("top URL = %q", top.URL)
	}
	// Params parsed from the urlencoded body.
	names := map[string]int{}
	for _, p := range top.Params {
		names[p.Name] = p.Size
	}
	if _, ok := names["keys"]; !ok {
		t.Errorf("expected a 'keys' param, got %v", top.Params)
	}
	if names["target_mem_no"] != 6 {
		t.Errorf("target_mem_no size = %d, want 6", names["target_mem_no"])
	}
}

func TestAnalyzeRejectsBadJSON(t *testing.T) {
	if _, err := analyze([]byte("not har")); err == nil {
		t.Fatal("expected error for invalid HAR")
	}
}
