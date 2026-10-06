package main

import (
	"path"
	"strings"

	ggen "github.com/sirkostya009/ggen/gen"
)

// typeKey identifies one Go type across both generators.
type typeKey struct{ pkg, name string }

// names fixes the TypeScript name of every Go type the output mentions, so
// the client and the types module agree without passing strings around.
//
// A type is declared twice: once as the server encodes it, once as the server
// decodes it. The two shapes differ — a field the encoder omits may be one the
// decoder requires — so a request body gets its own `…Input` declaration.
type names struct {
	qualify map[string]bool  // Go type name -> more than one package declares it
	known   map[typeKey]bool // types the ggen load actually produced
	used    map[string]bool  // TypeScript names the client references
}

func newNames(set *ggen.Set) *names {
	n := &names{
		qualify: map[string]bool{},
		known:   map[typeKey]bool{},
		used:    map[string]bool{},
	}
	pkgsOf := map[string]map[string]bool{}
	for _, d := range set.Types() {
		n.known[typeKey{d.Pkg.Path, d.Name}] = true
		if pkgsOf[d.Name] == nil {
			pkgsOf[d.Name] = map[string]bool{}
		}
		pkgsOf[d.Name][d.Pkg.Path] = true
	}
	for name, pkgs := range pkgsOf {
		if len(pkgs) > 1 {
			n.qualify[name] = true
		}
	}
	return n
}

// name is the TypeScript name for a Go type in one direction.
func (n *names) name(pkgPath, typeName string, input bool) string {
	s := typeName
	if n.qualify[typeName] {
		s = title(path.Base(pkgPath)) + typeName
	}
	if input {
		s += "Input"
	}
	return s
}

// ref is name, recorded as something the client imports. It reports ok=false
// for a type the types module does not declare, which the client renders as
// unknown rather than importing a name that is not there.
func (n *names) ref(pkgPath, typeName string, input bool) (string, bool) {
	if !n.known[typeKey{pkgPath, typeName}] {
		return "", false
	}
	s := n.name(pkgPath, typeName, input)
	n.used[s] = true
	return s, true
}

func title(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}

func lowerCamel(s string) string {
	if s == "" {
		return s
	}
	return strings.ToLower(s[:1]) + s[1:]
}

// tsIdent keeps a Go param name usable as a TypeScript identifier.
func tsIdent(s string) string {
	var b strings.Builder
	for i, r := range s {
		switch {
		case r >= 'a' && r <= 'z', r >= 'A' && r <= 'Z', r == '_', r == '$':
			b.WriteRune(r)
		case r >= '0' && r <= '9' && i > 0:
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	if b.Len() == 0 {
		return "_"
	}
	return b.String()
}
