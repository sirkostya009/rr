// Package bench compares rr's generated dispatch against httx, gin and
// httprouter on a realistic deeply-nested REST surface, adapted from
// httx's own bench/bench_test.go (github.com/sirkostya009/httx/bench).
//
//	cd bench && go test -bench . -benchmem .
package bench

import (
	"math/rand/v2"
	"net/http"
	"net/http/httptest"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/julienschmidt/httprouter"
	"github.com/sirkostya009/httx"
)

func init() {
	gin.SetMode(gin.ReleaseMode)
}

// Realistic deeply-nested API surface modeled after GitHub/GitLab/AWS-style
// services. Must stay in lockstep with bench/routes.go, which is the
// same surface pre-expanded (ver baked to v1/v2/v3, orderId/lineNo bound as
// typed int instead of regex) into //api:route directives for rr codegen.
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
		rs = append([]routerCase{{"rr", newRR()}}, rs...)
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
	// a typed int match instead of an inline regex, same digit-only guarantee).
	regexOnly = []routerCase{plain[0], plain[1]} // rr, httx
)

// benchmarkHits cycles through a precomputed deterministic-but-pseudorandom
// list of (method, path) hits, exercising multiple URLs per benchmark
// instead of one path repeated. Hits are pre-shuffled with a fixed seed so
// runs are reproducible.
func benchmarkHits(b *testing.B, rs []routerCase, hits []hit) {
	b.Helper()
	if len(hits) == 0 {
		b.Fatal("no hits")
	}
	for _, rc := range rs {
		b.Run(rc.name, func(b *testing.B) {
			// per-iteration request — reused so we measure pure dispatch, not
			// httptest.NewRequest allocations.
			req := httptest.NewRequest("GET", "/", nil)
			w := httptest.NewRecorder()

			// warm caches + lazy init.
			for i := range 200 {
				h := hits[i%len(hits)]
				req.Method = h.method
				req.URL.Path = h.path
				rc.h.ServeHTTP(w, req)
			}

			var start, end runtime.MemStats
			runtime.ReadMemStats(&start)
			b.ReportAllocs()
			b.ResetTimer()

			i := 0
			for b.Loop() {
				h := hits[i%len(hits)]
				req.Method = h.method
				req.URL.Path = h.path
				rc.h.ServeHTTP(w, req)
				i++
			}

			b.StopTimer()
			runtime.ReadMemStats(&end)
			totalBytes := end.TotalAlloc - start.TotalAlloc
			gcs := end.NumGC - start.NumGC
			b.ReportMetric(float64(end.HeapAlloc)/1024, "heap_KB")
			b.ReportMetric(float64(totalBytes)/1024, "total_KB")
			b.ReportMetric(float64(totalBytes)/float64(b.N), "totB/op")
			b.ReportMetric(float64(gcs)/float64(b.N)*1e6, "gc/Mop")
			b.ReportMetric(float64(gcs), "gc")
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
	benchmarkHits(b, plain, hits)
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
	// /inbox and /articles/published are the only two routes registered when f=true.
	// Hit with trailing slash to force the redirect.
	r := newRNG()
	registered := []string{"/inbox/", "/articles/published/"}
	hits := make([]hit, 64)
	for i := range hits {
		hits[i] = hit{method: pick(r, methods), path: pick(r, registered)}
	}
	benchmarkHits(b, tsrOnly, hits)
}

func BenchmarkCaseInsensitive(b *testing.B) {
	// Wrong-cased + trailing slash variants of the registered redirect-only routes.
	r := newRNG()
	variants := []string{"/ARTICLES/Published/", "/Articles/published/", "/INBOX/", "/Inbox/"}
	hits := make([]hit, 64)
	for i := range hits {
		hits[i] = hit{method: pick(r, methods), path: pick(r, variants)}
	}
	benchmarkHits(b, caseFix, hits)
}
