package services

import "slices"

// Post carries both key shapes: Num is the legacy v1 integer id, ID the
// uuid v2 serves.
type Post struct {
	ID    string
	Num   int
	Slug  string
	Title string
	Tags  []string
}

type PostService struct {
	list []Post
}

// Posts is the shared instance both api versions serve from.
var Posts = &PostService{list: []Post{
	{ID: "6b2d9c7e-49a1-45b6-8a3f-0d8f4f4e9a01", Num: 1, Slug: "hello-world", Title: "Hello, World", Tags: []string{"intro"}},
	{ID: "9f1a3d52-7c88-4a0b-b1de-52f0a1c6e402", Num: 2, Slug: "generics-in-anger", Title: "Generics in Anger", Tags: []string{"go", "rant"}},
}}

func (s *PostService) All() []Post {
	return s.list
}

func (s *PostService) ByTag(tag string) []Post {
	out := make([]Post, 0, len(s.list))
	for _, p := range s.list {
		if slices.Contains(p.Tags, tag) {
			out = append(out, p)
		}
	}
	return out
}

func (s *PostService) Create(slug, title string, tags []string) Post {
	p := Post{
		ID:    NewID(),
		Num:   s.list[len(s.list)-1].Num + 1,
		Slug:  slug,
		Title: title,
		Tags:  tags,
	}
	s.list = append(s.list, p)
	return p
}

func (s *PostService) ByNum(num int) (Post, error) {
	for _, p := range s.list {
		if p.Num == num {
			return p, nil
		}
	}
	return Post{}, ErrNotFound
}

func (s *PostService) ByID(id string) (Post, error) {
	for _, p := range s.list {
		if p.ID == id {
			return p, nil
		}
	}
	return Post{}, ErrNotFound
}

func (s *PostService) BySlug(slug string) (Post, error) {
	for _, p := range s.list {
		if p.Slug == slug {
			return p, nil
		}
	}
	return Post{}, ErrNotFound
}

func (s *PostService) Count() int {
	return len(s.list)
}
