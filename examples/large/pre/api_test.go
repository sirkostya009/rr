package pre

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestGuardValuesDoNotLeak(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/pre/values", nil)
	req.Header.Set("X-A", "5")
	w := httptest.NewRecorder()
	(&Api{}).ServeHTTP(w, req)
	if guarded != "5" {
		t.Fatalf("guard saw %q, want 5", guarded)
	}
	// X-B is absent: the route must get the zero value, not the guard's 5
	if got := strings.TrimSpace(w.Body.String()); got != `"0"` {
		t.Fatalf("route got %s, want \"0\"", got)
	}
}

func TestTransformerBoundAsDifferentTypes(t *testing.T) {
	for path, want := range map[string]string{"/pre/foo/a/any": `"any:a"`, "/pre/foo/b/ptr": `"ptr:b"`} {
		w := httptest.NewRecorder()
		(&Api{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		if got := strings.TrimSpace(w.Body.String()); w.Code != http.StatusOK || got != want {
			t.Errorf("%s: %d %s, want %s", path, w.Code, got, want)
		}
	}
}
