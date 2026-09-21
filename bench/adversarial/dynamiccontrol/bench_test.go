package dynamiccontrol

import (
	"strconv"
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

func dynamicHits() []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/api/v1/users/by-id/customer-" + strconv.Itoa(i)}
	}
	return hits
}

var dynamic10K = dynamicHits()

func TestRoutingParity(t *testing.T)       { stress.VerifyScale(t, &API{}, Routes) }
func BenchmarkDynamicOnly10K(b *testing.B) { stress.Run(b, &API{}, Routes, dynamic10K) }
