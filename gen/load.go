package gen

import (
	"context"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

// apiRaw holds the directive text of an api until every api in the package is
// known and refs can be resolved against it.
type apiRaw struct {
	mw      []string
	onError string
	on404   string
	on405   string
	on400   string
	fields  []fieldCand
	xfields []xFieldCand
}

// loader parses one package.
type loader struct {
	pkg  *Package
	idx  *Index
	fset *token.FileSet
	apis map[string]*API
	raw  map[*API]*apiRaw
}

// Load parses the rr: directives of every package the patterns match.
// Patterns are directories or `...` patterns, resolved the way `go list`
// resolves them.
//
// Load parses sources only: it never reads rr's own output, so a missing or
// stale generated file cannot change what it reports, and the result does not
// depend on generation order.
func Load(patterns ...string) (set *Set, err error) {
	defer catch(&err)
	locs, err := expand(patterns)
	if err != nil {
		return nil, err
	}
	set = &Set{}
	var loaders []*loader
	for _, loc := range locs {
		l := &loader{apis: map[string]*API{}, raw: map[*API]*apiRaw{}}
		if !l.parseDir(loc) {
			continue // no rr: directives here; a pattern may match many packages
		}
		loaders = append(loaders, l)
		set.Packages = append(set.Packages, l.pkg)
	}
	if len(loaders) == 0 {
		fatalf("no rr: directives found")
	}
	for _, l := range loaders {
		l.link()
	}
	for _, l := range loaders {
		l.resolveAll()
		l.prefixes()
	}
	linkXMounts(set)
	return set, nil
}

// pkgLoc is one package to parse.
type pkgLoc struct {
	path, dir, name string
}

// expand resolves patterns to package directories. `go list` does it when it
// can; a plain directory still loads when it cannot, only without an import
// path.
func expand(patterns []string) ([]pkgLoc, error) {
	if len(patterns) == 0 {
		return nil, &Error{Msg: "no packages given"}
	}
	args := make([]string, 0, len(patterns)+4)
	args = append(args, "list", "-e", "-f", "{{.ImportPath}}\t{{.Dir}}\t{{.Name}}")
	for _, p := range patterns {
		if !strings.HasPrefix(p, ".") && !filepath.IsAbs(p) {
			if fi, err := os.Stat(p); err == nil && fi.IsDir() {
				p = "./" + filepath.ToSlash(p)
			}
		}
		args = append(args, p)
	}
	out, err := exec.CommandContext(context.Background(), "go", args...).Output() //nolint:gosec // patterns come from the caller
	if err == nil {
		var locs []pkgLoc
		for line := range strings.SplitSeq(strings.TrimSpace(string(out)), "\n") {
			ip, rest, ok := strings.Cut(line, "\t")
			if !ok {
				continue
			}
			dir, name, _ := strings.Cut(rest, "\t")
			if dir == "" {
				continue
			}
			locs = append(locs, pkgLoc{ip, dir, name})
		}
		if len(locs) > 0 {
			return locs, nil
		}
	}
	// go list is unavailable or unhappy: a directory pattern still names a
	// package, it just has no import path to report.
	var locs []pkgLoc
	for _, p := range patterns {
		if fi, err := os.Stat(p); err == nil && fi.IsDir() {
			locs = append(locs, pkgLoc{dir: p})
		}
	}
	if len(locs) == 0 {
		return nil, &Error{Msg: "no packages match " + strings.Join(patterns, " ")}
	}
	return locs, nil
}

// parseDir parses one package directory and collects its directives. It
// reports whether the package declares any.
func (l *loader) parseDir(loc pkgLoc) bool {
	entries, err := os.ReadDir(loc.dir)
	if err != nil {
		fatalf("%v", err)
	}
	fset := token.NewFileSet()
	var files []*ast.File
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(loc.dir, name), nil, parser.ParseComments)
		if err != nil {
			fatalf("%v", err)
		}
		if isGenerated(f) { // our own output, about to be replaced
			continue
		}
		files = append(files, f)
	}
	if len(files) == 0 {
		return false
	}
	pkgName := loc.name
	if pkgName == "" {
		pkgName = files[0].Name.Name
	}
	// one directory holds one package; anything else here is a test variant
	kept := files[:0]
	for _, f := range files {
		if f.Name.Name == pkgName {
			kept = append(kept, f)
		}
	}
	files = kept

	l.fset = fset
	l.idx = &Index{
		Funcs:   map[string]*ast.FuncDecl{},
		Methods: map[string]map[string]*ast.FuncDecl{},
		Types:   map[string]*ast.TypeSpec{},
		Regexps: map[string]struct{}{},
		Imports: map[string]string{},
		FileOf:  map[*ast.FuncDecl]*ast.File{},
	}
	l.pkg = &Package{Path: loc.path, Name: pkgName, Dir: loc.dir, Fset: fset, Files: files, idx: l.idx}
	l.index(files)
	l.directives(files)
	return len(l.pkg.APIs) > 0
}

// index records every declaration an @ref or a type lookup can name.
func (l *loader) index(files []*ast.File) {
	for _, f := range files {
		for _, imp := range f.Imports {
			p, _ := strconv.Unquote(imp.Path.Value)
			name := path.Base(p)
			// a blank import still makes the package referenceable in directives
			if imp.Name != nil && imp.Name.Name != "_" {
				name = imp.Name.Name
			}
			l.idx.Imports[name] = p
		}
		for _, decl := range f.Decls {
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.TYPE {
				for _, spec := range gd.Specs {
					if ts, ok := spec.(*ast.TypeSpec); ok {
						l.idx.Types[ts.Name.Name] = ts
					}
				}
				continue
			}
			if gd, ok := decl.(*ast.GenDecl); ok && gd.Tok == token.VAR {
				for _, spec := range gd.Specs {
					vs, ok := spec.(*ast.ValueSpec)
					if !ok {
						continue
					}
					typed := isRegexpType(vs.Type)
					for i, nm := range vs.Names {
						if typed || (i < len(vs.Values) && isRegexpCompile(vs.Values[i])) {
							l.idx.Regexps[nm.Name] = struct{}{}
						}
					}
				}
				continue
			}
			fd, ok := decl.(*ast.FuncDecl)
			if !ok {
				continue
			}
			l.idx.FileOf[fd] = f
			if fd.Recv == nil {
				l.idx.Funcs[fd.Name.Name] = fd
				continue
			}
			t, _ := recvTypeName(fd.Recv)
			if t == "" {
				continue
			}
			if l.idx.Methods[t] == nil {
				l.idx.Methods[t] = map[string]*ast.FuncDecl{}
			}
			l.idx.Methods[t][fd.Name.Name] = fd
		}
	}
}

func (l *loader) api(name string) *API {
	a := l.apis[name]
	if a == nil {
		a = &API{Pkg: l.pkg, Name: name, Recv: "s", Ptr: true}
		l.apis[name] = a
		l.raw[a] = &apiRaw{}
		l.pkg.APIs = append(l.pkg.APIs, a)
	}
	return a
}

func refName(v, ctx string) string {
	if !strings.HasPrefix(v, "@") || len(v) == 1 {
		fatalf("%s: expected @name, got %q", ctx, v)
	}
	return v[1:]
}

// directives scans the whole package for rr: directives: go:generate usually
// passes a single $GOFILE, while routes live wherever the package keeps them.
func (l *loader) directives(files []*ast.File) {
	for _, f := range files {
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				if d.Tok != token.TYPE {
					continue
				}
				for _, spec := range d.Specs {
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					doc := ts.Doc
					if doc == nil && len(d.Specs) == 1 {
						doc = d.Doc
					}
					var a *API
					for _, dir := range parseDirectives(doc) {
						if a == nil {
							a = l.api(ts.Name.Name)
							a.Spec = ts
							a.Doc = docText(doc)
						}
						raw := l.raw[a]
						switch dir[0] {
						case "pre":
							// guard chain on one line: //rr:pre @f @g @h
							for m := range strings.FieldsSeq(dir[1]) {
								raw.mw = append(raw.mw, refName(m, ts.Name.Name))
							}
						case "api":
							a.Central = true
							st, ok := ts.Type.(*ast.StructType)
							if !ok {
								fatalf("%s: rr:api must be a struct", ts.Name.Name)
							}
							for _, fld := range st.Fields.List {
								fc, xc, ok := apiField(fld)
								if !ok {
									continue
								}
								if xc.field != "" {
									raw.xfields = append(raw.xfields, xc)
								} else {
									raw.fields = append(raw.fields, fc)
								}
							}
							parseHandlerOpts(raw, dir[1], ts.Name.Name)
						case "controller":
							// optional marker for a mounted api; carries the same
							// error-handler options as rr:api
							parseHandlerOpts(raw, dir[1], ts.Name.Name)
						default:
							fatalf("%s: unsupported directive rr:%s on a type", ts.Name.Name, dir[0])
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
				a := l.api(tname)
				a.Ptr = ptr
				if names := d.Recv.List[0].Names; len(names) == 1 && names[0].Name != "" && names[0].Name != "_" {
					a.Recv = names[0].Name
				}
				val, hasErr := resultShape(d)
				rt := &Route{
					API:     a,
					Handler: d.Name.Name,
					Doc:     docText(d.Doc),
					Decl:    d,
					Pos:     l.fset.Position(d.Pos()),
					Err:     hasErr,
					Args:    l.handlerArgs(f, d),
				}
				for _, dir := range dirs {
					switch dir[0] {
					case "route":
						if rt.Pattern != "" {
							fatalf("%s: multiple rr:route directives", d.Name.Name)
						}
						fields := strings.Fields(dir[1])
						switch len(fields) {
						case 1:
							rt.Pattern = fields[0]
						case 2:
							rt.Method, rt.Pattern = fields[0], fields[1]
						default:
							fatalf("%s: bad rr:route %q", d.Name.Name, dir[1])
						}
					default:
						fatalf("%s: unsupported directive rr:%s on a method", d.Name.Name, dir[0])
					}
				}
				if val != nil {
					tr := l.typeRef(val)
					rt.Result = &tr
				}
				if rt.Pattern == "" {
					fatalf("%s: rr: directives without rr:route", d.Name.Name)
				}
				rt.Segments = parsePattern(rt.Pattern, d.Name.Name)
				rt.Path = muxPath(rt.Segments)
				a.Routes = append(a.Routes, rt)
			}
		}
	}
}

// typeRef describes a Go type as a handler wrote it.
func (l *loader) typeRef(e ast.Expr) TypeRef {
	if e == nil {
		return TypeRef{}
	}
	tr := TypeRef{Node: e, Expr: renderType(l.fset, e)}
	t := e
	if st, ok := t.(*ast.StarExpr); ok {
		tr.Ptr = true
		t = st.X
	}
	if at, ok := t.(*ast.ArrayType); ok && at.Len == nil {
		tr.Slice = true
		t = at.Elt
	}
	if st, ok := t.(*ast.StarExpr); ok {
		tr.Ptr = true
		t = st.X
	}
	switch tt := t.(type) {
	case *ast.Ident:
		tr.Name = tt.Name
		if _, ok := l.idx.Types[tt.Name]; ok {
			tr.Pkg = l.pkg.Path
		}
	case *ast.SelectorExpr:
		if id, ok := tt.X.(*ast.Ident); ok {
			tr.Name = tt.Sel.Name
			tr.Pkg = l.idx.Imports[id.Name]
		}
	}
	return tr
}

// link attaches api-typed fields: in-package ones merge into this
// dispatcher, cross-package ones are delegated by the prefix their own
// sources say they own.
func (l *loader) link() {
	for _, a := range l.pkg.APIs {
		raw := l.raw[a]
		for _, fc := range raw.fields {
			if m := l.apis[fc.typeName]; m != nil && m != a {
				if m.Central {
					fatalf("%s: nested rr:api types are not supported (field %s)", a.Name, fc.field)
				}
				m.Mounted = true
				a.Mounts = append(a.Mounts, Mount{fc.field, m})
			}
		}
		for _, xc := range raw.xfields {
			pkgPath, ok := l.idx.Imports[xc.pkgIdent]
			if !ok {
				fatalf("%s: field %s: package %s is not imported", a.Name, xc.field, xc.pkgIdent)
			}
			prefix := discoverPrefix(pkgPath, xc.typeName)
			if prefix == "" {
				fatalf("%s: field %s: %s.%s owns no rr: routes", a.Name, xc.field, xc.pkgIdent, xc.typeName)
			}
			a.XMounts = append(a.XMounts, XMount{Field: xc.field, Pkg: pkgPath, Type: xc.typeName, Prefix: prefix})
		}
		if a.Central {
			if len(a.Routes)+len(a.Mounts)+len(a.XMounts) == 0 {
				fatalf("%s: rr:api has neither routes nor api-typed fields", a.Name)
			}
		} else if len(a.Routes) == 0 {
			fatalf("%s: has rr: directives but no routes", a.Name)
		}
	}
}

// resolve settles every @ref and every handler parameter.
func (l *loader) resolveAll() {
	for _, a := range l.pkg.APIs {
		raw := l.raw[a]
		for _, m := range raw.mw {
			a.Guards = append(a.Guards, l.guard(a, m))
		}
		if raw.onError != "" {
			a.OnError = l.ehandler(a, raw.onError, "onerror", true, true)
		}
		if raw.on404 != "" {
			a.On404 = l.ehandler(a, raw.on404, "on404", false, false)
		}
		if raw.on405 != "" {
			a.On405 = l.ehandler(a, raw.on405, "on405", false, false)
		}
		if raw.on400 != "" {
			a.On400 = l.ehandler(a, raw.on400, "on400", false, true)
		}
		for _, rt := range a.Routes {
			l.resolveRoute(a, rt)
		}
	}
}

// prefixes stamps the literal path prefix every dispatcher owns, by the rule
// its generated code cuts by: the common prefix of everything it serves,
// widened by the prefixes of what it mounts cross-package.
func (l *loader) prefixes() {
	for _, a := range l.pkg.APIs {
		if a.Central || a.Mounted {
			continue
		}
		a.Prefix = CommonPrefix(patternsOf(a.Routes))
	}
	for _, a := range l.pkg.APIs {
		if !a.Central {
			continue
		}
		if a.Recv == "" {
			a.Recv = "a"
		}
		var merged []string
		for rt := range a.Served() {
			merged = append(merged, rt.Pattern)
		}
		var cands []string
		if len(merged) > 0 {
			cands = append(cands, CommonPrefix(merged))
		}
		for _, xm := range a.XMounts {
			cands = append(cands, xm.Prefix)
		}
		a.Prefix = CommonPathPrefix(cands)
	}
}

func patternsOf(routes []*Route) []string {
	pats := make([]string, len(routes))
	for i, rt := range routes {
		pats[i] = rt.Pattern
	}
	return pats
}

// linkXMounts points each cross-package mount at its api when the set holds
// that package too. The prefix stands on its own either way.
func linkXMounts(set *Set) {
	for _, p := range set.Packages {
		for _, a := range p.APIs {
			for i := range a.XMounts {
				a.XMounts[i].API = set.API(a.XMounts[i].Pkg, a.XMounts[i].Type)
			}
		}
	}
}
