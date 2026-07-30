package api

import (
	v1 "versioned/api/v1"
	v2 "versioned/api/v2"
)

//go:generate go run ../../../cmd $GOFILE

// Api composes the versioned dispatchers. Each version is its own central in
// another package, stamping the prefix it owns; the generator discovers those
// prefixes and emits a ServeHTTP that delegates by prefix. Nesting works to any
// depth — a composition root stamps its own prefix too.
//
//rr:api
type Api struct {
	V1 v1.Api
	V2 v2.Api
}
