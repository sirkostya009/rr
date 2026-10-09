package gen_test

import (
	"net/http"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/sirkostya009/rr/gen"
)

const (
	servicesPkg = "versioned/services"
	centralAPI  = "Api" // the type every example calls its dispatcher root

	simple = "../examples/simple/api"
	v1     = "../examples/versioned/api/v1"
	large  = "../examples/large/api"
)

func load(t *testing.T, patterns ...string) *gen.Set {
	t.Helper()
	set, err := gen.Load(patterns...)
	if err != nil {
		t.Fatalf("Load(%v): %v", patterns, err)
	}
	return set
}

func api(t *testing.T, set *gen.Set, name string) *gen.API {
	t.Helper()
	for _, p := range set.Packages {
		for _, a := range p.APIs {
			if a.Name == name {
				return a
			}
		}
	}
	t.Fatalf("no api %s in the set", name)
	return nil
}

func route(t *testing.T, a *gen.API, handler string) *gen.Route {
	t.Helper()
	for _, rt := range a.Routes {
		if rt.Handler == handler {
			return rt
		}
	}
	t.Fatalf("%s has no route %s", a.Name, handler)
	return nil
}

func TestLoadSimple(t *testing.T) {
	set := load(t, simple)
	if len(set.Packages) != 1 {
		t.Fatalf("got %d packages, want 1", len(set.Packages))
	}
	p := set.Packages[0]
	if p.Path != "simple/api" || p.Name != "api" {
		t.Errorf("package = %q %q, want simple/api api", p.Path, p.Name)
	}

	central := api(t, set, centralAPI)
	if !central.Central || central.Mounted {
		t.Error("Api should be a central, not mounted")
	}
	if central.Prefix != "/api/" {
		t.Errorf("Api prefix = %q, want /api/", central.Prefix)
	}
	if central.OnError == nil || central.On400 == nil || central.On404 == nil || central.On405 == nil {
		t.Error("Api should carry all four error handlers")
	}
	if len(central.Mounts) != 6 {
		t.Errorf("Api mounts = %d, want 6", len(central.Mounts))
	}

	users := api(t, set, "UsersApi")
	if !users.Mounted {
		t.Error("UsersApi should be mounted")
	}

	get := route(t, users, "GetUser")
	if get.Method != http.MethodGet || get.Path != "/api/users/{i}" {
		t.Errorf("GetUser = %q %q, want GET /api/users/{i}", get.Method, get.Path)
	}
	if get.Static() {
		t.Error("GetUser is not a static route")
	}
	if get.Result == nil || get.Result.Name != "User" || get.Result.Pkg != "simple/api" || !get.Err {
		t.Errorf("GetUser result = %+v, err=%v", get.Result, get.Err)
	}
	params := slices.Collect(get.Params())
	if len(params) != 1 || params[0].Name != "i" || params[0].Type.Expr != "int" {
		t.Errorf("GetUser params = %+v", params)
	}
	if params[0].Arg == nil || params[0].Arg.Kind != gen.ArgPath {
		t.Error("the {i} param should be bound to a handler arg")
	}

	// a single header value, renamed by its annotation and transformed
	me := route(t, users, "GetMe")
	hdrs := slices.Collect(me.Headers())
	if len(hdrs) != 1 {
		t.Fatalf("GetMe headers = %d, want 1", len(hdrs))
	}
	h := hdrs[0]
	if h.Key != "X-User-Index" || h.Name != "i" || h.Shape != gen.BindValue {
		t.Errorf("GetMe header = %+v", h)
	}
	if h.Check == nil || h.Check.Name != "atoi" || !h.Check.Transform {
		t.Errorf("GetMe header check = %+v", h.Check)
	}

	// a struct bound to the whole query, keys from the `query` tag
	posts := api(t, set, "PostsApi")
	list := route(t, posts, "ListPosts")
	q := slices.Collect(list.Query())
	if len(q) != 1 || !q[0].Whole || q[0].Shape != gen.BindStruct {
		t.Fatalf("ListPosts query = %+v", q)
	}
	want := []gen.Field{
		{Name: "Page", Key: "page", Kind: gen.FieldInt},
		{Name: "Tag", Key: "tag", Kind: gen.FieldString},
	}
	if !slices.Equal(q[0].Fields, want) {
		t.Errorf("ListPosts query fields = %+v, want %+v", q[0].Fields, want)
	}

	// bodies: JSON pointer, urlencoded, multipart
	if b := route(t, posts, "CreatePost").Body(); b == nil || b.Body != gen.BodyJSON || !b.Type.Ptr || b.Type.Name != "Post" {
		t.Errorf("CreatePost body = %+v", b)
	}
	forms := api(t, set, "FormsApi")
	if b := route(t, forms, "Login").Body(); b == nil || b.Body != gen.BodyForm {
		t.Errorf("Login body = %+v", b)
	}
	if b := route(t, forms, "Upload").Body(); b == nil || b.Body != gen.BodyMultipart {
		t.Errorf("Upload body = %+v", b)
	}

	// a catch-all binds the raw rest of the path
	files := api(t, set, "FilesApi")
	stat := route(t, files, "StatFile")
	segs := stat.Segments
	last := segs[len(segs)-1]
	if last.Kind != gen.SegRest || last.Param == nil || !last.Param.Rest || last.Param.Name != "path" {
		t.Errorf("StatFile last segment = %+v", last)
	}
	if stat.Path != "/api/files/{path...}" {
		t.Errorf("StatFile path = %q", stat.Path)
	}

	// a guard chain, with its own parameter binding
	admin := api(t, set, "AdminApi")
	if len(admin.Guards) != 1 || admin.Guards[0].Ref.Name != "requireToken" || admin.Guards[0].RetErr {
		t.Fatalf("AdminApi guards = %+v", admin.Guards)
	}
	if got := admin.Guards[0].Args; len(got) != 2 || got[1].Kind != gen.ArgHeader || got[1].Key != "Authorization" {
		t.Errorf("requireToken args = %+v", got)
	}

	// a param of a named type embedding http.ResponseWriter is the writer
	if got := route(t, admin, "Probe").Args; len(got) != 1 || got[0].Kind != gen.ArgWriter || got[0].Type.Name != "statusWriter" {
		t.Errorf("Probe args = %+v", got)
	}
}

// TestSetRoutes checks the whole-set iterator yields every route once.
func TestSetRoutes(t *testing.T) {
	set := load(t, simple)
	n := 0
	seen := map[*gen.Route]bool{}
	for rt := range set.Routes() {
		if seen[rt] {
			t.Fatalf("route %s yielded twice", rt.Handler)
		}
		seen[rt] = true
		n++
	}
	total := 0
	for _, p := range set.Packages {
		for _, a := range p.APIs {
			total += len(a.Routes)
		}
	}
	if n != total {
		t.Errorf("Routes() yielded %d, want %d", n, total)
	}
}

func TestVersionedComposition(t *testing.T) {
	set := load(t, "../examples/versioned/...")

	root := set.API("versioned/api", centralAPI)
	if root == nil {
		t.Fatal("no versioned/api.Api")
	}
	if len(root.Routes) != 0 || len(root.Mounts) != 0 {
		t.Error("the composition root owns no routes of its own")
	}
	if root.Prefix != "/api/" {
		t.Errorf("root prefix = %q, want /api/", root.Prefix)
	}
	if len(root.XMounts) != 2 {
		t.Fatalf("root xmounts = %d, want 2", len(root.XMounts))
	}
	for i, want := range []struct{ field, pkg, prefix string }{
		{"V1", "versioned/api/v1", "/api/v1/"},
		{"V2", "versioned/api/v2", "/api/v2/"},
	} {
		got := root.XMounts[i]
		if got.Field != want.field || got.Pkg != want.pkg || got.Prefix != want.prefix {
			t.Errorf("xmount %d = %+v, want %+v", i, got, want)
		}
		if got.API == nil {
			t.Errorf("xmount %s should link to its loaded api", got.Field)
		}
	}

	// v2 binds body and result types from another package
	v2users := set.API("versioned/api/v2", "UsersApi")
	if v2users == nil {
		t.Fatal("no v2.UsersApi")
	}
	if b := route(t, v2users, "PostUser").Body(); b == nil || b.Type.Pkg != servicesPkg || b.Type.Name != "NewUser" || b.Type.Slice {
		t.Errorf("v2 PostUser body = %+v", b)
	}
	if b := route(t, v2users, "PostUsers").Body(); b == nil || b.Type.Pkg != servicesPkg || !b.Type.Slice {
		t.Errorf("v2 PostUsers body = %+v", b)
	}
	if b := route(t, v2users, "StreamUsers").Body(); b == nil || b.Type.Pkg != servicesPkg || b.Type.Name != "NewUser" || !b.Type.Seq || b.Type.Slice {
		t.Errorf("v2 StreamUsers body = %+v", b)
	}
	v2admin := set.API("versioned/api/v2", "AdminApi")
	if v2admin == nil {
		t.Fatal("no v2.AdminApi")
	}
	if r := route(t, v2admin, "Stats").Result; r == nil || r.Pkg != servicesPkg || r.Name != "Stats" {
		t.Errorf("v2 Stats result = %+v", r)
	}
}

// TestPrefixDiscovery reads a sub-package's prefix without loading it, the way
// a parent dispatcher does.
func TestPrefixDiscovery(t *testing.T) {
	got, err := gen.Prefix("versioned/api/v1", centralAPI)
	if err != nil {
		t.Fatal(err)
	}
	if got != "/api/v1/" {
		t.Errorf("prefix = %q, want /api/v1/", got)
	}
}

func TestLarge(t *testing.T) {
	set := load(t, large)
	a := api(t, set, centralAPI)
	if len(a.Routes) < 800 {
		t.Errorf("large api has %d routes, want the full fixture", len(a.Routes))
	}
	if a.Prefix != "/large/" {
		t.Errorf("large prefix = %q, want /large/", a.Prefix)
	}
	for _, rt := range a.Routes {
		if rt.Path == "" || !strings.HasPrefix(rt.Path, "/") {
			t.Fatalf("route %s has path %q", rt.Handler, rt.Path)
		}
	}
}

var patternStamp = regexp.MustCompile(`r\.Pattern = ("(?:[^"\\]|\\.)*")`)

// TestServedMatchesDispatcher pins the model against the generated code: the
// dispatcher stamps r.Pattern once per route it answers, so the patterns in
// the output must be exactly the ones Served() reports.
func TestServedMatchesDispatcher(t *testing.T) {
	for _, tc := range []struct{ dir, api, out string }{
		{simple, centralAPI, simple + "/api_gen.go"},
		{v1, centralAPI, v1 + "/api_gen.go"},
		{large, centralAPI, large + "/api_gen.go"},
	} {
		t.Run(tc.dir, func(t *testing.T) {
			set := load(t, tc.dir)
			a := api(t, set, tc.api)

			var want []string
			for rt := range a.Served() {
				p := rt.Path
				if rt.Method != "" {
					p = rt.Method + " " + p
				}
				want = append(want, p)
			}
			slices.Sort(want)

			src, err := os.ReadFile(tc.out)
			if err != nil {
				t.Skipf("no generated output to compare against: %v", err)
			}
			stamps := patternStamp.FindAllStringSubmatch(string(src), -1)
			got := make([]string, 0, len(stamps))
			for _, m := range stamps {
				s, err := strconv.Unquote(m[1])
				if err != nil {
					t.Fatalf("bad pattern literal %s: %v", m[1], err)
				}
				got = append(got, s)
			}
			slices.Sort(got)

			if !slices.Equal(got, want) {
				t.Errorf("dispatcher stamps %d patterns, model reports %d\nonly in output: %v\nonly in model: %v",
					len(got), len(want), missing(got, want), missing(want, got))
			}
		})
	}
}

func missing(from, in []string) []string {
	var out []string
	for _, s := range from {
		if !slices.Contains(in, s) {
			out = append(out, s)
		}
	}
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func TestLoadNothing(t *testing.T) {
	if _, err := gen.Load("../examples/simple/config"); err == nil {
		t.Error("a package with no rr: directives should not load")
	}
}
