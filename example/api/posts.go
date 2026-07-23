package api

import (
	"errors"
	"net/http"
	"slices"
)

// controller-level onerror: every PostsApi route that returns an error maps
// it to 404, overriding the central's 500
//
//rr:controller onerror=@postNotFound
type PostsApi struct{}

//ggen:generate
type Post struct {
	ID    int      `json:"id"`
	Slug  string   `json:"slug" pipe:"@isSlug"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

var posts = []Post{
	{ID: 1, Slug: "hello-world", Title: "Hello, World", Tags: []string{"intro"}},
	{ID: 2, Slug: "generics-in-anger", Title: "Generics in Anger", Tags: []string{"go", "rant"}},
}

func isSlug(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c != '-' && (c < 'a' || c > 'z') && (c < '0' || c > '9') {
			return false
		}
	}
	return len(s) != 0
}

type PostsQuery struct {
	Page int `query:"page"`
	Tag  string
}

// a param named query binds the whole query string (struct via `query:` tags)
//
//rr:route GET /api/posts -- ?page=2&tag=go
func (pa *PostsApi) ListPosts(query PostsQuery) []Post {
	if query.Tag == "" {
		return posts
	}
	out := make([]Post, 0, len(posts))
	for _, p := range posts {
		if slices.Contains(p.Tags, query.Tag) {
			out = append(out, p)
		}
	}
	return out
}

// a param named body, here a pointer; (T, error) return
//
//rr:route POST /api/posts
func (pa *PostsApi) CreatePost(body *Post) (Post, error) {
	body.ID = posts[len(posts)-1].ID + 1
	posts = append(posts, *body)
	return *body, nil
}

// Two handlers share GET /api/posts/{...}: their param types don't overlap,
// so they dispatch in declaration order — ints here, slugs below.
//
//rr:route GET /api/posts/{id}
func (pa *PostsApi) GetPost(id int) (Post, error) {
	for _, p := range posts {
		if p.ID == id {
			return p, nil
		}
	}
	return Post{}, errors.New("post not found")
}

//rr:route GET /api/posts/{slug=@isSlug} -- func checker takes what Atoi rejected
func (pa *PostsApi) GetPostBySlug(slug string) (Post, error) {
	for _, p := range posts {
		if p.Slug == slug {
			return p, nil
		}
	}
	return Post{}, errors.New("post not found")
}

//rr:route GET /api/posts/{pid}/comments/{cid} -- two auto-bound int params
func (pa *PostsApi) GetComment(pid, cid int) map[string]int {
	return map[string]int{"post": pid, "comment": cid}
}

func postNotFound(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusNotFound)
}
