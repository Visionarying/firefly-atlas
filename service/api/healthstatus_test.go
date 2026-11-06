package api

import (
	"testing"

	"firefly-atlas/models"
)

func TestParseClockMinutes(t *testing.T) {
	cases := []struct {
		in     string
		want   int
		wantOK bool
	}{
		{"00:00", 0, true},
		{"02:30", 150, true},
		{"23:59", 1439, true},
		{" 9:05 ", 545, true},
		{"24:00", 0, false},
		{"12:60", 0, false},
		{"noon", 0, false},
		{"", 0, false},
	}
	for _, tc := range cases {
		got, ok := parseClockMinutes(tc.in)
		if ok != tc.wantOK || (ok && got != tc.want) {
			t.Errorf("parseClockMinutes(%q) = (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.wantOK)
		}
	}
}

func TestHealthURLsPreferLanAndDedupe(t *testing.T) {
	item := models.ItemIcon{LanUrl: " http://192.168.1.10:5000 ", Url: "https://nas.example.com"}
	got := healthURLs(item)
	if len(got) != 2 || got[0] != "http://192.168.1.10:5000" || got[1] != "https://nas.example.com" {
		t.Fatalf("unexpected urls: %#v", got)
	}

	same := models.ItemIcon{LanUrl: "http://192.168.1.1", Url: "http://192.168.1.1"}
	if got := healthURLs(same); len(got) != 1 {
		t.Fatalf("expected deduplicated url, got %#v", got)
	}

	empty := models.ItemIcon{}
	if got := healthURLs(empty); len(got) != 0 {
		t.Fatalf("expected no urls, got %#v", got)
	}
}
