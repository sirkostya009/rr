package v1

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

//rr:route GET /api/v1/search -- whole query via a custom parser, one header value
func (sa *SearchApi) Search(
	/* rr:query @parseSearch */ sq SearchQuery,
	/* rr:header X-Request-Id */ rid string,
) map[string]any {
	return map[string]any{"query": sq, "rid": rid}
}

//rr:route GET /api/v1/debug/query -- url.Values named query is passed through
func (sa *SearchApi) DebugQuery(query url.Values) map[string][]string {
	return query
}

// Three routes on GET /api/v1/echo/{...}, dispatched purely by param type.
// The generator ranks numerics before bools (so ParseBool's lax "1"/"0"
// never claim a digit) and the catch-any string always goes last.

//rr:route GET /api/v1/echo/{f}
func (sa *SearchApi) EchoFloat(f float64) map[string]any {
	return map[string]any{"float": f}
}

//rr:route GET /api/v1/echo/{b}
func (sa *SearchApi) EchoBool(b bool) map[string]any {
	return map[string]any{"bool": b}
}

//rr:route GET /api/v1/echo/{str}
func (sa *SearchApi) EchoString(str string) map[string]any {
	return map[string]any{"string": str}
}
