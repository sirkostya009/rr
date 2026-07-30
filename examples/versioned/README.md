An example showing use of rr in a api layout that actively versions http api
using subpackages.

## Layout

```
api/
  api.go        //rr:api Api, cross-package: composes v1 + v2, no routes of its
                own — the generator delegates by discovered prefix
  api_gen.go    generated composing dispatcher (do not edit)
  api_test.go   exercises the whole surface below, both versions
  v1/           /api/v1 — legacy int ids (uuids accepted, not the default)
    api.go      //rr:api Api type + the four error-page handlers
    users.go    basic CRUD, (T, error) returns, `body` param, auto int path
                param, uuid checker-string route alongside the int one
    posts.go    `query` struct binding, `body` pointer, controller-level
                onerror, two params sharing one path position (int vs.
                checker-string), ggen-validated field
    admin.go    //rr:pre guard (bearer token), method catch-all route
    search.go   custom whole-query parser, single header value, url.Values
                query, typed-dispatch trio (bool/float/string, one position)
    forms.go    `headers` struct, url.Values body, multipart.Form body
    files.go    trailing wildcard path param
  v2/           /api/v2 — same resources, uuid string ids only
    api.go      //rr:api Api type + error handlers + the uuid checker
    users.go    CRUD keyed by uuid, plain string path param
    posts.go    two checker-string params on one position (uuid vs. slug)
    admin.go    guarded stats + method catch-all
  (each version has its own api_gen.go router and api_ggen.go codecs,
   go:generate'd, do not edit)
services/       storage/domain layer both api versions share; records carry
                both key shapes (v1 int + uuid)
config/         env-based config (ADDR, FOO)
cmd/            main.go: wires Config -> Api -> http.Server
```

## Regenerate after editing any api file

```sh
go generate ./...
```

Order doesn't matter: the composing root reads v1's and v2's prefixes out of
their *sources*, not their generated files, so it can be generated before,
after, or without them.

Within a package, ggen runs first (JSON codecs for `//ggen:generate` structs),
then rr (the router itself, which detects and uses whatever ggen produced).

Each generated package emits its own buffer pools. To share one set across the
tree, point `-helpers <import path>` at a package exporting `ReaderPool` and/or
`WriterPool` on the go:generate lines — same behaviour, one pool instead of N.

## Run

```sh
ADDR=:8080 FOO=bar go run ./cmd
```

## Test

```sh
go test ./...
```
