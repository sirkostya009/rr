// Package scale2000 benchmarks an enterprise gateway with 2,000 generated routes.
package scale2000

//go:generate go run ../../internal/genroutes -kind scale -count 2000 -package scale2000 -out routes.go
//go:generate go run github.com/sirkostya009/rr/cmd routes.go
