package api

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
)

type nopWriter struct{ h http.Header }

func (n *nopWriter) Header() http.Header         { return n.h }
func (n *nopWriter) Write(b []byte) (int, error) { return len(b), nil }
func (n *nopWriter) WriteHeader(int)             {}

func run(b *testing.B, paths []string) {
	a := &Api{}
	w := &nopWriter{h: http.Header{}}
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	b.ReportAllocs()
	i := 0
	for b.Loop() {
		req.URL.Path = paths[i%len(paths)]
		req.Pattern = ""
		a.ServeHTTP(w, req)
		i++
	}
}

func BenchmarkStatic(b *testing.B) {
	p := make([]string, Count)
	for i := range p {
		p[i] = fmt.Sprintf("/large/static/page-%03d", (i*7919)%Count)
	}
	run(b, p)
}

func BenchmarkTenant(b *testing.B) {
	p := make([]string, Count)
	for i := range p {
		p[i] = fmt.Sprintf("/large/tenants/tenant-%03d/items/x%d", (i*7919)%Count, i)
	}
	run(b, p)
}
