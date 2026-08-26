// Package stress contains the common, correctness-checked harness used by the
// adversarial benchmark packages. Keeping traffic generation here prevents the
// different generated route counts from drifting into different workloads.
package stress

import (
	"math"
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/sirkostya009/httx"
)

func init() { gin.SetMode(gin.ReleaseMode) }

type Route struct {
	Pattern         string
	Method          string
	Param           string
	SkipRRPathValue bool // transformed handler args have no raw string PathValue.
	VerifyValue     string
}

type Hit struct {
	Method string
	Path   string
}

func routeMethod(route Route) string {
	if route.Method == "" {
		return http.MethodGet
	}
	return route.Method
}

func hitMethod(hit Hit) string {
	if hit.Method == "" {
		return http.MethodGet
	}
	return hit.Method
}

type routerCase struct {
	name string
	h    http.Handler
}

type resetWriter struct {
	header http.Header
	code   int
}

func newResetWriter() *resetWriter         { return &resetWriter{header: make(http.Header)} }
func (w *resetWriter) Header() http.Header { return w.header }
func (w *resetWriter) Write(p []byte) (int, error) {
	if w.code == 0 {
		w.code = http.StatusOK
	}
	return len(p), nil
}
func (w *resetWriter) WriteHeader(code int) {
	if w.code == 0 {
		w.code = code
	}
}
func (w *resetWriter) reset() {
	clear(w.header)
	w.code = 0
}

func stripCheckers(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); {
		if pattern[i] != '{' {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		j := strings.IndexByte(pattern[i:], '}')
		if j < 0 {
			panic("malformed route " + pattern)
		}
		inner := pattern[i+1 : i+j]
		name, checker, checked := splitParam(inner)
		b.WriteByte('{')
		b.WriteString(name)
		if checked && checker == "*" {
			b.WriteString(":*")
		}
		b.WriteByte('}')
		i += j + 1
	}
	return b.String()
}

func splitParam(inner string) (name, checker string, checked bool) {
	colon := strings.IndexByte(inner, ':')
	equals := strings.IndexByte(inner, '=')
	cut := colon
	if cut < 0 || (equals >= 0 && equals < cut) {
		cut = equals
	}
	if cut < 0 {
		return inner, "", false
	}
	return inner[:cut], inner[cut+1:], true
}

func canonical(pattern string) string {
	pattern = stripCheckers(pattern)
	return strings.ReplaceAll(pattern, ":*}", "...}")
}

func colonPattern(pattern string) string {
	pattern = stripCheckers(pattern)
	var b strings.Builder
	for i := 0; i < len(pattern); {
		if pattern[i] != '{' {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		j := strings.IndexByte(pattern[i:], '}')
		inner := pattern[i+1 : i+j]
		name, checker, checked := splitParam(inner)
		if checked && checker == "*" {
			b.WriteByte('*')
		} else {
			b.WriteByte(':')
		}
		b.WriteString(name)
		i += j + 1
	}
	return b.String()
}

func chiPattern(pattern string) string {
	pattern = stripCheckers(pattern)
	if start := strings.Index(pattern, "{:*"); start >= 0 {
		return pattern[:start] + "*"
	}
	if start := strings.LastIndex(pattern, "{filepath:*"); start >= 0 {
		return pattern[:start] + "*"
	}
	return pattern
}

func newRouters(rr http.Handler, routes []Route) []routerCase {
	noop := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})

	hx := httx.NewMux()
	hx.RedirectTrailingSlash = false
	hx.RedirectCaseInsensitivePath = false
	hx.OnPanic = nil
	hx.GlobalOPTIONS = nil
	for _, route := range routes {
		hx.Handle(routeMethod(route), stripCheckers(route.Pattern), func(http.ResponseWriter, *http.Request) error { return nil })
	}

	hr := httprouter.New()
	hr.RedirectTrailingSlash = false
	hr.RedirectFixedPath = false
	hr.HandleOPTIONS = false
	for _, route := range routes {
		hr.Handle(routeMethod(route), colonPattern(route.Pattern), func(http.ResponseWriter, *http.Request, httprouter.Params) {})
	}

	g := gin.New()
	g.RedirectTrailingSlash = false
	g.RedirectFixedPath = false
	for _, route := range routes {
		g.Handle(routeMethod(route), colonPattern(route.Pattern), func(*gin.Context) {})
	}

	sm := http.NewServeMux()
	for _, route := range routes {
		sm.Handle(routeMethod(route)+" "+canonical(route.Pattern), noop)
	}

	cr := chi.NewRouter()
	for _, route := range routes {
		cr.Method(routeMethod(route), chiPattern(route.Pattern), noop)
	}

	return []routerCase{
		{"rr", rr},
		{"httx", hx},
		{"httprouter", hr},
		{"gin", g},
		{"stdlib", sm},
		{"chi", cr},
	}
}

// Run benchmarks a hit set with fresh request routing metadata on each
// iteration while keeping request construction itself outside the measurement.
func Run(b *testing.B, rr http.Handler, routes []Route, hits []Hit) {
	b.Helper()
	if len(hits) == 0 {
		b.Fatal("empty hit set")
	}
	urls := make([]*url.URL, len(hits))
	for i := range hits {
		urls[i] = &url.URL{Path: hits[i].Path}
	}
	for _, router := range newRouters(rr, routes) {
		b.Run(router.name, func(b *testing.B) {
			req := new(http.Request)
			w := newResetWriter()
			for i := range 200 {
				j := i % len(hits)
				*req = http.Request{Method: hitMethod(hits[j]), URL: urls[j]}
				w.reset()
				router.h.ServeHTTP(w, req)
			}
			b.ReportAllocs()
			b.ResetTimer()
			i := 0
			for b.Loop() {
				j := i % len(hits)
				*req = http.Request{Method: hitMethod(hits[j]), URL: urls[j]}
				w.reset()
				router.h.ServeHTTP(w, req)
				i++
			}
		})
	}
}

// ScaleHits produces the three branch-entropy controls used at every route
// count. Uniform10K and Zipf10K both contain 10,000 distinct concrete URLs.
func ScaleHits(routeCount int) (friendly64, uniform10K, zipf10K []Hit) {
	path := func(route, id int) string {
		return "/gateway/v1/services/service-" + fmtIndex(route) + "/resources/resource-" + strconv.Itoa(id)
	}
	friendly64 = make([]Hit, 64)
	for i := range friendly64 {
		friendly64[i] = Hit{Path: path(i%routeCount, i)}
	}

	rng := rand.New(rand.NewPCG(42, 42))
	uniform10K = make([]Hit, 10_000)
	for i := range uniform10K {
		uniform10K[i] = Hit{Path: path(rng.IntN(routeCount), i+10_000)}
	}

	cdf := make([]float64, routeCount)
	total := 0.0
	for rank := 1; rank <= routeCount; rank++ {
		total += 1 / math.Pow(float64(rank), 1.07)
		cdf[rank-1] = total
	}
	for i := range cdf {
		cdf[i] /= total
	}
	zipf10K = make([]Hit, 10_000)
	for i := range zipf10K {
		route := sort.SearchFloat64s(cdf, rng.Float64())
		if route == routeCount {
			route--
		}
		zipf10K[i] = Hit{Path: path(route, i+20_000)}
	}
	return friendly64, uniform10K, zipf10K
}

func fmtIndex(i int) string {
	s := strconv.Itoa(i)
	return strings.Repeat("0", 4-len(s)) + s
}

type observed struct {
	name string
	h    http.Handler
}

// VerifyScale checks every generated route and handler identity. The route
// index is returned by peer handlers; rr is checked through r.Pattern, which
// its generated dispatcher stamps immediately before the selected handler.
func VerifyScale(t *testing.T, rr http.Handler, routes []Route) {
	t.Helper()
	var peers []observed

	{
		m := httx.NewMux()
		for i, route := range routes {
			i := i
			m.Handle(routeMethod(route), stripCheckers(route.Pattern), func(w http.ResponseWriter, r *http.Request) error {
				w.Header().Set("X-Route", strconv.Itoa(i))
				w.Header().Set("X-Param", r.PathValue(route.Param))
				return nil
			})
		}
		peers = append(peers, observed{"httx", m})
	}
	{
		r := httprouter.New()
		for i, route := range routes {
			i := i
			r.Handle(routeMethod(route), colonPattern(route.Pattern), func(w http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
				w.Header().Set("X-Route", strconv.Itoa(i))
				w.Header().Set("X-Param", ps.ByName(route.Param))
			})
		}
		peers = append(peers, observed{"httprouter", r})
	}
	{
		r := gin.New()
		for i, route := range routes {
			i := i
			r.Handle(routeMethod(route), colonPattern(route.Pattern), func(c *gin.Context) {
				c.Header("X-Route", strconv.Itoa(i))
				c.Header("X-Param", c.Param(route.Param))
			})
		}
		peers = append(peers, observed{"gin", r})
	}
	{
		m := http.NewServeMux()
		for i, route := range routes {
			i := i
			m.HandleFunc(routeMethod(route)+" "+canonical(route.Pattern), func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("X-Route", strconv.Itoa(i))
				w.Header().Set("X-Param", r.PathValue(route.Param))
			})
		}
		peers = append(peers, observed{"stdlib", m})
	}
	{
		r := chi.NewRouter()
		for i, route := range routes {
			i := i
			r.MethodFunc(routeMethod(route), chiPattern(route.Pattern), func(w http.ResponseWriter, req *http.Request) {
				w.Header().Set("X-Route", strconv.Itoa(i))
				w.Header().Set("X-Param", chi.URLParam(req, route.Param))
			})
		}
		peers = append(peers, observed{"chi", r})
	}

	for i, route := range routes {
		method := routeMethod(route)
		param := ""
		path := route.Pattern
		if route.Param != "" {
			param = route.VerifyValue
			if param == "" {
				param = "verify-" + strconv.Itoa(i)
			}
			start := strings.Index(route.Pattern, "{"+route.Param)
			if start < 0 {
				t.Fatalf("route %d does not contain param %q", i, route.Param)
			}
			end := strings.IndexByte(route.Pattern[start:], '}')
			if end < 0 {
				t.Fatalf("route %d has malformed param", i)
			}
			path = route.Pattern[:start] + param + route.Pattern[start+end+1:]
		}
		req := httptest.NewRequest(method, path, nil)
		w := httptest.NewRecorder()
		rr.ServeHTTP(w, req)
		if w.Code != http.StatusOK || req.Pattern != method+" "+canonical(route.Pattern) || (route.Param != "" && !route.SkipRRPathValue && req.PathValue(route.Param) != param) {
			t.Fatalf("rr route %d: status=%d pattern=%q param=%q", i, w.Code, req.Pattern, req.PathValue(route.Param))
		}
		for _, peer := range peers {
			w := httptest.NewRecorder()
			peer.h.ServeHTTP(w, httptest.NewRequest(method, path, nil))
			if w.Code != http.StatusOK || w.Header().Get("X-Route") != strconv.Itoa(i) || w.Header().Get("X-Param") != param {
				t.Fatalf("%s route %d: status=%d selected=%q param=%q", peer.name, i, w.Code, w.Header().Get("X-Route"), w.Header().Get("X-Param"))
			}
		}
	}
}

// VerifyStatus checks targeted miss/method hit sets on every benchmark router.
func VerifyStatus(t *testing.T, rr http.Handler, routes []Route, hits []Hit, want int) {
	t.Helper()
	for _, router := range newRouters(rr, routes) {
		for _, hit := range hits {
			w := httptest.NewRecorder()
			router.h.ServeHTTP(w, httptest.NewRequest(hitMethod(hit), hit.Path, nil))
			if w.Code != want {
				t.Fatalf("%s %s: status=%d, want %d", router.name, hit.Path, w.Code, want)
			}
		}
	}
}
