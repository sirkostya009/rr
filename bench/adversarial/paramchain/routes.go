// Package paramchain measures typed/checker candidate fall-through at one
// segment in a plausible catalog lookup API.
package paramchain

import (
	"regexp"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

//go:generate go run github.com/sirkostya009/rr/cmd $GOFILE

var uuidRE = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[1-5][0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)

type API struct{}

var Routes = []stress.Route{
	{Pattern: "/catalog/{value}/number", Param: "value", SkipRRPathValue: true, VerifyValue: "12345"},
	{Pattern: "/catalog/{value}/toggle", Param: "value", SkipRRPathValue: true, VerifyValue: "true"},
	{Pattern: "/catalog/{value=@uuidRE}/uuid", Param: "value", VerifyValue: "123e4567-e89b-12d3-a456-426614174000"},
	{Pattern: "/catalog/{value}/profile", Param: "value", VerifyValue: "summer-catalog"},
}

//rr:route GET /catalog/{value}/number
func (a *API) Number(value int) {}

//rr:route GET /catalog/{value}/toggle
func (a *API) Toggle(value bool) {}

//rr:route GET /catalog/{value=@uuidRE}/uuid
func (a *API) UUID(value string) {}

//rr:route GET /catalog/{value}/profile
func (a *API) Profile(value string) {}
