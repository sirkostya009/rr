package mountchain

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

func serviceHits(service int) []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/gateway/service-" + twoDigits(service) + "/resources/resource-" + strconv.Itoa(i)}
	}
	return hits
}

func twoDigits(i int) string {
	if i < 10 {
		return "0" + strconv.Itoa(i)
	}
	return strconv.Itoa(i)
}

func missHits() []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		hits[i] = stress.Hit{Path: "/gateway/unmounted-" + strconv.Itoa(i) + "/resources/x"}
	}
	return hits
}

var first10K, last10K, misses10K = serviceHits(0), serviceHits(19), missHits()

func TestRoutingParity(t *testing.T) {
	stress.VerifyScale(t, &API{}, Routes)
	stress.VerifyStatus(t, &API{}, Routes, misses10K[:20], http.StatusNotFound)
}

func BenchmarkFirstMount10K(b *testing.B) { stress.Run(b, &API{}, Routes, first10K) }
func BenchmarkLastMount10K(b *testing.B)  { stress.Run(b, &API{}, Routes, last10K) }
func BenchmarkMountMiss10K(b *testing.B)  { stress.Run(b, &API{}, Routes, misses10K) }
