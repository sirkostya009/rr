package services

// User is the domain record: uuid-keyed, position-independent. v1 still
// addresses users by slice index, v2 by ID.
type User struct {
	ID   string
	Name string
}

type UserService struct {
	list []User
}

// Users is the shared instance both api versions serve from.
var Users = &UserService{list: []User{
	{ID: "5e0cf5ae-a2fd-4b71-96ea-1f2ee89f2b3a", Name: "Guy"},
}}

func (s *UserService) All() []User {
	return s.list
}

func (s *UserService) Create(name string) User {
	u := User{ID: NewID(), Name: name}
	s.list = append(s.list, u)
	return u
}

func (s *UserService) ByIndex(i int) (User, error) {
	if i < 0 || i >= len(s.list) {
		return User{}, ErrNotFound
	}
	return s.list[i], nil
}

func (s *UserService) ByID(id string) (User, error) {
	for _, u := range s.list {
		if u.ID == id {
			return u, nil
		}
	}
	return User{}, ErrNotFound
}

func (s *UserService) DeleteIndex(i int) error {
	if i < 0 || i >= len(s.list) {
		return ErrNotFound
	}
	s.list = append(s.list[:i], s.list[i+1:]...)
	return nil
}

func (s *UserService) DeleteByID(id string) error {
	for i, u := range s.list {
		if u.ID == id {
			s.list = append(s.list[:i], s.list[i+1:]...)
			return nil
		}
	}
	return ErrNotFound
}

func (s *UserService) Count() int {
	return len(s.list)
}
