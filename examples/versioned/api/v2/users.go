package v2

import "versioned/services"

type UsersApi struct{}

//ggen:generate
type User struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func toUser(u services.User) User {
	return User{ID: u.ID, Name: u.Name}
}

func toUsers(us []services.User) []User {
	out := make([]User, len(us))
	for i, u := range us {
		out[i] = toUser(u)
	}
	return out
}

//rr:route GET /api/v2/users
func (a *UsersApi) GetUsers() []User {
	return toUsers(services.Users.All())
}

//rr:route POST /api/v2/users -- returns the created user, id assigned by the service
func (a *UsersApi) PostUser(body User) User {
	return toUser(services.Users.Create(body.Name))
}

// ids are opaque uuids now: a plain string param, no int route to compete with
//
//rr:route GET /api/v2/users/{id}
func (a *UsersApi) GetUser(id string) (User, error) {
	u, err := services.Users.ByID(id)
	return toUser(u), err
}

//rr:route DELETE /api/v2/users/{id}
func (a *UsersApi) DeleteUser(id string) error {
	return services.Users.DeleteByID(id)
}
