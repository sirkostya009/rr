// Package dynamiccontrol isolates the dynamic route used by staticmiss so the
// cost of its failed 1,000-case static switch can be separated from PathValue.
package dynamiccontrol

import "github.com/sirkostya009/rr/bench/internal/stress"

//go:generate go run github.com/sirkostya009/rr/cmd $GOFILE

type API struct{}

var Routes = []stress.Route{{Pattern: "/api/v1/users/by-id/{userId}", Param: "userId"}}

//rr:route GET /api/v1/users/by-id/{userId}
func (a *API) DynamicUser() {}
