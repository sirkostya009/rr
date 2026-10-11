package miss

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
)

func serve(t *testing.T, path string) int {
	t.Helper()
	missed, bad = nil, nil
	w := httptest.NewRecorder()
	(&Api{}).ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
	return w.Code
}

func TestNotFoundError(t *testing.T) {
	for _, tc := range []struct {
		path    string
		convErr bool
	}{
		{"/miss/items/abc", true},
		{"/miss/deep/abc/x", true},
		{"/miss/ext/abc/int", true},
		{"/miss/hex/xyz", false},  // (T, bool) rejects with no error
		{"/miss/deep/5/y", false}, // Atoi never ran
		{"/miss/ext/5/y", false},  // Atoi succeeded, the literal missed
		{"/miss/items/5/x", false},
		{"/nope", false},
	} {
		if code := serve(t, tc.path); code != http.StatusNotFound {
			t.Errorf("%s: %d, want 404", tc.path, code)
			continue
		}
		var ne *strconv.NumError
		if got := errors.As(missed, &ne); got != tc.convErr {
			t.Errorf("%s: on404 got err %v, want conversion error: %v", tc.path, missed, tc.convErr)
		}
	}
}

func TestOKTransformer(t *testing.T) {
	for path, want := range map[string]int{
		"/miss/hex/ff":     http.StatusOK,
		"/miss/query?n=ff": http.StatusOK,
		"/miss/query":      http.StatusOK,
		"/miss/query?n=zz": http.StatusBadRequest,
	} {
		if code := serve(t, path); code != want {
			t.Errorf("%s: %d, want %d", path, code, want)
		}
		if bad != nil {
			t.Errorf("%s: on400 got err %v, want nil", path, bad)
		}
	}
}
