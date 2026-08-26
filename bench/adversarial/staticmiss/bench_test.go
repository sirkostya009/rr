package staticmiss

import (
	"strconv"
	"strings"
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

func index(i int) string {
	s := strconv.Itoa(i)
	return strings.Repeat("0", 4-len(s)) + s
}

func staticHits() []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/api/v1/users/reserved/action-" + index(i%RouteCount)}
	}
	return hits
}

func dynamicHits() []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/api/v1/users/by-id/customer-" + strconv.Itoa(i)}
	}
	return hits
}

var static10K, dynamic10K = staticHits(), dynamicHits()

func TestRoutingParity(t *testing.T)           { stress.VerifyScale(t, &API{}, Routes) }
func BenchmarkStaticControl10K(b *testing.B)   { stress.Run(b, &API{}, Routes, static10K) }
func BenchmarkDynamicNearMiss10K(b *testing.B) { stress.Run(b, &API{}, Routes, dynamic10K) }
