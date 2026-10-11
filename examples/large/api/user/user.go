// Package user is named like the types it holds, the common pkg.Pkg style:
// a slot named after user.User must not shadow the package.
package user

type User struct {
	Name string `json:"name"`
}

type Login struct {
	At string `json:"at"`
}
