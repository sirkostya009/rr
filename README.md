# rr

Codegen-first HTTP routing for Go. Annotate structs and methods with
`//api:` directives, run the generator, get one hand-written-looking
`ServeHTTP` per API — a common-prefix cut, a `switch` over static routes, and
a segment trie for dynamic ones. No runtime router, no reflection, nothing
imported at request time beyond stdlib (and [ggen](../ggen) if you use it).

Think of it as the annotation ergonomics of a framework like NestJS or Spring
compiled down to a plain `switch` statement.

```go
//api:central response=json onerror=@handleError on404=@notFound
type Api struct {
	UsersApi
	PostsApi
}

//api:route GET /api/users/{id}
func (a *UsersApi) GetUser(id int) (User, error) {
	...
}
```

```sh
go run ./cmd api.go   # -> api_gen.go, package-scoped
```

## Install

```sh
go install github.com/sirkostya009/rr/cmd
```

See [example/](example/) for a fully worked, multi-API service exercising
essentially every feature, and [CLAUDE.md](CLAUDE.md) for the complete
directive/semantics reference.

## Highlights

- **Zero-allocation static & typed-param matching.** Static routes are a
  string switch; the segment trie shares literal prefixes across routes and
  does at most one `IndexByte` per node.
- **Path params bind by declared type.** `func GetUser(id int)` derives an
  `Atoi` matcher automatically; `float64`/`float32`/`bool` too. Multiple
  handlers can share a path position as long as their param types don't
  overlap (`int` and `bool`, not `int` and `float64`) — dispatched in
  declaration order, validated at generate time.
- **`@ref` params**: `{slug=@isSlug}` (checker), `{id=@strconv.Atoi}`
  (transformer, binds as a typed handler arg), or `{id=@someRegexp}` (a
  `*regexp.Regexp` var, matched directly).
- **Full parameter binding**, not just path params: `api:query` (scalar,
  struct via `query:` tags, map, `url.Values`, or a whole-query custom
  parser), `api:header`, `api:body` (JSON, `T` or `*T`), `http.ResponseWriter`
  and `*http.Request` bind by type and are optional.
- **Composition.** `//api:central` merges any number of api-typed fields into
  one dispatcher — one prefix cut, one switch, one trie across every mounted
  API — instead of stacking `http.ServeMux` handlers (which cost real
  allocations per mount).
- **Middleware are guards**, not `func(Handler) Handler`: plain functions
  that bind params like handlers and return `bool` or `error`. No wrapping,
  no closures, no allocation.
- **[ggen](../ggen) integration.** If a request/response type has
  ggen-generated `DecodeFromStream`/`AppendJSON` methods, the generator uses
  them automatically through pooled buffers — no config. Falls back to
  `encoding/json` per type when ggen isn't in play.
- **`r.Pattern`** (Go 1.22+ `http.Request.Pattern`) is populated exactly like
  `http.ServeMux` would, so tracing/metrics middleware works unmodified.

## Benchmarks

| Benchmark                    | rr                    | [httx](github.com/sirkostya009/httx) | httprouter          | gin                 |
| ---------------------------- | --------------------- | ------------------------------------ | ------------------- | ------------------- |
| Simple (static)              | **5.4 ns**            | 9.6 ns                               | 13.8 ns             | 31.9 ns             |
| SingleParam                  | **24.5 ns**           | 43.5 ns                              | 54.9 ns (1 alloc)   | 45.7 ns             |
| MultiParam (5 params)        | **86.1 ns**           | 139.5 ns                             | 144.7 ns (1 alloc)  | 104.1 ns            |
| RegexParam (digit-validated) | **30.5 ns**           | 294.8 ns (4 allocs)                  | —                   | —                   |
| Wildcard                     | **94.0 ns**           | 158.8 ns                             | 149.4 ns (1 alloc)  | 114.1 ns            |
| MethodMismatch (405)         | **79.4 ns** (1 alloc) | 316.7 ns (1 alloc)                   | 673.0 ns (7 allocs) | 355.4 ns (4 allocs) |
| NotFound                     | **12.5 ns**           | 91.7 ns                              | 130.2 ns            | 126.3 ns (1 alloc)  |
| TrailingSlash                | n/a                   | **86.0 ns**                          | 176.2 ns            | 221.6 ns            |
| CaseInsensitive              | n/a                   | **87.2 ns**                          | 183.9 ns            | 220.1 ns            |

RegexParam only compares routers with real digit validation (httprouter/gin
have none). TrailingSlash/CaseInsensitive only compare routers with a
redirect feature — rr's route set is fixed at compile time and doesn't have
one yet.

## Status

Actively evolving; the directive surface and generated-code shape are not
stable across commits.
