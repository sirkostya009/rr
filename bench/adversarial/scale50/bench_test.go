package scale50

import (
	"testing"

	"github.com/sirkostya009/rr/bench/internal/stress"
)

var friendly64, uniform10K, zipf10K = stress.ScaleHits(RouteCount)

func TestRoutingParity(t *testing.T)   { stress.VerifyScale(t, &API{}, Routes) }
func BenchmarkFriendly64(b *testing.B) { stress.Run(b, &API{}, Routes, friendly64) }
func BenchmarkUniform10K(b *testing.B) { stress.Run(b, &API{}, Routes, uniform10K) }
func BenchmarkZipf10K(b *testing.B)    { stress.Run(b, &API{}, Routes, zipf10K) }
