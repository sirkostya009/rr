package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func do(t *testing.T, h http.Handler, method, path, body string, hdr ...string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	for i := 0; i+1 < len(hdr); i += 2 {
		req.Header.Set(hdr[i], hdr[i+1])
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func fromJSON[T any](t *testing.T, rec *httptest.ResponseRecorder) T {
	t.Helper()
	if ct := rec.Header().Get("Content-Type"); ct != "application/json" {
		t.Fatalf("Content-Type = %q, body %q", ct, rec.Body.String())
	}
	var v T
	if err := json.Unmarshal(rec.Body.Bytes(), &v); err != nil {
		t.Fatalf("bad json %q: %v", rec.Body.String(), err)
	}
	return v
}

func TestUsers(t *testing.T) {
	a := &Api{}

	if got := fromJSON[[]User](t, do(t, a, "GET", "/api/users", "")); len(got) != 1 || got[0].Name != "Guy" {
		t.Fatalf("GET /api/users: %v", got)
	}

	if got := fromJSON[[]User](t, do(t, a, "POST", "/api/users", `{"Name":"Gal"}`)); len(got) != 2 {
		t.Fatalf("POST /api/users: %v", got)
	}

	req := httptest.NewRequest("GET", "/api/users/1", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if u := fromJSON[User](t, rec); u.Name != "Gal" {
		t.Errorf("GET /api/users/1: %v", u)
	}
	if req.Pattern != "GET /api/users/{i}" {
		t.Errorf("Pattern = %q", req.Pattern)
	}

	// out of range: (User, error) routes through the central onerror
	if rec := do(t, a, "GET", "/api/users/99", ""); rec.Code != 500 {
		t.Errorf("GET /api/users/99: %d", rec.Code)
	}
	// non-numeric never matches the int param
	if rec := do(t, a, "GET", "/api/users/abc", ""); rec.Code != 404 {
		t.Errorf("GET /api/users/abc: %d", rec.Code)
	}

	if rec := do(t, a, "DELETE", "/api/users/1", ""); rec.Code != 200 {
		t.Errorf("DELETE /api/users/1: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, a, "DELETE", "/api/users/42", ""); rec.Code != 500 {
		t.Errorf("DELETE /api/users/42: %d", rec.Code)
	}
}

func TestPosts(t *testing.T) {
	a := &Api{}

	if got := fromJSON[[]Post](t, do(t, a, "GET", "/api/posts", "")); len(got) != 2 {
		t.Fatalf("GET /api/posts: %v", got)
	}
	// struct query binding filters by tag
	if got := fromJSON[[]Post](t, do(t, a, "GET", "/api/posts?tag=go&page=1", "")); len(got) != 1 || got[0].Slug != "generics-in-anger" {
		t.Fatalf("GET /api/posts?tag=go: %v", got)
	}
	if rec := do(t, a, "GET", "/api/posts?page=x", ""); rec.Code != 400 || rec.Header().Get("X-Bad-Request") != "1" {
		t.Errorf("bad page: %d", rec.Code)
	}

	// pointer body, (T, error) return
	if got := fromJSON[Post](t, do(t, a, "POST", "/api/posts", `{"slug":"third","title":"Third"}`)); got.ID != 3 {
		t.Fatalf("POST /api/posts: %+v", got)
	}
	// slug validation now runs inside the ggen decode (pipe:"@isSlug"), so a
	// bad slug is a 400 with the validation error, not a handler-level 500
	if rec := do(t, a, "POST", "/api/posts", `{"slug":"NOT OK"}`); rec.Code != 400 || rec.Body.Len() == 0 {
		t.Errorf("bad slug: %d %q", rec.Code, rec.Body.String())
	}
	if rec := do(t, a, "POST", "/api/posts", `{oops`); rec.Code != 400 {
		t.Errorf("bad body: %d", rec.Code)
	}

	// {id} int and {slug=@isSlug} share GET /api/posts/{...}: ints dispatch
	// to GetPost (with its route-level errorhandler), slugs to GetPostBySlug
	if got := fromJSON[Post](t, do(t, a, "GET", "/api/posts/1", "")); got.Slug != "hello-world" {
		t.Errorf("GET /api/posts/1: %+v", got)
	}
	if rec := do(t, a, "GET", "/api/posts/999", ""); rec.Code != 404 {
		t.Errorf("missing post: %d, want postNotFound's 404", rec.Code)
	}
	if got := fromJSON[Post](t, do(t, a, "GET", "/api/posts/third", "")); got.ID != 3 {
		t.Errorf("GET /api/posts/third: %+v", got)
	}
	if rec := do(t, a, "GET", "/api/posts/NOT-OK", ""); rec.Code != 404 {
		t.Errorf("checker reject: %d", rec.Code)
	}

	// string params always land in PathValue, argument binding or not
	req := httptest.NewRequest("GET", "/api/posts/third", nil)
	a.ServeHTTP(httptest.NewRecorder(), req)
	if req.PathValue("slug") != "third" {
		t.Errorf("PathValue(slug) = %q", req.PathValue("slug"))
	}

	// the posts subtree is GET-only, so its method gate is hoisted: a wrong
	// method beats a structural mismatch
	if rec := do(t, a, "PUT", "/api/posts/NOT-OK", ""); rec.Code != 405 {
		t.Errorf("hoisted method gate: %d, want 405", rec.Code)
	}

	// two auto-bound int params
	if got := fromJSON[map[string]int](t, do(t, a, "GET", "/api/posts/3/comments/9", "")); got["post"] != 3 || got["comment"] != 9 {
		t.Errorf("comments: %v", got)
	}
	if rec := do(t, a, "GET", "/api/posts/x/comments/9", ""); rec.Code != 404 {
		t.Errorf("bad pid: %d", rec.Code)
	}
}

func TestAdmin(t *testing.T) {
	a := &Api{}

	if rec := do(t, a, "GET", "/api/admin/stats", ""); rec.Code != 401 {
		t.Errorf("no token: %d", rec.Code)
	}
	got := fromJSON[map[string]int](t, do(t, a, "GET", "/api/admin/stats", "", "Authorization", "Bearer letmein"))
	if got["users"] < 1 || got["posts"] < 2 {
		t.Errorf("stats: %v", got)
	}
	// catch-all route takes any method
	if rec := do(t, a, "PURGE", "/api/admin/maintenance", "", "Authorization", "Bearer letmein"); rec.Code != 202 {
		t.Errorf("maintenance: %d", rec.Code)
	}

	// mounted apis serve only through the central, annotations or not
	var h any = &AdminApi{}
	if _, ok := h.(http.Handler); ok {
		t.Error("AdminApi should not implement http.Handler")
	}
}

func TestSearch(t *testing.T) {
	a := &Api{}

	got := fromJSON[map[string]any](t, do(t, a, "GET", "/api/search?q=go&tag=a&tag=b", "", "X-Request-Id", "r1"))
	q := got["query"].(map[string]any)
	if q["term"] != "go" || got["rid"] != "r1" || len(q["tags"].([]any)) != 2 {
		t.Errorf("search: %v", got)
	}
	if rec := do(t, a, "GET", "/api/search", ""); rec.Code != 400 {
		t.Errorf("missing q: %d", rec.Code)
	}

	if got := fromJSON[map[string][]string](t, do(t, a, "GET", "/api/debug/query?a=1&a=2&b=3", "")); len(got["a"]) != 2 || got["b"][0] != "3" {
		t.Errorf("debug query: %v", got)
	}
}

func TestEchoTypedDispatch(t *testing.T) {
	a := &Api{}
	cases := []struct {
		seg, key string
		want     any
	}{
		{"true", "bool", true},
		{"3.14", "float", 3.14},
		{"hello", "string", "hello"},
		{"1", "float", 1.0},    // numerics rank first, digits never reach ParseBool
		{"TRUE", "bool", true}, // ParseBool's lax spellings are back in
		{"maybe", "string", "maybe"},
	}
	for _, c := range cases {
		got := fromJSON[map[string]any](t, do(t, a, "GET", "/api/echo/"+c.seg, ""))
		if got[c.key] != c.want {
			t.Errorf("echo %s: got %v, want %s=%v", c.seg, got, c.key, c.want)
		}
	}
}

func TestServer(t *testing.T) {
	// with no sentry.Init the SDK is disabled: spans are no-ops, so the
	// wrapper must be a transparent passthrough
	srv := &Server{}

	if got := fromJSON[map[string]string](t, do(t, srv, "GET", "/api/files/a.css", "")); got["path"] != "a.css" {
		t.Errorf("through server: %v", got)
	}
	if rec := do(t, srv, "GET", "/api/nope", ""); rec.Code != 404 {
		t.Errorf("server 404: %d", rec.Code)
	}
	if rec := do(t, srv, "PUT", "/api/posts/1", ""); rec.Code != 405 {
		t.Errorf("server 405: %d", rec.Code)
	}
}

func TestFiles(t *testing.T) {
	a := &Api{}
	if got := fromJSON[map[string]string](t, do(t, a, "GET", "/api/files/css/site.css", "")); got["path"] != "css/site.css" {
		t.Errorf("files: %v", got)
	}
	// unannotated and mounted: no standalone handler
	var h any = &FilesApi{}
	if _, ok := h.(http.Handler); ok {
		t.Error("FilesApi should not implement http.Handler")
	}
}
