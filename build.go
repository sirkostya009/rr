package main

import (
	"go/ast"
	"go/token"
	"path"
	"strings"

	"github.com/sirkostya009/rr/gen"
)

const ggenPkg = "github.com/sirkostya009/ggen"

// builder turns the parsed route model into the emitter's view of it. The
// model is language-neutral on purpose, so everything Go-specific about a
// route lands here: the ggen fast-path shapes, the matcher derived from a
// path param's declared type, the rendered type names, and the imports the
// generated file needs.
type builder struct {
	fset *token.FileSet
	idx  *gen.Index

	// ggenOn: once ggen is in the picture (generated methods on any type, or
	// just imported), arbitrary response values ride ggen.AppendAny too
	// instead of falling back to encoding/json
	ggenOn bool

	extraImports map[string]bool
	importAlias  map[string]string // path -> the ident the sources refer to it by
	numHelpers   map[string]bool
}

// detectGgen decides whether the package is in ggen's world at all.
func (b *builder) detectGgen(pkg *gen.Package) {
	for _, mm := range b.idx.Methods {
		if mm["AppendJSON"] != nil {
			b.ggenOn = true
			return
		}
	}
	for _, p := range b.idx.Imports {
		if p == ggenPkg || strings.HasPrefix(p, ggenPkg+"/") {
			b.ggenOn = true
			return
		}
	}
	for _, a := range pkg.APIs {
		for _, rt := range a.Routes {
			types := make([]ast.Expr, 0, len(rt.Args)+1)
			if rt.Result != nil {
				types = append(types, rt.Result.Node)
			}
			for _, arg := range rt.Args {
				if arg.Type.Node != nil {
					types = append(types, arg.Type.Node)
				}
			}
			for _, t := range types {
				if st, ok := t.(*ast.StarExpr); ok {
					t = st.X
				}
				if sh, _ := ggShape(nil, b.idx.Imports, t, "AppendJSON"); sh != ggNone {
					b.ggenOn = true
					return
				}
			}
		}
	}
}

// build projects the package's apis onto the emitter's types, in the order
// they were first mentioned.
func (b *builder) build(pkg *gen.Package) []*apiType {
	byModel := make(map[*gen.API]*apiType, len(pkg.APIs))
	out := make([]*apiType, 0, len(pkg.APIs))
	for _, ga := range pkg.APIs {
		a := &apiType{
			name:    ga.Name,
			recv:    ga.Recv,
			ptr:     ga.Ptr,
			central: ga.Central,
			mounted: ga.Mounted,
			prefix:  ga.Prefix,
		}
		byModel[ga] = a
		out = append(out, a)
	}
	for _, ga := range pkg.APIs {
		a := byModel[ga]
		for _, m := range ga.Mounts {
			a.mounts = append(a.mounts, mount{m.Field, byModel[m.API]})
		}
		for _, x := range ga.XMounts {
			a.xmounts = append(a.xmounts, xmount{x.Field, x.Prefix})
		}
		for _, h := range ga.Guards {
			a.middlewares = append(a.middlewares, middleware{
				ref:    b.ref(h.Ref),
				retErr: h.RetErr,
				args:   b.args(h.Args),
			})
		}
		a.errHandler = b.ehandler(ga.OnError)
		a.notFound = b.ehandler(ga.On404)
		a.notAllowed = b.ehandler(ga.On405)
		a.badReq = b.ehandler(ga.On400)
		for _, gr := range ga.Routes {
			a.routes = append(a.routes, b.route(gr, a))
		}
	}
	return out
}

// ref renders a resolved @ref, noting the import a qualified one needs.
func (b *builder) ref(r gen.Ref) refExpr {
	if r.Kind == gen.RefExternal {
		b.noteRefImport(r.Name)
	}
	return refExpr{method: r.Kind == gen.RefMethod, name: r.Name}
}

// noteRefImport records the package a qualified @ref lives in. Not being
// imported by the sources is fine: assume a stdlib-style path
// (strconv.Atoi -> "strconv"); external packages need an import, a blank one
// is enough.
func (b *builder) noteRefImport(name string) {
	before, _, ok := strings.Cut(name, ".")
	if !ok {
		return
	}
	b.noteImportPath(before)
}

// noteImport records the package of a qualified type named in the output.
func (b *builder) noteImport(t ast.Expr) {
	sel, ok := t.(*ast.SelectorExpr)
	if !ok {
		return
	}
	if id, ok := sel.X.(*ast.Ident); ok {
		b.noteImportPath(id.Name)
	}
}

func (b *builder) noteImportPath(ident string) {
	p, ok := b.idx.Imports[ident]
	if !ok {
		p = ident
	}
	b.extraImports[p] = true
	if ident != path.Base(p) {
		b.importAlias[p] = ident
	}
}

func (b *builder) ehandler(h *gen.Handler) ehandler {
	if h == nil {
		return ehandler{}
	}
	return ehandler{ref: b.ref(h.Ref), args: b.args(h.Args), hasErr: h.TakesErr}
}

func (b *builder) args(in []*gen.Arg) []argSpec {
	if len(in) == 0 {
		return nil
	}
	out := make([]argSpec, len(in))
	for i, ga := range in {
		out[i] = b.arg(ga)
	}
	return out
}

func (b *builder) arg(ga *gen.Arg) argSpec {
	spec := argSpec{name: ga.Key, whole: ga.Whole, mapAny: ga.MapAny, typeExpr: ga.Type.Node}
	switch ga.Kind {
	case gen.ArgPath:
		spec.kind = argParam
	case gen.ArgBody:
		spec.kind = argBody
		b.bodyShape(&spec, ga)
	case gen.ArgQuery:
		spec.kind = argQuery
	case gen.ArgHeader:
		spec.kind = argHeader
	case gen.ArgWriter:
		// a named type embedding http.ResponseWriter is the writer itself,
		// type-asserted at the call site
		if ga.Type.Node == nil {
			spec.kind = argWriter
		} else {
			spec.kind, spec.typ = argWriterCast, ga.Type.Expr
		}
	case gen.ArgRequest:
		spec.kind = argRequest
	case gen.ArgContext:
		spec.kind = argContext
	case gen.ArgError:
		spec.kind = argError
	}
	if ga.Kind == gen.ArgQuery || ga.Kind == gen.ArgHeader {
		b.bindShape(&spec, ga)
	}
	if ga.Check != nil {
		spec.checker = ga.Check.Name
		spec.ref = b.ref(*ga.Check)
		spec.regex = ga.Check.Kind == gen.RefRegexp
		if ga.Shape == gen.BindParser {
			spec.transform = ga.Check.Err // a bare-T parser needs no error branch
		} else {
			spec.transform = ga.Check.Transform
			if spec.transform {
				spec.typ = ga.Type.Expr
			}
		}
	}
	return spec
}

func (b *builder) bodyShape(spec *argSpec, ga *gen.Arg) {
	t := ga.Type.Node
	if st, ok := t.(*ast.StarExpr); ok {
		spec.ptr = true
		t = st.X
	}
	// a form body is read off the request, so it needs no decode and no type
	//exhaustive:ignore BodyJSON is everything below
	switch ga.Body {
	case gen.BodyMultipart:
		spec.bodyKind = bodyMultipart
		return
	case gen.BodyForm:
		spec.bodyKind = bodyForm
		return
	}
	spec.bodyKind = bodyJSON
	spec.typ = renderType(b.fset, t)
	if il, ok := t.(*ast.IndexListExpr); ok && ga.Type.Seq {
		// only ggen's stream reads an array lazily
		elem := il.Indices[0]
		if spec.ptr {
			fatalf("body %s: a sequence is passed by value, not as a pointer", ga.Type.Expr)
		}
		if spec.fast, spec.elem = ggShape(b.idx.Methods, b.idx.Imports, elem, "DecodeFromStream"); spec.fast != ggOne {
			fatalf("body %s: the element type of an iter.Seq2 body needs ggen-generated methods", ga.Type.Expr)
		}
		spec.fast = ggSeq
		b.noteImport(elem)
		return
	}
	spec.fast, spec.elem = ggShape(b.idx.Methods, b.idx.Imports, t, "DecodeFromStream")
	if spec.ptr && spec.fast == ggSlice {
		spec.fast = ggNone
	}
	b.noteImport(t)
}

func (b *builder) bindShape(spec *argSpec, ga *gen.Arg) {
	switch ga.Shape {
	case gen.BindValue:
		spec.bind = bindScalar
	case gen.BindValues:
		spec.bind = bindValues
	case gen.BindMap:
		spec.bind = bindMap
	case gen.BindStruct:
		spec.bind = bindStruct
		spec.typ = ga.Type.Expr
	case gen.BindParser:
		spec.bind = bindParser
	}
	for _, f := range ga.Fields {
		qf := queryField{name: f.Name, key: f.Key}
		//exhaustive:ignore the default is FieldString
		switch f.Kind {
		case gen.FieldInt:
			qf.kind = fInt
			b.extraImports["strconv"] = true
		case gen.FieldBool:
			qf.kind = fBool
			b.extraImports["strconv"] = true
		default:
			qf.kind = fString
		}
		spec.fields = append(spec.fields, qf)
	}
}

func (b *builder) route(gr *gen.Route, a *apiType) route {
	rt := route{
		method:  gr.Method,
		pattern: gr.Pattern,
		display: gr.Path,
		handler: gr.Handler,
		owner:   a,
		errH:    a.errHandler, // owner onerror; central covers it at emission
		args:    b.args(gr.Args),
	}
	if gr.Method != "" {
		rt.display = gr.Method + " " + gr.Path
	}

	params := map[*gen.Param]*param{}
	for _, s := range gr.Segments {
		switch s.Kind {
		case gen.SegLit:
			rt.tokens = append(rt.tokens, tok{kind: tokLit, lit: s.Lit})
		case gen.SegParam:
			rt.tokens = append(rt.tokens, tok{kind: tokParam, p: b.param(s.Param, params)})
		case gen.SegRest:
			t := tok{kind: tokWild}
			if s.Param != nil {
				t.p = b.param(s.Param, params)
			}
			rt.tokens = append(rt.tokens, t)
		}
	}

	switch {
	case gr.Result == nil && !gr.Err:
		rt.retKind = retNone
	case gr.Result == nil:
		rt.retKind = retErr
	case !gr.Err:
		rt.retKind = retVal
	default:
		rt.retKind = retValErr
	}
	if gr.Result != nil {
		rt.retType = gr.Result.Node
		t := rt.retType
		if st, ok := t.(*ast.StarExpr); ok {
			t = st.X
		}
		rt.enc, _ = ggShape(b.idx.Methods, b.idx.Imports, t, "AppendJSON")
		if rt.enc == ggNone && b.ggenOn {
			rt.enc = ggAny
		}
		if rt.retKind == retValErr {
			rt.retTyp = gr.Result.Expr
			b.noteImport(t)
		}
	}

	// a path param with no matcher of its own derives one from the type its
	// handler declares
	for _, ga := range gr.Args {
		if ga.Kind != gen.ArgPath || ga.Param == nil || ga.Param.Rest {
			continue
		}
		pp := params[ga.Param]
		if pp == nil || pp.transform {
			continue
		}
		b.derive(a, &rt, pp, ga)
	}
	return rt
}

func (b *builder) param(gp *gen.Param, memo map[*gen.Param]*param) *param {
	if p, ok := memo[gp]; ok {
		return p
	}
	p := &param{name: gp.Name}
	if gp.Check != nil {
		p.checker = gp.Check.Name
		p.ref = b.ref(*gp.Check)
		p.transform = gp.Check.Transform
		p.regex = gp.Check.Kind == gen.RefRegexp
	}
	memo[gp] = p
	return p
}

// derive picks the matcher for a path param from its declared Go type.
// classNumeric before classBool is what keeps ParseBool's lax "1"/"0" from
// stealing digits at a shared position.
func (b *builder) derive(a *apiType, rt *route, pp *param, ga *gen.Arg) {
	tn := ""
	if id, _ := ga.Type.Node.(*ast.Ident); id != nil {
		tn = id.Name
	}
	if tn != typString && pp.checker != "" {
		fatalf("%s.%s: param {%s} already has a matcher; use {%s=@func} to transform it",
			a.name, rt.handler, pp.name, pp.name)
	}
	set := func(ref string, class int) {
		pp.ref = refExpr{false, ref}
		pp.transform = true
		pp.class = class
		pp.typ = tn
		b.extraImports["strconv"] = true
	}
	switch tn {
	case typString:
		// raw segment, nothing to derive
	case "int":
		set("strconv.Atoi", classNumeric)
	case "float64":
		set("strconv.ParseFloat", classNumeric)
		pp.extraArgs = ", 64"
	case "float32":
		set("parseFloat32", classNumeric)
		b.numHelpers["parseFloat32"] = true
	case typBool:
		set("strconv.ParseBool", classBool)
	default:
		fatalf("%s.%s: cannot bind path param %q to type %s: structs, interfaces and any are not path param material; use string, int, float64, float32, bool, or {%s=@func}",
			a.name, rt.handler, ga.Key, ga.Type.Expr, ga.Key)
	}
}
