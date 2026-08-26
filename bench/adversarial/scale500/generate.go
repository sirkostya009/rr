// Package scale500 benchmarks a large service gateway with 500 generated routes.
package scale500

//go:generate go run ../../internal/genroutes -kind scale -count 500 -package scale500 -out routes.go
//go:generate go run github.com/sirkostya009/rr/cmd routes.go
