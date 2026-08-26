# Benchmark audit verdict

Date: 2026-08-26. Machine: Apple M2 Pro, Darwin/arm64, Go 1.26.0. All timing
tables are single-core (`-cpu=1`) benchstat medians from 10 samples at 200 ms
per sample unless a row says otherwise. The `±` values are benchstat's reported
95% ranges. Local thermal state and frequency scaling still apply; use the
commands below on another machine before generalizing the absolute numbers.

## Bottom line

The audit found two realistic surfaces where rr loses decisively to compact
runtime routers and one composition cost worth fixing:

- On 2,000 gateway routes with 10,000 uniformly distributed concrete URLs, rr
  takes **260 ns**, versus **124 ns** for httprouter and **98 ns** for gin. rr
  remains faster than httx, ServeMux and chi on this fixture.
- A request that reaches the last member of an int → regexp → bool → string
  parameter candidate chain takes rr from **16 ns / 0 allocs** to **274 ns /
  6 allocs**. The generated code validates earlier candidates before checking
  their distinguishing literal suffixes.
- In a plausible 20-package gateway, the last generated `HasPrefix` mount is
  **21.3% slower** than the first (p<0.001, n=15).

The strongest initial explanation needs qualification. High-entropy traffic
does slow rr by about 17–21%, but it also slows httprouter and gin, often by a
larger percentage. The data confirms that rr can lose to runtime tries; it does
not show that generated dispatch is uniquely vulnerable to branch-predictor
saturation.

## Correctness and fairness audit

No benchmark number above was accepted until all routers reached the expected
route and exposed the expected parameter. The regular tests now verify all 295
method/template combinations in the original fixture and every route in each
adversarial package.

| Finding | Effect | Resolution |
|---|---|---|
| No route/handler/param oracle | A peer could silently benchmark a 404 or the wrong route. | Added route identity, status and native-param parity tests for rr, httx, httprouter, gin, ServeMux and chi. |
| Redirect hits randomized over five methods, but `/inbox` and `/articles/published` were GET-only | Most “redirect” iterations were actually misses/405s. | Redirect workloads now use GET, with a test requiring every peer response to be 3xx. |
| One request was reused forever | rr/httx could reuse a warmed `PathValue` map that a real incoming request does not have. | Each timed iteration resets the full `http.Request` value while keeping request parsing/allocation outside the router benchmark. |
| One `httptest.ResponseRecorder` was reused forever | Status, body and headers leaked across logical responses. | Replaced it with a resettable minimal writer and clear response state per iteration. |
| Custom `ReadMemStats` deltas around `RunParallel` | Process-global totals are not attributable to one parallel benchmark. | Removed them; the harness uses `testing.B.ReportAllocs`. |
| `gin.SetMode` ran in `init`, after package-level engines were constructed | Registration happened in debug mode and emitted debug work/output. | Release mode is established by an earlier package initializer. |
| Only author-related/runtime peers plus gin/httprouter | The comparison lacked neutral baselines. | Added Go 1.22+ `http.ServeMux` and chi v5.3.2. |
| 405 implementations do different work | rr writes a constant `Allow` and status; httx/httprouter/gin search method trees; ServeMux/chi also produce their default error body. | `Allow` semantics are tested and the 405 table is labeled as end-to-end default behavior, not pure lookup parity. |
| Regex support differs | httprouter, gin, ServeMux and the configured chi routes do not provide rr/httx's validator semantics. | The regexp benchmark remains rr/httx-only; a test records the expected non-digit behavior of every router. |
| Catch-all values differ | httprouter/gin include a leading slash; rr/httx/ServeMux/chi expose the tail. | The difference has an explicit test; httprouter/gin are excluded from the wildcard timing subset. |
| Escaped slash behavior differs | `/sessions/a%2Fb` matches in ServeMux/chi but not rr/httx/httprouter/gin with their current defaults. | Added a test documenting the split; encoded-path timings are not pooled as equivalent work. |

All routers still materialize their native route params even though the timing
handlers are no-ops: rr and httx use `Request.SetPathValue`, httprouter builds a
`Params` slice, gin fills its pooled context, ServeMux fills request matches,
and chi builds route context. These APIs have inherently different allocation
profiles. On a fresh one-param request rr and httx both report 336 B / 2 allocs,
httprouter 32 B / 1 alloc, gin 0, ServeMux 16 B / 1 alloc and chi 704 B / 4
allocs. Treat those as end-to-end router metadata costs, not an isolated trie
comparison.

## Benchstat summary

Times are ns/op except rows explicitly shown in µs.

### Corrected original surface

| Scenario | rr | httx | httprouter | gin | ServeMux | chi |
|---|---:|---:|---:|---:|---:|---:|
| shallow static | **10.21 ±1%** | 15.46 ±5% | 24.70 ±1% | 41.96 ±9% | 69.14 ±0% | 350.2 ±9% |
| five params | 331.2 ±9% | 390.8 ±2% | 228.7 ±15% | **129.1 ±1%** | 824.9 ±7% | 963.0 ±7% |
| 405 + default `Allow` behavior | **114.7 ±5%** | 502.9 ±2% | 1.113 µs ±8% | 671.5 ±3% | 5.738 µs ±3% | 918.4 ±4% |

The 405 row includes the response-work disparity described above and should not
be presented as lookup-only performance.

### Route-count and hit-entropy scaling

The route surface is a regular enterprise-gateway shape:
`/gateway/v1/services/service-N/resources/{resourceId}`. IDs make the 10K hit
sets concrete and distinct. “Friendly” is the old 64-hit cycle; Zipf uses
exponent 1.07.

| Routes / traffic | rr | httx | httprouter | gin | ServeMux | chi |
|---|---:|---:|---:|---:|---:|---:|
| 50 / friendly 64 | 178.1 ±16% | 190.8 ±4% | 87.40 ±4% | **57.80 ±1%** | 277.6 ±1% | 468.4 ±2% |
| 50 / uniform 10K | 212.1 ±10% | 223.2 ±11% | 95.35 ±7% | **62.16 ±1%** | 282.1 ±4% | 488.1 ±3% |
| 500 / friendly 64 | 185.7 ±2% | 208.8 ±5% | 89.16 ±1% | **62.40 ±15%** | 272.8 ±2% | 504.6 ±3% |
| 500 / uniform 10K | 217.7 ±3% | 241.5 ±4% | 111.4 ±2% | **79.21 ±1%** | 280.9 ±2% | 567.2 ±13% |
| 2,000 / friendly 64 | 215.4 ±7% | 237.3 ±3% | 91.47 ±1% | **66.92 ±0%** | 267.5 ±2% | 558.5 ±3% |
| 2,000 / uniform 10K | 260.2 ±8% | 295.3 ±1% | 123.5 ±2% | **97.89 ±0%** | 284.1 ±1% | 619.7 ±1% |
| 2,000 / Zipf 10K | 234.0 ±3% | 259.2 ±2% | 108.7 ±4% | **81.44 ±2%** | 276.5 ±2% | 578.8 ±3% |

Uniform-versus-friendly rr deltas are +19.1%, +17.2% and +20.8% at 50, 500
and 2,000 routes. The corresponding httprouter deltas are +9.1%, +24.9% and
+35.0%; gin's are +7.5%, +26.9% and +46.3%. Zipf consistently recovers part of
the entropy penalty.

### Targeted route shapes

| Scenario | rr | httx | httprouter | gin | ServeMux | chi |
|---|---:|---:|---:|---:|---:|---:|
| 1,000 static routes, static traffic | **22.88 ±9%** | 50.30 ±1% | 46.77 ±1% | 65.17 ±0% | 185.2 ±0% | 352.6 ±2% |
| same surface, dynamic near-misses | 201.5 ±5% | 192.9 ±12% | 72.38 ±2% | **42.44 ±21%** | 204.4 ±3% | 466.9 ±1% |
| param chain, first int candidate | **16.43 ±7%** | 178.5 ±8% | 70.31 ±4% | 40.02 ±0% | 156.0 ±3% | 444.4 ±3% |
| param chain, final string candidate | 274.1 ±4% | 188.4 ±2% | 79.26 ±4% | **46.47 ±1%** | 211.1 ±1% | 439.8 ±3% |
| uniform-method subtree | 210.0 ±9% | 220.7 ±4% | 96.45 ±10% | **66.51 ±1%** | 220.7 ±5% | 526.8 ±8% |
| mixed-method twin | 213.7 ±6% | 219.5 ±2% | 101.4 ±3% | **68.38 ±1%** | 227.2 ±3% | 515.5 ±3% |
| deep successful hit | 167.6 ±6% | 174.8 ±20% | 70.49 ±3% | **39.27 ±2%** | 513.1 ±7% | 507.9 ±9% |
| last-segment deep 404 | **14.39 ±7%** | 47.48 ±1% | 323.7 ±2% | 90.51 ±15% | 3.246 µs ±16% | 455.4 ±4% |
| first of 20 package mounts | 170.9 ±6% | 188.2 ±6% | 76.31 ±5% | **45.66 ±0%** | 224.3 ±6% | 487.3 ±18% |
| last of 20 package mounts | 219.7 ±5% | 192.4 ±5% | 78.89 ±3% | **49.84 ±1%** | 226.1 ±4% | 491.8 ±11% |
| miss all 20 mounts | 39.05 ±1% | **33.76 ±7%** | 261.8 ±3% | 66.82 ±11% | 852.2 ±2% | 404.9 ±7% |

Two focused 15-sample A/B tests make the attribution clearer:

- The identical dynamic rr route is **11.11% slower** when the dispatcher also
  contains 1,000 static routes (p<0.001). The same enlarged route surface costs
  httprouter 7.54% and gin 21.32%, so this is not uniquely an rr static-switch
  collapse.
- Uniform versus mixed method subtrees are statistically indistinguishable for
  rr: 227.4 ns versus 231.6 ns, p=0.631.
- First versus last mount is **+21.30%** for rr, p<0.001.

## Generated-code growth

These are forced full test-binary rebuilds (`go test -c -a`) on the same
machine. They recompile common dependencies, so the wall times understate the
relative package-only growth; the symbol size is the cleaner signal.

| Routes | generated `routes_gen.go` | optimized `(*API).ServeHTTP` text | full rebuild wall |
|---:|---:|---:|---:|
| 50 | 17,060 B | 32,576 B | 10.00 s |
| 500 | 166,011 B | 112,720 B | 11.14 s |
| 2,000 | 662,512 B | 457,200 B | 15.27 s |

The 2,000-route function is about 14× the machine-code size of the 50-route
function. Even on friendly traffic restricted to the first 64 routes, rr slows
from 178 ns to 215 ns (+21%), while httprouter moves from 87 ns to 91 ns (+5%).
That supports an instruction-footprint concern independent of hit entropy,
though it does not isolate front-end stalls with hardware counters.

## Hypothesis verdicts

| Hypothesis | Verdict | Mechanism/evidence |
|---|---|---|
| 64-hit traffic hides route-count/entropy losses | **Confirmed, but not rr-specific** | rr loses ~17–21% at 10K uniform entropy. Runtime tries also lose 9–46%. |
| Large generated dispatch can lose to compact tries | **Confirmed** | At 2,000 uniform routes rr is 2.11× httprouter and 2.66× gin, while still beating httx/ServeMux/chi. |
| Static-switch misses make dynamic traffic uniquely bad for rr | **Partly confirmed** | A 1,000-static surface adds 11.1% to identical rr dynamic traffic, but comparable peers also pay 7.5–21.3%. Static rr traffic remains exceptionally fast. |
| Late parameter candidates are dangerous | **Strongly confirmed** | Generated order is Atoi → regexp → ParseBool → string; the final case is 16.7× the first and adds 494 B / 6 allocs. |
| Defeating method-gate hoisting materially hurts | **Refuted for this 100+100-route fixture** | +1.8% median; focused A/B p=0.631. |
| Late deep 404s are an rr weakness | **Refuted** | 14.4 ns / 0 allocs; rr is fastest of all six. Params are stamped only after a complete match. |
| Linear cross-package mount chains hurt late mounts | **Confirmed** | Last versus first is +21.3%, p<0.001. A 20-prefix miss remains cheap at 39 ns. |
| Wildcard behavior is directly comparable across all peers | **Refuted** | httprouter/gin expose a leading slash; the parity subset is rr/httx/ServeMux/chi. Existing varied-tail traffic did not reveal an rr regression. |

## Generator recommendations supported by the data

1. **Discriminate descendant literals before running param validators.** For
   `/catalog/{value}/number`, `/uuid`, `/toggle` and `/profile`, slice the
   segment and suffix once, use the suffix to choose the only relevant
   candidate, then run `Atoi`, regexp or `ParseBool`. This directly attacks the
   16.7× late-candidate result and avoids allocating parse errors for unrelated
   routes.
2. **Compile cross-package mounts into a prefix trie or first-segment switch.**
   Preserve longest-prefix semantics, but avoid 19 failed `HasPrefix` calls
   before a late mount. The measured +21.3% makes this worthwhile at a
   realistic 20-package gateway size.
3. **Split very large generated dispatchers into cold helper subtrees.** A
   457 KB `ServeHTTP` symbol and the +21% friendly-traffic degradation support
   reducing the hot instruction footprint. A top-level literal switch calling
   non-inline helpers is a lower-risk experiment than replacing all codegen
   with a runtime trie.

The data does **not** support removing the static switch wholesale: it produces
22.9 ns static hits, and the dynamic penalty was not worse than every trie
peer. It also does not justify method-gate or deep-404 changes.

## Reproduction

From `bench/`:

```sh
go generate ./adversarial/...
go test ./...
go vet ./...

# Example ten-sample scale run.
go test ./adversarial/scale2000 -run '^$' \
  -bench 'Benchmark(Friendly64|Uniform10K|Zipf10K)/' \
  -benchmem -benchtime=200ms -count=10 -cpu=1

# Summarize a captured run.
benchstat scale2000.txt
```

Run the same command multiple times or interleave A/B inputs when comparing a
generator change. Pinning `-cpu=1` controls goroutine parallelism, not CPU
frequency, process scheduling or thermals.
