package api

import (
	"errors"
	"net/url"
)

type SearchApi struct{}

type SearchQuery struct {
	Term string   `json:"term"`
	Tags []string `json:"tags,omitempty"`
}

func parseSearch(q url.Values) (SearchQuery, error) {
	if q.Get("q") == "" {
		return SearchQuery{}, errors.New("missing q")
	}
	return SearchQuery{Term: q.Get("q"), Tags: q["tag"]}, nil
}

//api:route GET /api/search -- the whole query goes through a custom parser
func (sa *SearchApi) Search(
	/* api:query @parseSearch */ sq SearchQuery,
	/* api:header X-Request-Id */ rid string,
) map[string]any {
	return map[string]any{"query": sq, "rid": rid}
}

//api:route GET /api/debug/query -- the raw query, bound as a plain map
func (sa *SearchApi) DebugQuery( /* api:query */ q url.Values) map[string][]string {
	return q
}

// Three routes on GET /api/echo/{...}, dispatched purely by param type.
// The generator ranks numerics before bools (so ParseBool's lax "1"/"0"
// never claim a digit) and the catch-any string always goes last.

//api:route GET /api/echo/{f}
func (sa *SearchApi) EchoFloat(f float64) map[string]any {
	return map[string]any{"float": f}
}

//api:route GET /api/echo/{b}
func (sa *SearchApi) EchoBool(b bool) map[string]any {
	return map[string]any{"bool": b}
}

//api:route GET /api/echo/{str}
func (sa *SearchApi) EchoString(str string) map[string]any {
	return map[string]any{"string": str}
}
