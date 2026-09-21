// Package methodgate compares a hoistable GET subtree with a mixed-method twin.
package methodgate

//go:generate go run ../../internal/genmethodgate -count 100 -out routes.go
//go:generate go run github.com/sirkostya009/rr/cmd routes.go
