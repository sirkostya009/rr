package nopv

import (
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestRoutes(t *testing.T) {
	for path, want := range map[string]string{
		"/nopv/seg/q":         `"one:q"`,
		"/nopv/seg/q/b":       `"two"`,
		"/nopv/rest/true":     `"b:true"`,
		"/nopv/rest/true/x/y": `"w:true"`,
		"/nopv/rest/foo/x":    `"s"`,
	} {
		w := httptest.NewRecorder()
		(&Api{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if got := strings.TrimSpace(w.Body.String()); w.Code != http.StatusOK || got != want {
			t.Errorf("%s: %d %s, want %s", path, w.Code, got, want)
		}
	}
}

func TestNoPathValues(t *testing.T) {
	src, err := os.ReadFile("api_gen.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(src), "SetPathValue") {
		t.Fatal("api_gen.go calls SetPathValue: not generated with -nopathvalue")
	}
}
