# rr

Codegen-first HTTP framework for Go. A CLI (root package) parses `//api:`
directives off structs and methods and generates a single `ServeHTTP` per
dispatcher: common-prefix cut, whole-path switch for static routes, segment
trie for dynamic ones. No runtime library — generated code depends only on
stdlib (+ ggen when present). Think NestJS annotations, compiled to a switch.

## Layout

- `gen/` — the parser, exported as a consumer library (`gen.Load` ->
  `Set`/`Package`/`API`/`Route`/`Segment`/`Param`/`Arg`). Owns everything that
  decides MEANING: directives, patterns, handler args, annotations, query and
  header struct binds, mounts, cross-package prefix discovery, `@ref`
  classification. Stdlib + go/ast only. `gen/SKILL.md` is its user-facing doc
- `main.go` — the CLI and the whole emitter. Parses through `gen`, so there is
  one parser and it cannot drift from the library
- `build.go` — the adapter: projects the `gen` model onto the emitter's
  internal types (`apiType`/`route`/`param`/`argSpec`) and does everything
  Go-specific the model deliberately leaves out — ggen fast-path shapes,
  the matcher derived from a path param's declared type, rendered type names,
  the import set
- `examples/` — independent modules, each a living fixture + test suite
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
    `api_gen.go` stops containing 4 hashed switches or the catalog's
    literal-first switch (so a threshold or planner change can't silently
    drop coverage). `regressions.go` (+ `api/user`) holds one route group per
    generator bug fixed so far, each pinned by `regressions_test.go`: fixed
    width beside a terminal param, a body typed like a path param, a stdlib
    body costing unrelated routes allocs, slot names vs a `user` package and
    the receiver, an unnamed catch-all beside a param. `nopv/` is generated
    with `-nopathvalue` for the shapes only that flag reaches. `pre/` has a
    central `//rr:pre` guard binding what its routes bind (a guard's parsed
    header must not show through a route's absent one) and one `@fn`
    transformer bound as `any` and as `*Foo`. `miss/` has an on404 taking
    an error (which conversion it gets, scoped external transformer
    included) and `(T, bool)` transformers on a path and a query. The ggen-only
    case (a kept `*T` body overwritten by its own result) lives in
    `examples/simple` (`QueueDraft`, `TestKeptBodyNotOverwritten`)
  - `tsclient/` (module `tsclient`) — the consumer-library example: a script
    generating BOTH a TypeScript fetch client (from `rr/gen` routes) and the
    TypeScript types it sends and receives (from `ggen/gen` shapes), with
    golden files for the versioned and simple services. Its own module
    because `ggen/gen` pulls `golang.org/x/tools` and the rr root module
    stays dependency-free; a `replace` points at `../..` so it builds
    outside the workspace too. `names.go` is the one place the two
    generators agree on a type's TypeScript name. Every type is declared
    twice, `T` (encode) and `TInput` (decode), because the shapes differ
- `go.work` ties in root, the four example modules and `bench`; ggen comes
  from the pinned pseudo-version in each example's go.mod

## Commands

```sh
go build .                                 # root module: the generator
go test ./gen/...                          # the consumer library
cd examples/versioned && go generate ./... # any order; ggen runs before rr per pkg
cd examples/versioned && go vet ./... && go test ./...   # or examples/simple
cd examples/tsclient && go test .          # its own module; -update rewrites goldens
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
4. reserved name `body` — JSON body (T or *T, ggen or stdlib; `iter.Seq2[T,
   error]` with ggen T only, see below), OR
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
transformers `func(string) (T, error)` or `(T, bool)` bind T as handler arg
(`Ref.OK`, a shared `ok` var; a false is a reject with no error). Handler returns:
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
  declared (`readsRest`, the `used` check on segment vars). Both count only
  what emitNode(n) emits: n's own terminal routes are emitted by the PARENT
  against the parent's variable (`belowUsesPath` vs `subtreeUsesPath`) —
  counting them left `s0`/`p1` declared and unused.
- Error chains: handler AND encode errors → owner api onerror → central
  onerror → bare 500 (route-level errorhandler was removed; owner/central
  onerror is the only override, e.g. a controller mapping its errors →404).
- Error/condition handlers (on404/on405/on400/onerror) bind params like route
  handlers via the same buildArgs, NOT a fixed signature: `http.ResponseWriter`
  (or a type embedding it) required; `*http.Request`, `context.Context`,
  `error`, query and header binds optional; body and
  path params forbidden (no route context). onerror REQUIRES an error param;
  on400 may take one (nil for plain-bad); on404 may take one; on405 must
  NOT. Must be in-package (introspected for their args). `argError` binds the
  `err` var in scope, `nil` when plain-bad. on404's error is the shared `err`
  at the final fall-through: the last path param conversion that failed, as
  no successful one has reset it since. nil for a `(T, bool)` or checker
  reject, a structural miss, and at the prefix-mismatch 404. A scoped
  external transformer (`t%d`) copies its own `terrN` into `err`.
- Default error responses are bodyless `WriteHeader` calls.
- Generated output carries NO comments — only the `// Code generated by rr. DO
  NOT EDIT.` header (Go convention, and `isGenerated` keys off it when scanning
  sibling packages). Helper doc comments live on the consts in main.go, not
  in the emitted string.
- Literal prefix tests are emitted as plain compares (`hasPrefix`/`openCut`:
  `len(p) >= n && p[:n] == "lit"`), NEVER `strings.CutPrefix`. Go's inliner
  drops its budget from 80 to 20 inside functions over 5,000 nodes
  (`inline/inl.go` inlineBigFunctionMaxCost; `-l=4` and PGO don't override it,
  PGO says "hot-big check disallows inlining"), which a ~100+ route
  `ServeHTTP` is: CutPrefix (cost 42) and TrimPrefix (25) stop inlining.
  HasPrefix (18) still inlines, so the plain compare is just the uniform form,
  expanded by the compiler at any size. `strings.IndexByte` has no such form and stays a call in big
  dispatchers. `strings` is imported only when IndexByte is emitted
  (the emitter's `useStrings`) — a static-only dispatcher doesn't need it.
  IndexByte is emitted only where a segment scan is unavoidable: sibling
  descLits of ONE width with no param/wildcard siblings and NO TERMINAL PARAM
  check the slash at a constant offset and slice fixed (`if len(p) > k &&
  p[k] == '/' { switch p[:k]`). A terminal param forces the scan, and the
  literal arms must then slice by its `i`: slicing by the width anyway routed
  `/abcXd/q` to `/abc/{y}/q` and panicked on `/a/`. Terminal literals need no "no slash" guard at all (a whole
  compare fails on a slash anyway) — only a terminal PARAM keeps
  `IndexByte(p, '/') < 0`. Measured on 2000 routes: -7% friendly, -12%
  uniform/Zipf. For that terminal scan the call is the fastest form: an
  inline byte loop was +50% on uniform (no SIMD, mispredicted exit), a
  non-inlined local helper -3% but a wash on friendly; not worth a helper.
- Rest-of-path advances IN PLACE (`path = path[i+1:]`, `emitter.advance`) wherever
  the node reads it no more afterwards: lone literal chains always, a literal
  descent when the node has no param edges or catch-all, the LAST param descent
  when there's no catch-all. Otherwise a fresh `pN` is assigned, because a
  failed branch falls through to siblings that read the old value. A child owns
  the variable it's handed, so the rule is purely local. Why: Go spills a named
  variable to its own stack slot and never shares slots across switch cases —
  a fresh var per route made a 2000-route frame 32KB (now 80 bytes) and cost
  5-13% on big dispatchers.
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
- EVERY variable ServeHTTP uses is declared once at its top and only ever
  assigned (`x = ...`, never `x :=`): Go gives each declaration its own stack
  slot and never shares slots across switch cases, so per-route declarations
  grow the frame with the route count. Only one route runs per request, so
  routes share them. `path := r.URL.Path` is the one `:=`. Measured on the
  295-route root bench fixture: frame 768B -> 216B.
  - Trie variables are declared under their own names, per trie DEPTH
    (`emitter.fixedVar`): segment `sN`, rest-of-path `pN`, slash index `iN`
    (every node used to shadow its own `i`), mount `mpN`/`miN`. Depth only
    grows along a path, so one name is never live in two places; siblings at
    a depth assign it before reading it.
  - Typed values go through `declSlot`: params `slotAt` keyed by type AND
    depth (`if intVal, err = strconv.Atoi(s1)`, `intVal1`; custom `@fn`
    transformers take their own declared result type, or for an external
    `pkg.Fn` the type all handlers bind it as, else stay scoped), results `slot`
    reusing a param slot of the type (args are evaluated before the result
    is assigned), temporaries `tmp` keyed by role, type and ordinal within
    the route (query `url.Values`, raw strings, struct binds, parser results,
    map binds and their range vars). One shared `var err error`. A temp is
    shared by EVERY binding of the request, a central guard's included (the
    ordinal restarts per route), so the two that relied on a fresh zero
    value are zeroed before their conditional assignment: a transformed
    query/header value (`n = *new(int)`; else a guard's X-A=5 showed through
    a route's absent X-B) and a struct bind (`q = Filter{}`). Every other
    temp is always assigned before it is read.
  - Bodies have their OWN namespace (`bodySlot`, `bodyPtr`), never shared: a
    body sharing a param slot overwrote the param (`/items/7` + body 99 ->
    id=99); sharing a result slot made `post, err = s.Create(&post)` write
    the result over the `*T` the handler kept (a race under a queue).
  - An ADDRESS-TAKEN top-scope var is heap-allocated at function ENTRY, i.e.
    on every request: hoisting `var item Item` + `Decode(&item)` cost every
    unrelated route 1 alloc per body type (50 types = 50 allocs/3.2KB on
    GET /ping). So a body that escapes anyway is a top-scope `*T` the route
    points at `new(T)` (`postBodyPtr = new(Post)`): stdlib bodies (Decode
    takes an interface) and `*T` bodies the handler may keep. A `*T` body the
    handler provably doesn't keep (`keepsPointer` in build.go: every use is
    a field access, a dereference or a nil compare, no method, no `&`, no
    closure) decodes into the top-scope value and passes `&postBody`, which
    stays on the stack. A wrong "doesn't keep" only costs an allocation.
  - Names: `declSlot` hands out placeholders, `spliceDecls` names them once
    the body is emitted: the hint (`postBody`, a handler param name) or the
    type (`post`, `newUsers`, `Val` suffix for predeclared names like
    `int`), skipping every identifier the body or the slot types refer to
    and the receiver — `var user user.User` shadowed the package, `var a A`
    redeclared receiver `a`. Package qualifiers in the declared types are
    imported (`emitter.quals`); the old `:=` never named them.
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
  `examples/large`'s `versions/vNNN` set pins that. -23% on the 2000-route
  friendly cycle, -5% on a literal-then-param hit; code size is a wash
  (`examples/large` amd64 -0.3%). The plain switch (below the
  threshold) hoists the same head once it has 4+ arms. It does NOT replace the
  hash: measured on 2000 routes the hoisted switch is still 81 ns on uniform
  traffic vs 46 ns hashed — the binary search's mispredictions are the cost,
  not the compare width. No suffix hoist: a shared word mid-path becomes a
  prefix at that trie level, true common suffixes are rare in route tables.
  Measured and rejected: a perfect hash (the displacement load delays the jump,
  slower) and a shared table-driven body (1 miss even on repeating traffic).
  The trade: one indirect jump the predictor misses under random traffic, so
  it loses where the binary search predicts perfectly: a fixed repeating
  cycle (+46% at 500 routes; at 2000 within the ~8% layout noise of that
  shape) and keys walked in sorted order (+60%; the same 1000 static routes
  shuffled are -42%). Random orders at 2000 routes (`-nopathvalue`, amd64):
  uniform -41%, Zipf with permuted ranks -19%, 64 hot routes in random order
  -17%; branch misses ~1.0/request vs 1.6-4.2. With path values on the same
  wins are -8..-15% of a bigger total.
  JUMP TABLES EXIST ONLY ON amd64, arm64, loong64 (`sys.Arch.CanJumpTable`)
  and not under `-gcflags=-spectre=ret`. Elsewhere (386, arm, riscv64,
  ppc64le, s390x, wasm) the integer switch is a binary search again, so the
  hash is a small strict loss (+1..5% on 386, +2..27% with retpoline) and
  +5..7% code. The jump tables are also ~28KB of rodata on `examples/large`.
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
- An `iter.Seq2[T, error]` body (`gen.TypeRef.Seq`, emitter shape `ggSeq`) is
  `readJSONSeq[T](r)`, a lazy `Stream.Array` over the body passed straight
  into the handler call: no slot, no error check, no 400. Decode errors are
  yielded to the handler, which reports them through its own error return.
  The read buffer is drawn when the range starts and put back when it ends,
  inside the iterator, so ServeHTTP gains no defer. A non-ggen `T` or a
  pointer to the sequence is a generate-time error: stdlib has no lazy array.
- Buffer pools: `-helpers <import path>` points at a package exporting
  `ReaderPool` and/or `WriterPool` (`sync.Pool` or `*sync.Pool`), shared by
  every generated dispatcher in the tree — otherwise each package pools alone
  and N apis mean N pools. Each pool falls back INDEPENDENTLY: a missing export
  silently emits the package-local pool, a present-but-wrong-typed one is fatal.
  Helpers living in the package being generated emit unqualified (no
  self-import). Keep read and write pools SEPARATE: read buffers size to
  request bodies, write buffers to responses. Stream decode = strings copied
  out, buffer recycles immediately.

## Generator internals

Pipeline, front to back:

`gen.Load(dir)` — parse package dir (rr's own output skipped via `isGenerated`,
so `-o` and staleness can't poison it) → collect apis/routes/directives
(`API`, `Route`, `Segment`, `Arg`) → link central mounts + cross-package
`XMounts` (`discoverPrefix` → `scanPkg` re-parses the sibling package's
sources for its route patterns and nested xmounts; both memoized by package) →
resolve refs and classify checkers → stamp each dispatcher's `Prefix`.
Failures unwind by panic and come back from `Load` as a `*gen.Error` carrying
the same message the CLI prints.

`builder` (build.go) — project onto `apiType`/`route`/`tok`/`argSpec`, resolve
`refExpr{method,name}` (rendered against a receiver expr so `s.f` becomes
`s.Some.f` in merged dispatchers), classify ggen shapes, derive path-param
matchers from declared types, collect imports.

`emitter` (main.go) — `emitDispatcher` (per api/central; emits
`xmount` prefix-delegations before its own tree, and short-circuits to just
notFound when the central has no routes of its own) → static switch + trie
(`tnode`/`insert`/`compress`) → `emitNode` (one IndexByte per node, sibling
literals via segment switch, param edges term/desc split) → `emitLeaf` →
`emitDispatch` (method) → `emitCall` (Pattern, PathValue, guards, bindings,
call, response). Helpers (`readJSON` etc., float32 wrapper, pools) are string
consts appended once, gated by the emitter's `use*` flags. Output goes through
`format.Source`; on syntax errors the raw text is still written for debugging.

## ggen (../ggen)

Sibling project, same author. Structs need `//ggen:generate`; validation via
`pipe:"@fn"` tags (NOT `ggen:` — silently ignored). Everything lives in the root
package `ggen` (the old `scan`/`decode`/`encode` subpackages are gone); the cli
is a nested module, `github.com/sirkostya009/ggen/cmd/ggen`. Generic `Stream`
methods mean consumers need go 1.27. Decode-time validation failures surface as
readJSON errors → the 400 handler with the error attached.
\