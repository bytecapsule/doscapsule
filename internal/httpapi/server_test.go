package httpapi

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/bytecapsule/doscapsule/internal/app"
)

// TestHandlerHostAndCSRF exercises the security wrapper directly, without
// binding the fixed 8420 port or opening a browser.
func TestHandlerHostAndCSRF(t *testing.T) {
	t.Setenv("DOSCAPSULE_DATA_DIR", filepath.Join(t.TempDir(), "state"))
	a := app.New()
	ui := fstest.MapFS{
		"index.html": &fstest.MapFile{Data: []byte("<!doctype html><title>ok</title>")},
	}

	ts := httptest.NewUnstartedServer(nil)
	host := ts.Listener.Addr().String()
	ts.Config.Handler = newHandler(a, ui, []byte("jpeg"), host)
	ts.Start()
	defer ts.Close()

	client := ts.Client()

	get := func(h string) *http.Response {
		t.Helper()
		req, err := http.NewRequest(http.MethodGet, ts.URL+"/", nil)
		if err != nil {
			t.Fatal(err)
		}
		if h != "" {
			req.Host = h
		}
		resp, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		return resp
	}

	resp := get("")
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET / = %d, want 200", resp.StatusCode)
	}
	if got := resp.Header.Get("Cache-Control"); got != "no-store" {
		t.Errorf("HTML Cache-Control = %q, want no-store", got)
	}
	resp.Body.Close()

	resp = get("evil.example")
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("wrong Host = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	resp, err := client.Get(ts.URL + "/api/games")
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/games = %d, want 200", resp.StatusCode)
	}
	resp.Body.Close()

	// A cross-site browser POST must be rejected before it reaches a handler.
	req, err := http.NewRequest(http.MethodPost, ts.URL+"/api/stop", strings.NewReader(`{"id":"x"}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Origin", "https://evil.example")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("cross-site POST = %d, want 403", resp.StatusCode)
	}
	resp.Body.Close()

	// No Origin/Sec-Fetch (curl, CLI): CSRF passes and the handler sees the
	// request. An empty id is the handler's own 400, not the wrapper's 403.
	req, err = http.NewRequest(http.MethodPost, ts.URL+"/api/stop", strings.NewReader(`{"id":""}`))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err = client.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("same-origin POST = %d, want 400 from the handler", resp.StatusCode)
	}
	resp.Body.Close()
}
