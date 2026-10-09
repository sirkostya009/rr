package main

import (
	"errors"
	"fmt"
	"go/ast"
	"slices"
	"sort"
	"strings"

	rr "github.com/sirkostya009/rr/gen"
)

// loadRoutes parses the packages for their rr: routes.
func loadRoutes(patterns []string) (*rr.Set, error) {
	return rr.Load(patterns...)
}

type tsgen struct {
	set      *rr.Set
	names    *names
	typesMod string
	out      *strings.Builder
}

// renderClient walks every route and returns the whole client module.
func renderClient(set *rr.Set, nm *names, typesFile string) (string, error) {
	g := &tsgen{set: set, names: nm, typesMod: "./" + typesFile}

	var body strings.Builder
	g.out = &body
	roots := g.roots()
	if len(roots) == 0 {
		return "", errors.New("no dispatcher roots: every api is mounted or delegated to")
	}
	for _, a := range roots {
		g.group(a, 1)
	}

	var out strings.Builder
	out.WriteString(header + "\n\n")
	if imports := g.imported(); len(imports) > 0 {
		fmt.Fprintf(&out, "import type { %s } from %q;\n\n", strings.Join(imports, ", "), g.typesMod)
	}
	out.WriteString(runtime)
	out.WriteString("export function createClient(options: ClientOptions = {}) {\n")
	out.WriteString(prelude)
	out.WriteString("\n  return {\n")
	out.WriteString(body.String())
	out.WriteString("  };\n}\n\n")
	out.WriteString("export type Client = ReturnType<typeof createClient>;\n")
	return out.String(), nil
}

func (g *tsgen) imported() []string {
	out := make([]string, 0, len(g.names.used))
	for n := range g.names.used {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func (g *tsgen) roots() []*rr.API {
	mounted := map[*rr.API]bool{}
	for _, p := range g.set.Packages {
		for _, a := range p.APIs {
			for _, xm := range a.XMounts {
				if xm.API != nil {
					mounted[xm.API] = true
				}
			}
		}
	}
	var out []*rr.API
	for _, p := range g.set.Packages {
		for _, a := range p.APIs {
			if !a.Mounted && !mounted[a] {
				out = append(out, a)
			}
		}
	}
	return out
}

// group renders one api as a nested object: its own routes, the routes of
// everything it merges, then a sub-object per cross-package mount.
func (g *tsgen) group(a *rr.API, depth int) {
	pad := strings.Repeat("  ", depth+1)
	if len(a.Routes) > 0 {
		g.methods(groupName(a.Name), a.Routes, depth+1)
	}
	for _, m := range a.Mounts {
		g.methods(groupName(m.API.Name), m.API.Routes, depth+1)
	}
	for _, xm := range a.XMounts {
		if xm.API == nil {
			fmt.Fprintf(g.out, "%s// %s serves %s from another package, not loaded here\n", pad, xm.Field, xm.Prefix)
			continue
		}
		fmt.Fprintf(g.out, "%s%s: {\n", pad, lowerCamel(xm.Field))
		g.group(xm.API, depth+1)
		fmt.Fprintf(g.out, "%s},\n", pad)
	}
}

func (g *tsgen) methods(name string, routes []*rr.Route, depth int) {
	pad := strings.Repeat("  ", depth)
	fmt.Fprintf(g.out, "%s%s: {\n", pad, name)
	for _, rt := range routes {
		g.method(rt, depth+1)
	}
	fmt.Fprintf(g.out, "%s},\n", pad)
}

func (g *tsgen) method(rt *rr.Route, depth int) {
	pad := strings.Repeat("  ", depth)

	method := rt.Method
	if method == "" {
		method = "GET" // the route takes any method; override through init
	}
	fmt.Fprintf(g.out, "%s/** %s %s */\n", pad, method, rt.Path)

	var params []string
	for p := range rt.Params() {
		params = append(params, tsIdent(p.Name)+": "+g.tsType(p.Type, false))
	}
	body := rt.Body()
	if body != nil {
		params = append(params, "body: "+g.bodyType(body))
	}
	query := slices.Collect(rt.Query())
	if len(query) > 0 {
		params = append(params, "query: "+g.bagType(query, "string | string[]"))
	}
	headers := slices.Collect(rt.Headers())
	if len(headers) > 0 {
		params = append(params, "headers: "+g.bagType(headers, "string"))
	}
	params = append(params, "init?: RequestInit")

	var opts []string
	if query != nil {
		opts = append(opts, "query")
	}
	if headers != nil {
		opts = append(opts, "headers")
	}
	if body != nil {
		opts = append(opts, "body")
		if body.Body == rr.BodyJSON {
			opts = append(opts, "json: true")
		}
	}
	opts = append(opts, "init")

	call := fmt.Sprintf("send(%q, %s, { %s })", method, g.url(rt), strings.Join(opts, ", "))
	sig := fmt.Sprintf("%s(%s)", lowerCamel(rt.Handler), strings.Join(params, ", "))

	if rt.Result == nil {
		fmt.Fprintf(g.out, "%sasync %s: Promise<void> {\n%s  await %s;\n%s},\n", pad, sig, pad, call, pad)
		return
	}
	res := g.tsType(*rt.Result, false)
	fmt.Fprintf(g.out, "%s%s: Promise<%s> {\n%s  return json<%s>(%s);\n%s},\n", pad, sig, res, pad, res, call, pad)
}

// url renders the path as a template literal, interpolating each param.
func (g *tsgen) url(rt *rr.Route) string {
	var b strings.Builder
	b.WriteByte('`')
	for _, s := range rt.Segments {
		b.WriteByte('/')
		switch s.Kind {
		case rr.SegLit:
			b.WriteString(s.Lit)
		case rr.SegParam:
			fmt.Fprintf(&b, "${seg(%s)}", tsIdent(s.Param.Name))
		case rr.SegRest:
			if s.Param == nil {
				continue // a bare *: the client can only address the prefix
			}
			// the catch-all is the raw rest of the path, slashes included
			fmt.Fprintf(&b, "${%s}", tsIdent(s.Param.Name))
		}
	}
	b.WriteByte('`')
	return b.String()
}

func (g *tsgen) bodyType(a *rr.Arg) string {
	switch a.Body {
	case rr.BodyMultipart:
		return "FormData"
	case rr.BodyForm:
		return "URLSearchParams"
	}
	return g.tsType(a.Type, true)
}

// bagType renders the object a route's query or header arguments add up to.
// An opaque bind (a map, a passthrough, a custom parser) has no known keys, so
// the whole bag falls back to a record.
func (g *tsgen) bagType(args []*rr.Arg, valueType string) string {
	var fields []string
	for _, a := range args {
		switch a.Shape {
		case rr.BindStruct:
			for _, f := range a.Fields {
				fields = append(fields, fmt.Sprintf("%q?: %s", f.Key, fieldType(f.Kind)))
			}
		case rr.BindValue:
			fields = append(fields, fmt.Sprintf("%q?: %s", a.Key, g.tsType(a.Type, false)))
		default:
			return "Record<string, " + valueType + ">"
		}
	}
	if len(fields) == 0 {
		return "Record<string, " + valueType + ">"
	}
	return "{ " + strings.Join(fields, "; ") + " }"
}

func fieldType(k rr.FieldKind) string {
	switch k {
	case rr.FieldInt:
		return "number"
	case rr.FieldBool:
		return "boolean"
	}
	return "string"
}

func (g *tsgen) tsType(tr rr.TypeRef, input bool) string {
	var base string
	switch {
	case tr.Name == "" && tr.Node == nil:
		return "void"
	case scalar(tr.Name) != "":
		base = scalar(tr.Name)
	case tr.Name != "" && tr.Pkg != "":
		base = g.named(tr, input)
	case tr.Node != nil:
		base = tsUnnamed(tr.Node)
	default:
		base = "unknown"
	}
	if tr.Slice || tr.Seq { // a sequence is a json array on the wire
		base += "[]"
	}
	return base
}

// named is the TypeScript name of a referenced Go type, recorded as an
// import. A type the types module does not declare renders as unknown.
func (g *tsgen) named(tr rr.TypeRef, input bool) string {
	if name, ok := g.names.ref(tr.Pkg, tr.Name, input); ok {
		return name
	}
	return "unknown"
}

// tsUnnamed handles the types with no name of their own: maps, interfaces and
// anything the model does not resolve.
func tsUnnamed(e ast.Expr) string {
	switch t := e.(type) {
	case *ast.StarExpr:
		return tsUnnamed(t.X)
	case *ast.MapType:
		return "Record<string, " + tsUnnamed(t.Value) + ">"
	case *ast.ArrayType:
		if t.Len == nil {
			return tsUnnamed(t.Elt) + "[]"
		}
	case *ast.Ident:
		if s := scalar(t.Name); s != "" {
			return s
		}
	}
	return "unknown"
}

func scalar(name string) string {
	switch name {
	case "string":
		return "string"
	case "bool":
		return "boolean"
	case "int", "int8", "int16", "int32", "int64",
		"uint", "uint8", "uint16", "uint32", "uint64",
		"float32", "float64":
		return "number"
	case "any", "error":
		return "unknown"
	}
	return ""
}

// groupName turns an api type name into the property that holds its routes:
// UsersApi -> users, AdminApi -> admin.
func groupName(typeName string) string {
	n := strings.TrimSuffix(typeName, "Api")
	if n == "" {
		return "root"
	}
	return lowerCamel(n)
}

const runtime = `export type FetchLike = (input: string, init?: RequestInit) => Promise<Response>;

export interface ClientOptions {
  /** Prepended to every path. */
  baseUrl?: string;
  /** Defaults to the global fetch. */
  fetch?: FetchLike;
  /** Sent with every request. */
  headers?: Record<string, string>;
}

/** Thrown for any response the server did not mark ok. */
export class HttpError extends Error {
  readonly response: Response;
  readonly body: string;

  constructor(response: Response, body: string) {
    super(response.status + " " + response.statusText);
    this.name = "HttpError";
    this.response = response;
    this.body = body;
  }
}

interface SendOptions {
  query?: Record<string, unknown>;
  headers?: Record<string, unknown>;
  body?: unknown;
  json?: boolean;
  init?: RequestInit;
}

function seg(value: unknown): string {
  return encodeURIComponent(String(value));
}

function search(params: Record<string, unknown> | undefined): string {
  if (!params) return "";
  const sp = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === null) continue;
    if (Array.isArray(value)) for (const item of value) sp.append(key, String(item));
    else sp.append(key, String(value));
  }
  const s = sp.toString();
  return s ? "?" + s : "";
}

`

const prelude = `  const baseUrl = (options.baseUrl ?? "").replace(/\/+$/, "");
  const doFetch: FetchLike = options.fetch ?? globalThis.fetch.bind(globalThis);

  async function send(method: string, path: string, opts: SendOptions = {}): Promise<Response> {
    const headers = new Headers(options.headers);
    for (const [key, value] of Object.entries(opts.headers ?? {})) {
      if (value !== undefined && value !== null) headers.set(key, String(value));
    }
    let body: BodyInit | undefined;
    if (opts.body !== undefined) {
      if (opts.json) {
        headers.set("Content-Type", "application/json");
        body = JSON.stringify(opts.body);
      } else {
        body = opts.body as BodyInit;
      }
    }
    const res = await doFetch(baseUrl + path + search(opts.query), {
      method,
      headers,
      body,
      ...opts.init,
    });
    if (!res.ok) throw new HttpError(res, await res.text().catch(() => ""));
    return res;
  }

  async function json<T>(res: Promise<Response>): Promise<T> {
    return (await (await res).json()) as T;
  }
`
