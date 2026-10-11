// Package miss has an on404 taking an error: the path param conversion that
// rejected the request, nil for a (T, bool) transformer or a structural miss.
package miss

import (
	"net/http"
	"strconv"
)

//go:generate go run ../../.. $GOFILE

// missed and bad record the error each handler saw
var missed, bad error

func notFound(w http.ResponseWriter, err error) {
	missed = err
	w.WriteHeader(http.StatusNotFound)
}

func badRequest(w http.ResponseWriter, err error) {
	bad = err
	w.WriteHeader(http.StatusBadRequest)
}

func hex(s string) (int, bool) {
	n, err := strconv.ParseInt(s, 16, 64)
	return int(n), err == nil
}

//rr:api on404=@notFound on400=@badRequest
type Api struct{}

//rr:route GET /miss/items/{id}
func (a *Api) Item(id int) int { return id }

//rr:route GET /miss/deep/{id}/x
func (a *Api) Deep(id int) int { return id }

//rr:route GET /miss/hex/{n=@hex}
func (a *Api) Hex(n int) int { return n }

//rr:route GET /miss/query
func (a *Api) Query( /* rr:query n=@hex */ n int) int { return n }

// an external transformer bound as two types keeps its result scoped

//rr:route GET /miss/ext/{n=@strconv.Atoi}/int
func (a *Api) ExtInt(n int) int { return n }

//rr:route GET /miss/ext/{n=@strconv.Atoi}/any
func (a *Api) ExtAny(n any) any { return n }
