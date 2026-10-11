// Package pre has a central guard binding the same shapes its routes bind.
// Top-scope temporaries are shared by every binding of a request, so a value
// the guard parsed must not show through where a route's is absent.
package pre

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
)

//go:generate go run ../../.. $GOFILE

func atoi(s string) (int, error) { return strconv.Atoi(s) }

// guarded records what the guard saw, so a test can tell the two apart
var guarded string

func guard( /* rr:header X-A=@atoi */ n int) bool {
	guarded = fmt.Sprint(n)
	return true
}

func failed(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

//rr:api onerror=@failed
//rr:pre @guard
type Api struct{}

//rr:route GET /pre/values
func (a *Api) Values( /* rr:header X-B=@atoi */ n int) string { return fmt.Sprint(n) }

type Foo struct{ S string }

func parseFoo(s string) (*Foo, error) {
	if s == "" {
		return nil, errors.New("empty")
	}
	return &Foo{S: s}, nil
}

// one transformer, bound as different types: its slot takes parseFoo's result type

//rr:route GET /pre/foo/{x=@parseFoo}/any
func (a *Api) FooAny(x any) string { return "any:" + x.(*Foo).S }

//rr:route GET /pre/foo/{x=@parseFoo}/ptr
func (a *Api) FooPtr(x *Foo) string { return "ptr:" + x.S }
