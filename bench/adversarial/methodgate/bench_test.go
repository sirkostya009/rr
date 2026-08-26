package methodgate

import (
	"net/http"
	"strconv"
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

func methodHits(group string) []stress.Hit {
	hits := make([]stress.Hit, 10_000)
	for i := range hits {
		route := i % 100
		method := http.MethodGet
		if group == "mixed" && route%2 == 1 {
			method = http.MethodPost
		}
		hits[i] = stress.Hit{
			Method: method,
			Path:   "/api/v1/" + group + "/resource-" + threeDigits(route) + "/item-" + strconv.Itoa(i),
		}
	}
	return hits
}

func threeDigits(i int) string {
	s := strconv.Itoa(i)
	for len(s) < 3 {
		s = "0" + s
	}
	return s
}

var uniform10K, mixed10K = methodHits("uniform"), methodHits("mixed")

func TestRoutingParity(t *testing.T)         { stress.VerifyScale(t, &API{}, Routes) }
func BenchmarkUniformMethod10K(b *testing.B) { stress.Run(b, &API{}, Routes, uniform10K) }
func BenchmarkMixedMethod10K(b *testing.B)   { stress.Run(b, &API{}, Routes, mixed10K) }
