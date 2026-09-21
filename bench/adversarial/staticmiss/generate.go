// Package staticmiss measures the failed static-switch cost paid by dynamic hits.
package staticmiss

//go:generate go run ../../internal/genroutes -kind static -count 1000 -package staticmiss -out routes.go
//go:generate go run github.com/sirkostya009/rr/cmd routes.go
