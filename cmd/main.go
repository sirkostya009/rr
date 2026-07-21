// Command rr generates ServeHTTP routing code from //api: directives.
package main

import (
	"fmt"
	"go/ast"
	"go/format"
	"go/parser"
	"go/printer"
	"go/token"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

func fatalf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "rr: "+format+"\n", args...)
	os.Exit(1)
}

// refExpr is a resolved @ref. Method refs render relative to a receiver
// expression, so the same ref works in a standalone dispatcher ("s.handle")
// and in an app-merged one ("s.Some.handle").
type refExpr struct {
	method bool
	name   string
}

func (r refExpr) isSet() bool { return r.name != "" }

func (r refExpr) expr(recv string) string {
	if r.method {
		return recv + "." + r.name
	}
	return r.name
}

// param type classes; same-class params cannot share a trie position since
// their matchers overlap (any int parses as a float, etc.)
const (
	classCustom  = iota // checker/regex/plain: not statically comparable
	classNumeric        // int, float64, float32
	classBool
)

type param struct {
	name      string
	checker   string  // raw @ref from the pattern
	ref       refExpr // resolved checker
	extraArgs string  // appended to the call, e.g. ", 64" for strconv.ParseFloat
	transform bool    // checker returns (T, error); T becomes a handler argument
	regex     bool    // checker is a *regexp.Regexp var, matched directly
	class     int     // for auto-derived matchers
}

const (
	tokLit = iota
	tokParam
	tokWild
)

type tok struct {
	kind int
	lit  string // tokLit: one path segment, no slashes
	p    *param // tokParam; tokWild: nil for bare *, else capture name
}

const (
	argParam   = iota // a transformed path param, bound by name
	argBody           // JSON-decoded request body
	argQuery          // query string value(s)
	argHeader         // header value
	argWriter         // http.ResponseWriter, bound by type
	argRequest        // *http.Request, bound by type
)

// query binding shapes, decided by the declared param type
const (
	bindScalar = iota // one key, optionally checked/transformed
	bindValues        // url.Values or map[string][]string, passed through
	bindMap           // map[string]string or map[string]any, first values
	bindStruct        // in-package struct, fields bound by `query` tag or name
	bindParser        // @func taking the whole query/header, returning T or (T, error)
)

const (
	fString = iota
	fInt
	fBool
)

type queryField struct {
	name, key string
	kind      int
}

// argSpec describes one handler parameter, driven by its type and inline
// /* api:... */ annotations in the parameter list.
type argSpec struct {
	kind      int
	name      string  // path param / query key / header name
	checker   string  // raw @ref for query/header validators and parsers
	ref       refExpr // resolved checker
	transform bool
	regex     bool     // checker is a *regexp.Regexp var, matched directly
	typeExpr  ast.Expr // as declared in the handler signature
	typ       string   // rendered type, for body/transformed/struct binds
	ptr       bool     // body declared as *T
	fast      int      // ggen shape of the body type
	elem      string   // ggSlice: the element type name
	bind      int      // argQuery: binding shape
	mapAny    bool     // bindMap: map[string]any instead of map[string]string
	fields    []queryField
}

// handler return shapes
const (
	retNone = iota
	retErr
	retVal    // T, requires api:response
	retValErr // (T, error)
)

// ggen fast-path shapes for a body/response type
const (
	ggNone  = iota // no generated methods, use encoding/json
	ggOne          // T has DecodeFromStream/AppendJSON
	ggSlice        // []T with the methods on T
	ggAny          // arbitrary value via encode.AppendAny, when ggen is in play
)

type route struct {
	method   string
	pattern  string
	handler  string
	errRaw   string
	errRef   refExpr
	retKind  int
	retType  ast.Expr // first result, for retVal/retValErr
	response string   // "json": encode the returned value
	enc      int      // ggen shape of the response type
	owner    *apiType
	args     []argSpec
	tokens   []tok
}

type fieldCand struct {
	field, typeName string
}

type mount struct {
	field string
	api   *apiType
}

// middleware is a guard func: it takes anything a handler can (except path
// params and body) and returns bool (false = handled, stop) or error
// (non-nil goes to the error handler in scope).
type middleware struct {
	ref    refExpr
	retErr bool
	args   []argSpec
}

type apiType struct {
	name    string
	recv    string
	ptr     bool
	central bool // merges routes of api-typed fields into its dispatcher
	mounted bool // merged into some central dispatcher

	mwRaw    []string
	erRaw    string
	nfRaw    string
	naRaw    string
	brRaw    string
	response string // default response encoding for value-returning routes

	middlewares []middleware
	errHandler  refExpr
	notFound    refExpr // 404 handler
	notAllowed  refExpr // 405 handler
	badReq      refExpr // 400 handler
	brErr       bool    // 400 handler takes (w, r, err); err may be nil

	fieldCands []fieldCand
	mounts     []mount
	routes     []route
}

func main() {
	var inputs []string
	var out string
	args := os.Args[1:]
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "-o":
			i++
			if i == len(args) {
				fatalf("-o requires a value")
			}
			out = args[i]
		case "-h", "--help":
			fmt.Println("usage: rr [-o output.go] input.go...")
			return
		default:
			inputs = append(inputs, args[i])
		}
	}
	if len(inputs) == 0 {
		fatalf("no input files")
	}
	if out == "" {
		out = strings.TrimSuffix(inputs[0], ".go") + "_gen.go"
	} else if !strings.HasSuffix(out, ".go") {
		out += ".go"
	}
	outAbs, err := filepath.Abs(out)
	if err != nil {
		fatalf("%v", err)
	}

	inputAbs := map[string]bool{}
	for _, in := range inputs {
		a, err := filepath.Abs(in)
		if err != nil {
			fatalf("%v", err)
		}
		inputAbs[a] = true
	}

	// parse the whole package dir so @refs can resolve to any sibling declaration
	fset := token.NewFileSet()
	dir := filepath.Dir(inputs[0])
	entries, err := os.ReadDir(dir)
	if err != nil {
		fatalf("%v", err)
	}
	var inputFiles, pkgFiles []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		p := filepath.Join(dir, name)
		abs, _ := filepath.Abs(p)
		if abs == outAbs && !inputAbs[abs] {
			continue // stale output, about to be regenerated
		}
		f, err := parser.ParseFile(fset, p, nil, parser.ParseComments)
		if err != nil {
			fatalf("%v", err)
		}
		pkgFiles = append(pkgFiles, f)
		if inputAbs[abs] {
			inputFiles = append(inputFiles, f)
		}
	}
	if len(inputFiles) == 0 {
		fatalf("no input files found in %s", dir)
	}
	pkgName := inputFiles[0].Name.Name

	funcs := map[string]*ast.FuncDecl{}
	methods := map[string]map[string]*ast.FuncDecl{}
	types := map[string]*ast.TypeSpec{}
	regexpVars := map[string]bool{}
	fileOf := map[*ast.FuncDecl]*ast.File{}
	for _, f := range pkgFiles {
		if f.Name.Name != pkgName {
			continue
		}
		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					ts := spec.(*ast.TypeSpec)
					types[ts.Name.Name] = ts
				}
				continue
			}
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
				for _, spec := range gd.Specs {
					vs := spec.(*ast.ValueSpec)
					typed := isRegexpType(vs.Type)
					for i, nm := range vs.Names {
						if typed || (i < len(vs.Values) && isRegexpCompile(vs.Values[i])) {
							regexpVars[nm.Name] = true
						}
					}
				}
				continue
			}
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			fileOf[fd] = f
			if fd.Recv == nil {
				funcs[fd.Name.Name] = fd
				continue
			}
			t, _ := recvTypeName(fd.Recv)
			if t == "" {
				continue
			}
			if methods[t] == nil {
				methods[t] = map[string]*ast.FuncDecl{}
			}
			methods[t][fd.Name.Name] = fd
		}
	}

	importsByName := map[string]string{}
	for _, f := range pkgFiles {
		if f.Name.Name != pkgName {
			continue
		}
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			name := path.Base(p)
			// a blank import still makes the package referenceable in directives
			if imp.Name != nil && imp.Name.Name != "_" {
				name = imp.Name.Name
			}
			importsByName[name] = p
		}
	}

	apis := map[string]*apiType{}
	var order []string
	getAPI := func(name string) *apiType {
		a := apis[name]
		if a == nil {
			a = &apiType{name: name, recv: "s", ptr: true}
			apis[name] = a
			order = append(order, name)
		}
		return a
	}
	ref := func(v, ctx string) string {
		if !strings.HasPrefix(v, "@") || len(v) == 1 {
			fatalf("%s: expected @name, got %q", ctx, v)
		}
		return v[1:]
	}

	// scan the whole package for directives: go:generate usually passes a
	// single $GOFILE, while routes live wherever the package keeps them
	for _, f := range pkgFiles {
		if f.Name.Name != pkgName {
			continue
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts := spec.(*ast.TypeSpec)
					doc := ts.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					var a *apiType
					for _, dir := range parseDirectives(doc) {
						if a == nil {
							a = getAPI(ts.Name.Name)
						}
						switch dir[0] {
						case "middleware":
							a.mwRaw = append(a.mwRaw, ref(dir[1], ts.Name.Name))
						case "errorhandler":
							a.erRaw = ref(dir[1], ts.Name.Name)
						case "404":
							a.nfRaw = ref(dir[1], ts.Name.Name)
						case "405":
							a.naRaw = ref(dir[1], ts.Name.Name)
						case "400":
							a.brRaw = ref(dir[1], ts.Name.Name)
						case "central":
							a.central = true
							st, ok := ts.Type.(*ast.StructType)
							if !ok {
								fatalf("%s: api:central must be a struct", ts.Name.Name)
							}
							for _, fld := range st.Fields.List {
								t := fld.Type
								if s, ok := t.(*ast.StarExpr); ok {
									t = s.X
								}
								id, ok := t.(*ast.Ident)
								if !ok {
									continue
								}
								if len(fld.Names) == 0 { // embedded
									a.fieldCands = append(a.fieldCands, fieldCand{id.Name, id.Name})
									continue
								}
								for _, fn := range fld.Names {
									a.fieldCands = append(a.fieldCands, fieldCand{fn.Name, id.Name})
								}
							}
							// options: onerror=@f on404=@f on405=@f middleware=@f
							for _, opt := range strings.Fields(dir[1]) {
								k, v, ok := strings.Cut(opt, "=")
								v = strings.TrimPrefix(v, "@")
								if !ok || v == "" {
									fatalf("%s: bad api:central option %q, want key=@func", ts.Name.Name, opt)
								}
								switch k {
								case "onerror":
									a.erRaw = v
								case "on404":
									a.nfRaw = v
								case "on405":
									a.naRaw = v
								case "on400":
									a.brRaw = v
								case "response":
									if v != "json" {
										fatalf("%s: unsupported response %q, only json", ts.Name.Name, v)
									}
									a.response = v
								case "middleware":
									a.mwRaw = append(a.mwRaw, v)
								default:
									fatalf("%s: unknown api:central option %q", ts.Name.Name, k)
								}
							}
						default:
							fatalf("%s: unsupported directive api:%s on a type", ts.Name.Name, dir[0])
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil {
					continue
				}
				dirs := parseDirectives(d.Doc)
				if len(dirs) == 0 {
					continue
				}
				tname, ptr := recvTypeName(d.Recv)
				if tname == "" {
					fatalf("%s: cannot determine receiver type", d.Name.Name)
				}
				a := getAPI(tname)
				a.ptr = ptr
				if names := d.Recv.List[0].Names; len(names) == 1 && names[0].Name != "" && names[0].Name != "_" {
					a.recv = names[0].Name
				}
				rt := route{handler: d.Name.Name, retKind: returnKind(d), owner: a}
				rt.args = handlerArgs(f, d)
				for _, dir := range dirs {
					switch dir[0] {
					case "route":
						if rt.pattern != "" {
							fatalf("%s: multiple api:route directives", d.Name.Name)
						}
						fields := strings.Fields(dir[1])
						switch len(fields) {
						case 1:
							rt.pattern = fields[0]
						case 2:
							rt.method, rt.pattern = fields[0], fields[1]
						default:
							fatalf("%s: bad api:route %q", d.Name.Name, dir[1])
						}
					case "errorhandler":
						rt.errRaw = ref(dir[1], d.Name.Name)
					case "response":
						if dir[1] != "json" {
							fatalf("%s: unsupported api:response %q, only json", d.Name.Name, dir[1])
						}
						rt.response = dir[1]
					default:
						fatalf("%s: unsupported directive api:%s on a method", d.Name.Name, dir[0])
					}
				}
				switch rt.retKind {
				case retVal, retValErr:
					// a central's response= default may cover it; checked at emission
					rt.retType = resultTypes(d)[0]
				default:
					if rt.response != "" {
						fatalf("%s: api:response set but the handler returns no value", d.Name.Name)
					}
				}
				if rt.pattern == "" {
					fatalf("%s: api: directives without api:route", d.Name.Name)
				}
				rt.tokens = parsePattern(rt.pattern, d.Name.Name)
				a.routes = append(a.routes, rt)
			}
		}
	}
	if len(order) == 0 {
		fatalf("no api: directives found")
	}

	// link central mounts now that every api is known
	for _, name := range order {
		a := apis[name]
		for _, fc := range a.fieldCands {
			if m := apis[fc.typeName]; m != nil && m != a {
				if m.central {
					fatalf("%s: nested api:central types are not supported (field %s)", name, fc.field)
				}
				m.mounted = true
				a.mounts = append(a.mounts, mount{fc.field, m})
			}
		}
		if a.central {
			if len(a.routes)+len(a.mounts) == 0 {
				fatalf("%s: api:central has neither routes nor api-typed fields", name)
			}
		} else if len(a.routes) == 0 {
			fatalf("%s: has api: directives but no routes", name)
		}
	}

	// once ggen is in the picture (generated methods on any type, or just
	// imported), arbitrary response values ride encode.AppendAny too instead
	// of falling back to encoding/json
	ggenOn := false
	for _, mm := range methods {
		if mm["AppendJSON"] != nil {
			ggenOn = true
			break
		}
	}
	if !ggenOn {
		for _, p := range importsByName {
			if p == "github.com/sirkostya009/ggen" || strings.HasPrefix(p, "github.com/sirkostya009/ggen/") {
				ggenOn = true
				break
			}
		}
	}

	extraImports := map[string]bool{}
	numHelpers := map[string]bool{}
	resolve := func(a *apiType, name, ctx string) refExpr {
		if i := strings.IndexByte(name, '.'); i >= 0 {
			p, ok := importsByName[name[:i]]
			if !ok {
				// not imported by the input files: assume a stdlib-style path
				// (strconv.Atoi -> "strconv"); external packages need an import,
				// a blank one is enough
				p = name[:i]
			}
			extraImports[p] = true
			return refExpr{false, name}
		}
		if methods[a.name][name] != nil {
			return refExpr{true, name}
		}
		if funcs[name] != nil {
			return refExpr{false, name}
		}
		fatalf("%s: cannot resolve @%s to a package function or a method of %s", ctx, name, a.name)
		return refExpr{}
	}
	for _, name := range order {
		a := apis[name]
		for _, raw := range a.mwRaw {
			mw := middleware{ref: resolve(a, raw, name)}
			fd := methods[a.name][raw]
			if fd == nil {
				fd = funcs[raw]
			}
			if fd == nil {
				fatalf("%s: middleware @%s must be declared in this package", name, raw)
			}
			mw.retErr = middlewareRet(fd, name)
			mw.args = handlerArgs(fileOf[fd], fd)
			for _, spec := range mw.args {
				switch spec.kind {
				case argWriter, argRequest, argQuery, argHeader:
				default:
					fatalf("%s: middleware @%s params must be http.ResponseWriter, *http.Request, api:query or api:header", name, raw)
				}
			}
			a.middlewares = append(a.middlewares, mw)
		}
		if a.erRaw != "" {
			a.errHandler = resolve(a, a.erRaw, name)
		}
		if a.nfRaw != "" {
			a.notFound = resolve(a, a.nfRaw, name)
		}
		if a.naRaw != "" {
			a.notAllowed = resolve(a, a.naRaw, name)
		}
		if a.brRaw != "" {
			a.badReq = resolve(a, a.brRaw, name)
			fd := methods[a.name][a.brRaw]
			if fd == nil {
				fd = funcs[a.brRaw]
			}
			if fd != nil {
				switch countFields(fd.Type.Params) {
				case 2:
				case 3:
					a.brErr = true
				default:
					fatalf("%s: on400 handler @%s must take (w, r) or (w, r, err)", name, a.brRaw)
				}
			}
		}
		// resolve a validator ref: a *regexp.Regexp var is matched directly,
		// a func either checks (bool) or transforms ((T, error)); qualified
		// refs have unknown signatures, assume func(string) (T, error)
		classify := func(raw, ctx string) (ref refExpr, transform, regex bool) {
			if !strings.Contains(raw, ".") && regexpVars[raw] {
				return refExpr{false, raw}, false, true
			}
			resolved := resolve(a, raw, ctx)
			fd := methods[a.name][raw]
			if fd == nil {
				fd = funcs[raw]
			}
			if fd != nil {
				return resolved, isTransformer(fd, ctx), false
			}
			return resolved, true, false
		}
		// same, for whole-input parsers: T or (T, error)
		classifyParser := func(raw, ctx string) (refExpr, bool) {
			resolved := resolve(a, raw, ctx)
			fd := methods[a.name][raw]
			if fd == nil {
				fd = funcs[raw]
			}
			if fd != nil {
				return resolved, parserReturnsErr(fd, ctx)
			}
			return resolved, true
		}
		for i := range a.routes {
			rt := &a.routes[i]
			ctx := name + "." + rt.handler
			if rt.errRaw != "" {
				rt.errRef = resolve(a, rt.errRaw, ctx)
			} else {
				rt.errRef = a.errHandler
			}
			// a central's onerror may still cover the route; checked at emission
			if rt.retType != nil {
				t := rt.retType
				if st, ok := t.(*ast.StarExpr); ok {
					t = st.X
				}
				rt.enc, _ = ggShape(methods, t, "AppendJSON")
				if rt.enc == ggNone && ggenOn {
					rt.enc = ggAny
				}
			}
			for _, tk := range rt.tokens {
				if tk.kind == tokParam && tk.p.checker != "" {
					tk.p.ref, tk.p.transform, tk.p.regex = classify(tk.p.checker, ctx)
				}
			}
			bodySeen := false
			for j := range rt.args {
				spec := &rt.args[j]
				switch spec.kind {
				case argBody:
					if bodySeen {
						fatalf("%s.%s: multiple api:body params", name, rt.handler)
					}
					bodySeen = true
					t := spec.typeExpr
					if st, ok := t.(*ast.StarExpr); ok {
						spec.ptr = true
						t = st.X
					}
					spec.typ = renderType(fset, t)
					spec.fast, spec.elem = ggShape(methods, t, "DecodeFromStream")
					if spec.ptr && spec.fast == ggSlice {
						spec.fast = ggNone
					}
					if sel, ok := t.(*ast.SelectorExpr); ok {
						if id, ok := sel.X.(*ast.Ident); ok {
							p, ok := importsByName[id.Name]
							if !ok {
								p = id.Name
							}
							extraImports[p] = true
						}
					}
				case argHeader:
					if spec.bind == bindParser {
						spec.ref, spec.transform = classifyParser(spec.checker, ctx)
						break
					}
					if spec.checker != "" {
						spec.ref, spec.transform, spec.regex = classify(spec.checker, ctx)
						if spec.transform {
							spec.typ = renderType(fset, spec.typeExpr)
						}
					}
				case argQuery:
					if spec.bind == bindParser {
						spec.ref, spec.transform = classifyParser(spec.checker, ctx)
						break
					}
					if spec.checker != "" {
						spec.ref, spec.transform, spec.regex = classify(spec.checker, ctx)
						if spec.transform {
							spec.typ = renderType(fset, spec.typeExpr)
						}
						break
					}
					if analyzeQueryBind(spec, types, ctx) {
						extraImports["strconv"] = true
					}
				case argParam:
					var pp *param
					for _, tk := range rt.tokens {
						if tk.kind == tokParam && tk.p.name == spec.name {
							pp = tk.p
							break
						}
					}
					if pp == nil {
						fatalf("%s.%s: param %q matches no path param; use /* api:param <name> */, /* api:query */, /* api:header */ or /* api:body */",
							name, rt.handler, spec.name)
					}
					if pp.transform {
						break
					}
					// no explicit transformer: derive one from the declared type
					id, _ := spec.typeExpr.(*ast.Ident)
					tn := ""
					if id != nil {
						tn = id.Name
					}
					if tn != "string" && pp.checker != "" {
						fatalf("%s.%s: param {%s} already has a matcher; use {%s=@func} to transform it",
							name, rt.handler, pp.name, pp.name)
					}
					derive := func(ref string, class int) {
						pp.ref = refExpr{false, ref}
						pp.transform = true
						pp.class = class
						extraImports["strconv"] = true
					}
					switch tn {
					case "string":
						// raw segment, nothing to derive
					case "int":
						derive("strconv.Atoi", classNumeric)
					case "float64":
						derive("strconv.ParseFloat", classNumeric)
						pp.extraArgs = ", 64"
					case "float32":
						derive("parseFloat32", classNumeric)
						numHelpers["parseFloat32"] = true
					case "bool":
						// safe because bool ranks after numerics at a shared
						// position: digits are claimed before ParseBool sees them
						derive("strconv.ParseBool", classBool)
					default:
						fatalf("%s.%s: cannot bind path param %q to type %s: structs, interfaces and any are not path param material; use string, int, float64, float32, bool, or {%s=@func}",
							name, rt.handler, spec.name, renderType(fset, spec.typeExpr), spec.name)
					}
				}
			}
		}
	}

	g := &gen{}
	for _, name := range order {
		a := apis[name]
		if a.central {
			continue
		}
		// an api mounted into a central serves only through it
		if a.mounted {
			continue
		}
		g.emitDispatcher(a, a.routes, map[*apiType]string{a: a.recv})
	}
	for _, name := range order {
		a := apis[name]
		if !a.central {
			continue
		}
		merged := append([]route{}, a.routes...)
		recvOf := map[*apiType]string{a: a.recv}
		for _, m := range a.mounts {
			recvOf[m.api] = a.recv + "." + m.field
			merged = append(merged, m.api.routes...)
		}
		g.emitDispatcher(a, merged, recvOf)
	}

	var buf strings.Builder
	buf.WriteString("// Code generated by rr. DO NOT EDIT.\n\n")
	buf.WriteString("package ")
	buf.WriteString(pkgName)
	buf.WriteString("\n\n")
	imports := map[string]bool{"net/http": true, "strings": true}
	for _, name := range order {
		for _, rt := range apis[name].routes {
			if (rt.retKind == retVal || rt.retKind == retValErr) && rt.enc == ggNone {
				imports["encoding/json"] = true
			}
			for _, spec := range rt.args {
				if spec.kind == argBody && spec.fast == ggNone {
					imports["encoding/json"] = true
				}
			}
		}
	}
	if g.usesPool() {
		imports["sync"] = true
	}
	if g.useReadOne {
		imports["github.com/sirkostya009/ggen/scan"] = true
	}
	if g.useReadOne || g.useReadSlice {
		imports["github.com/sirkostya009/ggen/decode"] = true
	}
	if g.useWriteOne || g.useWriteSlice || g.useWriteAny {
		imports["github.com/sirkostya009/ggen/encode"] = true
	}
	for p := range extraImports {
		imports[p] = true
	}
	sorted := make([]string, 0, len(imports))
	for p := range imports {
		sorted = append(sorted, p)
	}
	sort.Strings(sorted)
	buf.WriteString("import (\n")
	for _, p := range sorted {
		buf.WriteString("\t")
		buf.WriteString(strconv.Quote(p))
		buf.WriteString("\n")
	}
	buf.WriteString(")\n\n")
	if g.useReadOne || g.useReadSlice {
		buf.WriteString("var readBufPool = sync.Pool{New: func() any { b := make([]byte, 0, 4096); return &b }}\n\n")
	}
	if g.useWriteOne || g.useWriteSlice {
		buf.WriteString("var writeBufPool = sync.Pool{New: func() any { b := make([]byte, 0, 4096); return &b }}\n\n")
	}
	if g.useReadOne {
		buf.WriteString(helperReadJSON)
	}
	if g.useReadSlice {
		buf.WriteString(helperReadJSONSlice)
	}
	if g.useWriteOne {
		buf.WriteString(helperWriteJSON)
	}
	if g.useWriteSlice {
		buf.WriteString(helperWriteJSONSlice)
	}
	if g.useWriteAny {
		buf.WriteString(helperWriteJSONAny)
	}
	for _, h := range []string{"parseFloat32"} {
		if numHelpers[h] {
			buf.WriteString(numHelperSrc[h])
		}
	}
	buf.WriteString(g.b.String())

	src, err := format.Source([]byte(buf.String()))
	if err != nil {
		_ = os.WriteFile(out, []byte(buf.String()), 0o644)
		fatalf("generated code has syntax errors (written unformatted to %s): %v", out, err)
	}
	if err := os.WriteFile(out, src, 0o644); err != nil {
		fatalf("%v", err)
	}
}

func parseDirectives(doc *ast.CommentGroup) [][2]string {
	if doc == nil {
		return nil
	}
	var out [][2]string
	for _, c := range doc.List {
		line, ok := strings.CutPrefix(c.Text, "//")
		if !ok {
			continue
		}
		line = strings.TrimSpace(line)
		rest, ok := strings.CutPrefix(line, "api:")
		if !ok {
			continue
		}
		kind, val, _ := strings.Cut(rest, " ")
		if i := strings.Index(val, " -- "); i >= 0 {
			val = val[:i]
		}
		if i := strings.Index(kind, " -- "); i >= 0 { // valueless directive with a comment
			kind = kind[:i]
		}
		out = append(out, [2]string{kind, strings.TrimSpace(val)})
	}
	return out
}

func parsePattern(pattern, handler string) []tok {
	if !strings.HasPrefix(pattern, "/") {
		fatalf("%s: route pattern %q must start with /", handler, pattern)
	}
	pieces := strings.Split(pattern[1:], "/")
	var toks []tok
	for i, piece := range pieces {
		last := i == len(pieces)-1
		switch {
		case piece == "*":
			if !last {
				fatalf("%s: * is only allowed at the end of a pattern", handler)
			}
			toks = append(toks, tok{kind: tokWild})
		case strings.HasPrefix(piece, "{"):
			if !strings.HasSuffix(piece, "}") {
				fatalf("%s: malformed param %q", handler, piece)
			}
			inner := piece[1 : len(piece)-1]
			if strings.ContainsRune(inner, ':') {
				fatalf("%s: {name:regex} params are gone; wrap the regex in a func and use {%s=@func}",
					handler, inner[:strings.IndexByte(inner, ':')])
			}
			name, checker, _ := strings.Cut(inner, "=")
			if wild := strings.TrimSuffix(name, "..."); wild != name {
				if !last {
					fatalf("%s: {%s} is only allowed at the end of a pattern", handler, name)
				}
				if checker != "" {
					fatalf("%s: wildcard {%s} cannot have a matcher", handler, name)
				}
				if wild == "" {
					fatalf("%s: empty wildcard name", handler)
				}
				toks = append(toks, tok{kind: tokWild, p: &param{name: wild}})
				break
			}
			if name == "" || strings.ContainsAny(name, "{}*") {
				fatalf("%s: bad param name %q", handler, name)
			}
			if checker != "" {
				if !strings.HasPrefix(checker, "@") || len(checker) == 1 {
					fatalf("%s: param checker must be @name, got %q", handler, checker)
				}
				checker = checker[1:]
			}
			toks = append(toks, tok{kind: tokParam, p: &param{name: name, checker: checker}})
		case strings.ContainsAny(piece, "{}*"):
			fatalf("%s: params must span a whole path segment, got %q", handler, piece)
		case piece == "" && !last:
			fatalf("%s: empty path segment in %q", handler, pattern)
		default:
			toks = append(toks, tok{kind: tokLit, lit: piece})
		}
	}
	return toks
}

func recvTypeName(recv *ast.FieldList) (string, bool) {
	if len(recv.List) != 1 {
		return "", false
	}
	t := recv.List[0].Type
	ptr := false
	if st, ok := t.(*ast.StarExpr); ok {
		ptr = true
		t = st.X
	}
	if idx, ok := t.(*ast.IndexExpr); ok {
		t = idx.X
	}
	id, ok := t.(*ast.Ident)
	if !ok {
		return "", false
	}
	return id.Name, ptr
}

func returnKind(d *ast.FuncDecl) int {
	types := resultTypes(d)
	switch len(types) {
	case 0:
		return retNone
	case 1:
		if isErrorIdent(types[0]) {
			return retErr
		}
		return retVal
	case 2:
		if isErrorIdent(types[1]) {
			return retValErr
		}
	}
	fatalf("%s: handler must return nothing, error, T, or (T, error)", d.Name.Name)
	return retNone
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// trie over path segments; shared literal prefixes are matched once
type tnode struct {
	routes []route // routes terminating exactly here
	lits   []litEdge
	params []*paramEdge
	wild   *wildEdge
}

type litEdge struct {
	seg   string
	child *tnode
}

type paramEdge struct {
	p     *param
	owner *apiType // for method checkers, whose receiver to call
	child *tnode
}

type wildEdge struct {
	name   string // capture name, "" for bare *
	routes []route
}

func (n *tnode) hasDesc() bool {
	return len(n.lits) > 0 || len(n.params) > 0 || n.wild != nil
}

func insert(root *tnode, rt route, skipSegs int) {
	n := root
	for _, tk := range rt.tokens[skipSegs:] {
		switch tk.kind {
		case tokLit:
			var child *tnode
			for i := range n.lits {
				if n.lits[i].seg == tk.lit {
					child = n.lits[i].child
					break
				}
			}
			if child == nil {
				child = &tnode{}
				n.lits = append(n.lits, litEdge{tk.lit, child})
			}
			n = child
		case tokParam:
			var child *tnode
			for _, pe := range n.params {
				if pe.p.name == tk.p.name && pe.p.ref == tk.p.ref && pe.p.extraArgs == tk.p.extraArgs &&
					(!tk.p.ref.method || pe.owner == rt.owner) {
					child = pe.child
					break
				}
			}
			if child == nil {
				child = &tnode{}
				n.params = append(n.params, &paramEdge{tk.p, rt.owner, child})
			}
			n = child
		case tokWild:
			name := ""
			if tk.p != nil {
				name = tk.p.name
			}
			if n.wild == nil {
				n.wild = &wildEdge{name: name}
			} else if n.wild.name != name {
				fatalf("%s: conflicting wildcard names %q and %q at the same position", rt.owner.name, n.wild.name, name)
			}
			n.wild.routes = append(n.wild.routes, rt)
			return
		}
	}
	n.routes = append(n.routes, rt)
}

type binding struct {
	name, expr string
	arg        bool // pass as a handler argument instead of SetPathValue
}

func withBinding(binds []binding, b binding) []binding {
	out := make([]binding, len(binds)+1)
	copy(out, binds)
	out[len(binds)] = b
	return out
}

type gen struct {
	b        strings.Builder
	depth    int
	n        int
	scope    *apiType            // the dispatcher being generated (api or central)
	curOwner *apiType            // owner of the route currently being emitted
	recvOf   map[*apiType]string // receiver expression per mounted api
	hoisted  string              // method already verified by a subtree-level check

	// ggen fast-path helpers used anywhere in the output
	useReadOne    bool
	useReadSlice  bool
	useWriteOne   bool
	useWriteSlice bool
	useWriteAny   bool
}

func (g *gen) usesPool() bool {
	return g.useReadOne || g.useReadSlice || g.useWriteOne || g.useWriteSlice || g.useWriteAny
}

func (g *gen) w(format string, args ...any) {
	g.b.WriteString(strings.Repeat("\t", g.depth))
	fmt.Fprintf(&g.b, format, args...)
	g.b.WriteByte('\n')
}

func (g *gen) open(format string, args ...any) {
	g.w(format, args...)
	g.depth++
}

func (g *gen) close() {
	g.depth--
	g.w("}")
}

func (g *gen) newVar() string {
	g.n++
	return fmt.Sprintf("p%d", g.n)
}

func (g *gen) cond(p *param, recv, v string) string {
	switch {
	case p.regex:
		return p.ref.expr(recv) + ".MatchString(" + v + ")"
	case p.ref.isSet():
		return p.ref.expr(recv) + "(" + v + ")"
	default:
		return v + ` != ""`
	}
}

// emitDispatcher generates the ServeHTTP (and middleware plumbing) for one
// dispatcher: a standalone api, or an app with the routes of all its mounted
// apis merged into a single match tree.
func (g *gen) emitDispatcher(scope *apiType, routes []route, recvOf map[*apiType]string) {
	g.n = 0
	g.scope = scope
	g.recvOf = recvOf
	g.hoisted = ""
	star := ""
	if scope.ptr {
		star = "*"
	}
	recv := scope.recv
	prefix := commonPrefix(routes)

	g.open("func (%s %s%s) ServeHTTP(w http.ResponseWriter, r *http.Request) {", recv, star, scope.name)
	g.curOwner = nil
	for _, mw := range scope.middlewares {
		g.emitGuard(mw, recv, nil)
	}
	g.w("path, ok := strings.CutPrefix(r.URL.Path, %q)", prefix)
	g.open("if !ok {")
	g.notFound()
	g.w("return")
	g.close()

	staticByPath := map[string][]route{}
	var keys []string
	var dynamic []route
	for _, rt := range routes {
		if isDynamic(rt) {
			dynamic = append(dynamic, rt)
			continue
		}
		key := rt.pattern[len(prefix):]
		if _, ok := staticByPath[key]; !ok {
			keys = append(keys, key)
		}
		staticByPath[key] = append(staticByPath[key], rt)
	}

	if len(keys) > 0 {
		sort.Strings(keys)
		g.w("switch path {")
		for _, k := range keys {
			g.w("case %q:", k)
			g.depth++
			g.emitDispatch(staticByPath[k], nil, nil)
			g.depth--
		}
		g.w("}")
	}

	exhaustive := false
	if len(dynamic) > 0 {
		root := &tnode{}
		skipSegs := strings.Count(prefix, "/") - 1
		for _, rt := range dynamic {
			insert(root, rt, skipSegs)
		}
		exhaustive = g.emitNode(root, "path", nil)
	}
	if !exhaustive {
		g.notFound()
	}
	g.close()
	g.w("")
}

func (g *gen) notFound() {
	if g.scope.notFound.isSet() {
		g.w("%s(w, r)", g.scope.notFound.expr(g.recvOf[g.scope]))
	} else {
		g.w("w.WriteHeader(http.StatusNotFound)")
	}
}

// notAllowed sets the Allow header (also for a custom handler) and responds
// 405, preferring the owning api's handler when the leaf belongs to one api.
func (g *gen) notAllowed(routes []route, allow string) {
	target := g.scope
	if len(routes) > 0 {
		owner := routes[0].owner
		uniform := true
		for _, rt := range routes[1:] {
			if rt.owner != owner {
				uniform = false
				break
			}
		}
		if uniform && owner.notAllowed.isSet() {
			target = owner
		}
	}
	g.w(`w.Header().Set("Allow", %q)`, allow)
	if target.notAllowed.isSet() {
		g.w("%s(w, r)", target.notAllowed.expr(g.recvOf[target]))
	} else {
		g.w("w.WriteHeader(http.StatusMethodNotAllowed)")
	}
}

// openParamCheck opens the if-block validating v against p and returns the
// binding the leaf should record. used=false discards a transformed value no
// downstream handler takes, avoiding an unused variable.
func (g *gen) openParamCheck(pe *paramEdge, v string, used bool) binding {
	p := pe.p
	if p.transform {
		val := "_"
		if used {
			val = g.newVar()
		}
		g.open("if %s, err := %s(%s%s); err == nil {", val, p.ref.expr(g.recvOf[pe.owner]), v, p.extraArgs)
		return binding{p.name, val, true}
	}
	g.open("if %s {", g.cond(p, g.recvOf[pe.owner], v))
	return binding{p.name, v, false}
}

// subtreeUsesPath reports whether any route below n takes the transformed
// path param as a handler argument.
func subtreeUsesPath(n *tnode, name string) bool {
	uses := func(rts []route) bool {
		for _, rt := range rts {
			for _, spec := range rt.args {
				if spec.kind == argParam && spec.name == name {
					return true
				}
			}
		}
		return false
	}
	if uses(n.routes) {
		return true
	}
	for _, e := range n.lits {
		if subtreeUsesPath(e.child, name) {
			return true
		}
	}
	for _, pe := range n.params {
		if subtreeUsesPath(pe.child, name) {
			return true
		}
	}
	return n.wild != nil && uses(n.wild.routes)
}

// emitNode generates matching code for one trie node. Reports whether it
// ends in an unconditional dispatch (a wildcard), making later code dead.
//
// A node does at most one IndexByte to split the rest of the path into
// "final segment" vs "more segments follow"; sibling literals in each half
// dispatch through a switch on that segment.
func (g *gen) emitNode(n *tnode, cur string, binds []binding) bool {
	// when every route below shares one method, verify it once at the top of
	// the branch instead of per leaf (method mismatch then wins over a
	// deeper structural mismatch: 405, not 404)
	if g.hoistMethod(n) {
		defer func() { g.hoisted = "" }()
	}

	// lone literal chain: a single comparison, no segment scan
	if len(n.lits) == 1 && len(n.params) == 0 && n.wild == nil {
		seg, child := compress(n.lits[0])
		term := len(child.routes) > 0
		switch {
		case term && !child.hasDesc():
			g.open("if %s == %q {", cur, seg)
			g.emitLeaf(child.routes, binds)
			g.close()
		case !term:
			v := g.newVar()
			g.open("if %s, ok := strings.CutPrefix(%s, %q); ok {", v, cur, seg+"/")
			g.emitNode(child, v, binds)
			g.close()
		default:
			v := g.newVar()
			g.open("if %s, ok := strings.CutPrefix(%s, %q); ok {", v, cur, seg)
			g.open(`if %s == "" {`, v)
			g.emitLeaf(child.routes, binds)
			g.close()
			v2 := g.newVar()
			g.open(`if %s, ok := strings.CutPrefix(%s, "/"); ok {`, v2, v)
			g.emitNode(child, v2, binds)
			g.close()
			g.close()
		}
		return false
	}

	// constrained params are more specific: try them before catch-any ones
	// (auto-derived transformers like Atoi count as constraints too), and
	// bools after numerics so ParseBool's lax "1"/"0" never claim a digit
	constrained := func(p *param) bool { return p.checker != "" || p.ref.isSet() }
	rank := func(p *param) int {
		switch {
		case !constrained(p):
			return 3
		case p.class == classBool:
			return 2
		default:
			return 1
		}
	}
	sort.SliceStable(n.params, func(i, j int) bool {
		return rank(n.params[i].p) < rank(n.params[j].p)
	})

	// several params may share a position when their matchers differ: they
	// are tried in declaration order, catch-any last. Reject the provably
	// ambiguous shapes.
	for i, pe := range n.params {
		for _, pe2 := range n.params[i+1:] {
			// edges only compete when both can claim the same branch: a
			// final-segment param and a more-segments-follow param coexist
			terms := len(pe.child.routes) > 0 && len(pe2.child.routes) > 0
			descs := pe.child.hasDesc() && pe2.child.hasDesc()
			if !terms && !descs {
				continue
			}
			p1, p2 := pe.p, pe2.p
			why := ""
			switch {
			case p1.ref == p2.ref && (!p1.ref.method || pe.owner == pe2.owner):
				if p1.ref.isSet() {
					why = "a matcher; the second can never match"
				} else {
					why = "the whole segment space; both match anything"
				}
			case p1.class == classNumeric && p2.class == classNumeric:
				why = "the numeric segment space; every int is also a valid float"
			case p1.class == classBool && p2.class == classBool:
				why = "the bool segment space"
			}
			if why != "" {
				fatalf("params {%s} (%s) and {%s} (%s) share a position and %s; give them non-overlapping types or matchers",
					p1.name, pe.owner.name, p2.name, pe2.owner.name, why)
			}
		}
	}

	var termLits, descLits []litEdge
	for _, e := range n.lits {
		if len(e.child.routes) > 0 {
			termLits = append(termLits, e)
		}
		if e.child.hasDesc() {
			descLits = append(descLits, e)
		}
	}
	var termParams, descParams []*paramEdge
	for _, pe := range n.params {
		if len(pe.child.routes) > 0 {
			termParams = append(termParams, pe)
		}
		if pe.child.hasDesc() {
			descParams = append(descParams, pe)
		}
	}

	emitTerm := func() {
		switch len(termLits) {
		case 0:
		case 1:
			g.open("if %s == %q {", cur, termLits[0].seg)
			g.emitLeaf(termLits[0].child.routes, binds)
			g.close()
		default:
			g.w("switch %s {", cur)
			for _, e := range termLits {
				g.w("case %q:", e.seg)
				g.depth++
				g.emitLeaf(e.child.routes, binds)
				g.depth--
			}
			g.w("}")
		}
		for _, pe := range termParams {
			b := g.openParamCheck(pe, cur, subtreeUsesPath(pe.child, pe.p.name))
			g.emitLeaf(pe.child.routes, withBinding(binds, b))
			g.close()
		}
	}
	emitDesc := func() {
		descend := func(e litEdge) {
			// gate the method before even slicing the rest of the path
			hoisted := g.hoistMethod(e.child)
			v := g.newVar()
			g.w("%s := %s[i+1:]", v, cur)
			g.emitNode(e.child, v, binds)
			if hoisted {
				g.hoisted = ""
			}
		}
		switch len(descLits) {
		case 0:
		case 1:
			g.open("if %s[:i] == %q {", cur, descLits[0].seg)
			descend(descLits[0])
			g.close()
		default:
			g.w("switch %s[:i] {", cur)
			for _, e := range descLits {
				g.w("case %q:", e.seg)
				g.depth++
				descend(e)
				g.depth--
			}
			g.w("}")
		}
		if len(descParams) > 0 {
			seg := g.newVar()
			g.w("%s := %s[:i]", seg, cur)
			for _, pe := range descParams {
				b := binding{pe.p.name, seg, false}
				wrapped := constrained(pe.p) // i > 0 already guarantees a non-empty segment
				if wrapped {
					b = g.openParamCheck(pe, seg, subtreeUsesPath(pe.child, pe.p.name))
				}
				rest := g.newVar()
				g.w("%s := %s[i+1:]", rest, cur)
				g.emitNode(pe.child, rest, withBinding(binds, b))
				if wrapped {
					g.close()
				}
			}
		}
	}

	hasTerm := len(termLits)+len(termParams) > 0
	hasDesc := len(descLits)+len(descParams) > 0
	switch {
	case hasTerm && hasDesc:
		g.open("if i := strings.IndexByte(%s, '/'); i < 0 {", cur)
		emitTerm()
		g.depth--
		g.w("} else if i > 0 {")
		g.depth++
		emitDesc()
		g.close()
	case hasTerm:
		g.open("if strings.IndexByte(%s, '/') < 0 {", cur)
		emitTerm()
		g.close()
	case hasDesc:
		g.open("if i := strings.IndexByte(%s, '/'); i > 0 {", cur)
		emitDesc()
		g.close()
	}

	if n.wild != nil {
		b := binds
		if n.wild.name != "" {
			b = withBinding(binds, binding{n.wild.name, cur, false})
		}
		g.emitLeaf(n.wild.routes, b)
		return true
	}
	return false
}

// compress folds chains of single-literal-child nodes into one multi-segment
// literal so the prefix is matched with a single comparison.
func compress(e litEdge) (string, *tnode) {
	seg, child := e.seg, e.child
	for len(child.routes) == 0 && child.wild == nil && len(child.params) == 0 && len(child.lits) == 1 {
		seg += "/" + child.lits[0].seg
		child = child.lits[0].child
	}
	return seg, child
}

// emitLeaf dispatches by method; bindings travel along so the selected
// route can bind them as args or path values.
func (g *gen) emitLeaf(routes []route, binds []binding) {
	args := map[string]string{}
	for _, b := range binds {
		args[b.name] = b.expr
	}
	g.emitDispatch(routes, args, binds)
}

func (g *gen) emitDispatch(routes []route, args map[string]string, binds []binding) {
	var catch *route
	var methods []route
	seen := map[string]*apiType{}
	for i := range routes {
		rt := &routes[i]
		if rt.method == "" {
			if catch != nil {
				fatalf("conflicting catch-all routes for %q (%s and %s)", rt.pattern, catch.owner.name, rt.owner.name)
			}
			catch = rt
			continue
		}
		if prev := seen[rt.method]; prev != nil {
			fatalf("conflicting routes %s %q (%s and %s)", rt.method, rt.pattern, prev.name, rt.owner.name)
		}
		seen[rt.method] = rt.owner
		methods = append(methods, *rt)
	}
	switch {
	case catch != nil && len(methods) == 0:
		g.emitCall(*catch, args, binds)
	case catch != nil && len(methods) == 1:
		g.open("if r.Method == %q {", methods[0].method)
		g.emitCall(methods[0], args, binds)
		g.depth--
		g.w("} else {")
		g.depth++
		g.emitCall(*catch, args, binds)
		g.close()
	case catch == nil && len(methods) == 1:
		if methods[0].method != g.hoisted {
			g.open("if r.Method != %q {", methods[0].method)
			g.notAllowed(routes, methods[0].method)
			g.w("return")
			g.close()
		}
		g.emitCall(methods[0], args, binds)
	default:
		g.w("switch r.Method {")
		for _, m := range methods {
			g.w("case %q:", m.method)
			g.depth++
			g.emitCall(m, args, binds)
			g.depth--
		}
		g.w("default:")
		g.depth++
		if catch != nil {
			g.emitCall(*catch, args, binds)
		} else {
			allow := make([]string, len(methods))
			for i, m := range methods {
				allow[i] = m.method
			}
			g.notAllowed(routes, strings.Join(allow, ", "))
		}
		g.depth--
		g.w("}")
	}
	g.w("return")
}

// emitCall invokes the handler. In an app dispatcher, a route whose owning
// api declares middleware gets it wrapped around just this call.
func (g *gen) emitCall(rt route, args map[string]string, binds []binding) {
	g.w("r.Pattern = %q", displayPattern(rt))
	// every string param lands in PathValue: it is the request's public match
	// metadata, argument binding or not; transformed values have no raw string
	for _, b := range binds {
		if b.arg {
			continue
		}
		g.w("r.SetPathValue(%q, %s)", b.name, b.expr)
	}
	owner := rt.owner
	g.curOwner = owner
	recv := g.recvOf[owner]
	if owner != g.scope {
		for _, mw := range owner.middlewares {
			g.emitGuard(mw, recv, &rt)
		}
	}
	g.emitCallBare(rt, args, recv)
}

// subtreeRoutes collects every route reachable below n, wildcards included.
func subtreeRoutes(n *tnode, out []route) []route {
	out = append(out, n.routes...)
	if n.wild != nil {
		out = append(out, n.wild.routes...)
	}
	for _, e := range n.lits {
		out = subtreeRoutes(e.child, out)
	}
	for _, pe := range n.params {
		out = subtreeRoutes(pe.child, out)
	}
	return out
}

// hoistMethod emits one method gate for a method-uniform subtree; leaves
// under it skip their own checks. Caller clears g.hoisted afterwards.
func (g *gen) hoistMethod(n *tnode) bool {
	if g.hoisted != "" {
		return false
	}
	rts := subtreeRoutes(n, nil)
	if len(rts) == 0 || rts[0].method == "" {
		return false
	}
	m := rts[0].method
	for _, rt := range rts[1:] {
		if rt.method != m {
			return false
		}
	}
	g.open("if r.Method != %q {", m)
	g.notAllowed(rts, m)
	g.w("return")
	g.close()
	g.hoisted = m
	return true
}

// emitGuard emits one middleware call: a false bool return means the guard
// wrote the response, an error goes to the error handler in scope.
func (g *gen) emitGuard(mw middleware, recv string, rt *route) {
	call := fmt.Sprintf("%s(%s)", mw.ref.expr(recv), g.buildArgs(mw.args, nil, recv))
	if mw.retErr {
		var eh string
		if rt != nil {
			eh = g.errExpr(*rt)
		} else if g.scope.errHandler.isSet() {
			eh = g.scope.errHandler.expr(g.recvOf[g.scope])
		} else {
			fatalf("%s: middleware returns error but there is no onerror in scope", g.scope.name)
		}
		g.open("if err := %s; err != nil {", call)
		g.w("%s(w, r, err)", eh)
		g.w("return")
		g.close()
	} else {
		g.open("if !%s {", call)
		g.w("return")
		g.close()
	}
}

// buildArgs emits the extraction preamble for the given arg specs and
// returns the call argument list.
func (g *gen) buildArgs(specs []argSpec, args map[string]string, recv string) string {
	qv := ""
	query := func() string {
		if qv == "" {
			qv = g.newVar()
			g.w("%s := r.URL.Query()", qv)
		}
		return qv
	}
	var parts []string
	for _, spec := range specs {
		switch spec.kind {
		case argWriter:
			parts = append(parts, "w")
		case argRequest:
			parts = append(parts, "r")
		case argParam:
			parts = append(parts, args[spec.name])
		case argBody:
			parts = append(parts, g.emitBodyDecode(spec))
		case argQuery:
			switch spec.bind {
			case bindParser:
				parts = append(parts, g.emitParser(spec, query(), recv))
			case bindScalar:
				parts = append(parts, g.emitExtract(spec, fmt.Sprintf("%s.Get(%q)", query(), spec.name), recv))
			default:
				parts = append(parts, g.emitQueryBind(spec, query()))
			}
		case argHeader:
			if spec.bind == bindParser {
				parts = append(parts, g.emitParser(spec, "r.Header", recv))
			} else {
				parts = append(parts, g.emitExtract(spec, fmt.Sprintf("r.Header.Get(%q)", spec.name), recv))
			}
		}
	}
	return strings.Join(parts, ", ")
}

func (g *gen) emitCallBare(rt route, args map[string]string, recv string) {
	call := fmt.Sprintf("%s.%s(%s)", recv, rt.handler, g.buildArgs(rt.args, args, recv))
	if rt.retKind == retVal || rt.retKind == retValErr {
		if rt.response == "" && g.scope.response == "" {
			fatalf("%s.%s returns a value; add //api:response json or response=json on the central", rt.owner.name, rt.handler)
		}
	}
	switch rt.retKind {
	case retNone:
		g.w("%s", call)
	case retErr:
		g.open("if err := %s; err != nil {", call)
		g.w("%s(w, r, err)", g.errExpr(rt))
		g.close()
	case retVal:
		if rt.enc != ggNone {
			g.emitFastEncode(rt, call)
			return
		}
		g.w(`w.Header().Set("Content-Type", "application/json")`)
		g.w("_ = json.NewEncoder(w).Encode(%s)", call)
	case retValErr:
		v := g.newVar()
		g.w("%s, err := %s", v, call)
		g.open("if err != nil {")
		g.w("%s(w, r, err)", g.errExpr(rt))
		g.w("return")
		g.close()
		if rt.enc != ggNone {
			g.emitFastEncode(rt, v)
			return
		}
		g.w(`w.Header().Set("Content-Type", "application/json")`)
		g.w("_ = json.NewEncoder(w).Encode(%s)", v)
	}
}

// emitFastEncode writes the value through the pregenerated pooled-buffer
// helper; only encode errors surface, they arrive before anything hit the
// wire. An encode failure is an infrastructure error, not a handler error:
// route-level errorhandler overrides (which map domain errors to statuses)
// are deliberately skipped in favor of the api/central onerror.
func (g *gen) emitFastEncode(rt route, v string) {
	fn := "writeJSON"
	switch rt.enc {
	case ggSlice:
		g.useWriteSlice = true
		fn = "writeJSONSlice"
	case ggAny:
		g.useWriteAny = true
		fn = "writeJSONAny"
	default:
		g.useWriteOne = true
	}
	g.open("if err := %s(w, %s); err != nil {", fn, v)
	switch {
	case rt.errRaw == "" && rt.errRef.isSet(): // the owning api's onerror
		g.w("%s(w, r, err)", rt.errRef.expr(g.recvOf[rt.owner]))
	case g.scope.errHandler.isSet():
		g.w("%s(w, r, err)", g.scope.errHandler.expr(g.recvOf[g.scope]))
	default:
		g.w("w.WriteHeader(http.StatusInternalServerError)")
	}
	g.w("return")
	g.close()
}

// errExpr picks the error handler: route/owner level first, then the
// dispatcher's own (a central's onerror covers mounted apis).
func (g *gen) errExpr(rt route) string {
	if rt.errRef.isSet() {
		return rt.errRef.expr(g.recvOf[rt.owner])
	}
	if g.scope.errHandler.isSet() {
		return g.scope.errHandler.expr(g.recvOf[g.scope])
	}
	fatalf("%s.%s returns error but no error handler in scope of %s", rt.owner.name, rt.handler, g.scope.name)
	return ""
}

// emitParser calls a whole-input parser; a bare-T parser inlines into the
// call, a (T, error) one gets a temp and a 400 check.
func (g *gen) emitParser(spec argSpec, input, recv string) string {
	fn := spec.ref.expr(recv)
	if !spec.transform {
		return fn + "(" + input + ")"
	}
	v := spec.name
	if !safeVarName(v) {
		v = g.newVar()
	}
	g.w("%s, err := %s(%s)", v, fn, input)
	g.open("if err != nil {")
	g.badRequest(true)
	g.close()
	return v
}

// emitQueryBind materializes a whole-query binding (values, map or struct)
// and returns the variable to pass to the handler.
func (g *gen) emitQueryBind(spec argSpec, q string) string {
	switch spec.bind {
	case bindValues:
		return q
	case bindMap:
		vt := "string"
		if spec.mapAny {
			vt = "any"
		}
		v := g.newVar()
		g.w("%s := make(map[string]%s, len(%s))", v, vt, q)
		g.open("for k, vs := range %s {", q)
		g.w("%s[k] = vs[0]", v)
		g.close()
		return v
	default: // bindStruct
		v := spec.name
		if !safeVarName(v) {
			v = g.newVar()
		}
		g.w("var %s %s", v, spec.typ)
		for _, fd := range spec.fields {
			switch fd.kind {
			case fString:
				g.w("%s.%s = %s.Get(%q)", v, fd.name, q, fd.key)
			default:
				fn := "strconv.Atoi"
				if fd.kind == fBool {
					fn = "strconv.ParseBool"
				}
				rv := g.newVar()
				g.open(`if %s := %s.Get(%q); %s != "" {`, rv, q, fd.key, rv)
				g.w("var err error")
				g.open("if %s.%s, err = %s(%s); err != nil {", v, fd.name, fn, rv)
				g.badRequest(true)
				g.close()
				g.close()
			}
		}
		return v
	}
}

// badRequest responds 400 through the owning api's handler, the dispatcher's,
// or a bare status write; no body by default. hasErr says whether an err
// variable is in scope at the call site — a 3-arg handler gets it, or nil
// when the request is just bad without a specific error.
func (g *gen) badRequest(hasErr bool) {
	target := g.scope
	if g.curOwner != nil && g.curOwner.badReq.isSet() {
		target = g.curOwner
	}
	switch {
	case !target.badReq.isSet():
		g.w("w.WriteHeader(http.StatusBadRequest)")
	case !target.brErr:
		g.w("%s(w, r)", target.badReq.expr(g.recvOf[target]))
	case hasErr:
		g.w("%s(w, r, err)", target.badReq.expr(g.recvOf[target]))
	default:
		g.w("%s(w, r, nil)", target.badReq.expr(g.recvOf[target]))
	}
	g.w("return")
}

// emitBodyDecode declares the body value, decodes JSON into it (a failure is
// a 400), and returns the expression to pass. Types with ggen-generated
// methods stream-decode through a pooled buffer; the stream path copies
// strings out, so the buffer recycles immediately.
func (g *gen) emitBodyDecode(spec argSpec) string {
	v := spec.name
	if !safeVarName(v) {
		v = g.newVar()
	}
	switch spec.fast {
	case ggOne:
		g.useReadOne = true
		g.w("%s, err := readJSON[%s](r)", v, spec.typ)
		g.open("if err != nil {")
		g.badRequest(true)
		g.close()
		if spec.ptr {
			return "&" + v
		}
		return v
	case ggSlice:
		g.useReadSlice = true
		g.w("%s, err := readJSONSlice[%s](r)", v, spec.elem)
		g.open("if err != nil {")
		g.badRequest(true)
		g.close()
		return v
	}
	if spec.ptr {
		g.w("%s := new(%s)", v, spec.typ)
		g.open("if err := json.NewDecoder(r.Body).Decode(%s); err != nil {", v)
	} else {
		g.w("var %s %s", v, spec.typ)
		g.open("if err := json.NewDecoder(r.Body).Decode(&%s); err != nil {", v)
	}
	g.badRequest(true)
	g.close()
	return v
}

// emitExtract turns a raw query/header expression into the handler argument:
// passed through as-is, validated (false is a 400), or transformed (a missing
// value stays the zero value, an invalid one is a 400).
func (g *gen) emitExtract(spec argSpec, raw, recv string) string {
	if spec.checker == "" {
		return raw
	}
	fn := spec.ref.expr(recv)
	if !spec.transform {
		if spec.regex {
			fn += ".MatchString"
		}
		v := g.newVar()
		g.w("%s := %s", v, raw)
		g.open("if !%s(%s) {", fn, v)
		g.badRequest(false)
		g.close()
		return v
	}
	v := g.newVar()
	g.w("var %s %s", v, spec.typ)
	rv := g.newVar()
	g.open("if %s := %s; %s != \"\" {", rv, raw, rv)
	g.w("var err error")
	g.open("if %s, err = %s(%s); err != nil {", v, fn, rv)
	g.badRequest(true)
	g.close()
	g.close()
	return v
}

// safeVarName reports whether the handler's own param name can be reused in
// generated code without clashing with router locals.
func safeVarName(v string) bool {
	switch v {
	case "", "_", "w", "r", "path", "err", "ok", "i":
		return false
	}
	if v[0] == 'p' && len(v) > 1 && strings.TrimLeft(v[1:], "0123456789") == "" {
		return false
	}
	return true
}

func resultTypes(fd *ast.FuncDecl) []ast.Expr {
	var types []ast.Expr
	if fd.Type.Results != nil {
		for _, f := range fd.Type.Results.List {
			n := len(f.Names)
			if n == 0 {
				n = 1
			}
			for j := 0; j < n; j++ {
				types = append(types, f.Type)
			}
		}
	}
	return types
}

func isErrorIdent(e ast.Expr) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == "error"
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
		if id, ok := types[0].(*ast.Ident); ok && id.Name == "bool" {
			return false
		}
	}
	if len(types) == 2 && isErrorIdent(types[1]) {
		return true
	}
	fatalf("%s: validator @%s must be func(string) bool or func(string) (T, error)", ctx, fd.Name.Name)
	return false
}

// pooled-buffer JSON helpers, emitted once per output when a fast path is used
const helperReadJSON = `// readJSON stream-decodes a request body through a pooled buffer. The
// stream path copies strings out, so the buffer recycles immediately.
func readJSON[T decode.Decoder[T]](r *http.Request) (T, error) {
	bp := readBufPool.Get().(*[]byte)
	defer readBufPool.Put(bp)
	var s scan.Stream
	s.Reset(r.Body, *bp)
	var zero T
	v, err := zero.DecodeFromStream(&s)
	*bp = s.Bytes()
	return v, err
}

`

const helperReadJSONSlice = `// readJSONSlice stream-decodes a JSON array body through a pooled buffer.
func readJSONSlice[T decode.Decoder[T]](r *http.Request) ([]T, error) {
	bp := readBufPool.Get().(*[]byte)
	defer readBufPool.Put(bp)
	vs, nb, err := decode.UnmarshalSliceStream[T](r.Body, (*bp)[:0])
	*bp = nb
	return vs, err
}

`

const helperWriteJSON = `// writeJSON appends the value's JSON into a pooled buffer and pipes it to
// the ResponseWriter in one Write. Only encode errors are reported; they
// arrive before anything hits the wire.
func writeJSON[T encode.Marshaler](w http.ResponseWriter, v T) error {
	bp := writeBufPool.Get().(*[]byte)
	defer writeBufPool.Put(bp)
	b, err := v.AppendJSON((*bp)[:0])
	*bp = b
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
	return nil
}

`

const helperWriteJSONSlice = `// writeJSONSlice is writeJSON for slices of ggen-generated types.
func writeJSONSlice[T encode.Marshaler](w http.ResponseWriter, vs []T) error {
	bp := writeBufPool.Get().(*[]byte)
	defer writeBufPool.Put(bp)
	b, err := encode.AppendSlice((*bp)[:0], vs)
	*bp = b
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
	return nil
}

`

const helperWriteJSONAny = `// writeJSONAny is writeJSON for values without generated methods (maps,
// mixed types); encode.AppendAny takes them through the same pooled buffer.
func writeJSONAny(w http.ResponseWriter, v any) error {
	bp := writeBufPool.Get().(*[]byte)
	defer writeBufPool.Put(bp)
	b, err := encode.AppendAny((*bp)[:0], v)
	*bp = b
	if err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	_, _ = w.Write(b)
	return nil
}

`

// float32 path params need a wrapper for the down-conversion
var numHelperSrc = map[string]string{
	"parseFloat32": `func parseFloat32(s string) (float32, error) {
	f, err := strconv.ParseFloat(s, 32)
	return float32(f), err
}

`,
}

// ggShape reports whether t is a type (or slice of one) with the given
// ggen-generated method, enabling the fast JSON paths.
func ggShape(methods map[string]map[string]*ast.FuncDecl, t ast.Expr, method string) (int, string) {
	switch tt := t.(type) {
	case *ast.Ident:
		if mm := methods[tt.Name]; mm != nil && mm[method] != nil {
			return ggOne, tt.Name
		}
	case *ast.ArrayType:
		if tt.Len == nil {
			if id, ok := tt.Elt.(*ast.Ident); ok {
				if mm := methods[id.Name]; mm != nil && mm[method] != nil {
					return ggSlice, id.Name
				}
			}
		}
	}
	return ggNone, ""
}

// middlewareRet reports whether a middleware returns error (true) or bool.
func middlewareRet(fd *ast.FuncDecl, ctx string) bool {
	types := resultTypes(fd)
	if len(types) == 1 {
		if id, ok := types[0].(*ast.Ident); ok {
			switch id.Name {
			case "bool":
				return false
			case "error":
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

func countFields(fl *ast.FieldList) int {
	if fl == nil {
		return 0
	}
	n := 0
	for _, f := range fl.List {
		c := len(f.Names)
		if c == 0 {
			c = 1
		}
		n += c
	}
	return n
}

// wrKind reports whether t is http.ResponseWriter or *http.Request, which
// bind by type and need no annotation.
func wrKind(t ast.Expr) (int, bool) {
	if st, ok := t.(*ast.StarExpr); ok {
		if sel, ok := st.X.(*ast.SelectorExpr); ok {
			if id, ok := sel.X.(*ast.Ident); ok && id.Name == "http" && sel.Sel.Name == "Request" {
				return argRequest, true
			}
		}
		return 0, false
	}
	if sel, ok := t.(*ast.SelectorExpr); ok {
		if id, ok := sel.X.(*ast.Ident); ok && id.Name == "http" && sel.Sel.Name == "ResponseWriter" {
			return argWriter, true
		}
	}
	return 0, false
}

// handlerArgs maps handler params to argSpecs. http.ResponseWriter and
// *http.Request bind by type and are optional; the rest is driven by inline
// /* api:... */ annotations, with an unannotated param binding to the
// transformed path param of the same name.
func handlerArgs(f *ast.File, d *ast.FuncDecl) []argSpec {
	params := d.Type.Params
	if params == nil {
		return nil
	}
	var anns []*ast.CommentGroup
	for _, cg := range f.Comments {
		if cg.Pos() > params.Opening && cg.End() < params.Closing && strings.HasPrefix(annText(cg), "api:") {
			anns = append(anns, cg)
		}
	}
	var specs []argSpec
	prev := params.Opening
	for fi, fl := range params.List {
		if fi > 0 {
			prev = params.List[fi-1].End()
		}
		if k, ok := wrKind(fl.Type); ok {
			n := len(fl.Names)
			if n == 0 {
				n = 1
			}
			for j := 0; j < n; j++ {
				specs = append(specs, argSpec{kind: k})
			}
			continue
		}
		if len(fl.Names) == 0 {
			fatalf("%s: handler params other than http.ResponseWriter and *http.Request must be named", d.Name.Name)
		}
		for _, nm := range fl.Names {
			spec := argSpec{kind: argParam, name: nm.Name, typeExpr: fl.Type}
			for _, cg := range anns {
				if cg.Pos() <= prev || cg.End() >= nm.Pos() {
					continue
				}
				fields := strings.Fields(annText(cg))
				switch fields[0] {
				case "api:param":
					if len(fields) > 1 {
						spec.name = fields[1]
					}
				case "api:body":
					if len(fields) > 1 && fields[1] != "json" {
						fatalf("%s: unsupported api:body format %q, only json", d.Name.Name, fields[1])
					}
					spec.kind = argBody
				case "api:query", "api:header":
					if fields[0] == "api:query" {
						spec.kind = argQuery
					} else {
						spec.kind = argHeader
					}
					if len(fields) > 1 {
						if strings.HasPrefix(fields[1], "@") {
							// whole-input parser: @func(url.Values|http.Header) (T[, error])
							if len(fields[1]) == 1 {
								fatalf("%s: parser in %q must be @name", d.Name.Name, fields[1])
							}
							spec.checker = fields[1][1:]
							spec.bind = bindParser
							break
						}
						nm, chk, has := strings.Cut(fields[1], "=")
						if nm != "" {
							spec.name = nm
						}
						if has {
							if !strings.HasPrefix(chk, "@") || len(chk) == 1 {
								fatalf("%s: checker in %q must be @name", d.Name.Name, fields[1])
							}
							spec.checker = chk[1:]
						}
					}
				default:
					fatalf("%s: unknown param annotation %q", d.Name.Name, fields[0])
				}
			}
			specs = append(specs, spec)
			prev = nm.End()
		}
	}
	return specs
}

// annText extracts annotation text, tolerating doc-comment forms like
// /** api:body json */ whose Text() keeps the leading asterisk.
func annText(cg *ast.CommentGroup) string {
	t := strings.TrimSpace(cg.Text())
	t = strings.TrimLeft(t, "*")
	return strings.TrimSpace(t)
}

func renderType(fset *token.FileSet, e ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

// analyzeQueryBind picks the binding shape for an unchecked api:query param
// from its declared type. Reports whether generated code will need strconv.
func analyzeQueryBind(spec *argSpec, types map[string]*ast.TypeSpec, ctx string) bool {
	isIdent := func(e ast.Expr, name string) bool {
		id, ok := e.(*ast.Ident)
		return ok && id.Name == name
	}
	switch t := spec.typeExpr.(type) {
	case *ast.Ident:
		if t.Name == "string" {
			return false // scalar
		}
		ts := types[t.Name]
		if ts == nil {
			fatalf("%s: api:query type %s is not declared in this package", ctx, t.Name)
		}
		st, ok := ts.Type.(*ast.StructType)
		if !ok {
			fatalf("%s: api:query type %s must be a struct, a map, or string", ctx, t.Name)
		}
		spec.bind = bindStruct
		spec.typ = t.Name
		needStrconv := false
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
				qf := queryField{name: fn.Name, key: key}
				switch {
				case isIdent(fld.Type, "string"):
					qf.kind = fString
				case isIdent(fld.Type, "int"):
					qf.kind = fInt
					needStrconv = true
				case isIdent(fld.Type, "bool"):
					qf.kind = fBool
					needStrconv = true
				default:
					fatalf("%s: query struct field %s.%s must be string, int or bool", ctx, t.Name, fn.Name)
				}
				spec.fields = append(spec.fields, qf)
			}
		}
		return needStrconv
	case *ast.MapType:
		if !isIdent(t.Key, "string") {
			fatalf("%s: api:query map key must be string", ctx)
		}
		switch v := t.Value.(type) {
		case *ast.Ident:
			switch v.Name {
			case "string":
				spec.bind = bindMap
			case "any":
				spec.bind = bindMap
				spec.mapAny = true
			default:
				fatalf("%s: api:query map value must be string, any or []string", ctx)
			}
		case *ast.ArrayType:
			if v.Len != nil || !isIdent(v.Elt, "string") {
				fatalf("%s: api:query map value must be string, any or []string", ctx)
			}
			spec.bind = bindValues
		case *ast.InterfaceType:
			if len(v.Methods.List) != 0 {
				fatalf("%s: api:query map value must be string, any or []string", ctx)
			}
			spec.bind = bindMap
			spec.mapAny = true
		default:
			fatalf("%s: api:query map value must be string, any or []string", ctx)
		}
		return false
	case *ast.SelectorExpr:
		if id, ok := t.X.(*ast.Ident); ok && id.Name == "url" && t.Sel.Name == "Values" {
			spec.bind = bindValues
			return false
		}
		fatalf("%s: unsupported api:query type", ctx)
	default:
		fatalf("%s: unsupported api:query type", ctx)
	}
	return false
}

// displayPattern renders the route in ServeMux Pattern form: method-prefixed,
// params as plain {name}, without checker internals.
func displayPattern(rt route) string {
	var b strings.Builder
	if rt.method != "" {
		b.WriteString(rt.method)
		b.WriteByte(' ')
	}
	for _, tk := range rt.tokens {
		b.WriteByte('/')
		switch tk.kind {
		case tokLit:
			b.WriteString(tk.lit)
		case tokParam:
			b.WriteString("{")
			b.WriteString(tk.p.name)
			b.WriteString("}")
		case tokWild:
			if tk.p != nil {
				b.WriteString("{")
				b.WriteString(tk.p.name)
				b.WriteString("...}")
			} else {
				b.WriteByte('*')
			}
		}
	}
	return b.String()
}

func isDynamic(rt route) bool {
	for _, t := range rt.tokens {
		if t.kind != tokLit {
			return true
		}
	}
	return false
}

// commonPrefix returns the longest literal prefix shared by all patterns,
// cut back to the last '/' so the switch keys stay whole segments.
func commonPrefix(routes []route) string {
	lit := func(p string) string {
		if i := strings.IndexAny(p, "{*"); i >= 0 {
			p = p[:i]
		}
		return p
	}
	pfx := lit(routes[0].pattern)
	for _, rt := range routes[1:] {
		l := lit(rt.pattern)
		n := 0
		for n < len(pfx) && n < len(l) && pfx[n] == l[n] {
			n++
		}
		pfx = pfx[:n]
	}
	if i := strings.LastIndexByte(pfx, '/'); i >= 0 {
		pfx = pfx[:i+1]
	}
	return pfx
}
