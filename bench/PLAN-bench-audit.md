# Plan: stress rr's routing under adversarial route shapes + audit benchmark fairness

## Context

rr (this repo) is a codegen-first router: `cmd/main.go` compiles `//rr:route`
directives into a single generated `ServeHTTP` — common-prefix cut, whole-path
`switch` for static routes, a compiled segment trie for dynamic ones (one
`strings.IndexByte` per node, sibling literals via segment switch, param edges
tried in declaration order with checker/transformer fallthrough).

`bench/` benchmarks the generated dispatcher (`routes.go` → `routes_gen.go`)
against httx, gin and httprouter over a fixed 18-template × 3-version × 5-method
surface (`bench_test.go`). Current results presumably favor rr.

Two hypotheses to test, honestly and adversarially:

1. **The routing codegen is elementary and there exist realistic route sets /
   traffic patterns where it does much worse than the runtime tries.**
2. **The current benchmark harness may hand rr (or a peer) an undue advantage
   through measurement artifacts or feature disparity.**

The goal is NOT to make rr look good. The goal is to find where it breaks, and
to make the benchmark trustworthy. Negative findings are the deliverable.

You have full latitude on how to structure the work; the sections below are
directions and known suspicions, not a spec. Correctness of any new bench
fixture matters: every route set must actually generate (`go run
github.com/sirkostya009/rr/cmd routes.go` from the package dir) and every hit
must be verified to reach the intended handler on every router before its
numbers count (see Direction B1).

---

## Direction A — adversarial route sets

Design route sets that are *plausible for a real service* (this matters — a
pathological-only fixture proves nothing) but hit the generator's known
structural weak points. For each, generate the rr dispatcher, register the same
surface on the peers, and bench with hit distributions targeted at the weak
point. Ideas, roughly ordered by expected payoff:

1. **Branch-predictor saturation / icache pressure.** The current bench cycles
   64 hits — a modern predictor memorizes that. A giant generated `ServeHTTP`
   (whole-path string switch + deep nested if/else) should suffer more than a
   compact trie loop once the predictor can't memorize. Scale two axes
   independently: route count (say 50 → 500 → 2000 routes, realistic REST
   naming) and hit-set size/entropy (64 → 10k+ distinct URLs, uniformly and
   Zipf-distributed). Also worth reporting: generated file size, build time of
   the bench package, and `ServeHTTP` object-code size (`go tool nm` /
   `-gcflags -S`) as route count grows.

2. **Static-switch near-misses.** Static routes are tried first via a whole-path
   `switch`; a dynamic-route request pays the failed switch before the trie
   walk. Craft a surface with many static routes overlapping dynamic prefixes
   (e.g. `/users/settings`, `/users/billing`, … next to `/users/{id}`) and
   traffic that's mostly dynamic. Compare against a radix router that resolves
   both in one walk. Go compiles big string switches via length+binary-search,
   so also probe many same-length static paths sharing long common prefixes.

3. **Param-edge candidate chains.** Same-position params dispatch in
   declaration order with parse-fail → next candidate (int before bool before
   string, checkers via `MatchString`). Craft positions with several typed/
   checked candidates (`{id=@digits}`, `{uuid=@uuidRe}`, typed int, catch-all
   string) and traffic that mostly falls through to the *last* candidate — every
   request pays every failed parse/regex. Compare with routers that don't
   re-validate.

4. **Method-gate hoisting defeated.** Hoisting (405 checks lifted above path
   slicing) only fires on method-uniform subtrees. Build realistic mixed-method
   subtrees (GET-only reads mixed with POST-only actions under one prefix) and
   check whether per-leaf method dispatch + 405 detection degrades vs peers.

5. **Deep 404s / late mismatches.** Paths that match the trie for 6–8 segments
   and diverge at the last one; also long-common-prefix misses against the
   static switch. Does the failed static switch + full trie walk double-pay?

6. **Cross-package composition chains.** The `strings.HasPrefix` delegation for
   mounted sub-apis is a linear scan, longest-prefix-first. Compose, say, 10–20
   mounted packages (a plausible microservice-gateway shape) and measure
   dispatch for the *last* prefix and for misses, vs one flat surface and vs
   peers.

7. **Wildcard/catch-all interaction.** Tails of very different lengths, and
   catch-alls competing with deeper literal routes.

For each adversarial set, keep a control: the same routes on rr vs peers under
*friendly* traffic too, so a regression is attributable to the traffic shape
vs the route shape.

## Direction B — benchmark-correctness audit

Audit `bench/bench_test.go` for anything that makes the comparison unfair in
either direction. Known suspicions to confirm or dismiss (and look for others):

1. **No correctness cross-check.** Nothing verifies that a given (method, path)
   hit actually routes to the equivalent handler with the same status and the
   same extracted params on every router. A bench that's silently 404ing on one
   router measures the wrong thing. Build a verification pass (counting
   handlers that record status + params) that runs over every hit set before
   benching; make it a regular `Test*`.

2. **Work parity per request.** Check what each router actually does per hit:
   - rr stamps `r.Pattern` and calls `r.SetPathValue` for every string param —
     what does `SetPathValue` cost on a stdlib request (map/slice alloc?), and
     do gin/httprouter do equivalent param materialization in their no-op
     handlers, or less, or more (httprouter's `Params` slice, gin's context
     pool)?
   - 405 parity: rr emits an `Allow` header; confirm peers are configured to do
     the same work (httprouter builds Allow dynamically; gin?).
   - Escaping: rr matches `r.URL.Path` raw; peers may consult
     `RawPath`/unescape. Feature disparity → document it, and add a hit set
     with percent-encoded segments to expose behavioral divergence.

3. **Harness artifacts.** Reused `httptest.NewRecorder` across iterations
   (header map accumulation, no reset), reused request, 64-hit cycle
   (predictor memorization — see A1), warmup asymmetry, whether the no-op
   handlers allow any router's dispatch to be dead-code-eliminated or inlined
   in a way a real handler wouldn't. Also whether `reportMem`'s ReadMemStats
   pairs are meaningful under `b.RunParallel`.

4. **Peer configuration fairness.** gin release mode ✓, but check: gin without
   its context pool being bypassed, httprouter options, whether httx (the same
   author's runtime router) is configured comparably. Add neutral, widely-used
   baselines so the comparison isn't only against the author's other project:
   stdlib `http.ServeMux` (1.22 patterns) and chi at minimum; echo if cheap.

5. **Statistics.** Single-run `go test -bench` numbers are noise-prone. Run
   via `benchstat` with ≥10 counts, report deltas with confidence intervals,
   pin down GOMAXPROCS/`-cpu` choices, and note thermal/noise caveats for
   local runs.

## Direction C — synthesis

Whatever you find, end with a written verdict in `bench/` (markdown):

- a table of scenarios × routers with benchstat summaries,
- which adversarial hypotheses were confirmed vs refuted, with the mechanism
  (e.g. "predictor memorization inflated rr by ~Nx; at 10k-URL entropy rr is
  slower than httprouter on X"),
- the harness bugs/disparities found and whether they were fixed or documented,
- concrete, prioritized suggestions for the generator (only if the data
  supports them — e.g. "merge static routes into the trie", "hash-dispatch
  static paths", "hoist checker regexes").

Keep new fixtures in separate packages under `bench/` (each needs its own
`//go:generate` + generated file, mirroring `bench/routes.go`) so the original
comparison stays intact. Everything must build with `go vet ./... && go test
./...` from `bench/`.
