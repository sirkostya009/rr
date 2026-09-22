// Package api is a route surface wide enough to push rr past hashSwitchMin:
// every literal set here dispatches through the generated hash, not a string
// switch. routes.go is written by ./gen.
package api

//go:generate go run ./gen
//go:generate go run ../../.. $GOFILE

type Api struct{}
