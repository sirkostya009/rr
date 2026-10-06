# Consumer library plan — SHIPPED

Status: implemented. `gen/` is the parser, `build.go` the adapter, `main.go`
the CLI and emitter, `examples/tsclient` the worked example. The gate held:
regenerating all three example services produces byte-identical output.

Goal: a Go library that hands a small script every route rr can find, with
its metadata (owning api struct, method, raw pattern, mux-style path, parsed
segments, path/query/header/body bindings, result type, error handlers,
guards, composition), so the script can write an HTTP client or API wrapper in
any language. The library describes ROUTES, not types: a body or response
type is a reference (import path + name), and a script that needs the shape
pairs it with `github.com/sirkostya009/ggen/gen` (`set.Type(pkg, name)`).

This is deliberately smaller than ggen/gen: no Output/File/Place/Emitter
framework, no shipped emitters, no lowering modes. Load, iterate, write files
yourself. If a second consumer ever wants shared file plumbing, it can borrow
ggen/gen's Output, which is already language-neutral.

## Decisions

- Package `github.com/sirkostya009/rr/gen`, in the ROOT module (not a nested
  module like ggen/gen: rr has no deps to fence off, and `go install
  github.com/sirkostya009/rr` keeps working). Stdlib only, go/ast only, no
  go/types, no x/tools. Same as the CLI today.
- The CLI parses THROUGH the library. main.go keeps only what decides Go
  output (ref rendering against a receiver, ggen fast-path shapes, derived
  matcher classes, slots, trie, emission). Everything that reads sources and
  decides MEANING (directives, patterns, handler args, annotations, query and
  header struct binds, mounts, cross-package prefix discovery, ref
  classification: checker vs transformer vs regexp) moves to `gen`. One
  parser, so a directive change cannot drift between CLI and library, and the
  three example suites cover the library's front-end for free.
- Acceptance gate for the refactor: `go generate` over every example and
  `git diff --exit-code examples/` must be empty. The emitter is untouched, so
  the output must be byte-identical.
- Errors: `Load` returns `error`. Internally the moved code keeps its
  `fatalf` call sites, but `fatalf` panics with `*gen.Error` and `Load`
  recovers it. Zero churn in the moved functions, real errors for scripts,
  and the CLI prints `rr: <msg>` exactly as before. Positions
  (`token.Position`) on errors are a follow-up, not part of step 1.
- Route patterns are absolute as written (`/v1/users/{id}`), so composition
  (mounts, xmounts, prefixes) is METADATA for a script, never something it
  has to join. `API.Prefix` and `XMount.Prefix` are what the dispatcher cuts,
  exposed because they are cheap and true.
- Route handler args stay POSITIONAL in the model (`Route.Args`, one entry per
  Go parameter, declaration order). That is the single representation the CLI
  needs to emit a call; scripts get grouped accessors (`Body()`, `Query()`,
  `Headers()`, `Params()`) over the same slice. No second representation to
  keep in sync.
- The model exposes the AST it came from (`Package.Fset`, `Package.Files`,
  `Route.Decl`, `API.Spec`). Stdlib types, honest, and the CLI's leftover
  analysis (`embedsWriter`, `ggShape`) runs off them without a private
  back-channel. Scripts that want a doc comment or a struct tag the model
  does not carry can read it.

## Package layout

```
gen/
├── model.go    Set, Package, API, Route, Segment, Param, Arg, TypeRef, Ref, Handler, kinds
├── load.go     Load (patterns → dirs via `go list`), per-dir parse, index maps, link, recover
├── parse.go    parseDirectives, parsePattern, handlerArgs, parseParamAnnotation,
│               analyzeQueryBind, analyzeHeaderBind, returnKind, recvTypeName (moved)
├── link.go     mounts, xmounts, scanPkg, discoverPrefix, pkgDir, commonPrefixOf, lcpTrim (moved)
├── resolve.go  ref resolution + classification (the `resolve`/`classify`/`classifyParser`
│               closures in main today, minus refExpr rendering)
├── error.go    Error, fatalf → panic, recover
└── gen_test.go loads examples/{simple,versioned,large}, asserts the model
main.go         flags, Load, CLI-side per-route emit state, emission (unchanged)
examples/tsclient/  script + golden test: TS fetch client for examples/versioned
```

## Model

```go
package gen

func Load(patterns ...string) (*Set, error) // dirs or ./... patterns, via `go list -f`

type Set struct {
	Packages []*Package // one per loaded package, go list order
}
func (s *Set) Routes() iter.Seq[*Route] // every route: package → api → declaration order

type Package struct {
	Path, Name, Dir string
	APIs            []*API // types carrying rr: directives, declaration order
	Fset            *token.FileSet
	Files           []*ast.File // non-test, non-generated
}

type API struct {
	Pkg     *Package
	Name    string // Go type name, e.g. UsersApi
	Doc     string // doc comment minus directive lines
	Spec    *ast.TypeSpec
	Central bool // //rr:api
	Mounted bool // merged into a central of this package: serves through it
	Prefix  string // literal prefix its dispatcher cuts ("" if mounted)
	Mounts  []Mount  // in-package api-typed fields, field order
	XMounts []XMount // cross-package api-typed fields, prefix-delegated
	Routes  []*Route // own routes, declaration order
	Guards  []Handler // //rr:pre chain, in order
	OnError, On400, On404, On405 *Handler
}
func (a *API) Served() iter.Seq[*Route] // own + mounted routes: what its ServeHTTP handles

type Mount struct{ Field string; API *API }
type XMount struct {
	Field, Pkg, Type string // import path + type name
	Prefix           string // discovered from that package's sources
	API              *API   // nil unless that package was loaded too
}

type Route struct {
	API      *API   // owner
	Method   string // "" = any method
	Pattern  string // as written: /users/{id=@isUUID}/posts/{rest...}
	Path     string // mux form: /users/{id}/posts/{rest...} (r.Pattern minus method)
	Segments []Segment
	Handler  string // method name
	Doc      string
	Decl     *ast.FuncDecl
	Pos      token.Position
	Args     []Arg    // handler params, declaration order
	Result   *TypeRef // nil for no value
	Err      bool     // returns error / (T, error)
}
func (r *Route) Static() bool
func (r *Route) Params() iter.Seq[*Param]  // path params in pattern order, bound or not
func (r *Route) Body() *Arg
func (r *Route) Query() iter.Seq[*Arg]
func (r *Route) Headers() iter.Seq[*Arg]

type SegKind int // Literal, Param, Rest ({name...} or *)
type Segment struct {
	Kind  SegKind
	Lit   string
	Param *Param // Param and Rest; nil for a bare *
}

type Param struct {
	Name  string
	Type  TypeRef // the bound handler arg's type; string when unbound
	Check *Ref    // {name=@ref}: checker (bool), transformer ((T, error)) or regexp var
	Rest  bool
	Arg   *Arg    // the handler arg reading it, nil if only matched
}

type ArgKind int // Path, Body, Query, Header, Writer, Request, Context, Error
type Arg struct {
	Kind ArgKind
	Name string
	Type TypeRef
	// Path
	Param *Param
	// Body
	Body BodyKind // JSON, Multipart, Form
	// Query / Header
	Shape  BindShape // Value, Struct, Map, Values, Parser
	Key    string    // Value: the query key / header name
	Check  *Ref      // Value: =@check or =@transform; Parser: the @parser
	Fields []Field   // Struct: exported fields bound by tag or name
}
type Field struct{ Name, Key, Type string } // "string", "int", "bool"

type TypeRef struct {
	Expr  string // as written in the signature: "*services.User", "[]User", "map[string]any"
	Pkg   string // import path of the outermost named type; "" for builtins/unnamed
	Name  string // "User"
	Ptr   bool
	Slice bool
}

type RefKind int // Func, Method, Regexp, External (qualified pkg.Fn, signature unknown)
type Ref struct {
	Name      string // as written after @
	Kind      RefKind
	Transform bool // returns (T, error) / T: binds a value, else a bool checker
}

type Handler struct { // guard or error/condition handler
	Ref  Ref
	Args []Arg
	Err  bool // returns error (guard) / takes error (on400, onerror)
}
```

`Load` runs the same validation the CLI does today at parse and link time
(malformed patterns, unresolvable refs, bad param types, multiple bodies,
missing routes on a marked type). Dispatch-level checks (two same-class params
at one position, hash planning) belong to the emitter and stay in main.go:
`Load` succeeding does not promise `rr` generates. Scripts run after `go
generate` in practice, so this is fine and documented.

## What a script looks like

```go
set, err := gen.Load("./api/...")
if err != nil {
	log.Fatal(err)
}
types, _ := ggen.Load("./api/...", "./services") // only if the target needs shapes

for _, pkg := range set.Packages {
	for _, api := range pkg.APIs {
		if api.Mounted || api.Central && len(api.Routes) == 0 {
			continue // served through a central, or a pure composition root
		}
		for rt := range api.Served() {
			// rt.Method, rt.Path, rt.Segments, rt.API.Name, rt.Handler,
			// rt.Params(), rt.Body(), rt.Query(), rt.Headers(), rt.Result
			if b := rt.Body(); b != nil && b.Type.Pkg != "" {
				shape := types.Type(b.Type.Pkg, b.Type.Name) // ggen/gen shape for the target language
			}
		}
	}
}
```

Client method name = `rt.Handler`; client class = `rt.API.Name`; URL template
= segments (Param → interpolate, Rest → append raw); query = `rt.Query()`
(Struct → one key per Field, Value → one key, Map/Values/Parser → opaque
record); headers likewise; body = `rt.Body()` (JSON → serialize the type,
Multipart → FormData, Form → URLSearchParams); response = `rt.Result`
(always JSON) or void.

## Steps (all done)

1. **Extracted the front end.** `gen/` owns the model plus the moved
   parse/link/resolve code. `fatalf` panics with `*gen.Error`; `Load`, `Scan`
   and `Prefix` recover it, so the CLI prints the same messages as before.
   `Load` skips files carrying the rr header (`isGenerated`) rather than
   special-casing the output path, which covers `-o` too.
   `build.go` projects the model onto the emitter's internal types and keeps
   everything Go-specific: ggen shapes, derived matchers, rendered type names,
   the import set. The emitter is untouched apart from renaming its `gen`
   struct to `emitter` (the package name now needs that identifier).
2. **Library tests.** `gen/gen_test.go` loads all three examples and asserts
   the model. `TestServedMatchesDispatcher` compares `Served()` against the
   `r.Pattern` stamps in the generated output, which pins model and emitter
   together across simple, v1 and the 860-route large fixture.
3. **Worked example.** `examples/tsclient` generates a TypeScript fetch
   client, with golden files for the versioned and simple services.
4. **Docs.** `gen/SKILL.md`, a README section, `examples/tsclient/README.md`,
   CLAUDE.md layout and internals.

## Decisions taken during the build

- **Adapter, not a rewrite of the emitter.** The emitter keeps its own
  `apiType`/`route`/`param`/`argSpec`, built from the model in one place.
  Sharing the public types would have meant side tables keyed by pointer for
  emit-only state (ggen shapes, derived matchers) and a 1,500-line diff
  through the emitter, for no gain the byte-identical gate could verify.
- **`Api` -> `API`** everywhere in the model: revive's var-naming is enabled
  and the repo lints clean.
- **Guards now resolve their query and header binds.** The old CLI skipped
  `resolveQH` for `//rr:pre` args, so a guard binding a whole query struct or
  a checked value emitted an unresolved ref. Only programs that already failed
  to compile change.
- **`gen.CommonPrefix` / `gen.CommonPathPrefix` are exported**: the emitter
  needs both, and duplicating the prefix rule would break the invariant that a
  parent dispatcher and its child agree by construction.

## Open questions, answered

- **Package name**: `gen`, for symmetry with `ggen/gen`.
- **Does `Load` accept a single file?** No. Directories and `...` patterns
  only; the CLI passes `filepath.Dir(input)`. Patterns resolve through
  `go list`, with a plain-directory fallback when it is unavailable.
- **Does the tsclient example depend on `ggen/gen`?** No. It emits body and
  response types as named imports from a sibling module, which is the honest
  split: rr describes routes, ggen describes types. That also let the example
  live in the root module instead of needing a pinned requirement on rr.

## Not done, on purpose

- `rr -dump` writing the model as JSON for non-Go scripts.
- `token.Position` on `*gen.Error` (routes already carry `Route.Pos`).
- A shipped OpenAPI emitter.
