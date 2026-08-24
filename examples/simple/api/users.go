package api

import (
	"errors"
	"strconv"
)

type UsersApi struct {
	Foo string
}

//ggen:generate
type User struct {
	Name string `json:"name"`
}

var users = []User{
	{Name: "Guy"},
}

//rr:route GET /api/users
func (a *UsersApi) GetUsers() []User {
	return users
}

// a param named body is the JSON request body — no annotation needed;
// /* rr:body */ forces the role on a differently-named param
//
//rr:route POST /api/users
func (a *UsersApi) PostUser( /* rr:body */ u User) []User {
	users = append(users, u)
	return users
}

// codegen picks up "i" as the {i} path param via its int type;
// a non-numeric segment never reaches the handler
//
//rr:route GET /api/users/{i}
func (a *UsersApi) GetUser(i int) (User, error) {
	if i < 0 || i >= len(users) {
		return User{}, errors.New("no such user")
	}
	return users[i], nil
}

// error-only return: the central onerror covers mounted apis
//
//rr:route DELETE /api/users/{i}
func (a *UsersApi) DeleteUser(i int) error {
	if i < 0 || i >= len(users) {
		return errors.New("no such user")
	}
	users = append(users[:i], users[i+1:]...)
	return nil
}

// a static segment always wins over a param at the same position: /api/users/me
// lands in the whole-path switch, {i} never sees "me".
// single header values are strings unless =@transform converts them.
//
//rr:route GET /api/users/me
func (a *UsersApi) GetMe( /* rr:header X-User-Index=@atoi */ i int) (User, error) {
	return a.GetUser(i)
}

func atoi(s string) (int, error) {
	return strconv.Atoi(s)
}
