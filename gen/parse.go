package gen

import (
	"go/ast"
	"go/printer"
	"go/token"
	"strings"
)

// Go spellings the parser matches on.
const (
	typString = "string"
	typBool   = "bool"
	typError  = "error"
	pkgHTTP   = "http"
)

// parseDirectives pulls the `rr:` lines out of a doc comment, each split into
// its kind and the rest of the line. A ` -- comment` suffix is dropped.
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
		rest, ok := strings.CutPrefix(line, "rr:")
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

// docText is the doc comment with its rr: directive lines removed.
func docText(doc *ast.CommentGroup) string {
	if doc == nil {
		return ""
	}
	var keep []string
	for line := range strings.SplitSeq(doc.Text(), "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "rr:") {
			continue
		}
		keep = append(keep, line)
	}
	return strings.TrimSpace(strings.Join(keep, "\n"))
}

// parsePattern splits a route pattern into segments. Params span a whole
// segment; a catch-all ({name...} or *) is only allowed last.
func parsePattern(pattern, handler string) []Segment {
	if !strings.HasPrefix(pattern, "/") {
		fatalf("%s: route pattern %q must start with /", handler, pattern)
	}
	pieces := strings.Split(pattern[1:], "/")
	var segs []Segment
	for i, piece := range pieces {
		last := i == len(pieces)-1
		switch {
		case piece == "*":
			if !last {
				fatalf("%s: * is only allowed at the end of a pattern", handler)
			}
			segs = append(segs, Segment{Kind: SegRest})
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
			if wild, ok := strings.CutSuffix(name, "..."); ok {
				if !last {
					fatalf("%s: {%s} is only allowed at the end of a pattern", handler, name)
				}
				if checker != "" {
					fatalf("%s: wildcard {%s} cannot have a matcher", handler, name)
				}
				if wild == "" {
					fatalf("%s: empty wildcard name", handler)
				}
				segs = append(segs, Segment{Kind: SegRest, Param: &Param{Name: wild, Rest: true, Type: stringType}})
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
			segs = append(segs, Segment{Kind: SegParam, Param: &Param{Name: name, checker: checker, Type: stringType}})
		case strings.ContainsAny(piece, "{}*"):
			fatalf("%s: params must span a whole path segment, got %q", handler, piece)
		case piece == "" && !last:
			fatalf("%s: empty path segment in %q", handler, pattern)
		default:
			segs = append(segs, Segment{Kind: SegLit, Lit: piece})
		}
	}
	return segs
}

var stringType = TypeRef{Expr: typString, Name: typString}

// muxPath renders the segments in net/http ServeMux form: params as plain
// {name}, without any checker internals.
func muxPath(segs []Segment) string {
	var b strings.Builder
	for _, s := range segs {
		b.WriteByte('/')
		switch s.Kind {
		case SegLit:
			b.WriteString(s.Lit)
		case SegParam:
			b.WriteString("{")
			b.WriteString(s.Param.Name)
			b.WriteString("}")
		case SegRest:
			if s.Param != nil {
				b.WriteString("{")
				b.WriteString(s.Param.Name)
				b.WriteString("...}")
			} else {
				b.WriteByte('*')
			}
		}
	}
	return b.String()
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

func resultTypes(fd *ast.FuncDecl) []ast.Expr {
	var types []ast.Expr
	if fd.Type.Results != nil {
		for _, f := range fd.Type.Results.List {
			n := len(f.Names)
			if n == 0 {
				n = 1
			}
			for range n {
				types = append(types, f.Type)
			}
		}
	}
	return types
}

func isIdent(e ast.Expr, name string) bool {
	id, ok := e.(*ast.Ident)
	return ok && id.Name == name
}

func isErrorIdent(e ast.Expr) bool { return isIdent(e, typError) }

// resultShape reports what a handler returns: the value type (nil for none)
// and whether an error comes with it.
func resultShape(d *ast.FuncDecl) (val ast.Expr, err bool) {
	types := resultTypes(d)
	switch len(types) {
	case 0:
		return nil, false
	case 1:
		if isErrorIdent(types[0]) {
			return nil, true
		}
		return types[0], false
	case 2:
		if isErrorIdent(types[1]) {
			return types[0], true
		}
	}
	fatalf("%s: handler must return nothing, error, T, or (T, error)", d.Name.Name)
	return nil, false
}

func lowerFirst(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

func renderType(fset *token.FileSet, e ast.Expr) string {
	var b strings.Builder
	_ = printer.Fprint(&b, fset, e)
	return b.String()
}

// annText extracts annotation text, tolerating doc-comment forms like
// /** rr:query whose Text() keeps the leading asterisk.
func annText(cg *ast.CommentGroup) string {
	t := strings.TrimSpace(cg.Text())
	t = strings.TrimLeft(t, "*")
	return strings.TrimSpace(t)
}

// parseHandlerOpts reads the shared error-handler options off a rr:api /
// rr:controller directive: onerror=@f on400=@f on404=@f on405=@f.
func parseHandlerOpts(raw *apiRaw, opts, ctx string) {
	for opt := range strings.FieldsSeq(opts) {
		k, v, ok := strings.Cut(opt, "=")
		if !ok || !strings.HasPrefix(v, "@") || len(v) == 1 {
			fatalf("%s: bad option %q, want key=@func", ctx, opt)
		}
		v = v[1:]
		switch k {
		case "onerror":
			raw.onError = v
		case "on404":
			raw.on404 = v
		case "on405":
			raw.on405 = v
		case "on400":
			raw.on400 = v
		default:
			fatalf("%s: unknown option %q (want onerror/on400/on404/on405)", ctx, k)
		}
	}
}

// handlerArgs maps handler params to Args. Roles are inferred:
//   - http.ResponseWriter / *http.Request / context.Context bind by type, any
//     position
//   - a param named body / query / headers is that whole-object role (a route
//     token of the same name overrides it back to a path param, resolved later)
//   - an inline /* rr:query key=@check */ or /* rr:header Name=@check */ binds
//     one keyed value
//   - anything else is a path param (must appear in the route, else an error)
func (l *loader) handlerArgs(f *ast.File, d *ast.FuncDecl) []*Arg {
	params := d.Type.Params
	if params == nil {
		return nil
	}
	var anns []*ast.CommentGroup
	for _, cg := range f.Comments {
		if cg.Pos() > params.Opening && cg.End() < params.Closing && strings.HasPrefix(annText(cg), "rr:") {
			anns = append(anns, cg)
		}
	}
	var specs []*Arg
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
			for range n {
				specs = append(specs, &Arg{Kind: k})
			}
			continue
		}
		if len(fl.Names) == 0 {
			fatalf("%s: handler params other than http.ResponseWriter, *http.Request and context.Context must be named", d.Name.Name)
		}
		for _, nm := range fl.Names {
			spec := &Arg{Kind: ArgPath, Name: nm.Name, Key: nm.Name, Type: l.typeRef(fl.Type)}
			// a single-name param accepts its annotation before the name or
			// between the name and the type; multi-name fields keep it before
			// each name to stay unambiguous
			upper := nm.Pos()
			if len(fl.Names) == 1 {
				upper = fl.End()
			}
			var cg *ast.CommentGroup
			for _, c := range anns {
				if c.Pos() > prev && c.End() < upper {
					cg = c
				}
			}
			if cg != nil {
				parseParamAnnotation(spec, annText(cg), d.Name.Name)
			} else {
				switch nm.Name { // reserved names are the annotation-free defaults
				case "body":
					spec.Kind, spec.byName = ArgBody, true
				case "query":
					spec.Kind, spec.byName, spec.Whole = ArgQuery, true, true
				case "headers":
					spec.Kind, spec.byName, spec.Whole = ArgHeader, true, true
				}
			}
			specs = append(specs, spec)
			prev = nm.End()
		}
	}
	return specs
}

// parseParamAnnotation sets a param's role from its explicit inline comment,
// overriding the name-based default. Forms:
//
//	/* rr:body */                  the request body (type decides the format)
//	/* rr:param [name] */          a path param (optionally renamed)
//	/* rr:query [key][=@check] */  one query value; bare = whole query by type
//	/* rr:query @parser */         whole query through a custom parser
//	/* rr:header ... */            same, for headers
func parseParamAnnotation(spec *Arg, text, handler string) {
	fields := strings.Fields(text)
	switch fields[0] {
	case "rr:body":
		spec.Kind = ArgBody
		if len(fields) > 1 && fields[1] != "json" {
			fatalf("%s: rr:body takes no format, the param type decides it (got %q)", handler, fields[1])
		}
		return
	case "rr:param":
		spec.Kind = ArgPath
		if len(fields) > 1 {
			spec.Key = fields[1]
		}
		return
	case "rr:query":
		spec.Kind = ArgQuery
	case "rr:header":
		spec.Kind = ArgHeader
	default:
		fatalf("%s: unknown param annotation %q", handler, fields[0])
	}
	if len(fields) <= 1 { // bare rr:query / rr:header: bind whole by type
		spec.Whole = true
		return
	}
	if strings.HasPrefix(fields[1], "@") {
		// whole-input parser: @func(url.Values|http.Header) (T[, error])
		if len(fields[1]) == 1 {
			fatalf("%s: parser in %q must be @name", handler, fields[1])
		}
		spec.checker = fields[1][1:]
		spec.Shape = BindParser
		return
	}
	key, chk, has := strings.Cut(fields[1], "=")
	if key != "" {
		spec.Key = key
	}
	if has {
		if !strings.HasPrefix(chk, "@") || len(chk) == 1 {
			fatalf("%s: checker in %q must be @name", handler, fields[1])
		}
		spec.checker = chk[1:]
	}
}
