// Package mountchain measures rr's linear cross-package HasPrefix delegation.
package mountchain

//go:generate go run ../../internal/genmounts -root . -count 20
//go:generate go generate ./service00 ./service01 ./service02 ./service03 ./service04 ./service05 ./service06 ./service07 ./service08 ./service09 ./service10 ./service11 ./service12 ./service13 ./service14 ./service15 ./service16 ./service17 ./service18 ./service19
//go:generate go run github.com/sirkostya009/rr/cmd api.go
