package v1

import "versioned/services"

type UsersApi struct {
	Foo string
}

//ggen:generate
type User struct {
	Name string `json:"name"`
}

func toUser(u services.User) User {
	return User{Name: u.Name}
}

func toUsers(us []services.User) []User {
	out := make([]User, len(us))
	for i, u := range us {
		out[i] = toUser(u)
	}
	return out
}

//rr:route GET /api/v1/users
func (a *UsersApi) GetUsers() []User {
	return toUsers(services.Users.All())
}

// a param named body is the JSON request body — no annotation needed;
// /* rr:body */ forces the role on a differently-named param
//
//rr:route POST /api/v1/users
func (a *UsersApi) PostUser( /* rr:body */ u User) []User {
	services.Users.Create(u.Name)
	return toUsers(services.Users.All())
}

// codegen picks up "i" as the {i} path param via its int type;
// a non-numeric segment never reaches the handler
//
//rr:route GET /api/v1/users/{i}
func (a *UsersApi) GetUser(i int) (User, error) {
	u, err := services.Users.ByIndex(i)
	return toUser(u), err
}

// v1 also takes uuids, but int indexes stay the default: the numeric route
// above claims digit segments, the checker below claims uuid-shaped ones
//
//rr:route GET /api/v1/users/{id=@isUUID}
func (a *UsersApi) GetUserByID(id string) (User, error) {
	u, err := services.Users.ByID(id)
	return toUser(u), err
}

// error-only return: the central onerror covers mounted apis
//
//rr:route DELETE /api/v1/users/{i}
func (a *UsersApi) DeleteUser(i int) error {
	return services.Users.DeleteIndex(i)
}

func isUUID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			continue
		}
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
