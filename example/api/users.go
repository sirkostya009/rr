package api

import "errors"

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
