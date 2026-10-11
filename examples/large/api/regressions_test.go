package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func send(t *testing.T, method, path, body string) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	(&Api{}).ServeHTTP(w, httptest.NewRequest(method, path, strings.NewReader(body)))
	return w.Code, strings.TrimSpace(w.Body.String())
}

func TestFixedWidthUnderTerminalParam(t *testing.T) {
	for path, want := range map[string]string{
		"/large/fw/zz":      "x:zz",
		"/large/fw/abc/7/q": "abc:7",
		"/large/fw/cde/8/q": "cde:8",
	} {
		if code, got := get(t, path); code != http.StatusOK || got != want {
			t.Errorf("%s: %d %q, want %q", path, code, got, want)
		}
	}
	// these used to reach ABC/CDE with a bogus y, or panic on a short segment
	for _, path := range []string{"/large/fw/abcXd/q", "/large/fw/cdeZZ/q", "/large/fw/a/", "/large/fw/ab/x"} {
		if code, _ := get(t, path); code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, code)
		}
	}
}

func TestBodyKeepsParam(t *testing.T) {
	if code, got := send(t, http.MethodPost, "/large/counters/7", "99"); code != http.StatusOK || got != `"7:99"` {
		t.Fatalf("got %d %s, want 200 \"7:99\"", code, got)
	}
}

func TestKeptBodyNotOverwritten(t *testing.T) {
	code, got := send(t, http.MethodPost, "/large/jobs", `{"id":5,"status":"new"}`)
	var res Job
	if err := json.Unmarshal([]byte(got), &res); code != http.StatusOK || err != nil || res.ID != 1005 {
		t.Fatalf("response %d %s", code, got)
	}
	if queued == nil || *queued != (Job{ID: 5, Status: "new"}) {
		t.Fatalf("queued body = %+v, want the decoded body {5 new}", queued)
	}
}

// a body type elsewhere in the dispatcher must cost unrelated routes nothing
func TestUnrelatedRouteAllocs(t *testing.T) {
	a := &Api{}
	w := &nopWriter{h: http.Header{}}
	for _, path := range []string{"/large/ping", "/large/nope"} {
		req := httptest.NewRequest(http.MethodGet, path, nil)
		if n := testing.AllocsPerRun(100, func() { a.ServeHTTP(w, req) }); n != 0 {
			t.Errorf("%s: %v allocs/op, want 0", path, n)
		}
	}
}

func TestSlotNamesDoNotShadow(t *testing.T) {
	for path, want := range map[string]string{
		"/large/users/me":    `{"name":"me"}`,
		"/large/users/login": `{"at":"now"}`,
		"/large/a":           `{"n":1}`,
		"/large/items":       `"item:x"`,
	} {
		method := http.MethodGet
		if path == "/large/items" || path == "/large/users/login" {
			method = http.MethodPost
		}
		if code, got := send(t, method, path, `{"name":"x","at":"now"}`); code != http.StatusOK || got != want {
			t.Errorf("%s %s: %d %s, want %s", method, path, code, got, want)
		}
	}
}

func TestUnnamedCatchAllBesideParam(t *testing.T) {
	for path, want := range map[string]string{
		"/large/star/p/a":       "a:p",
		"/large/star/p/a/x/y/z": "any:p",
		"/large/star/p/q/b":     "b:p/q",
	} {
		if code, got := get(t, path); code != http.StatusOK || got != want {
			t.Errorf("%s: %d %q, want %q", path, code, got, want)
		}
	}
}
