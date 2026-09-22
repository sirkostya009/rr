# rr

Codegen-first HTTP framework for Go. A CLI (root package) parses `//api:`
directives off structs and methods and generates a single `ServeHTTP` per
dispatcher: common-prefix cut, whole-path switch for static routes, segment
trie for dynamic ones. No runtime library — generated code depends only on
stdlib (+ ggen when present). Think NestJS annotations, compiled to a switch.

## Layout

- `main.go` — the whole generator, single file
- `examples/` — two independent modules, each a living fixture + test suite
  - `simple/` (module `simple`) — the flat single-package example: one
    `//rr:api` central with a file per sub-API (`api.go`, `users.go`, …),
    `server.go` the sentry/slog wrapper, generated `api_gen.go`/`api_ggen.go`
  - `versioned/` (module `versioned`) — the cross-package composition example:
    - `api/` — `api.go` is a `//rr:api` composing v1+v2 (generated `api_gen.go`
      delegates by prefix), `server.go` the wrapper, `api_test.go`
    - `api/v1/`, `api/v2/` — one `//rr:api` central each, one file per sub-API;
      v1 = int ids (uuid checker route as non-default), v2 = uuid string ids.
      Each has generated `api_gen.go` (router) + `api_ggen.go` (ggen codecs),
      each emitting its own buffer pools (no `-helpers`)
    - `services/` — shared storage layer; records carry both id shapes. Also
      owns ggen types (`NewUser`, `Stats`) v2 routes bind cross-package —
      `TestForeignGgen` is the fixture for foreign ggen detection
  - `large/` (module `large`, no deps) — the hashed-dispatch fixture: `api/gen`
    writes `routes.go` with three literal sets of 260 routes (just past
    the default `-hashmin`): whole-path static, literal-before-param, and
    mixed-length words, and `versions/vNNN/{id}` whose keys share exactly one
    leading byte. `candidates.go` adds the two same-position param
    shapes: four candidates each followed by a distinct terminal literal
    (literal-first switch) and a `{id int}`/`{slug}` pair sharing `edit`
    (ordered chain). `api_test.go` hits every route, checks impostors that
    share a hash slot get 404, checks candidate order, and FAILS if
    `api_gen.go` stops containing 4 hashed switches or 1 literal-first switch
    (so a threshold or planner change can't silently drop coverage)
- `go.work` ties in root, the three examples and `bench`; ggen comes from the
  pinned pseudo-version in each example's go.mod

## Commands

```sh
go build .                                 # root module: the generator
cd examples/versioned && go generate ./... # any order; ggen runs before rr per pkg
cd examples/versioned && go vet ./... && go test ./...   # or examples/simple
```

Generation is whole-package: the input file arg only anchors the directory and
the output name (`api.go` → `api_gen.go`); directives are scanned from every
file in the package. The stale output file is skipped while parsing.

## Directive surface

ALL directives are `rr:` prefixed. Patterns:
`/x/{name}`, `/x/{name=@ref}`, trailing `/{name...}` or `/*`. `{name:regex}`
was removed — point `@ref` at a `*regexp.Regexp` var instead (used as-is via
MatchString, NOT auto-anchored).

Type directives:
- `//rr:api [onerror=@f on400=@f on404=@f on405=@f]` — the central; api-typed
  fields (incl. embedded) get their routes merged into one dispatcher.
- `//rr:controller [onerror=@f on400=@f on404=@f on405=@f]` — OPTIONAL marker
  on a mounted sub-api, only to attach its error handlers. A struct with route
  methods is a controller with or without it.
- `//rr:pre @f @g @h` — guard chain on one line (on api or controller). Was
  `//rr:middleware`.
Cross-package composition: a `//rr:api` field whose type is `pkg.Type` (another
package's api) can't have its routes merged — the generator instead delegates by
prefix (`emitXMounts`): the mounts' shared head is cut once, then a switch on
the next segment picks the mount, recursively — no linear `HasPrefix` chain.
Equal-width sibling segments slice fixed (`switch p[:3] { case "v1/":`, behind
a `len` guard that also elides the bounds check); mixed widths use IndexByte. A
mount whose prefix ends at a level is tried last there, so the longest prefix
still wins and a failed match falls through to own routes. It discovers the sub-api's prefix by `go list
-e`ing the import and re-deriving it from THAT PACKAGE'S SOURCES — its
`//rr:route` directives and, recursively, its own cross-package fields
(`discoverPrefix`/`scanPkg`, memoized). Generated output is never read, so
GENERATION ORDER DOES NOT MATTER: `go generate ./...` works cold, and a stale or
missing sub-package `api_gen.go` can't poison a parent. Parent and child agree
by construction because both run `commonPrefixOf` over the same patterns.
A root with only cross-package fields (no own routes) is allowed; it stamps its
own prefix (the LCP of its mounts) so it nests to any depth.

Mounted APIs never get a standalone ServeHTTP; unmounted route-owners and
composition roots always do. Responses are ALWAYS JSON (no `response`).

Method directives: `//rr:route [METHOD] /path` only. Route-level
`//rr:errorhandler` was dropped — put onerror on the owning controller/api.
` -- comment` suffixes allowed everywhere.

Handler params bind by NAME + TYPE for the common cases; an inline
/* rr:... */ annotation is the explicit override (e.g. a body param not named
`body`). Resolution order per param:
1. `http.ResponseWriter` / `*http.Request` / `context.Context` — by type, any
   position, optional; the context is emitted as `r.Context()`
2. explicit annotation, if present (see below) — overrides everything after
3. name ∈ route `{tokens}` — path param; T ∈ string/int/float64/float32/bool
   derives the matcher (Atoi, ParseFloat, strict ParseBool); struct/iface/any
   is a generate-time error. A route token overrides a reserved NAME (not an
   explicit annotation). A named catch-all `{name...}` binds the same way but
   string ONLY (the raw rest of the path, slashes included) — under
   `-nopathvalue` an arg is the only way to read it
4. reserved name `body` — JSON body (T or *T, ggen or stdlib), OR
   `multipart.Form`/`*multipart.Form` (ParseMultipartForm→r.MultipartForm),
   OR `url.Values` (ParseForm→r.PostForm)
5. reserved name `query` — whole query: struct (`query:` tags), map[string]
   string/any, or `url.Values` (r.URL.Query() passthrough)
6. reserved name `headers` — whole header struct (`header:` tags) or http.Header
7. named type (or *T) embedding `http.ResponseWriter` — the writer itself,
   emitted as `w.(T)`; embeds are followed recursively, in-package types only.
   A failing assertion panics — the wrapper is expected to always pass one
8. anything else (bare scalar not in route, struct not body/query/headers) — error

Annotations (override a name; `whole` = dispatch on type like the reserved name):
- `/* rr:body */` — the body (type decides json/multipart/urlencoded)
- `/* rr:param [name] */` — a path param, optionally renamed to match a token
- `/* rr:query */` (bare) — whole query by type; `/* rr:query [key][=@check] */`
  a single value; `/* rr:query @parser */` whole via a custom parser
- `/* rr:header ... */` — same three forms, for headers

`@ref` resolution: package func, method of the api (receiver-relative), regexp
var, or qualified `pkg.Fn` (signature unknown → assumed transformer).
Checkers: `func(string) bool` filters (fail → next candidate / 404);
transformers `func(string) (T, error)` bind T as handler arg. Handler returns:
nothing, `error`, `T`, `(T, error)` — `T`/`(T,error)` always JSON-encoded.

Middleware are guards, not `func(http.Handler) http.Handler`: any binding
params, must return bool (false = handled, stop) or error (→ onerror). They
cannot wrap the ResponseWriter — writer-wrapping concerns (gzip, tracing)
belong in an outer wrapper like `example/api/server.go`.

## Semantics that were argued about (do not regress)

- Same-position params dispatch in declaration order; generator forces
  numerics before bools (ParseBool's lax "1"/"0" must not steal digits) and
  catch-any string last. Same class twice at a position = generate error;
  terminal vs descend edges never compete. When every candidate at a position
  is followed by exactly one terminal literal and no two share it
  (`{id int}/number`, `{v=@re}/uuid`, `{s}/profile`), the generator switches
  on the literal FIRST and runs only that candidate's validator
  (`terminalLits`): distinct literals make the candidates mutually exclusive,
  so order is preserved by construction. Before, the last candidate paid
  every earlier Atoi/regexp/ParseBool (and their error allocs): 340 -> 190 ns,
  6 -> 2 allocs on the paramchain fixture, the rest being SetPathValue. A
  shared literal (`{id int}/edit` + `{slug}/edit`) keeps the ordered chain.
- Method gates hoist to the top of method-uniform subtrees, before the
  rest-of-path slicing: wrong method beats structural mismatch (405 > 404).
- `r.Pattern` (ServeMux format, `{name}` without checker internals) and
  SetPathValue are stamped only after the route is fully selected, before
  guards. Every string param lands in PathValue even when also an arg;
  transformed params don't (no raw string). `-nopathvalue` (OPT-IN, never the
  default) drops every SetPathValue call: the lazily allocated PathValue map
  plus its GC is ~90% of a dynamic hit (189 → 13 ns). `r.Pattern` is still
  stamped. Rest/segment vars whose only reader was SetPathValue are not
  declared (`readsRest`, the `used` check on segment vars).
- Error chains: handler AND encode errors → owner api onerror → central
  onerror → bare 500 (route-level errorhandler was removed; owner/central
  onerror is the only override, e.g. a controller mapping its errors →404).
- Error/condition handlers (on404/on405/on400/onerror) bind params like route
  handlers via the same buildArgs, NOT a fixed signature: `http.ResponseWriter`
  (or a type embedding it) required; `*http.Request`, `context.Context`,
  `error`, query and header binds optional; body and
  path params forbidden (no route context). onerror REQUIRES an error param;
  on400 may take one (nil for plain-bad); on404/on405 must NOT. Must be
  in-package (introspected for their args). `argError` binds the `err` var in
  scope, `nil` when plain-bad.
- Default error responses are bodyless `WriteHeader` calls.
- Generated output carries NO comments — only the `// Code generated by rr. DO
  NOT EDIT.` header (Go convention, and `isGenerated` keys off it when scanning
  sibling packages). Helper doc comments live on the consts in main.go, not
  in the emitted string.
- Literal prefix tests are emitted as plain compares (`hasPrefix`/`openCut`:
  `len(p) >= n && p[:n] == "lit"`), NEVER `strings.CutPrefix`/`HasPrefix`. Go's
  inliner drops its budget from 80 to 20 inside functions over 5,000 nodes (no
  directive, flag or PGO overrides it), which a ~100+ route `ServeHTTP` is, so
  those calls stop inlining; the plain compare is expanded by the compiler at
  any size. `strings.IndexByte` has no such form and stays a call in big
  dispatchers. `strings` is imported only when IndexByte is emitted
  (`gen.useStrings`) — a static-only dispatcher doesn't need it.
  IndexByte is emitted only where a segment scan is unavoidable: sibling
  descLits of ONE width with no param/wildcard siblings check the slash at a
  constant offset and slice fixed (`if len(p) > k && p[k] == '/' { switch
  p[:k]`), and terminal literals need no "no slash" guard at all (a whole
  compare fails on a slash anyway) — only a terminal PARAM keeps
  `IndexByte(p, '/') < 0`. Measured on 2000 routes: -7% friendly, -12%
  uniform/Zipf. For that terminal scan the call is the fastest form: an
  inline byte loop was +50% on uniform (no SIMD, mispredicted exit), a
  non-inlined local helper -3% but a wash on friendly; not worth a helper.
- Rest-of-path advances IN PLACE (`path = path[i+1:]`, `gen.advance`) wherever
  the node reads it no more afterwards: lone literal chains always, a literal
  descent when the node has no param edges or catch-all, the LAST param descent
  when there's no catch-all. Otherwise a fresh `pN` is declared, because a
  failed branch falls through to siblings that read the old value. A child owns
  the variable it's handed, so the rule is purely local. Why: Go spills a named
  variable to its own stack slot and never shares slots across switch cases —
  a fresh var per route made a 2000-route frame 32KB (now 80 bytes) and cost
  5-13% on big dispatchers. Segment vars (`sN`) are still one per param node.
- Same rule for anything stack-allocated per route: the encoding/json write
  path is the generated `writeJSON(w, v)` helper, NOT inline
  `json.NewEncoder(w).Encode(v)` — NewEncoder inlines, its Encoder doesn't
  escape, and each route's copy got its own 120 byte slot (780 routes = 91KB
  frame, 88 bytes with the helper; ServeHTTP also shrank 460KB -> 254KB).
  Check `SUBQ $..., SP` in the ServeHTTP prologue when adding per-route code.
- Content-Type is stamped as `w.Header()["Content-Type"] = __jsonCT`, a
  package-level shared `[]string`, NEVER `Header.Set`: Set canonicalizes the
  key (byte-validating walk) and allocates a fresh slice per response — ~30%
  of a small ggen reply's CPU and its only allocation. Safe because the key
  is canonical already and net/http never appends to the value slice in place.
  Emitted once (`helperJSONCT`) whenever any JSON write path is in use.
  Measured on 780 routes: ggen reply -45% and 0 allocs, encoding/json -18%.
- `(T, error)` results and decoded JSON bodies assign into ONE variable per
  type, declared at the top of ServeHTTP (`gen.slot`, named after the type: `post`, `newUsers`, ... plus a
  shared `var err error`, spliced in by `spliceDecls` after the body is
  emitted). A `v1, err :=` per route gave every 64B `Post` result its own
  slot. Only one route runs per request, so sharing is safe. Derived path
  param matchers assign the same way (`if intVal, err = strconv.Atoi(s1)`),
  keyed by type AND trie position (`gen.slotAt`: a route has one param per
  position, so `intVal`, `intVal1`), and a same-typed handler result reuses
  the first such slot (`gen.slot` -> `gen.byType`; args are passed before the
  result lands). Predeclared names get `Val` (`int` is not a keyword; `var int
  int` compiles and then breaks every later `int`). Measured: scalars were
  already register-resident with disjoint scopes, so this is uniformity, not
  frame — the frame wins are the address-taken values above. `new(T)` bodies
  (heap) and custom `@fn` transformers (return type unknown) are not hoisted.
  Remainder in examples/simple (~1.3KB) is inlined HANDLER bodies (ListPosts'
  make+range, Search's map literal) — the handler author's, not rr's.
- Literal sibling sets of `-hashmin` (default 256; 0 = every set the planner
  can hash, even below the jump-table minimums; -1 = never) keys or more dispatch through a
  generated hash, not a string switch (`emitStrSwitch`/`planHash`; all five
  literal-switch sites go through it, smaller sets emit the plain switch). Go
  compiles a string switch to a binary search — ~log2(n) conditional branches
  that mispredict under spread-out traffic (4.6 misses/request at 2000 routes).
  The hash is a multiply-shift over up to 4 greedily chosen distinguishing
  bytes (+ length when lengths differ) into a dense integer switch, which the
  compiler turns into ONE jump table; the key is verified inside the arm and
  colliding keys chain as `else if` (max 3, else it falls back to the switch).
  Mixed lengths: every key must be readable at the chosen positions, so they
  are limited to the shortest key's length (counted from the start, or from
  the end when lengths differ, plus the length itself). One very short key
  would starve the hash, so the shortest keys peel off into a plain switch in
  the `else` of the length guard until the rest plan well.
  The plan must keep slots dense (range <= 4x arms) or Go builds no jump table.
  The keys' shared head (their byte-wise LCP, when >1 byte) is verified ONCE
  before the hash and each arm compares only its distinguishing tail
  (`path[:12] == "static/page-"` then `path[12:] == "038"`). A head under 2
  bytes is NOT hoisted and the arms then MUST keep the whole key — stripping
  it anyway 404'd every hashed set sharing one leading byte (`r000`..`r299`);
  `examples/large`'s `versions/vNNN` set pins that. 6-12% less
  ServeHTTP code, -23% on the 2000-route friendly cycle, -5% on a
  literal-then-param hit, ~neutral elsewhere. The plain switch (below the
  threshold) hoists the same head once it has 4+ arms. It does NOT replace the
  hash: measured on 2000 routes the hoisted switch is still 81 ns on uniform
  traffic vs 46 ns hashed — the binary search's mispredictions are the cost,
  not the compare width. No suffix hoist: a shared word mid-path becomes a
  prefix at that trie level, true common suffixes are rare in route tables.
  Measured and rejected: a perfect hash (the displacement load delays the jump,
  slower) and a shared table-driven body (1 miss even on repeating traffic).
  The trade: one mispredicted jump ALWAYS, so it loses when the binary search
  predicts well — skewed traffic under ~256 keys (hence the threshold), and
  traffic walking the keys in sorted order (+52% on the 1000-static fixture).
  At 2000 routes: uniform -44%, Zipf -19%, fixed 64-route cycle +5%.
- ggen integration: types with generated `DecodeFromStream`/`AppendJSON` use
  pooled fast paths. Foreign `pkg.T` / `[]pkg.T` count too: `foreignGgen`
  scans that package's sources (`scanPkg`, stdlib skipped) for the method OR a
  `//ggen:generate` directive on the type, so the sibling's ggen run need not
  come first. A foreign ggen type in any route also flips `ggenOn`. Writes of
  Marshalers are NOT generated — they call `ggen.WriteTo` /
  `ggen.WriteSliceTo`, which own ggen's pool, so
  Content-Type is stamped BEFORE the call (it writes bare). Consequence: on an
  encode error the header is already set; `http.Error` overwrites it, a custom
  onerror that only calls WriteHeader does not. Still generated: `readJSON[T]`
  / `readJSONSlice[T]` (ggen has no pooled reader) and `writeJSONAny`
  (`ggen.AppendAny` has no pooled writer) — once ggen is in play at all,
  arbitrary values go through it instead of encoding/json.
- Buffer pools: `-helpers <import path>` points at a package exporting
  `ReaderPool` and/or `WriterPool` (`sync.Pool` or `*sync.Pool`), shared by
  every generated dispatcher in the tree — otherwise each package pools alone
  and N apis mean N pools. Each pool falls back INDEPENDENTLY: a missing export
  silently emits the package-local pool, a present-but-wrong-typed one is fatal.
  Helpers living in the package being generated emit unqualified (no
  self-import). Keep read and write pools SEPARATE: read buffers size to
  request bodies, write buffers to responses. Stream decode = strings copied
  out, buffer recycles immediately.

## Generator internals (main.go)

Pipeline: parse package dir → collect apis/routes/directives (`apiType`,
`route`, `tok`, `argSpec`) → link central mounts + cross-package `xmounts`
(`discoverPrefix` → `scanPkg` re-parses the sibling package's sources for its
route patterns and nested xmounts; both memoized by package) →
resolve refs (`refExpr{method,name}` rendered against a receiver expr so `s.f`
becomes `s.Some.f` in merged dispatchers) → classify checkers/ggen shapes → emit.

Emission: `emitDispatcher` (per api/central; emits
`xmount` prefix-delegations before its own tree, and short-circuits to just
notFound when the central has no routes of its own) → static switch + trie
(`tnode`/`insert`/`compress`) → `emitNode` (one IndexByte per node, sibling
literals via segment switch, param edges term/desc split) → `emitLeaf` →
`emitDispatch` (method) → `emitCall` (Pattern, PathValue, guards, bindings,
call, response). Helpers (`readJSON` etc., float32 wrapper, pools) are string
consts appended once, gated by `gen.use*` flags. Output goes through
`format.Source`; on syntax errors the raw text is still written for debugging.

## ggen (../ggen)

Sibling project, same author. Structs need `//ggen:generate`; validation via
`pipe:"@fn"` tags (NOT `ggen:` — silently ignored). Everything lives in the root
package `ggen` (the old `scan`/`decode`/`encode` subpackages are gone); the cli
is a nested module, `github.com/sirkostya009/ggen/cmd/ggen`. Generic `Stream`
methods mean consumers need go 1.27. Decode-time validation failures surface as
readJSON errors → the 400 handler with the error attached.
\