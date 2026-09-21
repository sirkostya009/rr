// Package deep404 measures successful deep routing against a last-segment miss.
package deep404

import "github.com/sirkostya009/rr/bench/internal/stress"

//go:generate go run github.com/sirkostya009/rr/cmd $GOFILE

type API struct{}

var Routes = []stress.Route{{
	Pattern: "/api/v1/orgs/acme/projects/core/repos/main/branches/trunk/commits/head/artifacts/{artifactId}/detail",
	Param:   "artifactId",
}}

//rr:route GET /api/v1/orgs/acme/projects/core/repos/main/branches/trunk/commits/head/artifacts/{artifactId}/detail
func (a *API) ArtifactDetail() {}
