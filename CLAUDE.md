# rr

Codegen-first HTTP framework for Go. A CLI (`cmd`) parses `//api:`
directives off structs and methods and generates a single `ServeHTTP` per
dispatcher: common-prefix cut, whole-path switch for static routes, segment
trie for dynamic ones. No runtime library — generated code depends only on
stdlib (+ ggen when present). Think NestJS annotations, compiled to a switch.

## Layout

- `cmd/main.go` — the whole generator, single file
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
    - `services/` — shared storage layer; records carry both id shapes
- `go.work` ties in both examples plus `../ggen` and `../ggen/cli` (local,
  unreleased module)

## Commands

```sh
go build ./cmd                            # root module: the generator
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
prefix, emitting `if strings.HasPrefix(path, prefix) { field.ServeHTTP(w, r);
return }` (longest prefix first). It discovers the sub-api's prefix by `go list
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
1. `http.ResponseWriter` / `*http.Request` — by type, any position, optional
2. explicit annotation, if present (see below) — overrides everything after
3. name ∈ route `{tokens}` — path param; T ∈ string/int/float64/float32/bool
   derives the matcher (Atoi, ParseFloat, strict ParseBool); struct/iface/any
   is a generate-time error. A route token overrides a reserved NAME (not an
   explicit annotation)
4. reserved name `body` — JSON body (T or *T, ggen or stdlib), OR
   `multipart.Form`/`*multipart.Form` (ParseMultipartForm→r.MultipartForm),
   OR `url.Values` (ParseForm→r.PostForm)
5. reserved name `query` — whole query: struct (`query:` tags), map[string]
   string/any, or `url.Values` (r.URL.Query() passthrough)
6. reserved name `headers` — whole header struct (`header:` tags) or http.Header
7. anything else (bare scalar not in route, struct not body/query/headers) — error

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
  terminal vs descend edges never compete.
- Method gates hoist to the top of method-uniform subtrees, before the
  rest-of-path slicing: wrong method beats structural mismatch (405 > 404).
- `r.Pattern` (ServeMux format, `{name}` without checker internals) and
  SetPathValue are stamped only after the route is fully selected, before
  guards. Every string param lands in PathValue even when also an arg;
  transformed params don't (no raw string).
- Error chains: handler AND encode errors → owner api onerror → central
  onerror → bare 500 (route-level errorhandler was removed; owner/central
  onerror is the only override, e.g. a controller mapping its errors →404).
- Error/condition handlers (on404/on405/on400/onerror) bind params like route
  handlers via the same buildArgs, NOT a fixed signature: `http.ResponseWriter`
  required; `*http.Request`, `error`, query and header binds optional; body and
  path params forbidden (no route context). onerror REQUIRES an error param;
  on400 may take one (nil for plain-bad); on404/on405 must NOT. Must be
  in-package (introspected for their args). `argError` binds the `err` var in
  scope, `nil` when plain-bad.
- Default error responses are bodyless `WriteHeader` calls.
- Generated output carries NO comments — only the `// Code generated by rr. DO
  NOT EDIT.` header (Go convention, and `isGenerated` keys off it when scanning
  sibling packages). Helper doc comments live on the consts in cmd/main.go, not
  in the emitted string.
- ggen integration: types with generated `DecodeFromStream`/`AppendJSON` use
  pooled fast paths. Writes of Marshalers are NOT generated — they call
  `encode.WriteTo` / `encode.WriteSliceTo`, which own ggen's pool, so
  Content-Type is stamped BEFORE the call (it writes bare). Consequence: on an
  encode error the header is already set; `http.Error` overwrites it, a custom
  onerror that only calls WriteHeader does not. Still generated: `readJSON[T]`
  / `readJSONSlice[T]` (ggen has no pooled reader) and `writeJSONAny`
  (`encode.AppendAny` has no pooled writer) — once ggen is in play at all,
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

## Generator internals (cmd/main.go)

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
`pipe:"@fn"` tags (NOT `ggen:` — silently ignored). Its cli is a nested
module (`../ggen/cli` in go.work). Decode-time validation failures surface as
readJSON errors → the 400 handler with the error attached.
\