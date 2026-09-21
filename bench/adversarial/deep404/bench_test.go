package deep404

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

const prefix = "/api/v1/orgs/acme/projects/core/repos/main/branches/trunk/commits/head/artifacts/"

func deepHits(suffix string) []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: prefix + "artifact-" + strconv.Itoa(i) + suffix}
	}
	return hits
}

var valid10K, miss10K = deepHits("/detail"), deepHits("/missing")

func TestRoutingParity(t *testing.T) {
	stress.VerifyScale(t, &API{}, Routes)
	stress.VerifyStatus(t, &API{}, Routes, miss10K[:20], http.StatusNotFound)
}

func BenchmarkDeepHit10K(b *testing.B)      { stress.Run(b, &API{}, Routes, valid10K) }
func BenchmarkDeepLate404_10K(b *testing.B) { stress.Run(b, &API{}, Routes, miss10K) }
