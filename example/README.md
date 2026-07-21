# example

A small but complete service built on [rr](../), exercising nearly
every generator feature. Separate Go module so it depends on the generator
only as a dev tool (`go run rr/cmd`), never at runtime.

## Layout

```
api/
  api.go        //api:central Api type + the four error-page handlers
  server.go     Server: wraps Api with sentry tracing + slog request logging
  users.go      basic CRUD, (T, error) returns, auto int path param
  posts.go      struct query binding, pointer body, two params sharing one
                path position (int vs. checker-string), ggen-validated field
  admin.go      //api:middleware guard (bearer token), method catch-all route
  search.go     custom whole-query parser, header binding, typed-dispatch
                trio (bool/float/string sharing one path position)
  files.go      trailing wildcard path param
  api_gen.go    generated router (go:generate'd, do not edit)
  api_ggen.go   generated JSON codecs via ggen (do not edit)
  api_test.go   exercises the whole surface above
config/         env-based config (includes optional SENTRY_DSN)
cmd/            main.go: wires Config -> Api -> Server -> http.Server
```

## Regenerate after editing any api/*.go file

```sh
cd api && go generate .
```

Runs ggen first (JSON codecs for `//ggen:generate` structs), then rr
(the router itself, which detects and uses whatever ggen produced).

## Run

```sh
ADDR=:8080 FOO=bar go run ./cmd
```

`SENTRY_DSN` is optional — unset, the SDK stays disabled and tracing is a
no-op; `LOG_LEVEL=info` turns on the per-request slog line.

## Test

```sh
go test ./...
```
