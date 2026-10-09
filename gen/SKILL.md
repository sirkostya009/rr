---
name: rr-gen
description: Drive github.com/sirkostya009/rr/gen. Emit HTTP clients and API wrappers in any language (TypeScript, Kotlin, Swift, Dart, OpenAPI, …) from the //rr: directives of a Go backend, by writing a small Go script. Use when a frontend, a mobile app or another service needs a client that tracks the Go routes without anyone maintaining it. Covers the model, script shape, binding shapes, and the split with ggen.
---

# gen — the route model behind the rr CLI

A library, not a CLI. Write a small Go script: load the packages, walk the
routes, write the target language yourself. gen has no opinion about the
output — no templates, no file layout, no emitters.

Module: the rr root module, `github.com/sirkostya009/rr/gen`. Standard library
only, go/ast only, no type checking.

The rr CLI parses through this package. A directive the CLI understands is a
directive a script sees, and there is no second parser to drift.

## When to reach for gen

Use it when any of these hold:

- A frontend, a mobile app or another service calls an rr backend and its
  client is written by hand.
- A hand-maintained client has already drifted from the routes once.
- You want an OpenAPI or Postman description of a service whose routes already
  live in `//rr:route` directives.

Skip it when:

- You want the shape of the JSON types, not the routes. gen describes routes.
  Pair it with `github.com/sirkostya009/ggen/gen`, which describes types.
- The backend does not use rr.

## Install

```sh
go get github.com/sirkostya009/rr
```

Put the script in its own directory and run it with `go run ./cmd/client`.
Nothing runs it for you; no `//go:generate` convention is imposed.

## Minimal script

```go
package main

import (
	"fmt"
	"log"

	"github.com/sirkostya009/rr/gen"
)

func main() {
	set, err := gen.Load("./api/...")
	if err != nil {
		log.Fatal(err)
	}
	for rt := range set.Routes() {
		fmt.Println(rt.Method, rt.Path, "->", rt.API.Name+"."+rt.Handler)
	}
}
```

Three steps, always: **load → walk → write**.

## Step 1: load

`gen.Load(patterns...)` takes directories or `...` patterns, resolved the way
`go list` resolves them. It returns a `*Set` of `*Package`.

Load parses sources only. It never reads rr's own output, so a missing or
stale `api_gen.go` cannot change what it reports and the result does not
depend on generation order. Test files are skipped.

A package with no rr: directives is skipped silently; a load that finds none
at all is an error.

## Step 2: the model

```
Set        Packages, Routes(), API(pkgPath, name)
Package    Path, Name, Dir, APIs, Fset, Files, Index()
API        Name, Doc, Central, Mounted, Prefix, Mounts, XMounts, Routes,
           Guards, OnError, On400, On404, On405, Served()
Route      API, Method, Pattern, Path, Segments, Handler, Doc, Decl, Pos,
           Args, Result, Err, Static(), Params(), Body(), Query(), Headers()
Segment    Kind (SegLit | SegParam | SegRest), Lit, Param
Param      Name, Type, Check, Rest, Arg
Arg        Kind, Name, Type, Param, Body, Whole, Shape, Key, Fields, MapAny, Check
TypeRef    Node, Expr, Pkg, Name, Ptr, Slice, Seq
Ref        Name, Kind, Transform, Err, Decl
Handler    Ref, Args, RetErr, TakesErr
```

### Routes

`Pattern` is the route as written, checkers and all
(`/users/{id=@isUUID}`). `Path` is the same route in net/http ServeMux form
(`/users/{id}`), which is also what the generated dispatcher stamps into
`r.Pattern`. Build URLs from `Segments`, not from either string.

`Method` is empty when the route answers any method.

`Result` is the value the handler returns, always encoded as JSON. It is nil
when the handler returns nothing or only an error. `Err` says whether an error
comes with it; a client cannot see the difference, so it usually only needs
`Result`.

Patterns are absolute as written, so composition never has to be joined by
hand. `API.Prefix` and `XMount.Prefix` are there because they are cheap and
true, not because a client needs them.

### Composition

An api is one of three things:

- `Central` (`//rr:api`): merges the routes of its api-typed fields into one
  dispatcher. `Mounts` holds those fields, `Served()` iterates its own routes
  plus theirs.
- `Mounted`: merged into a central of the same package. It has no dispatcher
  of its own, so a client should nest it under its central, not treat it as a
  root.
- Neither: a plain route owner with its own dispatcher.

`XMounts` are api-typed fields from another package. Their routes cannot be
merged, so requests under `Prefix` are delegated to that api's own dispatcher.
`XMount.API` points at the mounted api when the load covered its package too,
which is how a script nests `client.v1.users` under `client`.

The roots of a client are the apis that are neither `Mounted` nor the target
of an `XMount` in the set.

### Parameter binding

`Route.Args` is one entry per Go parameter, in declaration order. `Arg.Kind`
says what it binds to:

| Kind         | Meaning                                                                   |
| ------------ | ------------------------------------------------------------------------- |
| `ArgPath`    | a `{token}` from the pattern; `Arg.Param` points at it                    |
| `ArgBody`    | the request body; `Arg.Body` is `BodyJSON`, `BodyMultipart` or `BodyForm` |
| `ArgQuery`   | the query string, whole or one key                                        |
| `ArgHeader`  | request headers, whole or one name                                        |
| `ArgWriter`  | `http.ResponseWriter`, or a named type embedding it                       |
| `ArgRequest` | `*http.Request`                                                           |
| `ArgContext` | `context.Context`                                                         |
| `ArgError`   | an `error`, only on an error handler                                      |

The last four are server plumbing. A client generator ignores them.

For a query or header arg, `Shape` says how it binds:

| Shape        | Wire meaning                                                             |
| ------------ | ------------------------------------------------------------------------ |
| `BindValue`  | one key, named by `Arg.Key`                                              |
| `BindStruct` | a struct; `Arg.Fields` lists each key and its `string`/`int`/`bool` kind |
| `BindMap`    | `map[string]string` or `map[string]any`, keys unknown                    |
| `BindValues` | `url.Values`, `http.Header` or `map[string][]string`, keys unknown       |
| `BindParser` | a custom `@func` took the whole thing, keys unknown                      |

The last three have no key list. Fall back to a record type.

`Arg.Name` is the Go parameter name; `Arg.Key` is the wire key, which an
annotation may have renamed. Use `Key` on the wire and `Name` for a readable
signature.

### Path params

`Route.Params()` iterates the params in pattern order, whether or not a
handler reads them: a param only matched still has to appear in the URL.

`Param.Type` is the Go type the handler declared, so `{id}` bound to an `int`
tells a client to send a number. A param no handler reads, and every
catch-all, is a `string`.

`Param.Rest` marks a trailing `{name...}`: the raw rest of the path, slashes
included. Interpolate it without escaping. A bare `*` is a `SegRest` with no
`Param` — the client can only address the prefix.

`Param.Check` is the `{name=@ref}` validator. `Transform` tells a checker
(`func(string) bool`, filters) from a transformer (`func(string) (T, error)`,
produces `Param.Type`). A `RefRegexp` is a package-level `*regexp.Regexp`. A
client usually only needs `Param.Type`.

### Types

`TypeRef.Pkg` and `TypeRef.Name` identify the named type at the core of the
Go type, after any pointer or slice. `Pkg` is an import path, empty for a
builtin or an unnamed type. `Expr` is the type as written, and `Node` is the
AST node for anything the model does not resolve, such as a map.

Two packages may declare types with the same name (`v1.Post` and `v2.Post`).
Key by `Pkg` plus `Name` and qualify the emitted name when a bare one would
collide.

gen does not describe what is inside those types. Feed the same packages to
`github.com/sirkostya009/ggen/gen` and place the shape of every `TypeRef.Pkg`
plus `Name` the client mentions.

## Step 3: write

There is no output framework. Build the text and write the file. If the target
needs more than one file, decide that yourself.

## Escape hatches

`Package.Index()` gives the raw AST index the model was built from (funcs,
methods, types, imports, regexp vars, and the file each func came from), and
`Route.Decl`, `API.Spec`, `Package.Fset` and `TypeRef.Node` give the nodes
themselves. Reach for them when a target needs a struct tag or a doc comment
the model does not carry.

`gen.Scan(importPath)` reads another package's sources without loading it: the
methods its types declare, the types carrying `//ggen:generate`, and its
package-level vars. `gen.Prefix(importPath, typeName)` reports the path prefix
that package's dispatcher owns.

`gen.CommonPrefix` and `gen.CommonPathPrefix` are the two prefix rules the
generated dispatchers cut by, useful for deriving a base URL.

## Errors

Every rejection is a `*gen.Error` with the same message the rr CLI prints.
`Load` validates directives, patterns, `@ref` resolution and parameter roles.
It does not validate dispatch: two same-class params at one trie position are
a CLI error, not a Load error, so `Load` succeeding does not promise `rr`
generates. Scripts run after `go generate` in practice, so this rarely shows.

## Pitfalls

- **Do not treat every api as a root.** A `Mounted` api serves through its
  central; emitting it separately gives a client that calls the same route
  twice under two names.
- **Do not build URLs from `Path`.** It carries `{name}` placeholders. Walk
  `Segments`.
- **Do not skip `Params()` entries with no `Arg`.** The server matches the
  segment without reading it; the client still has to send one.
- **Do not escape a `Rest` param.** It is a path, slashes included.
- **Do not assume a query bind has keys.** `BindMap`, `BindValues` and
  `BindParser` do not.
- **Do not assume type names are unique.** Qualify by `Pkg` when they clash.

## Worked example

`examples/tsclient` generates a TypeScript fetch client from the routes AND
the TypeScript types it sends and receives, by running this package and
`ggen/gen` over the same packages in one pass. Golden files cover both example
services. Its `names.go` is the one place the two generators agree on a type's
name, which is all the coordination the pairing needs.
