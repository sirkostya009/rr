package api

import (
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	v1 "versioned/api/v1"
	v2 "versioned/api/v2"
)

// seed ids from versioned/services
const (
	guyID        = "5e0cf5ae-a2fd-4b71-96ea-1f2ee89f2b3a"
	helloWorldID = "6b2d9c7e-49a1-45b6-8a3f-0d8f4f4e9a01"
	genericsID   = "9f1a3d52-7c88-4a0b-b1de-52f0a1c6e402"
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

func TestForms(t *testing.T) {
	a := &Api{}

	// headers struct via `header:` tags
	got := fromJSON[map[string]string](t, do(t, a, "GET", "/api/v1/meta", "", "X-Request-Id", "abc", "X-Trace", "t1"))
	if got["rid"] != "abc" || got["trace"] != "t1" {
		t.Errorf("meta: %v", got)
	}

	// url.Values body (urlencoded)
	req := httptest.NewRequest("POST", "/api/v1/forms/login", strings.NewReader("user=bob&pw=x"))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if login := fromJSON[map[string]string](t, rec); login["user"] != "bob" {
		t.Errorf("login: %v", login)
	}

	// multipart.Form body
	var buf strings.Builder
	mw := multipart.NewWriter(&buf)
	_ = mw.WriteField("title", "hi")
	fw, _ := mw.CreateFormFile("doc", "a.txt")
	fw.Write([]byte("hello"))
	mw.Close()
	req = httptest.NewRequest("POST", "/api/v1/forms/upload", strings.NewReader(buf.String()))
	req.Header.Set("Content-Type", mw.FormDataContentType())
	rec = httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if up := fromJSON[map[string]int](t, rec); up["fields"] != 1 || up["files"] != 1 {
		t.Errorf("upload: %v", up)
	}
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

	if got := fromJSON[[]v1.User](t, do(t, a, "GET", "/api/v1/users", "")); len(got) != 1 || got[0].Name != "Guy" {
		t.Fatalf("GET /api/v1/users: %v", got)
	}

	if got := fromJSON[[]v1.User](t, do(t, a, "POST", "/api/v1/users", `{"name":"Gal"}`)); len(got) != 2 {
		t.Fatalf("POST /api/v1/users: %v", got)
	}

	req := httptest.NewRequest("GET", "/api/v1/users/1", nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if u := fromJSON[v1.User](t, rec); u.Name != "Gal" {
		t.Errorf("GET /api/v1/users/1: %v", u)
	}
	if req.Pattern != "GET /api/v1/users/{i}" {
		t.Errorf("Pattern = %q", req.Pattern)
	}

	// v1 supports uuids too, they just aren't the default: uuid-shaped
	// segments route past the int param to the @isUUID checker
	if u := fromJSON[v1.User](t, do(t, a, "GET", "/api/v1/users/"+guyID, "")); u.Name != "Guy" {
		t.Errorf("GET /api/v1/users/{uuid}: %v", u)
	}

	// out of range: (User, error) routes through the central onerror
	if rec := do(t, a, "GET", "/api/v1/users/99", ""); rec.Code != 500 {
		t.Errorf("GET /api/v1/users/99: %d", rec.Code)
	}
	// neither numeric nor uuid-shaped never reaches a handler
	if rec := do(t, a, "GET", "/api/v1/users/abc", ""); rec.Code != 404 {
		t.Errorf("GET /api/v1/users/abc: %d", rec.Code)
	}

	if rec := do(t, a, "DELETE", "/api/v1/users/1", ""); rec.Code != 200 {
		t.Errorf("DELETE /api/v1/users/1: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, a, "DELETE", "/api/v1/users/42", ""); rec.Code != 500 {
		t.Errorf("DELETE /api/v1/users/42: %d", rec.Code)
	}
}

func TestPosts(t *testing.T) {
	a := &Api{}

	if got := fromJSON[[]v1.Post](t, do(t, a, "GET", "/api/v1/posts", "")); len(got) != 2 {
		t.Fatalf("GET /api/v1/posts: %v", got)
	}
	// struct query binding filters by tag
	if got := fromJSON[[]v1.Post](t, do(t, a, "GET", "/api/v1/posts?tag=go&page=1", "")); len(got) != 1 || got[0].Slug != "generics-in-anger" {
		t.Fatalf("GET /api/v1/posts?tag=go: %v", got)
	}
	// the 400 handler binds X-Request-Id (beyond w) and echoes it back
	rec := do(t, a, "GET", "/api/v1/posts?page=x", "", "X-Request-Id", "rq-9")
	if rec.Code != 400 || rec.Header().Get("X-Bad-Request") != "1" || rec.Header().Get("X-Request-Id") != "rq-9" {
		t.Errorf("bad page: %d, X-Request-Id %q", rec.Code, rec.Header().Get("X-Request-Id"))
	}

	// pointer body, (T, error) return; the service assigns the next int id
	if got := fromJSON[v1.Post](t, do(t, a, "POST", "/api/v1/posts", `{"slug":"third","title":"Third"}`)); got.ID != 3 {
		t.Fatalf("POST /api/v1/posts: %+v", got)
	}
	// slug validation runs inside the ggen decode (pipe:"@isSlug"), so a
	// bad slug is a 400 with the validation error, not a handler-level 500
	if rec := do(t, a, "POST", "/api/v1/posts", `{"slug":"NOT OK"}`); rec.Code != 400 || rec.Body.Len() == 0 {
		t.Errorf("bad slug: %d %q", rec.Code, rec.Body.String())
	}
	if rec := do(t, a, "POST", "/api/v1/posts", `{oops`); rec.Code != 400 {
		t.Errorf("bad body: %d", rec.Code)
	}

	// {id} int and {slug=@isSlug} share GET /api/v1/posts/{...}: ints dispatch
	// to GetPost, slugs to GetPostBySlug
	if got := fromJSON[v1.Post](t, do(t, a, "GET", "/api/v1/posts/1", "")); got.Slug != "hello-world" {
		t.Errorf("GET /api/v1/posts/1: %+v", got)
	}
	if rec := do(t, a, "GET", "/api/v1/posts/999", ""); rec.Code != 404 {
		t.Errorf("missing post: %d, want postNotFound's 404", rec.Code)
	}
	if got := fromJSON[v1.Post](t, do(t, a, "GET", "/api/v1/posts/third", "")); got.ID != 3 {
		t.Errorf("GET /api/v1/posts/third: %+v", got)
	}
	if rec := do(t, a, "GET", "/api/v1/posts/NOT-OK", ""); rec.Code != 404 {
		t.Errorf("checker reject: %d", rec.Code)
	}

	// string params always land in PathValue, argument binding or not
	req := httptest.NewRequest("GET", "/api/v1/posts/third", nil)
	a.ServeHTTP(httptest.NewRecorder(), req)
	if req.PathValue("slug") != "third" {
		t.Errorf("PathValue(slug) = %q", req.PathValue("slug"))
	}

	// the posts subtree is GET-only, so its method gate is hoisted: a wrong
	// method beats a structural mismatch
	if rec := do(t, a, "PUT", "/api/v1/posts/NOT-OK", ""); rec.Code != 405 {
		t.Errorf("hoisted method gate: %d, want 405", rec.Code)
	}

	// two auto-bound int params
	if got := fromJSON[map[string]int](t, do(t, a, "GET", "/api/v1/posts/3/comments/9", "")); got["post"] != 3 || got["comment"] != 9 {
		t.Errorf("comments: %v", got)
	}
	if rec := do(t, a, "GET", "/api/v1/posts/x/comments/9", ""); rec.Code != 404 {
		t.Errorf("bad pid: %d", rec.Code)
	}
}

func TestAdmin(t *testing.T) {
	a := &Api{}

	if rec := do(t, a, "GET", "/api/v1/admin/stats", ""); rec.Code != 401 {
		t.Errorf("no token: %d", rec.Code)
	}
	got := fromJSON[map[string]int](t, do(t, a, "GET", "/api/v1/admin/stats", "", "Authorization", "Bearer letmein"))
	if got["users"] < 1 || got["posts"] < 2 {
		t.Errorf("stats: %v", got)
	}
	// catch-all route takes any method
	if rec := do(t, a, "PURGE", "/api/v1/admin/maintenance", "", "Authorization", "Bearer letmein"); rec.Code != 202 {
		t.Errorf("maintenance: %d", rec.Code)
	}

	// mounted apis serve only through their version's central
	var h any = &v1.AdminApi{}
	if _, ok := h.(http.Handler); ok {
		t.Error("v1.AdminApi should not implement http.Handler")
	}
}

func TestSearch(t *testing.T) {
	a := &Api{}

	got := fromJSON[map[string]any](t, do(t, a, "GET", "/api/v1/search?q=go&tag=a&tag=b", "", "X-Request-Id", "r1"))
	q := got["query"].(map[string]any)
	if q["term"] != "go" || got["rid"] != "r1" || len(q["tags"].([]any)) != 2 {
		t.Errorf("search: %v", got)
	}
	if rec := do(t, a, "GET", "/api/v1/search", ""); rec.Code != 400 {
		t.Errorf("missing q: %d", rec.Code)
	}

	if got := fromJSON[map[string][]string](t, do(t, a, "GET", "/api/v1/debug/query?a=1&a=2&b=3", "")); len(got["a"]) != 2 || got["b"][0] != "3" {
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
		got := fromJSON[map[string]any](t, do(t, a, "GET", "/api/v1/echo/"+c.seg, ""))
		if got[c.key] != c.want {
			t.Errorf("echo %s: got %v, want %s=%v", c.seg, got, c.key, c.want)
		}
	}
}

func TestV2Users(t *testing.T) {
	a := &Api{}

	// v1 tests above left only Guy in the shared service
	got := fromJSON[[]v2.User](t, do(t, a, "GET", "/api/v2/users", ""))
	if len(got) != 1 || got[0].ID != guyID {
		t.Fatalf("GET /api/v2/users: %v", got)
	}

	created := fromJSON[v2.User](t, do(t, a, "POST", "/api/v2/users", `{"name":"Gal"}`))
	if created.Name != "Gal" || len(created.ID) != 36 {
		t.Fatalf("POST /api/v2/users: %+v", created)
	}

	req := httptest.NewRequest("GET", "/api/v2/users/"+created.ID, nil)
	rec := httptest.NewRecorder()
	a.ServeHTTP(rec, req)
	if u := fromJSON[v2.User](t, rec); u.Name != "Gal" {
		t.Errorf("GET /api/v2/users/{id}: %v", u)
	}
	if req.Pattern != "GET /api/v2/users/{id}" {
		t.Errorf("Pattern = %q", req.Pattern)
	}

	// unknown uuid: (User, error) routes through the v2 central onerror
	if rec := do(t, a, "GET", "/api/v2/users/"+missingID, ""); rec.Code != 500 {
		t.Errorf("missing user: %d", rec.Code)
	}

	if rec := do(t, a, "DELETE", "/api/v2/users/"+created.ID, ""); rec.Code != 200 {
		t.Errorf("DELETE /api/v2/users/{id}: %d %s", rec.Code, rec.Body.String())
	}
	if rec := do(t, a, "DELETE", "/api/v2/users/"+created.ID, ""); rec.Code != 500 {
		t.Errorf("double delete: %d", rec.Code)
	}
}

const missingID = "00000000-0000-4000-8000-000000000000"

func TestV2Posts(t *testing.T) {
	a := &Api{}

	// list carries uuid ids, no ints anywhere
	got := fromJSON[[]v2.Post](t, do(t, a, "GET", "/api/v2/posts", ""))
	if len(got) < 2 || got[0].ID != helloWorldID {
		t.Fatalf("GET /api/v2/posts: %v", got)
	}
	if gotag := fromJSON[[]v2.Post](t, do(t, a, "GET", "/api/v2/posts?tag=rant", "")); len(gotag) != 1 || gotag[0].ID != genericsID {
		t.Fatalf("GET /api/v2/posts?tag=rant: %v", gotag)
	}

	created := fromJSON[v2.Post](t, do(t, a, "POST", "/api/v2/posts", `{"slug":"fourth","title":"Fourth"}`))
	if len(created.ID) != 36 {
		t.Fatalf("POST /api/v2/posts: %+v", created)
	}

	// uuid and slug segments are both strings: checkers split them, uuid first
	if p := fromJSON[v2.Post](t, do(t, a, "GET", "/api/v2/posts/"+created.ID, "")); p.Slug != "fourth" {
		t.Errorf("GET /api/v2/posts/{id}: %+v", p)
	}
	if p := fromJSON[v2.Post](t, do(t, a, "GET", "/api/v2/posts/hello-world", "")); p.ID != helloWorldID {
		t.Errorf("GET /api/v2/posts/{slug}: %+v", p)
	}
	// controller onerror maps service errors to 404
	if rec := do(t, a, "GET", "/api/v2/posts/"+missingID, ""); rec.Code != 404 {
		t.Errorf("missing post: %d", rec.Code)
	}
	if rec := do(t, a, "GET", "/api/v2/posts/NOT-OK", ""); rec.Code != 404 {
		t.Errorf("checker reject: %d", rec.Code)
	}

	// two uuid-checked string params
	if got := fromJSON[map[string]string](t, do(t, a, "GET", "/api/v2/posts/"+created.ID+"/comments/"+genericsID, "")); got["post"] != created.ID || got["comment"] != genericsID {
		t.Errorf("comments: %v", got)
	}
	if rec := do(t, a, "GET", "/api/v2/posts/nope/comments/"+genericsID, ""); rec.Code != 404 {
		t.Errorf("bad pid: %d", rec.Code)
	}
}

func TestV2Admin(t *testing.T) {
	a := &Api{}

	if rec := do(t, a, "GET", "/api/v2/admin/stats", ""); rec.Code != 401 {
		t.Errorf("no token: %d", rec.Code)
	}
	got := fromJSON[map[string]int](t, do(t, a, "GET", "/api/v2/admin/stats", "", "Authorization", "Bearer letmein"))
	if got["users"] < 1 || got["posts"] < 2 {
		t.Errorf("stats: %v", got)
	}
	if rec := do(t, a, "PURGE", "/api/v2/admin/maintenance", "", "Authorization", "Bearer letmein"); rec.Code != 202 {
		t.Errorf("maintenance: %d", rec.Code)
	}
}

func TestComposition(t *testing.T) {
	// the composed Api delegates by prefix: v1 and v2 both route through the
	// one generated ServeHTTP, each version keeping its own error handlers
	a := &Api{}

	if got := fromJSON[map[string]string](t, do(t, a, "GET", "/api/v1/files/a.css", "")); got["path"] != "a.css" {
		t.Errorf("v1 through composition: %v", got)
	}
	if rec := do(t, a, "GET", "/api/v1/nope", ""); rec.Code != 404 {
		t.Errorf("v1 404: %d", rec.Code)
	}
	if rec := do(t, a, "PUT", "/api/v1/posts/1", ""); rec.Code != 405 {
		t.Errorf("v1 405: %d", rec.Code)
	}
	// v2 paths route through the same handler
	if rec := do(t, a, "GET", "/api/v2/nope", ""); rec.Code != 404 {
		t.Errorf("v2 404: %d", rec.Code)
	}
}

func TestFiles(t *testing.T) {
	a := &Api{}
	if got := fromJSON[map[string]string](t, do(t, a, "GET", "/api/v1/files/css/site.css", "")); got["path"] != "css/site.css" {
		t.Errorf("files: %v", got)
	}
	// unannotated and mounted: no standalone handler
	var h any = &v1.FilesApi{}
	if _, ok := h.(http.Handler); ok {
		t.Error("v1.FilesApi should not implement http.Handler")
	}
}
