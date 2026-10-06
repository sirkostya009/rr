package main

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

var update = flag.Bool("update", false, "rewrite the golden files")

// run generates into a temporary directory and returns both files.
func run(t *testing.T, patterns ...string) (client, types string) {
	t.Helper()
	dir := t.TempDir()
	if _, err := generate(dir, "client.ts", "types.ts", patterns); err != nil {
		t.Fatal(err)
	}
	return read(t, filepath.Join(dir, "client.ts")), read(t, filepath.Join(dir, "types.ts"))
}

func read(t *testing.T, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func golden(t *testing.T, name, got string) {
	t.Helper()
	p := filepath.Join("testdata", name)
	if *update {
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(got), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	want, err := os.ReadFile(p)
	if err != nil {
		t.Fatalf("%v (run go test ./examples/tsclient -update)", err)
	}
	if got == string(want) {
		return
	}
	gl, wl := strings.Split(got, "\n"), strings.Split(string(want), "\n")
	for i := range max(len(gl), len(wl)) {
		g, w := "", ""
		if i < len(gl) {
			g = gl[i]
		}
		if i < len(wl) {
			w = wl[i]
		}
		if g != w {
			t.Fatalf("%s differs at line %d\n got: %s\nwant: %s", name, i+1, g, w)
		}
	}
	t.Fatalf("%s differs", name)
}

// TestVersioned pins the cross-package example: two versions nested under the
// composition root, types shared through a third package, every binding shape
// rr knows.
func TestVersioned(t *testing.T) {
	client, types := run(t, "../versioned/...")
	golden(t, "versioned/client.ts", client)
	golden(t, "versioned/types.ts", types)
}

// TestSimple covers the flat single-package shape: one central, six merged
// controllers, no cross-package delegation.
func TestSimple(t *testing.T) {
	client, types := run(t, "../simple/api")
	golden(t, "simple/client.ts", client)
	golden(t, "simple/types.ts", types)
}

// TestBodiesUseInputShapes checks the two generators agree on names: a body
// takes the shape the server decodes, a response the shape it encodes.
func TestBodiesUseInputShapes(t *testing.T) {
	client, types := run(t, "../versioned/...")
	for _, want := range []string{
		"postUser(body: NewUserInput",
		"createPost(body: V1PostInput",
		"Promise<V1Post>",
		"Promise<Stats>",
	} {
		if !strings.Contains(client, want) {
			t.Errorf("client is missing %q", want)
		}
	}
	// every name the client imports has to be declared in the types module
	_, imports, ok := strings.Cut(client, "import type { ")
	if !ok {
		t.Fatal("the client imports no types")
	}
	imports, _, _ = strings.Cut(imports, " }")
	for name := range strings.SplitSeq(imports, ", ") {
		if !strings.Contains(types, "export interface "+name+" ") &&
			!strings.Contains(types, "export type "+name+" ") {
			t.Errorf("the client imports %s, which the types module does not declare", name)
		}
	}
}

// TestUnknownPackage reports rather than panics.
func TestUnknownPackage(t *testing.T) {
	if _, err := generate(t.TempDir(), "client.ts", "types.ts", []string{"../simple/config"}); err == nil {
		t.Error("a package with no rr: directives should not generate a client")
	}
}
