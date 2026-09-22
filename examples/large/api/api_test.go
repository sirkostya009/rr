package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func get(t *testing.T, path string) (int, string) {
	t.Helper()
	w := httptest.NewRecorder()
	(&Api{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	var body string
	if w.Code == http.StatusOK {
		if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
	}
	return w.Code, body
}

func TestEveryRoute(t *testing.T) {
	for i := range Count {
		for path, want := range map[string]string{
			fmt.Sprintf("/large/static/page-%03d", i):            fmt.Sprintf("static-%03d", i),
			fmt.Sprintf("/large/versions/v%03d/items", i):        fmt.Sprintf("v%03d:items", i),
			fmt.Sprintf("/large/tenants/tenant-%03d/items/x", i): fmt.Sprintf("tenant-%03d:x", i),
			fmt.Sprintf("/large/words/%s/y", Words[i]):           fmt.Sprintf("word-%03d:y", i),
		} {
			if code, got := get(t, path); code != http.StatusOK || got != want {
				t.Fatalf("%s: %d %q, want %q", path, code, got, want)
			}
		}
	}
}

// Impostors land in a real key's hash slot (the hashed bytes and length
// match) and must be turned away by the verifying compare.
func TestImpostors(t *testing.T) {
	for _, path := range []string{
		"/large/static/pagE-017",
		"/large/static/page-999",
		"/large/static/page-01",
		"/large/tenants/tenanT-017/items/x",
		"/large/tenants/tenant-260/items/x",
		"/large/tenants/tenant-017/item/x",
		"/large/words/" + strings.ToUpper(Words[17][:1]) + Words[17][1:] + "/y",
		"/large/words/" + Words[17] + "x/y",
		"/large/words//y",
	} {
		if code, _ := get(t, path); code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, code)
		}
	}
}

// The fixture is only worth keeping while it still reaches the hash path.
func TestCandidates(t *testing.T) {
	for path, want := range map[string]string{
		"/large/catalog/42/number":        "number:42",
		"/large/catalog/deadbeef/hex":     "hex:deadbeef",
		"/large/catalog/true/toggle":      "toggle:true",
		"/large/catalog/anything/profile": "profile:anything",
		"/large/catalog/42/profile":       "profile:42", // literal picks the candidate, not the value's shape
		"/large/docs/7/edit":              "edit-id:7",
		"/large/docs/intro/edit":          "edit-slug:intro",
	} {
		if code, got := get(t, path); code != http.StatusOK || got != want {
			t.Errorf("%s: %d %q, want %q", path, code, got, want)
		}
	}
	for _, path := range []string{
		"/large/catalog/x/number",    // fails the int validator
		"/large/catalog/nothex/hex",  // fails the regexp
		"/large/catalog/2/toggle",    // ParseBool rejects "2"
		"/large/catalog/42/nope",     // no such literal
		"/large/catalog/42/number/x", // trailing segment
	} {
		if code, _ := get(t, path); code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", path, code)
		}
	}
}

// The candidate shapes must keep generating what they are here to pin.
func TestCandidateShapes(t *testing.T) {
	src, err := os.ReadFile("api_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(src), "switch path[i+1:] {"); n != 1 {
		t.Fatalf("%d literal-first candidate switches, want 1 (catalog)", n)
	}
	// the docs pair shares "edit": ordered chain, int candidate tried first
	i := strings.Index(string(src), `r.Pattern = "GET /large/docs/{id}/edit"`)
	j := strings.Index(string(src), `r.Pattern = "GET /large/docs/{slug}/edit"`)
	if i < 0 || j < 0 || i > j {
		t.Fatalf("docs candidates: id at %d, slug at %d, want id first", i, j)
	}
}

func TestHashed(t *testing.T) {
	src, err := os.ReadFile("api_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(src), "switch (("); n != 4 {
		t.Fatalf("%d hashed switches in api_gen.go, want 4", n)
	}
}
