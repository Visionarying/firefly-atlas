package api

import (
	"net/http"
	"testing"
)

func mustRequest(t *testing.T, raw string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, raw, nil)
	if err != nil {
		t.Fatalf("new request %q: %v", raw, err)
	}
	return req
}

func TestIsSafeHealthURLRejectsMetadataAndBadSchemes(t *testing.T) {
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://[::ffff:169.254.169.254]/latest/meta-data/",
		"http://100.100.100.200/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
		"ftp://192.168.1.10/",
		"file:///etc/passwd",
		"not-a-url",
	}
	for _, raw := range blocked {
		if _, err := isSafeHealthURL(raw); err == nil {
			t.Errorf("isSafeHealthURL(%q) = nil error, want rejection", raw)
		}
	}
}

func TestIsSafeHealthURLAllowsLANAndLoopback(t *testing.T) {
	allowed := []string{
		"http://192.168.1.10:5000",
		"https://nas.example.com",
		"http://127.0.0.1:8080",
		"http://[::1]:9000",
	}
	for _, raw := range allowed {
		if _, err := isSafeHealthURL(raw); err != nil {
			t.Errorf("isSafeHealthURL(%q) = %v, want allowed", raw, err)
		}
	}
}

// A redirect hop must go through the same policy as the original address.
func TestHealthHTTPRejectsRedirectToMetadata(t *testing.T) {
	blocked := []string{
		"http://169.254.169.254/latest/meta-data/",
		"http://metadata.google.internal/computeMetadata/v1/",
	}
	for _, raw := range blocked {
		if err := healthHTTP.CheckRedirect(mustRequest(t, raw), nil); err == nil {
			t.Errorf("redirect to %q should be rejected", raw)
		}
	}
	if err := healthHTTP.CheckRedirect(mustRequest(t, "http://192.168.1.10:5000/"), nil); err != nil {
		t.Errorf("redirect to a LAN address should be allowed, got %v", err)
	}
}

func TestHealthHTTPRejectsTooManyRedirects(t *testing.T) {
	via := make([]*http.Request, 5)
	if err := healthHTTP.CheckRedirect(mustRequest(t, "http://example.com"), via); err == nil {
		t.Error("expected more than five redirects to be rejected")
	}
}
