// Package scale50 benchmarks a gateway-sized surface with 50 generated routes.
package scale50

//go:generate go run ../../internal/genroutes -kind scale -count 50 -package scale50 -out routes.go
//go:generate go run github.com/sirkostya009/rr/cmd routes.go
