# rr

Codegen-first HTTP routing for Go. Annotate structs and methods with `//rr:`
directives, run the generator, and get one `ServeHTTP` per API: a common-prefix
cut, a switch over static routes, and a segment trie for dynamic ones. No
runtime router, no reflection. Generated code imports only the standard library
(plus [ggen](https://github.com/sirkostya009/ggen) if you use it).

```go
//go:generate go run github.com/sirkostya009/rr@latest $GOFILE

//rr:api onerror=@handleError on404=@notFound
type Api struct {
	UsersApi
	PostsApi
}

type UsersApi struct{ db *sql.DB }

//rr:route GET /api/users/{id}
func (a *UsersApi) GetUser(ctx context.Context, id int) (User, error) {
	...
}

//rr:route POST /api/users
func (a *UsersApi) CreateUser(body NewUser) (User, error) {
	...
}
```

`id int` makes the route match only numeric segments. `body` is decoded from
JSON. The `User` result is encoded as JSON. Errors go to `handleError`.

## Install

```sh
go install github.com/sirkostya009/rr@latest
```

Or run it from `go generate` without installing, as above. The input file only
anchors the package and names the output (`api.go` → `api_gen.go`). Directives
are read from every file in the package.

```
rr [-o output.go] [-helpers import/path] [-nopathvalue] [-hashmin n] input.go
```

| Flag           | Effect                                                                                                                            |
| -------------- | --------------------------------------------------------------------------------------------------------------------------------- |
| `-o`           | output file, defaults to `<input>_gen.go`                                                                                         |
| `-helpers`     | package exporting `ReaderPool` / `WriterPool` (`sync.Pool`), shared by every generated dispatcher instead of one pool per package |
| `-nopathvalue` | skip `r.SetPathValue`. Path params then reach the handler only as arguments, but a dynamic hit gets ~10x cheaper                  |
| `-hashmin`     | literal sets this large dispatch through a generated hash instead of a string switch. Default 256, `0` always, `-1` never         |

## Directives

| Directive                                                 | On     | Meaning                                                              |
| --------------------------------------------------------- | ------ | -------------------------------------------------------------------- |
| `//rr:api [onerror=@f on400=@f on404=@f on405=@f]`        | struct | a dispatcher. Its api-typed fields (embedded or named) merge into it |
| `//rr:controller [onerror=@f on400=@f on404=@f on405=@f]` | struct | optional, attaches error handlers to a mounted sub-api               |
| `//rr:pre @f @g`                                          | struct | guard chain run before every route of that api                       |
| `//rr:route [METHOD] /path`                               | method | a route. No method matches any method                                |

Any directive may end with a `-- comment`.

### Patterns

- `/users/{id}`: one segment
- `/users/{id=@ref}`: one segment, checked or converted by `@ref`
- `/files/{path...}`: the rest of the path, slashes included
- `/files/*`: same, unnamed

Static segments always beat params at the same position. Several params may
share a position (`/items/{id}` bound as `int` and `/items/{slug}` as
`string`) and are tried in
declaration order, with numbers before bools and plain strings last. Two of
the same kind at one position is a generate error.

### `@ref`

A `@ref` names a package func, a method of the api, a `*regexp.Regexp` var, or
a qualified `pkg.Fn`.

- `func(string) bool` is a checker: false means try the next candidate, then 404
- `func(string) (T, error)` or `func(string) (T, bool)` is a transformer: its
  `T` is passed to the handler, an error or false rejects the value
- a regexp is matched with `MatchString` as written, so anchor it yourself

## Handler parameters

Parameters bind by name and type. No annotation is needed for the usual cases.

| Parameter                                                 | Binds                                                                                                             |
| --------------------------------------------------------- | ----------------------------------------------------------------------------------------------------------------- |
| `http.ResponseWriter`, `*http.Request`, `context.Context` | by type, any position, all optional                                                                               |
| name matches a `{token}`                                  | path param. `string`, `int`, `float64`, `float32`, `bool`. The type picks the matcher (`int` only matches digits) |
| `body`                                                    | JSON (`T`, `*T` or `iter.Seq2[T, error]`), `multipart.Form`, or `url.Values` (form post)                          |
| `query`                                                   | struct with `query:` tags, `map[string]string`, `map[string]any`, or `url.Values`                                 |
| `headers`                                                 | struct with `header:` tags, or `http.Header`                                                                      |
| type embedding `http.ResponseWriter`                      | the writer itself, type-asserted from `w`                                                                         |

Inline annotations override the name:

```go
func (a *Api) Search(
	in /* rr:body */ SearchRequest,
	page /* rr:query page=@atoi */ int,
	rid /* rr:header X-Request-Id */ string,
	slug /* rr:param name */ string,
) ([]Result, error)
```

`/* rr:query */` and `/* rr:header */` take three forms: bare (whole value by
type), `key[=@check]` (one value), or `@parser` (whole value through a custom
func).

## Results and errors

A handler returns nothing, `error`, `T`, or `(T, error)`. A `T` is always
written as JSON.

Handler and encode errors go to the owning api's `onerror`, then the central
`onerror`, then a bare 500. Error handlers bind parameters like handlers do.
They need an `http.ResponseWriter`, and may take `*http.Request`,
`context.Context`, query and header binds:

- `onerror` must take an `error`
- `on400` may take an `error` (nil when the request is just malformed)
- `on404` may take an `error`: the path param conversion that failed, nil when
  none did or the transformer returns `(T, bool)`
- `on405` takes none

Without a handler, the response is a bodyless status code. A wrong method on
an existing path is a 405, not a 404.

## Guards

`//rr:pre` functions are guards, not `func(http.Handler) http.Handler`. They
bind parameters like handlers and return `bool` (false = already handled, stop)
or `error` (sent to `onerror`).

```go
//rr:pre @requireToken
type AdminApi struct{}

func requireToken(w http.ResponseWriter, auth /* rr:header Authorization */ string) bool {
	if auth != "Bearer letmein" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}
```

Guards can't wrap the writer. Gzip, tracing and the like belong in an outer
`http.Handler` around the generated one.

## Composition

Fields of an `//rr:api` from the same package merge into one dispatcher: one
prefix cut, one switch and one trie for every mounted api.

A field from another package (`v1.Api`) delegates by prefix instead. rr reads
that package's sources to learn its prefix, so generation order doesn't matter
and `go generate ./...` works from a clean tree. See
[examples/versioned](examples/versioned).

## ggen

When a body or result type has [ggen](https://github.com/sirkostya009/ggen)
`DecodeFromStream` / `AppendJSON` methods, rr uses them with pooled buffers.
This also works for types from other packages. Other types fall back to
`encoding/json`.

A body declared as `iter.Seq2[T, error]`, with ggen methods on `T`, is a JSON
array read one element at a time as the handler ranges over it, so nothing is
read before the loop and the array is never held whole. A malformed body
shows up as an error inside the loop, not as a 400, so reporting it is the
handler's job.

```go
//rr:route POST /users/import
func (a *UsersApi) Import(body iter.Seq2[NewUser, error]) error {
	for u, err := range body {
		if err != nil {
			return err
		}
		// ...
	}
	return nil
}
```

## Request metadata

`r.Pattern` is set the way `http.ServeMux` sets it, and string path params are
available through `r.PathValue`, so existing middleware and tracing work
unchanged.

## Generating clients

The parser is a library. `github.com/sirkostya009/rr/gen` returns every route
with its api, handler, pattern (as written and in ServeMux form), segments and
parameter bindings. The CLI uses the same package, so scripts see exactly what
rr sees.

```go
set, err := gen.Load("./api/...")
if err != nil {
	log.Fatal(err)
}
for rt := range set.Routes() {
	fmt.Println(rt.Method, rt.Path, "->", rt.API.Name+"."+rt.Handler)
}
```

`gen` describes routes, not types. Pair it with `ggen/gen` for the JSON
shapes. [examples/tsclient](examples/tsclient) builds a TypeScript fetch
client and its types this way. The model is documented in
[gen/SKILL.md](gen/SKILL.md).

## Examples

- [examples/simple](examples/simple): one package, nearly every feature
- [examples/versioned](examples/versioned): `/v1` and `/v2` packages composed under one api
- [examples/large](examples/large): hashed dispatch over sets of hundreds of routes
- [examples/tsclient](examples/tsclient): TypeScript client generation

## Benchmarks

From [bench](bench), a deeply nested REST surface, single goroutine, median of
6 runs (`cd bench && go test -bench . -benchmem -count 6`):

| Benchmark                    | rr                    | [httx](https://github.com/sirkostya009/httx) | httprouter          | gin                 |
| ---------------------------- | --------------------- | -------------------------------------------- | ------------------- | ------------------- |
| Simple (static)              | **6.4 ns**            | 16.0 ns                                      | 25.3 ns             | 50.2 ns             |
| SingleParam                  | **28.9 ns**           | 74.5 ns                                      | 77.4 ns (1 alloc)   | 77.4 ns             |
| MultiParam (5 params)        | 235.5 ns              | 233.3 ns                                     | 206.1 ns (1 alloc)  | **178.4 ns**        |
| RegexParam (digit-validated) | **241.4 ns**          | 421.9 ns (4 allocs)                          | —                   | —                   |
| Wildcard                     | 289.9 ns              | 254.8 ns                                     | 211.8 ns (1 alloc)  | **186.5 ns**        |
| MethodMismatch (405)         | **75.9 ns** (1 alloc) | 487.0 ns (1 alloc)                           | 893.5 ns (7 allocs) | 509.2 ns (4 allocs) |
| NotFound                     | **9.6 ns**            | 148.8 ns                                     | 226.7 ns            | 172.2 ns (1 alloc)  |

httprouter and gin don't validate digits, so they're left out of RegexParam.

## Status

The directive surface and generated code are still changing between commits.
