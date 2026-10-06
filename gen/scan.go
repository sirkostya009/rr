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

// fieldCand is an api-typed struct field naming a type in the same package.
type fieldCand struct {
	field, typeName string
}

// xFieldCand is an api-typed field whose type lives in another package
// (`v1.API`); its routes can't be merged, only delegated by prefix.
type xFieldCand struct {
	field, pkgIdent, typeName string
}

// pkgScan is the slice of another package the parser needs to read: route
// patterns per receiver type, api-typed fields per struct, the imports to
// resolve cross-package field types against, and package-level var types (for
// finding a shared helpers package's pools).
type pkgScan struct {
	name    string
	dir     string
	routes  map[string][]string
	fields  map[string][]fieldCand
	xfields map[string][]xFieldCand
	imports map[string]string
	vars    map[string]string
	methods map[string]map[string]struct{}
	ggen    map[string]struct{} // types carrying //ggen:generate
}

var (
	pkgDirs    = map[string]string{}
	pkgStd     = map[string]struct{}{}
	pkgScans   = map[string]*pkgScan{}
	prefixMemo = map[string]string{}
)

// PkgInfo is a source-only read of a package other than the one being
// generated: no type checking, no dependency on that package having been
// generated yet.
type PkgInfo struct {
	Path string
	Name string
	Dir  string
	Std  bool // a standard library package

	Methods   map[string]map[string]struct{} // receiver type name -> method names
	GgenTypes map[string]struct{}            // types carrying //ggen:generate
	Vars      map[string]string              // package-level var name -> declared type, or its initializer
}

// Scan reads a package's sources for what a generator needs to know about it
// without loading it: the methods its types declare, the types promising ggen
// codecs, and its package-level vars. Results are memoized per import path.
func Scan(pkgPath string) (info *PkgInfo, err error) {
	defer catch(&err)
	s := scanPkg(pkgPath)
	_, std := pkgStd[pkgPath]
	return &PkgInfo{
		Path:      pkgPath,
		Name:      s.name,
		Dir:       s.dir,
		Std:       std,
		Methods:   s.methods,
		GgenTypes: s.ggen,
		Vars:      s.vars,
	}, nil
}

// Prefix returns the path prefix pkgPath's typeName dispatcher owns, derived
// from that package's sources. Empty means the type owns no rr: routes.
func Prefix(pkgPath, typeName string) (prefix string, err error) {
	defer catch(&err)
	return discoverPrefix(pkgPath, typeName), nil
}

func pkgDir(pkgPath string) string {
	if d, ok := pkgDirs[pkgPath]; ok {
		return d
	}
	// -e: the package need not typecheck, and it usually doesn't yet — its own
	// generated file may be missing or stale while we're discovering it
	out, err := exec.CommandContext(context.Background(), "go", "list", "-e", "-f", "{{.Standard}} {{.Dir}}", pkgPath).Output() //nolint:gosec // pkgPath is from parsed imports
	if err != nil {
		fatalf("go list %s: %v", pkgPath, err)
	}
	std, d, _ := strings.Cut(strings.TrimSpace(string(out)), " ")
	if d == "" {
		fatalf("go list %s: package not found", pkgPath)
	}
	if std == "true" {
		pkgStd[pkgPath] = struct{}{}
	}
	pkgDirs[pkgPath] = d
	return d
}

// scanPkg reads directives straight out of a package's sources. It never looks
// at generated output, so discovery does not depend on generation order.
func scanPkg(pkgPath string) *pkgScan {
	if s, ok := pkgScans[pkgPath]; ok {
		return s
	}
	dir := pkgDir(pkgPath)
	s := &pkgScan{
		dir:     dir,
		routes:  map[string][]string{},
		fields:  map[string][]fieldCand{},
		xfields: map[string][]xFieldCand{},
		imports: map[string]string{},
		vars:    map[string]string{},
		methods: map[string]map[string]struct{}{},
		ggen:    map[string]struct{}{},
	}
	pkgScans[pkgPath] = s
	if _, std := pkgStd[pkgPath]; std {
		return s // nothing in the standard library carries rr: or ggen: directives
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		fatalf("%v", err)
	}
	fset := token.NewFileSet()
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		f, err := parser.ParseFile(fset, filepath.Join(dir, name), nil, parser.ParseComments)
		if err != nil || isGenerated(f) {
			continue
		}
		s.name = f.Name.Name
		for _, im := range f.Imports {
			p, _ := strconv.Unquote(im.Path.Value)
			id := path.Base(p)
			if im.Name != nil {
				id = im.Name.Name
			}
			s.imports[id] = p
		}
		for _, decl := range f.Decls {
			switch d := decl.(type) {
			case *ast.GenDecl:
				for _, spec := range d.Specs {
					if vs, ok := spec.(*ast.ValueSpec); ok && d.Tok == token.VAR {
						for i, n := range vs.Names {
							switch {
							case vs.Type != nil:
								s.vars[n.Name] = renderType(fset, vs.Type)
							case i < len(vs.Values):
								s.vars[n.Name] = renderType(fset, vs.Values[i])
							}
						}
						continue
					}
					ts, ok := spec.(*ast.TypeSpec)
					if !ok {
						continue
					}
					if hasGgenDirective(ts.Doc) || len(d.Specs) == 1 && hasGgenDirective(d.Doc) {
						s.ggen[ts.Name.Name] = struct{}{}
					}
					st, ok := ts.Type.(*ast.StructType)
					if !ok {
						continue
					}
					for _, fld := range st.Fields.List {
						fc, xc, ok := apiField(fld)
						if !ok {
							continue
						}
						if xc.field != "" {
							s.xfields[ts.Name.Name] = append(s.xfields[ts.Name.Name], xc)
						} else {
							s.fields[ts.Name.Name] = append(s.fields[ts.Name.Name], fc)
						}
					}
				}
			case *ast.FuncDecl:
				if d.Recv == nil {
					continue
				}
				tname, _ := recvTypeName(d.Recv)
				if tname == "" {
					continue
				}
				if s.methods[tname] == nil {
					s.methods[tname] = map[string]struct{}{}
				}
				s.methods[tname][d.Name.Name] = struct{}{}
				for _, dir := range parseDirectives(d.Doc) {
					if dir[0] != "route" {
						continue
					}
					if fs := strings.Fields(dir[1]); len(fs) == 1 {
						s.routes[tname] = append(s.routes[tname], fs[0])
					} else if len(fs) == 2 {
						s.routes[tname] = append(s.routes[tname], fs[1])
					}
				}
			}
		}
	}
	return s
}

// discoverPrefix computes the path prefix pkgPath's typeName dispatcher owns,
// by the same rule its own generation uses: the common prefix of every route it
// merges, widened by the prefixes of anything it mounts cross-package. Reading
// sources (not the sub-package's generated marker) keeps `go generate ./...`
// order-independent. Empty means the type owns no routes.
func discoverPrefix(pkgPath, typeName string) string {
	key := pkgPath + "." + typeName
	if p, ok := prefixMemo[key]; ok {
		return p
	}
	prefixMemo[key] = "" // depth guard; import cycles can't happen, self-mounts can
	s := scanPkg(pkgPath)
	// own routes plus the routes of every in-package api it merges: one level,
	// matching what the dispatcher itself flattens
	merged := append([]string{}, s.routes[typeName]...)
	for _, fc := range s.fields[typeName] {
		if fc.typeName != typeName {
			merged = append(merged, s.routes[fc.typeName]...)
		}
	}
	var cands []string
	if len(merged) > 0 {
		cands = append(cands, CommonPrefix(merged))
	}
	for _, xc := range s.xfields[typeName] {
		sub, ok := s.imports[xc.pkgIdent]
		if !ok {
			fatalf("%s.%s: field %s: package %s is not imported", pkgPath, typeName, xc.field, xc.pkgIdent)
		}
		if p := discoverPrefix(sub, xc.typeName); p != "" {
			cands = append(cands, p)
		}
	}
	p := CommonPathPrefix(cands)
	prefixMemo[key] = p
	return p
}

// apiField splits a struct field into an in-package or cross-package api
// candidate. ok=false for anything that can't name an api type.
func apiField(fld *ast.Field) (fieldCand, xFieldCand, bool) {
	t := fld.Type
	if s, ok := t.(*ast.StarExpr); ok {
		t = s.X
	}
	switch v := t.(type) {
	case *ast.Ident:
		if len(fld.Names) == 0 {
			return fieldCand{v.Name, v.Name}, xFieldCand{}, true
		}
		return fieldCand{fld.Names[0].Name, v.Name}, xFieldCand{}, true
	case *ast.SelectorExpr:
		pid, ok := v.X.(*ast.Ident)
		if !ok {
			return fieldCand{}, xFieldCand{}, false
		}
		if len(fld.Names) == 0 {
			return fieldCand{}, xFieldCand{v.Sel.Name, pid.Name, v.Sel.Name}, true
		}
		return fieldCand{}, xFieldCand{fld.Names[0].Name, pid.Name, v.Sel.Name}, true
	}
	return fieldCand{}, xFieldCand{}, false
}

func hasGgenDirective(doc *ast.CommentGroup) bool {
	if doc == nil {
		return false
	}
	for _, c := range doc.List {
		if c.Text == "//ggen:generate" || strings.HasPrefix(c.Text, "//ggen:generate ") {
			return true
		}
	}
	return false
}

// isGenerated reports whether f is rr's own output. Such a file is never
// parsed: a stale or missing one must not change what the parser sees.
func isGenerated(f *ast.File) bool {
	for _, cg := range f.Comments {
		for _, c := range cg.List {
			if strings.HasPrefix(c.Text, "// Code generated by rr.") {
				return true
			}
		}
	}
	return false
}

// CommonPathPrefix is the longest prefix every string shares, cut back to the
// last slash so it lands on a path boundary.
func CommonPathPrefix(strs []string) string {
	if len(strs) == 0 {
		return ""
	}
	p := strs[0]
	for _, s := range strs[1:] {
		n := 0
		for n < len(p) && n < len(s) && p[n] == s[n] {
			n++
		}
		p = p[:n]
	}
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		p = p[:i+1]
	}
	return p
}

// CommonPrefix is the literal path prefix every route pattern shares: each
// pattern is cut at its first wildcard, then the shared head is taken. Prefix
// discovery runs the same rule over another package's routes, so a parent
// always agrees with the child.
func CommonPrefix(patterns []string) string {
	if len(patterns) == 0 {
		return ""
	}
	lit := func(p string) string {
		if i := strings.IndexAny(p, "{*"); i >= 0 {
			p = p[:i]
		}
		return p
	}
	pfx := lit(patterns[0])
	for _, p := range patterns[1:] {
		l := lit(p)
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
