// Package bench compares rr's generated dispatch against httx, gin,
// httprouter, the standard ServeMux, and chi on a realistic deeply-nested REST
// surface adapted from httx's own benchmark.
//
//	cd bench && go test -bench . -benchmem .
package bench

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"net/url"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/go-chi/chi/v5"
	"github.com/julienschmidt/httprouter"
	"github.com/sirkostya009/httx"
)

// Package-level routers are built before init functions run, so release mode
// must itself be a preceding package initializer. The old init() left every
// benchmark Engine constructed in gin's debug mode.
var _ = func() struct{} {
	gin.SetMode(gin.ReleaseMode)
	return struct{}{}
}()

// Realistic deeply-nested API surface modeled after GitHub/GitLab/AWS-style
// services. Must stay in lockstep with bench/routes.go, which is the
// same surface pre-expanded (ver baked to v1/v2/v3, orderId/lineNo validated
// by a @digits regexp checker) into //rr:route directives for rr codegen.
var deepTemplates = []string{
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/files/{filepath:*}",
	"/api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/issues/{issueId}/comments/{commentId}",
	"/api/v{ver}/organizations/{orgId}/teams/{teamSlug}/members/{userId}",
	"/api/v{ver}/billing/accounts/{accountId}/subscriptions/{subId}/invoices/{invoiceId}/line_items/{lineItemId}",
	"/api/v{ver}/billing/accounts/{accountId}/payment_methods/{pmId}/transactions/{txnId}",
	"/api/v{ver}/marketplace/categories/{catSlug}/subcategories/{subSlug}/items/{itemId}/variants/{variantId}",
	"/api/v{ver}/observability/dashboards/{dashId}/panels/{panelId}/queries/{queryId}",
	"/api/v{ver}/observability/incidents/{incidentId}/timeline/{eventId}/responders/{responderId}",
	"/api/v{ver}/datasets/{datasetId}/tables/{tableId}/columns/{columnId}",
	"/api/v{ver}/ml/models/{modelId}/versions/{versionId}/deployments/{deployId}/predictions/{predId}",
	"/api/v{ver}/webhooks/{whId}/deliveries/{deliveryId}",
	"/api/v{ver}/integrations/{provSlug}/connections/{connId}/syncs/{syncId}",
	`/api/v{ver}/orders/{orderId:\d+}/lines/{lineNo:\d+}`,
	"/api/v{ver}/sessions/{sessionId}",
}

var topLevel = []string{"/", "/healthz", "/livez", "/readyz", "/metrics"}

// braceToColon converts httx/rr brace syntax to gin/httprouter colon syntax.
func braceToColon(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			i++
			isWild := false
			start := i
			for i < len(s) && s[i] != '}' && s[i] != ':' {
				i++
			}
			name := s[start:i]
			if i < len(s) && s[i] == ':' {
				i++
				if i < len(s) && s[i] == '*' {
					isWild = true
					i++
				}
				for i < len(s) && s[i] != '}' {
					i++
				}
			}
			i++
			if isWild {
				b.WriteByte('*')
			} else {
				b.WriteByte(':')
			}
			b.WriteString(name)
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

// stripRegex removes ":<pattern>" from inside braces, leaving just {name}.
// For routers that don't support regex (httprouter, gin).
func stripRegex(s string) string {
	var b strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '{' {
			j := strings.IndexByte(s[i:], '}')
			if j < 0 {
				b.WriteString(s[i:])
				break
			}
			seg := s[i : i+j+1]
			if before, _, ok := strings.Cut(seg, ":"); ok {
				b.WriteString(before)
				b.WriteByte('}')
			} else {
				b.WriteString(seg)
			}
			i += j + 1
		} else {
			b.WriteByte(s[i])
			i++
		}
	}
	return b.String()
}

type tmpl struct{ brace, colon string }

func buildTemplates() []tmpl {
	all := make([]tmpl, 0, 64)
	for _, p := range topLevel {
		all = append(all, tmpl{p, p})
	}
	for ver := 1; ver <= 3; ver++ {
		for _, t := range deepTemplates {
			brace := strings.ReplaceAll(t, "{ver}", strconv.Itoa(ver))
			all = append(all, tmpl{brace, braceToColon(brace)})
		}
	}
	return all
}

// methods is the set of methods we register each template with — exercises
// 405 detection and creates realistic per-resource handler counts.
var methods = []string{"GET", "POST", "PUT", "DELETE", "PATCH"}

func status405(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(405)
}

func status404(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(404)
}

// hit is one bench iteration: a method + concrete URL path to dispatch.
type hit struct {
	method string
	path   string
}

// newRNG returns a deterministic rng seeded the same way every run so bench
// results are reproducible across runs and CI.
func newRNG() *rand.Rand {
	return rand.New(rand.NewPCG(42, 42))
}

func pick[T any](r *rand.Rand, xs []T) T { return xs[r.IntN(len(xs))] }

func randID(r *rand.Rand, n int) string {
	const alphabet = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = alphabet[r.IntN(len(alphabet))]
	}
	return string(b)
}

// newRR returns rr's generated dispatcher: routes come from bench/routes.go
// (codegen'd into routes_gen.go), the compile-time equivalent of the runtime
// registration the other routers do below.
func newRR() *Api {
	return &Api{}
}

func newHTTX(f bool) *httx.Mux {
	m := httx.NewMux()
	m.RedirectTrailingSlash = f
	m.RedirectCaseInsensitivePath = f
	m.OnPanic = nil       // disable panic recovery — match peers' default
	m.GlobalOPTIONS = nil // disable OPTIONS auto-handling — match peers' default
	h := func(w http.ResponseWriter, r *http.Request) error { return nil }
	for _, t := range buildTemplates() {
		for _, meth := range methods {
			m.Handle(meth, t.brace, h)
		}
	}
	if f {
		m.GET("/articles/published", h)
		m.GET("/inbox", h)
	}
	return m
}

func newHTTPRouter(f bool) *httprouter.Router {
	r := httprouter.New()
	r.RedirectTrailingSlash = f
	r.RedirectFixedPath = f
	r.HandleOPTIONS = false // match httx's nil GlobalOPTIONS
	r.NotFound = http.HandlerFunc(status404)
	r.MethodNotAllowed = http.HandlerFunc(status405)
	h := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {}
	for _, t := range buildTemplates() {
		path := stripRegex(t.colon) // httprouter doesn't support regex
		for _, meth := range methods {
			r.Handle(meth, path, h)
		}
	}
	if f {
		r.GET("/articles/published", h)
		r.GET("/inbox", h)
	}
	return r
}

func newGin(f bool) *gin.Engine {
	r := gin.New()
	r.RedirectTrailingSlash = f
	r.RedirectFixedPath = f
	r.HandleMethodNotAllowed = true
	r.NoRoute(func(c *gin.Context) { c.Writer.WriteHeader(404) })
	r.NoMethod(func(c *gin.Context) { c.Writer.WriteHeader(405) })
	h := func(c *gin.Context) {}
	for _, t := range buildTemplates() {
		path := stripRegex(t.colon)
		for _, meth := range methods {
			r.Handle(meth, path, h)
		}
	}
	if f {
		r.GET("/articles/published", h)
		r.GET("/inbox", h)
	}
	return r
}

func serveMuxPattern(method, pattern string) string {
	pattern = canonicalPath(pattern)
	if pattern == "/" {
		// A bare "/" is a subtree match in ServeMux, unlike rr's exact route.
		pattern = "/{$}"
	}
	return method + " " + pattern
}

func newServeMux() *http.ServeMux {
	m := http.NewServeMux()
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, tmpl := range buildTemplates() {
		for _, method := range methods {
			m.Handle(serveMuxPattern(method, tmpl.brace), h)
		}
	}
	return m
}

func chiPath(pattern string) string {
	pattern = stripRegex(pattern)
	if i := strings.Index(pattern, "{filepath}"); i >= 0 && strings.HasSuffix(pattern, "{filepath}") {
		return pattern[:i] + "*"
	}
	return pattern
}

func newChi() *chi.Mux {
	r := chi.NewRouter()
	r.NotFound(status404)
	// Keep chi's default 405 handler: unlike a custom handler, it receives the
	// matched method set and therefore emits the same required Allow work.
	h := http.HandlerFunc(func(http.ResponseWriter, *http.Request) {})
	for _, tmpl := range buildTemplates() {
		for _, method := range methods {
			r.Method(method, chiPath(tmpl.brace), h)
		}
	}
	return r
}

type routerCase struct {
	name string
	h    http.Handler
}

func routers(f bool) []routerCase {
	rs := []routerCase{
		{"httx", newHTTX(f)},
		{"httprouter", newHTTPRouter(f)},
		{"gin", newGin(f)},
	}
	if !f {
		// rr's route set is fixed at compile time (bench/routes.go) and
		// has no trailing-slash/case-insensitive redirect feature, so it
		// only takes part in the "plain" registration.
		rs = append([]routerCase{
			{"rr", newRR()},
			{"stdlib", newServeMux()},
			{"chi", newChi()},
		}, rs...)
	}
	return rs
}

var (
	plain = routers(false)
	fix   = routers(true)
	// httprouter and gin support trailing-slash/case-insensitive redirects;
	// httx does too. rr, chi-equivalents, and stdlib-equivalents don't, so
	// they're excluded from these two subsets (mirrors upstream httx bench).
	tsrOnly = fix
	caseFix = fix
	// httprouter and gin have no regex param support; rr and httx do (rr via
	// a {name=@digits} checker ref to a real *regexp.Regexp, same digit-only
	// guarantee as httx's inline {orderId:\d+}).
	regexOnly = []routerCase{{"rr", newRR()}, {"httx", newHTTX(false)}}
	// httprouter/gin expose catch-all params with a leading slash while rr,
	// httx, ServeMux and chi expose the tail. Keep the timing comparison on
	// routers with equivalent handler-visible values.
	wildcardOnly = []routerCase{
		{"rr", newRR()},
		{"httx", newHTTX(false)},
		{"stdlib", newServeMux()},
		{"chi", newChi()},
	}
)

// resetWriter models the server's per-request response state without bringing
// httptest.ResponseRecorder's retained status/body into later iterations.
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

func hitURLs(hits []hit) []*url.URL {
	urls := make([]*url.URL, len(hits))
	for i := range hits {
		urls[i] = &url.URL{Path: hits[i].path}
	}
	return urls
}

// warmup drives 200 hits through h to warm caches + lazy init.
func warmup(h http.Handler, hits []hit, urls []*url.URL) {
	req := new(http.Request)
	w := newResetWriter()
	for i := range 200 {
		j := i % len(hits)
		ht := hits[j]
		*req = http.Request{Method: ht.method, URL: urls[j]}
		w.reset()
		h.ServeHTTP(w, req)
	}
}

// benchmarkHits cycles through a precomputed deterministic-but-pseudorandom
// list of (method, path) hits, exercising multiple URLs per benchmark
// instead of one path repeated. Hits are pre-shuffled with a fixed seed so
// runs are reproducible.
//
// Each router runs twice: "serial" (single goroutine, single-core dispatch
// cost) and "parallel" (GOMAXPROCS goroutines via b.RunParallel — contention
// on shared router state, pools, GC). Scale cores with -cpu 1,4,8.
func benchmarkHits(b *testing.B, rs []routerCase, hits []hit) {
	b.Helper()
	if len(hits) == 0 {
		b.Fatal("no hits")
	}
	urls := hitURLs(hits)
	for _, rc := range rs {
		b.Run(rc.name+"/serial", func(b *testing.B) {
			warmup(rc.h, hits, urls)

			// Keep the request allocation out of the timing, but reset its entire
			// routing metadata each iteration. This prevents rr/httx/stdlib from
			// reusing a warmed PathValue map that a real incoming request lacks.
			req := new(http.Request)
			w := newResetWriter()

			b.ReportAllocs()
			b.ResetTimer()

			i := 0
			for b.Loop() {
				j := i % len(hits)
				h := hits[j]
				*req = http.Request{Method: h.method, URL: urls[j]}
				w.reset()
				rc.h.ServeHTTP(w, req)
				i++
			}
		})
		b.Run(rc.name+"/parallel", func(b *testing.B) {
			warmup(rc.h, hits, urls)

			b.ReportAllocs()
			b.ResetTimer()

			var goroutines atomic.Int64
			b.RunParallel(func(pb *testing.PB) {
				req := new(http.Request)
				w := newResetWriter()
				// prime-stride offset so goroutines walk different hit sequences.
				i := int(goroutines.Add(1)) * 7919
				for pb.Next() {
					j := i % len(hits)
					h := hits[j]
					*req = http.Request{Method: h.method, URL: urls[j]}
					w.reset()
					rc.h.ServeHTTP(w, req)
					i++
				}
			})
		})
	}
}

// Each Bench* generates a deterministic-but-pseudorandom set of hits (varying
// method, path params, and API version where applicable). Hits cycle inside
// benchmarkHits so the bench exercises many distinct URLs, not one path
// repeated. Seed is fixed → replayable.

func BenchmarkSimple(b *testing.B) {
	// shallow static paths, varied methods. All topLevel routes are
	// registered with every method in `methods`.
	r := newRNG()
	hits := make([]hit, 64)
	for i := range hits {
		hits[i] = hit{method: pick(r, methods), path: pick(r, topLevel)}
	}
	benchmarkHits(b, plain, hits)
}

func BenchmarkSingleParam(b *testing.B) {
	// /api/v{ver}/sessions/{sessionId} — 1 param. Varied ver, sessionId, method.
	r := newRNG()
	hits := make([]hit, 64)
	for i := range hits {
		ver := 1 + r.IntN(3)
		hits[i] = hit{
			method: pick(r, methods),
			path:   "/api/v" + strconv.Itoa(ver) + "/sessions/sess-" + randID(r, 12),
		}
	}
	benchmarkHits(b, plain, hits)
}

func BenchmarkMultiParam(b *testing.B) {
	// /api/v{ver}/organizations/{orgId}/projects/{projectId}/repositories/{repoId}/branches/{branchName}/commits/{commitSha}/diff — 5 params.
	r := newRNG()
	hits := make([]hit, 64)
	for i := range hits {
		ver := 1 + r.IntN(3)
		hits[i] = hit{
			method: pick(r, methods),
			path: "/api/v" + strconv.Itoa(ver) + "/organizations/" + randID(r, 10) +
				"/projects/" + randID(r, 10) +
				"/repositories/" + randID(r, 12) +
				"/branches/" + randID(r, 8) +
				"/commits/" + randID(r, 40) +
				"/diff",
		}
	}
	benchmarkHits(b, plain, hits)
}

func BenchmarkRegexParam(b *testing.B) {
	// /api/v{ver}/orders/{orderId:\d+}/lines/{lineNo:\d+} — 2 digit-validated params.
	r := newRNG()
	hits := make([]hit, 64)
	for i := range hits {
		ver := 1 + r.IntN(3)
		hits[i] = hit{
			method: pick(r, methods),
			path:   "/api/v" + strconv.Itoa(ver) + "/orders/" + strconv.Itoa(r.IntN(1<<20)) + "/lines/" + strconv.Itoa(r.IntN(1024)),
		}
	}
	benchmarkHits(b, regexOnly, hits)
}

func BenchmarkWildcard(b *testing.B) {
	// .../commits/{commitSha}/files/{filepath:*} — catchall at depth, 5 params + tail.
	r := newRNG()
	tails := []string{
		"src/main.go",
		"src/internal/auth/middleware/oidc.go",
		"docs/api/v1/reference.md",
		"vendor/github.com/some/dep/file.go",
		"pkg/util/helpers_test.go",
		".github/workflows/ci.yml",
	}
	hits := make([]hit, 64)
	for i := range hits {
		ver := 1 + r.IntN(3)
		hits[i] = hit{
			method: pick(r, methods),
			path: "/api/v" + strconv.Itoa(ver) + "/organizations/" + randID(r, 10) +
				"/projects/" + randID(r, 10) +
				"/repositories/" + randID(r, 12) +
				"/branches/" + randID(r, 8) +
				"/commits/" + randID(r, 40) +
				"/files/" + pick(r, tails),
		}
	}
	benchmarkHits(b, wildcardOnly, hits)
}

func BenchmarkMethodMismatch(b *testing.B) {
	// OPTIONS / TRACE on registered paths — exercises 405 + Allow-header build.
	// OPTIONS and TRACE are not in `methods`, so any deep template path with
	// one of these methods triggers the 405 path.
	mismatchMethods := []string{"OPTIONS", "TRACE"}
	r := newRNG()
	hits := make([]hit, 64)
	for i := range hits {
		ver := 1 + r.IntN(3)
		hits[i] = hit{
			method: pick(r, mismatchMethods),
			path: "/api/v" + strconv.Itoa(ver) + "/billing/accounts/" + randID(r, 10) +
				"/payment_methods/pm-" + randID(r, 8) +
				"/transactions/txn-" + randID(r, 8),
		}
	}
	benchmarkHits(b, plain, hits)
}

func BenchmarkNotFound(b *testing.B) {
	// Random unregistered paths.
	r := newRNG()
	prefixes := []string{"/api/v9", "/admin", "/totally", "/does/not", "/api/v2/missing"}
	hits := make([]hit, 64)
	for i := range hits {
		hits[i] = hit{
			method: pick(r, methods),
			path:   pick(r, prefixes) + "/" + randID(r, 8) + "/" + randID(r, 6),
		}
	}
	benchmarkHits(b, plain, hits)
}

func BenchmarkTrailingSlash(b *testing.B) {
	// /inbox and /articles/published are GET-only routes. Keep every hit on GET:
	// the old varied-method workload silently measured 404/405 for most hits.
	r := newRNG()
	registered := []string{"/inbox/", "/articles/published/"}
	hits := make([]hit, 64)
	for i := range hits {
		hits[i] = hit{method: http.MethodGet, path: pick(r, registered)}
	}
	benchmarkHits(b, tsrOnly, hits)
}

func BenchmarkCaseInsensitive(b *testing.B) {
	// Wrong-cased + trailing slash variants of the registered redirect-only routes.
	r := newRNG()
	variants := []string{"/ARTICLES/Published/", "/Articles/published/", "/INBOX/", "/Inbox/"}
	hits := make([]hit, 64)
	for i := range hits {
		hits[i] = hit{method: http.MethodGet, path: pick(r, variants)}
	}
	benchmarkHits(b, caseFix, hits)
}

// canonicalPath renders rr/httx route syntax as the public Go 1.22 route
// pattern used by rr's generated dispatcher.
func canonicalPath(pattern string) string {
	var b strings.Builder
	for i := 0; i < len(pattern); {
		if pattern[i] != '{' {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		j := strings.IndexByte(pattern[i:], '}')
		if j < 0 {
			b.WriteString(pattern[i:])
			break
		}
		inner := pattern[i+1 : i+j]
		name, checker, hasChecker := strings.Cut(inner, ":")
		b.WriteByte('{')
		b.WriteString(name)
		if hasChecker && checker == "*" {
			b.WriteString("...")
		}
		b.WriteByte('}')
		i += j + 1
	}
	return b.String()
}

func pathParamNames(pattern string) []string {
	var names []string
	for i := 0; i < len(pattern); {
		start := strings.IndexByte(pattern[i:], '{')
		if start < 0 {
			break
		}
		start += i
		end := strings.IndexByte(pattern[start:], '}')
		if end < 0 {
			break
		}
		end += start
		inner := pattern[start+1 : end]
		name, _, _ := strings.Cut(inner, ":")
		names = append(names, name)
		i = end + 1
	}
	return names
}

func concreteRoute(pattern string, n int) (string, map[string]string) {
	params := make(map[string]string)
	var b strings.Builder
	for i := 0; i < len(pattern); {
		if pattern[i] != '{' {
			b.WriteByte(pattern[i])
			i++
			continue
		}
		j := strings.IndexByte(pattern[i:], '}')
		if j < 0 {
			panic("malformed benchmark route: " + pattern)
		}
		inner := pattern[i+1 : i+j]
		name, checker, _ := strings.Cut(inner, ":")
		value := name + "-value-" + strconv.Itoa(n)
		switch {
		case name == "orderId":
			value = strconv.Itoa(100000 + n)
		case name == "lineNo":
			value = strconv.Itoa(10 + n%90)
		case checker == "*":
			value = "src/internal/file-" + strconv.Itoa(n) + ".go"
		}
		params[name] = value
		b.WriteString(value)
		i += j + 1
	}
	return b.String(), params
}

type routeObservation struct {
	pattern string
	params  map[string]string
}

func (o *routeObservation) reset() {
	o.pattern = ""
	o.params = nil
}

func (o *routeObservation) record(pattern string, names []string, value func(string) string) {
	o.pattern = pattern
	o.params = make(map[string]string, len(names))
	for _, name := range names {
		o.params[name] = value(name)
	}
}

type observedRouter struct {
	name string
	h    http.Handler
	obs  *routeObservation
}

func observedRouters() []observedRouter {
	var out []observedRouter

	{
		o := new(routeObservation)
		// Rebuild so route-specific handlers can expose which route and params
		// were selected; the benchmark constructors deliberately use no-ops.
		m := httx.NewMux()
		m.RedirectTrailingSlash = false
		m.RedirectCaseInsensitivePath = false
		m.OnPanic = nil
		m.GlobalOPTIONS = nil
		for _, tmpl := range buildTemplates() {
			for _, method := range methods {
				pattern := method + " " + canonicalPath(tmpl.brace)
				names := pathParamNames(tmpl.brace)
				m.Handle(method, tmpl.brace, func(_ http.ResponseWriter, r *http.Request) error {
					o.record(pattern, names, r.PathValue)
					return nil
				})
			}
		}
		out = append(out, observedRouter{"httx", m, o})
	}

	{
		o := new(routeObservation)
		r := httprouter.New()
		r.RedirectTrailingSlash = false
		r.RedirectFixedPath = false
		r.HandleOPTIONS = false
		r.NotFound = http.HandlerFunc(status404)
		r.MethodNotAllowed = http.HandlerFunc(status405)
		for _, tmpl := range buildTemplates() {
			for _, method := range methods {
				pattern := method + " " + canonicalPath(tmpl.brace)
				names := pathParamNames(tmpl.brace)
				r.Handle(method, braceToColon(tmpl.brace), func(_ http.ResponseWriter, _ *http.Request, ps httprouter.Params) {
					o.record(pattern, names, ps.ByName)
				})
			}
		}
		out = append(out, observedRouter{"httprouter", r, o})
	}

	{
		o := new(routeObservation)
		r := gin.New()
		r.RedirectTrailingSlash = false
		r.RedirectFixedPath = false
		r.HandleMethodNotAllowed = true
		r.NoRoute(func(c *gin.Context) { c.Status(http.StatusNotFound) })
		r.NoMethod(func(c *gin.Context) { c.Status(http.StatusMethodNotAllowed) })
		for _, tmpl := range buildTemplates() {
			for _, method := range methods {
				pattern := method + " " + canonicalPath(tmpl.brace)
				names := pathParamNames(tmpl.brace)
				r.Handle(method, braceToColon(tmpl.brace), func(c *gin.Context) {
					o.record(pattern, names, c.Param)
				})
			}
		}
		out = append(out, observedRouter{"gin", r, o})
	}

	{
		o := new(routeObservation)
		m := http.NewServeMux()
		for _, tmpl := range buildTemplates() {
			for _, method := range methods {
				pattern := method + " " + canonicalPath(tmpl.brace)
				names := pathParamNames(tmpl.brace)
				m.HandleFunc(serveMuxPattern(method, tmpl.brace), func(_ http.ResponseWriter, r *http.Request) {
					o.record(pattern, names, r.PathValue)
				})
			}
		}
		out = append(out, observedRouter{"stdlib", m, o})
	}

	{
		o := new(routeObservation)
		r := chi.NewRouter()
		for _, tmpl := range buildTemplates() {
			for _, method := range methods {
				pattern := method + " " + canonicalPath(tmpl.brace)
				names := pathParamNames(tmpl.brace)
				wildcard := strings.Contains(tmpl.brace, ":*")
				r.MethodFunc(method, chiPath(tmpl.brace), func(_ http.ResponseWriter, r *http.Request) {
					o.record(pattern, names, func(name string) string {
						if wildcard && name == "filepath" {
							return chi.URLParam(r, "*")
						}
						return chi.URLParam(r, name)
					})
				})
			}
		}
		out = append(out, observedRouter{"chi", r, o})
	}

	return out
}

func TestRegisteredRoutesAgree(t *testing.T) {
	rr := newRR()
	peers := observedRouters()
	n := 0
	for _, tmpl := range buildTemplates() {
		for _, method := range methods {
			n++
			path, params := concreteRoute(tmpl.brace, n)
			wantPattern := method + " " + canonicalPath(tmpl.brace)

			req := httptest.NewRequest(method, path, nil)
			w := httptest.NewRecorder()
			rr.ServeHTTP(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("rr %s %s: status=%d, want 200", method, path, w.Code)
			}
			if req.Pattern != wantPattern {
				t.Fatalf("rr %s %s: pattern=%q, want %q", method, path, req.Pattern, wantPattern)
			}
			for name, want := range params {
				if got := req.PathValue(name); got != want {
					t.Fatalf("rr %s %s: %s=%q, want %q", method, path, name, got, want)
				}
			}

			for _, peer := range peers {
				peer.obs.reset()
				req := httptest.NewRequest(method, path, nil)
				w := httptest.NewRecorder()
				peer.h.ServeHTTP(w, req)
				if w.Code != http.StatusOK {
					t.Fatalf("%s %s %s: status=%d, want 200", peer.name, method, path, w.Code)
				}
				if peer.obs.pattern != wantPattern {
					t.Fatalf("%s %s %s: pattern=%q, want %q", peer.name, method, path, peer.obs.pattern, wantPattern)
				}
				gotParams := peer.obs.params
				if strings.Contains(tmpl.brace, ":*") && (peer.name == "httprouter" || peer.name == "gin") {
					gotParams = make(map[string]string, len(peer.obs.params))
					for name, value := range peer.obs.params {
						gotParams[name] = value
					}
					gotParams["filepath"] = strings.TrimPrefix(gotParams["filepath"], "/")
				}
				if !reflect.DeepEqual(gotParams, params) {
					t.Fatalf("%s %s %s: params=%v, want %v", peer.name, method, path, peer.obs.params, params)
				}
			}
		}
	}
}

func TestMissAndMethodSemantics(t *testing.T) {
	for _, rc := range plain {
		t.Run(rc.name+"/404", func(t *testing.T) {
			w := httptest.NewRecorder()
			rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/not/a/registered/path", nil))
			if w.Code != http.StatusNotFound {
				t.Fatalf("status=%d, want 404", w.Code)
			}
		})
		t.Run(rc.name+"/405", func(t *testing.T) {
			w := httptest.NewRecorder()
			rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/api/v1/sessions/session-1", nil))
			if w.Code != http.StatusMethodNotAllowed {
				t.Fatalf("status=%d, want 405", w.Code)
			}
			if len(w.Header().Values("Allow")) == 0 {
				t.Fatal("405 response omitted Allow")
			}
		})
	}
}

func TestRegexFeatureParity(t *testing.T) {
	path := "/api/v1/orders/not-digits/lines/7"
	for _, rc := range plain {
		w := httptest.NewRecorder()
		rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		want := http.StatusOK
		if rc.name == "rr" || rc.name == "httx" {
			want = http.StatusNotFound
		}
		if w.Code != want {
			t.Errorf("%s status=%d, want %d", rc.name, w.Code, want)
		}
	}
}

func TestRedirectHitSetsAreActualRedirects(t *testing.T) {
	for _, rc := range tsrOnly {
		w := httptest.NewRecorder()
		rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/articles/published/", nil))
		if w.Code < 300 || w.Code >= 400 {
			t.Errorf("trailing slash: %s status=%d, want redirect", rc.name, w.Code)
		}
	}
	for _, rc := range caseFix {
		w := httptest.NewRecorder()
		rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, "/ARTICLES/Published/", nil))
		if w.Code < 300 || w.Code >= 400 {
			t.Errorf("case fix: %s status=%d, want redirect", rc.name, w.Code)
		}
	}
}

func TestAllowSets(t *testing.T) {
	want := append([]string(nil), methods...)
	sort.Strings(want)
	for _, rc := range plain {
		w := httptest.NewRecorder()
		rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, "/api/v1/sessions/session-1", nil))
		var got []string
		for _, line := range w.Header().Values("Allow") {
			for _, method := range strings.Split(line, ",") {
				got = append(got, strings.TrimSpace(method))
			}
		}
		sort.Strings(got)
		// ServeMux and chi treat GET as also allowing HEAD. rr and the legacy
		// peers do not; record that intentional semantic disparity explicitly.
		got = slicesDelete(got, http.MethodHead)
		got = slicesDelete(got, http.MethodOptions)
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%s Allow=%v, want %v (ignoring automatic HEAD)", rc.name, got, want)
		}
	}
}

func TestWildcardParamRepresentation(t *testing.T) {
	const path = "/api/v1/organizations/o/projects/p/repositories/r/branches/b/commits/c/files/src/main.go"
	want := map[string]string{
		"rr":         "src/main.go",
		"httx":       "src/main.go",
		"httprouter": "/src/main.go",
		"gin":        "/src/main.go",
		"stdlib":     "src/main.go",
		"chi":        "src/main.go",
	}

	req := httptest.NewRequest(http.MethodGet, path, nil)
	newRR().ServeHTTP(httptest.NewRecorder(), req)
	if got := req.PathValue("filepath"); got != want["rr"] {
		t.Errorf("rr filepath=%q, want %q", got, want["rr"])
	}
	for _, peer := range observedRouters() {
		peer.obs.reset()
		peer.h.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, path, nil))
		if got := peer.obs.params["filepath"]; got != want[peer.name] {
			t.Errorf("%s filepath=%q, want %q", peer.name, got, want[peer.name])
		}
	}
}

func TestPercentEncodedSlashSemantics(t *testing.T) {
	const path = "/api/v1/sessions/a%2Fb"
	for _, rc := range plain {
		w := httptest.NewRecorder()
		rc.h.ServeHTTP(w, httptest.NewRequest(http.MethodGet, path, nil))
		want := http.StatusNotFound
		if rc.name == "stdlib" || rc.name == "chi" {
			// ServeMux unescapes path segments individually, preserving an
			// encoded slash inside the wildcard; chi likewise routes the escaped
			// segment. rr, httx, httprouter and gin use decoded URL.Path.
			want = http.StatusOK
		}
		if w.Code != want {
			t.Errorf("%s status=%d, want %d", rc.name, w.Code, want)
		}
	}
}

func slicesDelete(xs []string, value string) []string {
	for i, x := range xs {
		if x == value {
			return append(xs[:i], xs[i+1:]...)
		}
	}
	return xs
}
