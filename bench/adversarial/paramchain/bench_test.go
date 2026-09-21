package paramchain

import (
	"strconv"
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

func firstCandidateHits() []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/catalog/" + strconv.Itoa(i+1) + "/number"}
	}
	return hits
}

func lastCandidateHits() []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/catalog/not-a-number-or-uuid-" + strconv.Itoa(i) + "/profile"}
	}
	return hits
}

var first10K, last10K = firstCandidateHits(), lastCandidateHits()

func TestRoutingParity(t *testing.T)          { stress.VerifyScale(t, &API{}, Routes) }
func BenchmarkFirstCandidate10K(b *testing.B) { stress.Run(b, &API{}, Routes, first10K) }
func BenchmarkLastCandidate10K(b *testing.B)  { stress.Run(b, &API{}, Routes, last10K) }
