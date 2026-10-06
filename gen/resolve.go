package gen

import (
	"go/ast"
	"reflect"
	"strconv"
	"strings"
)

// wrKind reports whether t is http.ResponseWriter, *http.Request,
// context.Context or error — all bound by type, no annotation needed.
func wrKind(t ast.Expr) (ArgKind, bool) {
	if id, ok := t.(*ast.Ident); ok && id.Name == typError {
		return ArgError, true
	}
	if st, ok := t.(*ast.StarExpr); ok {
		if sel, ok := st.X.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == pkgHTTP && sel.Sel.Name == "Request" {
				return ArgRequest, true
			}
		}
		return 0, false
	}
	if sel, ok := t.(*ast.SelectorExpr); ok {
		id, ok := sel.X.(*ast.Ident)
		if !ok {
			return 0, false
		}
		switch {
		case id.Name == pkgHTTP && sel.Sel.Name == "ResponseWriter":
			return ArgWriter, true
		case id.Name == "context" && sel.Sel.Name == "Context":
			return ArgContext, true
		}
	}
	return 0, false
}

// embedsWriter reports whether t is an in-package named type (or pointer to
// one) that embeds http.ResponseWriter, directly or through another such type.
// Handlers taking one get `w` type-asserted into it.
func embedsWriter(t ast.Expr, types map[string]*ast.TypeSpec, depth int) bool {
	if depth > 8 {
		return false
	}
	if st, ok := t.(*ast.StarExpr); ok {
		t = st.X
	}
	id, ok := t.(*ast.Ident)
	if !ok {
		return false
	}
	ts := types[id.Name]
	if ts == nil {
		return false
	}
	var fields *ast.FieldList
	switch u := ts.Type.(type) {
	case *ast.StructType:
		fields = u.Fields
	case *ast.InterfaceType:
		fields = u.Methods
	default:
		return false
	}
	if fields == nil {
		return false
	}
	for _, f := range fields.List {
		if len(f.Names) > 0 { // not embedded
			continue
		}
		if k, ok := wrKind(f.Type); ok && k == ArgWriter {
			return true
		}
		if embedsWriter(f.Type, types, depth+1) {
			return true
		}
	}
	return false
}

// isRegexpType reports whether t is *regexp.Regexp.
func isRegexpType(t ast.Expr) bool {
	st, ok := t.(*ast.StarExpr)
	if !ok {
		return false
	}
	sel, ok := st.X.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "regexp" && sel.Sel.Name == "Regexp"
}

// isRegexpCompile reports whether e is a regexp.MustCompile(...) call.
func isRegexpCompile(e ast.Expr) bool {
	call, ok := e.(*ast.CallExpr)
	if !ok {
		return false
	}
	sel, ok := call.Fun.(*ast.SelectorExpr)
	if !ok {
		return false
	}
	id, ok := sel.X.(*ast.Ident)
	return ok && id.Name == "regexp" && strings.HasPrefix(sel.Sel.Name, "MustCompile")
}

func isTransformer(fd *ast.FuncDecl, ctx string) bool {
	types := resultTypes(fd)
	if len(types) == 1 {
		if id, ok := types[0].(*ast.Ident); ok && id.Name == typBool {
			return false
		}
	}
	if len(types) == 2 && isErrorIdent(types[1]) {
		return true
	}
	fatalf("%s: validator @%s must be func(string) bool or func(string) (T, error)", ctx, fd.Name.Name)
	return false
}

// middlewareRet reports whether a middleware returns error (true) or bool.
func middlewareRet(fd *ast.FuncDecl, ctx string) bool {
	types := resultTypes(fd)
	if len(types) == 1 {
		if id, ok := types[0].(*ast.Ident); ok {
			switch id.Name {
			case typBool:
				return false
			case typError:
				return true
			}
		}
	}
	fatalf("%s: middleware @%s must return bool or error", ctx, fd.Name.Name)
	return false
}

// parserReturnsErr reports whether a whole-input parser returns (T, error)
// rather than a bare T.
func parserReturnsErr(fd *ast.FuncDecl, ctx string) bool {
	types := resultTypes(fd)
	switch len(types) {
	case 1:
		return false
	case 2:
		if isErrorIdent(types[1]) {
			return true
		}
	}
	fatalf("%s: parser @%s must return T or (T, error)", ctx, fd.Name.Name)
	return false
}

// decl finds an @ref's declaration: a method of the owning api first, then a
// package func. Nil for a regexp var or a qualified ref.
func (l *loader) decl(a *API, name string) *ast.FuncDecl {
	if fd := l.idx.Methods[a.Name][name]; fd != nil {
		return fd
	}
	return l.idx.Funcs[name]
}

// resolve turns an @ref into a Ref. A qualified name is assumed to live in an
// imported package; a bare one must be a method of a or a package func.
func (l *loader) resolveRef(a *API, name, ctx string) Ref {
	if _, _, ok := strings.Cut(name, "."); ok {
		return Ref{Name: name, Kind: RefExternal}
	}
	if fd := l.idx.Methods[a.Name][name]; fd != nil {
		return Ref{Name: name, Kind: RefMethod, Decl: fd}
	}
	if fd := l.idx.Funcs[name]; fd != nil {
		return Ref{Name: name, Kind: RefFunc, Decl: fd}
	}
	fatalf("%s: cannot resolve @%s to a package function or a method of %s", ctx, name, a.Name)
	return Ref{}
}

// classify resolves a validator ref: a *regexp.Regexp var is matched directly,
// a func either checks (bool) or transforms ((T, error)); qualified refs have
// unknown signatures, assume func(string) (T, error).
func (l *loader) classify(a *API, raw, ctx string) *Ref {
	if !strings.Contains(raw, ".") {
		if _, ok := l.idx.Regexps[raw]; ok {
			return &Ref{Name: raw, Kind: RefRegexp}
		}
	}
	ref := l.resolveRef(a, raw, ctx)
	if fd := l.decl(a, raw); fd != nil {
		ref.Transform = isTransformer(fd, ctx)
		ref.Err = ref.Transform
		return &ref
	}
	ref.Transform, ref.Err = true, true
	return &ref
}

// classifyParser resolves a whole-input parser: T or (T, error).
func (l *loader) classifyParser(a *API, raw, ctx string) *Ref {
	ref := l.resolveRef(a, raw, ctx)
	ref.Transform = true
	if fd := l.decl(a, raw); fd != nil {
		ref.Err = parserReturnsErr(fd, ctx)
		return &ref
	}
	ref.Err = true
	return &ref
}

// castWriter turns a param of a named type embedding http.ResponseWriter into
// the writer itself, type-asserted at the call site.
func (l *loader) castWriter(spec *Arg) bool {
	if spec.Kind != ArgPath || spec.Type.Node == nil || !embedsWriter(spec.Type.Node, l.idx.Types, 0) {
		return false
	}
	spec.Kind = ArgWriter
	return true
}

// resolveQH resolves one query or header arg, whole-object or single value.
func (l *loader) resolveQH(a *API, spec *Arg, ctx string) {
	//exhaustive:ignore only query and header args bind this way
	switch spec.Kind {
	case ArgHeader:
		if spec.Shape == BindParser {
			spec.Check = l.classifyParser(a, spec.checker, ctx)
			return
		}
		if spec.Whole {
			l.analyzeHeaderBind(spec, ctx)
			return
		}
		if spec.checker != "" {
			spec.Check = l.classify(a, spec.checker, ctx)
			return
		}
		if id, ok := spec.Type.Node.(*ast.Ident); !ok || id.Name != typString {
			fatalf("%s: single header value %q must be string, or add a =@transform", ctx, spec.Key)
		}
	case ArgQuery:
		if spec.Shape == BindParser {
			spec.Check = l.classifyParser(a, spec.checker, ctx)
			return
		}
		if spec.Whole {
			l.analyzeQueryBind(spec, ctx)
			return
		}
		if spec.checker != "" {
			spec.Check = l.classify(a, spec.checker, ctx)
			return
		}
		if id, ok := spec.Type.Node.(*ast.Ident); !ok || id.Name != typString {
			fatalf("%s: single query value %q must be string, or add a =@transform", ctx, spec.Key)
		}
	}
}

// analyzeQueryBind picks the whole-query binding from the declared type: a
// struct (fields keyed by `query` tag or name), a map, url.Values passed
// through, or a bare string, which is one value after all.
func (l *loader) analyzeQueryBind(spec *Arg, ctx string) {
	isIdent := func(e ast.Expr, name string) bool {
		id, ok := e.(*ast.Ident)
		return ok && id.Name == name
	}
	switch t := spec.Type.Node.(type) {
	case *ast.Ident:
		if t.Name == typString {
			return // scalar
		}
		ts := l.idx.Types[t.Name]
		if ts == nil {
			fatalf("%s: query type %s is not declared in this package", ctx, t.Name)
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			fatalf("%s: query type %s must be a struct, a map, or string", ctx, t.Name)
		}
		spec.Shape = BindStruct
		for _, fld := range st.Fields.List {
			for _, fn := range fld.Names {
				if !fn.IsExported() {
					continue
				}
				key := lowerFirst(fn.Name)
				if fld.Tag != nil {
					tag, _ := strconv.Unquote(fld.Tag.Value)
					if v := reflect.StructTag(tag).Get("query"); v != "" {
						if v == "-" {
							continue
						}
						key = v
					}
				}
				qf := Field{Name: fn.Name, Key: key}
				switch {
				case isIdent(fld.Type, typString):
					qf.Kind = FieldString
				case isIdent(fld.Type, "int"):
					qf.Kind = FieldInt
				case isIdent(fld.Type, typBool):
					qf.Kind = FieldBool
				default:
					fatalf("%s: query struct field %s.%s must be string, int or bool", ctx, t.Name, fn.Name)
				}
				spec.Fields = append(spec.Fields, qf)
			}
		}
	case *ast.MapType:
		if !isIdent(t.Key, typString) {
			fatalf("%s: query map key must be string", ctx)
		}
		switch v := t.Value.(type) {
		case *ast.Ident:
			switch v.Name {
			case typString:
				spec.Shape = BindMap
			case "any":
				spec.Shape = BindMap
				spec.MapAny = true
			default:
				fatalf("%s: query map value must be string, any or []string", ctx)
			}
		case *ast.ArrayType:
			if v.Len != nil || !isIdent(v.Elt, typString) {
				fatalf("%s: query map value must be string, any or []string", ctx)
			}
			spec.Shape = BindValues
		case *ast.InterfaceType:
			if len(v.Methods.List) != 0 {
				fatalf("%s: query map value must be string, any or []string", ctx)
			}
			spec.Shape = BindMap
			spec.MapAny = true
		default:
			fatalf("%s: query map value must be string, any or []string", ctx)
		}
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok && id.Name == "url" && t.Sel.Name == "Values" {
			spec.Shape = BindValues
			return
		}
		fatalf("%s: unsupported query type", ctx)
	default:
		fatalf("%s: unsupported query type", ctx)
	}
}

// analyzeHeaderBind picks the whole-header binding: a struct (fields keyed by
// `header` tag or name) or http.Header passed through directly.
func (l *loader) analyzeHeaderBind(spec *Arg, ctx string) {
	isIdent := func(e ast.Expr, name string) bool {
		id, ok := e.(*ast.Ident)
		return ok && id.Name == name
	}
	switch t := spec.Type.Node.(type) {
	case *ast.Ident:
		ts := l.idx.Types[t.Name]
		if ts == nil {
			fatalf("%s: headers type %s is not declared in this package", ctx, t.Name)
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			fatalf("%s: headers type %s must be a struct or http.Header", ctx, t.Name)
		}
		spec.Shape = BindStruct
		for _, fld := range st.Fields.List {
			for _, fn := range fld.Names {
				if !fn.IsExported() {
					continue
				}
				key := fn.Name
				if fld.Tag != nil {
					tag, _ := strconv.Unquote(fld.Tag.Value)
					if v := reflect.StructTag(tag).Get("header"); v != "" {
						if v == "-" {
							continue
						}
						key = v
					}
				}
				if !isIdent(fld.Type, typString) {
					fatalf("%s: header struct field %s.%s must be string", ctx, t.Name, fn.Name)
				}
				spec.Fields = append(spec.Fields, Field{Name: fn.Name, Key: key, Kind: FieldString})
			}
		}
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok && id.Name == pkgHTTP && t.Sel.Name == "Header" {
			spec.Shape = BindValues // pass r.Header through
			return
		}
		fatalf("%s: unsupported headers type", ctx)
	default:
		fatalf("%s: unsupported headers type", ctx)
	}
}

// guard resolves one //rr:pre entry.
func (l *loader) guard(a *API, raw string) *Handler {
	fd := l.decl(a, raw)
	if fd == nil {
		fatalf("%s: middleware @%s must be declared in this package", a.Name, raw)
	}
	h := &Handler{Ref: l.resolveRef(a, raw, a.Name)}
	h.RetErr = middlewareRet(fd, a.Name)
	h.Args = l.handlerArgs(l.idx.FileOf[fd], fd)
	for _, spec := range h.Args {
		l.castWriter(spec)
		//exhaustive:ignore the default rejects every other kind
		switch spec.Kind {
		case ArgWriter, ArgRequest, ArgContext, ArgQuery, ArgHeader:
		default:
			fatalf("%s: middleware @%s params must be http.ResponseWriter, *http.Request, context.Context, rr:query or rr:header", a.Name, raw)
		}
	}
	for _, spec := range h.Args {
		if spec.Kind == ArgQuery || spec.Kind == ArgHeader {
			l.resolveQH(a, spec, a.Name)
		}
	}
	return h
}

// ehandler resolves an error or condition handler. Like a route handler it
// binds params — http.ResponseWriter required, error required when wantErr
// (onerror), *http.Request/context/query/header optional.
func (l *loader) ehandler(a *API, raw, kind string, wantErr, allowErr bool) *Handler {
	ctx := a.Name + " " + kind
	fd := l.decl(a, raw)
	if fd == nil {
		fatalf("%s: %s handler @%s must be a func or method declared in this package", a.Name, kind, raw)
	}
	h := &Handler{Ref: l.resolveRef(a, raw, ctx), Args: l.handlerArgs(l.idx.FileOf[fd], fd)}
	hasW := false
	for _, spec := range h.Args {
		//exhaustive:ignore the default rejects every other kind
		switch spec.Kind {
		case ArgWriter:
			hasW = true
		case ArgPath:
			if !l.castWriter(spec) {
				fatalf("%s: %s handler @%s can only bind http.ResponseWriter, *http.Request, context.Context, error, query and headers", a.Name, kind, raw)
			}
			hasW = true
		case ArgRequest, ArgContext:
		case ArgError:
			if !allowErr {
				fatalf("%s: %s handler @%s must not take an error param", a.Name, kind, raw)
			}
			h.TakesErr = true
		case ArgQuery, ArgHeader:
			l.resolveQH(a, spec, ctx)
		default:
			fatalf("%s: %s handler @%s can only bind http.ResponseWriter, *http.Request, context.Context, error, query and headers", a.Name, kind, raw)
		}
	}
	if !hasW {
		fatalf("%s: %s handler @%s must take http.ResponseWriter", a.Name, kind, raw)
	}
	if wantErr && !h.TakesErr {
		fatalf("%s: %s handler @%s must take an error param", a.Name, kind, raw)
	}
	return h
}

// resolveRoute settles every handler parameter of one route against its
// pattern: a route token beats a reserved name, a body is classified by its
// type, query and header binds resolve, and each remaining param must name a
// token in the pattern.
func (l *loader) resolveRoute(a *API, rt *Route) {
	ctx := a.Name + "." + rt.Handler

	for p := range rt.Params() {
		if p.checker != "" {
			p.Check = l.classify(a, p.checker, ctx)
		}
	}

	// a route token of the same name overrides a by-name role
	tokens := map[string]*Param{}
	for p := range rt.Params() {
		tokens[p.Name] = p
	}
	bodySeen := false
	for _, spec := range rt.Args {
		if spec.byName && tokens[spec.Key] != nil {
			spec.Kind, spec.byName, spec.Whole = ArgPath, false, false
		}
		//exhaustive:ignore the writer, request and context kinds need no resolving
		switch spec.Kind {
		case ArgBody:
			if bodySeen {
				fatalf("%s.%s: multiple body params", a.Name, rt.Handler)
			}
			bodySeen = true
			t := spec.Type.Node
			if st, ok := t.(*ast.StarExpr); ok {
				t = st.X
			}
			// form bodies pass request fields directly, no named type
			if sel, ok := t.(*ast.SelectorExpr); ok {
				if id, ok := sel.X.(*ast.Ident); ok {
					switch {
					case id.Name == "multipart" && sel.Sel.Name == "Form":
						spec.Body = BodyMultipart
						continue
					case id.Name == "url" && sel.Sel.Name == "Values":
						spec.Body = BodyForm
						continue
					}
				}
			}
		case ArgHeader, ArgQuery:
			l.resolveQH(a, spec, ctx)
		case ArgError:
			fatalf("%s.%s: handlers receive errors via return, not a param", a.Name, rt.Handler)
		case ArgPath:
			if l.castWriter(spec) {
				break
			}
			pp := tokens[spec.Key]
			if pp == nil {
				fatalf("%s.%s: param %q is not a route token and has no role; name it body/query/headers, add {%s} to the route, or annotate /* rr:query */ or /* rr:header */",
					a.Name, rt.Handler, spec.Key, spec.Key)
			}
			if pp.Rest {
				if id, _ := spec.Type.Node.(*ast.Ident); id == nil || id.Name != typString {
					fatalf("%s.%s: catch-all {%s...} binds the raw rest of the path, its param must be a string, not %s",
						a.Name, rt.Handler, spec.Key, spec.Type.Expr)
				}
			}
			pp.Arg, spec.Param = spec, pp
			pp.Type = spec.Type
		}
	}
}
