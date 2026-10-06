package main

import (
	ggen "github.com/sirkostya009/ggen/gen"
	"github.com/sirkostya009/ggen/gen/ts"
)

// loadTypes parses the same packages for their JSON type shapes. rr describes
// routes; ggen describes what is inside the types those routes carry.
func loadTypes(patterns []string) (*ggen.Set, error) {
	return ggen.Load(patterns...)
}

// writeTypes renders every JSON type of the loaded packages into one module,
// each in both directions. Placing both is also what keeps the module closed:
// a shape referencing a type that was not placed fails Write, and a request
// shape reaches request shapes.
func writeTypes(dir, file string, set *ggen.Set, nm *names) ([]string, error) {
	decls := set.Types()
	if len(decls) == 0 {
		return nil, nil
	}
	out := ggen.NewOutput(dir)
	f := out.File(file, ts.New(ts.Header(header), ts.ImportExt(".ts")))
	for _, d := range decls {
		f.Place(d.Out(), nm.name(d.Pkg.Path, d.Name, false))
		f.Place(d.In(), nm.name(d.Pkg.Path, d.Name, true))
	}
	if err := out.Write(); err != nil {
		return nil, err
	}
	var issues []string
	for _, i := range out.Report().Issues {
		issues = append(issues, i.String())
	}
	return issues, nil
}
