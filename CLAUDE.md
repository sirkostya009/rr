# rr

Codegen-first HTTP framework for Go. A CLI (`cmd`) parses `//api:`
directives off structs and methods and generates a single `ServeHTTP` per
dispatcher: common-prefix cut, whole-path switch for static routes, segment
trie for dynamic ones. No runtime library — generated code depends only on
stdlib (+ ggen when present). Think NestJS annotations, compiled to a switch.

## Layout

- `cmd/main.go` — the whole generator, single file
- `example/` — separate module (`example`), the living fixture + test suite
  - `api/` — one file per sub-API, `api.go` holds the `//api:central` type,
    `server.go` the sentry/slog instrumentation wrapper
  - `api/api_gen.go` (router, generated), `api/api_ggen.go` (ggen codecs)
- `go.work` ties in `../ggen` and `../ggen/cli` (local, unreleased module)

## Commands

```sh
go build ./cmd                            # root module: the generator
cd example/api && go generate .           # runs ggen FIRST, then rr
cd example && go vet ./... && go test ./...
```

Generation is whole-package: the input file arg only anchors the directory and
the output name (`api.go` → `api_gen.go`); directives are scanned from every
file in the package. The stale output file is skipped while parsing.

## Directive surface

Patterns: `/x/{name}`, `/x/{name=@ref}`, trailing `/{name...}` or `/*`.
`{name:regex}` was removed — point `@ref` at a `*regexp.Regexp` var instead
(used as-is via MatchString, NOT auto-anchored).

On the central struct:
`//api:central response=json onerror=@f on400=@f on404=@f on405=@f middleware=@f`
— api-typed fields (incl. embedded) get their routes merged into one dispatcher.
Mounted APIs never get a standalone ServeHTTP; unmounted ones always do.

On any api type: `//api:middleware @f`, `//api:errorhandler @f`, `//api:400/404/405 @f`.

On handler methods: `//api:route [METHOD] /path`, `//api:errorhandler @f`,
`//api:response json`. ` -- comment` suffixes allowed everywhere.

Handler params bind by type and inline annotations; w/r are optional:
- `http.ResponseWriter`, `*http.Request` — by type, any position
- bare `name T` — path param by name; T ∈ string/int/float64/float32/bool
  derives the matcher (Atoi, ParseFloat, ParseBool); structs/interfaces/any
  are a generate-time error
- `/* api:param name */`, `/* api:body [json] */` (T or *T, ggen or stdlib),
  `/* api:query [key][=@check] */` (scalar, struct via `query:` tags,
  map[string]string/any, url.Values, or whole-query `@parser`),
  `/* api:header [Name][=@check] */`

`@ref` resolution: package func, method of the api (receiver-relative), regexp
var, or qualified `pkg.Fn` (signature unknown → assumed transformer).
Checkers: `func(string) bool` filters (fail → next candidate / 404);
transformers `func(string) (T, error)` bind T as handler arg. Handler returns:
nothing, `error`, `T`, `(T, error)` — values encode per `api:response`/central
default.

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
- Error chains: handler errors → route `//api:errorhandler` → owner api →
  central onerror. Encode failures deliberately SKIP route-level overrides
  (those map domain errors, e.g. →404) and go owner→central→bare 500.
  400 handlers may be `(w,r)` or `(w,r,err)`; err is nil for plain-bad.
- Default error responses are bodyless `WriteHeader` calls.
- ggen integration: types with generated `DecodeFromStream`/`AppendJSON` use
  pooled fast paths (`readJSON[T]`, `writeJSON`, `writeJSONSlice`); once ggen
  is in play at all, arbitrary values go through `encode.AppendAny`
  (`writeJSONAny`) instead of encoding/json. Separate read/write buffer
  pools. Stream decode = strings copied out, buffer recycles immediately.

## Generator internals (cmd/main.go)

Pipeline: parse package dir → collect apis/routes/directives (`apiType`,
`route`, `tok`, `argSpec`) → link central mounts → resolve refs
(`refExpr{method,name}` rendered against a receiver expr so `s.f` becomes
`s.Some.f` in merged dispatchers) → classify checkers/ggen shapes → emit.

Emission: `emitDispatcher` (per api/central) → static switch + trie
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
