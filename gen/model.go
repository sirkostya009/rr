// Package gen parses rr: directives out of Go sources and hands back every
// route it can find, with the metadata a code generator needs: the owning api
// struct, the HTTP method, the pattern as written, the same pattern in
// net/http mux form, the parsed segments, and what each handler parameter
// binds to (path, body, query, header).
//
// It is the front end of the rr CLI, exported so a script can generate an
// HTTP client or an API wrapper in any language without maintaining a second
// description of the routes. The CLI itself parses through this package, so a
// directive the CLI understands is a directive a script sees.
//
// gen describes ROUTES, not types. A body or response type is a reference
// (import path plus name); a script that also needs the shape of those types
// pairs this package with github.com/sirkostya009/ggen/gen.
//
// A minimal script:
//
//	set, err := gen.Load("./api/...")
//	if err != nil {
//		log.Fatal(err)
//	}
//	for rt := range set.Routes() {
//		fmt.Println(rt.Method, rt.Path, "->", rt.API.Name+"."+rt.Handler)
//	}
package gen

import (
	"go/ast"
	"go/token"
	"iter"
)

// Set is everything Load parsed.
type Set struct {
	Packages []*Package // one per loaded package, in the order Load resolved them
}

// Package is one Go package holding rr: directives.
type Package struct {
	Path string // import path, "" when it could not be resolved
	Name string // package clause
	Dir  string // directory on disk

	APIs  []*API // types carrying rr: directives, first-mention order
	Fset  *token.FileSet
	Files []*ast.File // parsed sources, excluding tests and rr's own output

	idx *Index
}

// Index is the raw AST index Package was built from. It is the escape hatch
// for a generator that needs a declaration this model does not carry.
type Index struct {
	Funcs   map[string]*ast.FuncDecl            // package-level funcs by name
	Methods map[string]map[string]*ast.FuncDecl // receiver type name -> method name
	Types   map[string]*ast.TypeSpec            // named types by name
	Regexps map[string]struct{}                 // package-level *regexp.Regexp vars
	Imports map[string]string                   // ident -> import path
	FileOf  map[*ast.FuncDecl]*ast.File
}

// Index returns the AST index behind the package.
func (p *Package) Index() *Index { return p.idx }

// API is one type carrying rr: directives: a dispatcher root (//rr:api), a
// controller mounted into one, or a plain route owner.
type API struct {
	Pkg  *Package
	Name string // Go type name
	Doc  string // doc comment with the rr: directive lines removed
	Spec *ast.TypeSpec

	Recv string // receiver name its route methods use
	Ptr  bool   // its route methods take a pointer receiver

	Central bool // carries //rr:api: merges its api-typed fields into one dispatcher
	Mounted bool // merged into a central of this package; it has no ServeHTTP of its own

	// Prefix is the literal path prefix this dispatcher cuts before matching.
	// Empty for a mounted api, which serves through its central.
	Prefix string

	Mounts  []Mount  // api-typed fields in this package, merged into this dispatcher
	XMounts []XMount // api-typed fields from another package, delegated by prefix
	Routes  []*Route // routes declared on this type, declaration order

	Guards []*Handler // //rr:pre chain, in order

	OnError *Handler // onerror=@f
	On400   *Handler // on400=@f
	On404   *Handler // on404=@f
	On405   *Handler // on405=@f
}

// Served iterates every route this api's dispatcher answers: its own, plus
// those of the apis it merges. Cross-package mounts are not included; they
// are delegated to the mounted api's own dispatcher.
func (a *API) Served() iter.Seq[*Route] {
	return func(yield func(*Route) bool) {
		for _, rt := range a.Routes {
			if !yield(rt) {
				return
			}
		}
		for _, m := range a.Mounts {
			for _, rt := range m.API.Routes {
				if !yield(rt) {
					return
				}
			}
		}
	}
}

// Mount is an api-typed field whose routes are merged into the central's
// dispatcher.
type Mount struct {
	Field string
	API   *API
}

// XMount is an api-typed field from another package. Its routes cannot be
// merged, so requests under Prefix are handed to that api's own dispatcher.
type XMount struct {
	Field  string
	Pkg    string // import path of the mounted api's package
	Type   string // its type name
	Prefix string // the prefix it owns, read from that package's sources
	API    *API   // the mounted api, when Load also parsed its package; else nil
}

// Route is one //rr:route.
type Route struct {
	API     *API
	Method  string // "" matches any method
	Pattern string // as written: /users/{id=@isUUID}/posts/{rest...}
	Path    string // net/http mux form: /users/{id}/posts/{rest...}

	Segments []Segment

	Handler string // method name
	Doc     string // doc comment with the rr: directive lines removed
	Decl    *ast.FuncDecl
	Pos     token.Position

	Args []*Arg // handler parameters, declaration order

	// Result is the value the handler returns, always encoded as JSON. Nil
	// when it returns nothing or only an error.
	Result *TypeRef
	Err    bool // the handler returns an error
}

// Static reports whether the route is a fixed path with no params or
// catch-all.
func (r *Route) Static() bool {
	for _, s := range r.Segments {
		if s.Kind != SegLit {
			return false
		}
	}
	return true
}

// Params iterates the route's path params in pattern order, including a
// trailing catch-all and whether or not a handler parameter reads them. A
// bare * has no param and is skipped.
func (r *Route) Params() iter.Seq[*Param] {
	return func(yield func(*Param) bool) {
		for i := range r.Segments {
			if p := r.Segments[i].Param; p != nil && !yield(p) {
				return
			}
		}
	}
}

// Body returns the handler parameter carrying the request body, or nil.
func (r *Route) Body() *Arg {
	for _, a := range r.Args {
		if a.Kind == ArgBody {
			return a
		}
	}
	return nil
}

// Query iterates the handler parameters bound to the query string.
func (r *Route) Query() iter.Seq[*Arg] { return argsOfKind(r.Args, ArgQuery) }

// Headers iterates the handler parameters bound to request headers.
func (r *Route) Headers() iter.Seq[*Arg] { return argsOfKind(r.Args, ArgHeader) }

func argsOfKind(args []*Arg, k ArgKind) iter.Seq[*Arg] {
	return func(yield func(*Arg) bool) {
		for _, a := range args {
			if a.Kind == k && !yield(a) {
				return
			}
		}
	}
}

// SegKind tells a path segment apart.
type SegKind int

// Path segment kinds.
const (
	SegLit   SegKind = iota // a literal segment
	SegParam                // {name} or {name=@check}
	SegRest                 // a trailing {name...} or *
)

// Segment is one path segment of a route pattern.
type Segment struct {
	Kind  SegKind
	Lit   string // SegLit
	Param *Param // SegParam, and SegRest unless the pattern ends in a bare *
}

// Param is a path param: one whole segment, or the rest of the path.
type Param struct {
	Name string // the {token} name

	// Type is the Go type the handler binds it to. A param no handler reads,
	// and any catch-all, is a string.
	Type TypeRef

	// Check is the {name=@ref} validator: a checker that filters candidates,
	// a transformer that produces Type, or a regexp var.
	Check *Ref

	Rest bool // a trailing {name...}: the raw rest of the path, slashes included
	Arg  *Arg // the handler parameter reading it, nil when only matched

	checker string // raw @ref from the pattern, before resolution
}

// ArgKind is what a handler parameter binds to.
type ArgKind int

// What a handler parameter binds to.
const (
	ArgPath    ArgKind = iota // a path param
	ArgBody                   // the request body
	ArgQuery                  // the query string, whole or one key
	ArgHeader                 // request headers, whole or one name
	ArgWriter                 // http.ResponseWriter, or a named type embedding it
	ArgRequest                // *http.Request
	ArgContext                // context.Context, bound to r.Context()
	ArgError                  // error; only on an error handler
)

// BodyKind is the wire format of a body parameter, decided by its Go type.
type BodyKind int

// Request body formats.
const (
	BodyJSON      BodyKind = iota // a struct, slice or map decoded from JSON
	BodyMultipart                 // multipart.Form
	BodyForm                      // url.Values, from an urlencoded body
)

// BindShape is how a query or header parameter is bound.
type BindShape int

// Query and header binding shapes.
const (
	BindValue  BindShape = iota // one key, optionally checked or transformed
	BindValues                  // url.Values / http.Header / map[string][]string, passed through
	BindMap                     // map[string]string or map[string]any
	BindStruct                  // a struct, fields keyed by tag or name
	BindParser                  // a custom @func taking the whole query or header
)

// Arg is one handler parameter and what it binds to.
type Arg struct {
	Kind ArgKind
	Name string  // the Go parameter name
	Type TypeRef // as declared; zero for a plain http.ResponseWriter, *http.Request, context.Context or error

	Param *Param // ArgPath: the route param it reads

	Body BodyKind // ArgBody

	// ArgQuery / ArgHeader
	Whole  bool      // bind the whole query or header, dispatched by type
	Shape  BindShape // how the value is bound
	Key    string    // BindValue: the query key or header name
	Fields []Field   // BindStruct: the exported fields bound
	MapAny bool      // BindMap: map[string]any rather than map[string]string

	// Check is a validator or transformer: on a path param the one from the
	// pattern, on a BindValue query or header the one from the annotation, on
	// a BindParser the parser itself.
	Check *Ref

	byName  bool   // role came from a reserved parameter name, so a route token overrides it
	checker string // raw @ref from the annotation, before resolution
}

// FieldKind is the Go type of a bound query or header struct field.
type FieldKind int

// Go types a bound query or header struct field may have.
const (
	FieldString FieldKind = iota
	FieldInt
	FieldBool
)

// Field is one field of a struct bound to the whole query or header.
type Field struct {
	Name string // Go field name
	Key  string // query key or header name
	Kind FieldKind
}

// TypeRef is a Go type as a handler wrote it.
type TypeRef struct {
	Node ast.Expr // the declared type, nil when there is none
	Expr string   // rendered as written, e.g. "*services.User" or "[]User"

	// Pkg and Name identify the named type at the core of Expr, after any
	// pointer or slice. Pkg is its import path, empty for a builtin, an
	// unnamed type, or a package Load could not resolve.
	Pkg  string
	Name string

	Ptr   bool // declared as *T
	Slice bool // declared as []T
	// Seq is a body declared as iter.Seq2[T, error]: a JSON array of T, read
	// an element at a time as the handler ranges over it.
	Seq bool
}

// RefKind is what an @ref resolved to.
type RefKind int

// What an @ref resolved to.
const (
	RefFunc     RefKind = iota // a func in the same package
	RefMethod                  // a method of the owning api
	RefRegexp                  // a package-level *regexp.Regexp var, used via MatchString
	RefExternal                // a qualified pkg.Fn; its signature is unknown, so it is assumed to transform
)

// Ref is a resolved @ref from a directive or an annotation.
type Ref struct {
	Name string // as written after the @
	Kind RefKind

	// Transform reports whether the ref produces a value the handler
	// receives, rather than filtering the request. A checker is
	// func(string) bool; a transformer is func(string) (T, error).
	Transform bool

	// Err reports whether the ref returns an error alongside its value. A
	// whole-query or whole-header parser may return a bare T; a path or value
	// transformer always returns (T, error).
	Err bool

	Decl *ast.FuncDecl // nil for a regexp var or an external ref
}

// Handler is a guard from //rr:pre, or one of the error and condition
// handlers (onerror, on400, on404, on405). Both bind parameters the way a
// route handler does.
type Handler struct {
	Ref  Ref
	Args []*Arg

	RetErr   bool // a guard returning error rather than bool
	TakesErr bool // takes an error parameter
}

// Routes iterates every route in the set: package order, api order,
// declaration order. Each route is yielded once, by its owning api.
func (s *Set) Routes() iter.Seq[*Route] {
	return func(yield func(*Route) bool) {
		for _, p := range s.Packages {
			for _, a := range p.APIs {
				for _, rt := range a.Routes {
					if !yield(rt) {
						return
					}
				}
			}
		}
	}
}

// API finds a loaded api by import path and type name.
func (s *Set) API(pkgPath, name string) *API {
	for _, p := range s.Packages {
		if p.Path != pkgPath {
			continue
		}
		for _, a := range p.APIs {
			if a.Name == name {
				return a
			}
		}
	}
	return nil
}
