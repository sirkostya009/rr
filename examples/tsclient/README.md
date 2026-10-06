# tsclient

A worked example of [`rr/gen`](../../gen): a Go script that turns the
`//rr:route` directives of a Go backend into a TypeScript fetch client, and
the Go structs those routes carry into the TypeScript types the client uses.

Nothing here is part of rr. Copy it, change the output, keep the traversal.

```sh
cd examples/tsclient
go run . -o ../../web/src ../versioned/...
go run . -o /tmp/out ../simple/api
```

Flags: `-o <dir>` (default `.`), `-client <file>` (default `client.ts`),
`-types <file>` (default `types.ts`).

## Two generators, one pass

| Library                         | Knows                                                          |
| ------------------------------- | -------------------------------------------------------------- |
| [`rr/gen`](../../gen)           | every route, its method, path, segments and parameter bindings |
| [`ggen/gen`](../../../ggen/gen) | what is inside every JSON type those routes carry              |

Neither knows the other's half, which is why this program uses both. The
naming lives in [names.go] so the client and the types module agree without
passing strings around.

That is also why this example is its own Go module: `ggen/gen` pulls in
`golang.org/x/tools`, and the rr root module stays dependency-free so
`go install github.com/sirkostya009/rr` never drags a toolchain library along.
It is in `go.work`, with a `replace` so it builds on its own too. Its tests
need the workspace, because the fixtures they generate from are sibling
modules.

## What it emits

`client.ts` holds one `createClient(options)` factory returning a nested
object that mirrors the api composition. `examples/versioned` composes two
versions cross-package, so the client nests the same way:

```ts
import { createClient } from "./client.ts";

const api = createClient({ baseUrl: "https://example.com" });

const users = await api.v1.users.getUsers();
const user = await api.v2.users.getUser("0f8b…");
const post = await api.v2.posts.createPost({ slug: "hi", title: "Hi", tags: [] });
const page = await api.v1.posts.listPosts({ page: 2, tag: "go" });
const me = await api.v1.users.getMe({ "X-User-Index": 0 });
```

Per route it emits path params as positional arguments, then `body`, `query`
and `headers` when the route binds them, then an optional `RequestInit`.
Responses are parsed as JSON; a handler returning nothing or only an error
gives a `Promise<void>`. A response the server did not mark ok throws
`HttpError`.

Multipart bodies take a `FormData`, urlencoded bodies a `URLSearchParams`,
JSON bodies the named type.

## Why every type is declared twice

`types.ts` declares each Go type in both directions, because the two shapes
genuinely differ: the encoder always writes a field the decoder is happy to do
without.

```ts
export interface V1Post {
	id: number;
	slug: string;
	title: string;
	tags?: string[];
}

export interface V1PostInput {
	id?: number;
	slug?: string;
	title?: string;
	tags?: string[] | null;
}
```

A request body takes the `…Input` shape, a response the plain one. Placing
both also keeps the module closed: a shape referencing a type that was not
placed fails, and a request shape reaches request shapes.

Names are bare unless two packages declare the same one, which is why
`v1.Post` and `v2.Post` come out as `V1Post` and `V2Post`.

A type with no `//ggen:generate` is not in the types module, and the client
falls back to `unknown` for it rather than importing a name that is not there.

## Tests

Golden files under `testdata/` cover the cross-package shape and the flat
single-package shape, client and types for each.

```sh
go test .
go test . -update   # after changing either emitter
```

They run in this module, not from the repo root, and they need the workspace:
the packages they generate from are the sibling example modules.
