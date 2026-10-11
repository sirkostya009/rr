// Package api is a route surface wide enough to push rr past hashSwitchMin:
// every literal set here dispatches through the generated hash, not a string
// switch. routes.go is written by ./gen.
package api

import "net/http"

//go:generate go run ./gen
//go:generate go run ../../.. $GOFILE

//rr:api onerror=@failed
type Api struct{}

func failed(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}
