// Package nopv is generated with -nopathvalue: no SetPathValue anywhere, so a
// segment or rest-of-path variable is declared only where an argument reads it.
package nopv

import "strconv"

//go:generate go run ../../.. -nopathvalue $GOFILE

type Api struct{}

// a param bound by its terminal route but not by the deeper one

//rr:route GET /nopv/seg/{a}
func (n *Api) One(a string) string { return "one:" + a }

//rr:route GET /nopv/seg/{a}/b
func (n *Api) Two() string { return "two" }

// a param child with its own terminal route and a named catch-all nobody reads

//rr:route GET /nopv/rest/{b}
func (n *Api) B(b bool) string { return "b:" + strconv.FormatBool(b) }

//rr:route GET /nopv/rest/{b}/{rest...}
func (n *Api) W(b bool) string { return "w:" + strconv.FormatBool(b) }

//rr:route GET /nopv/rest/{s}/x
func (n *Api) S() string { return "s" }
