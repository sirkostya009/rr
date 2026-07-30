package v1

import (
	"net/http"

	"versioned/services"
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

func toPost(p services.Post) Post {
	return Post{ID: p.Num, Slug: p.Slug, Title: p.Title, Tags: p.Tags}
}

func toPosts(ps []services.Post) []Post {
	out := make([]Post, len(ps))
	for i, p := range ps {
		out[i] = toPost(p)
	}
	return out
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
//rr:route GET /api/v1/posts -- ?page=2&tag=go
func (pa *PostsApi) ListPosts(query PostsQuery) []Post {
	if query.Tag == "" {
		return toPosts(services.Posts.All())
	}
	return toPosts(services.Posts.ByTag(query.Tag))
}

// a param named body, here a pointer; (T, error) return
//
//rr:route POST /api/v1/posts
func (pa *PostsApi) CreatePost(body *Post) (Post, error) {
	return toPost(services.Posts.Create(body.Slug, body.Title, body.Tags)), nil
}

// Two handlers share GET /api/v1/posts/{...}: their param types don't overlap,
// so they dispatch in declaration order — ints here, slugs below.
//
//rr:route GET /api/v1/posts/{id}
func (pa *PostsApi) GetPost(id int) (Post, error) {
	p, err := services.Posts.ByNum(id)
	return toPost(p), err
}

//rr:route GET /api/v1/posts/{slug=@isSlug} -- func checker takes what Atoi rejected
func (pa *PostsApi) GetPostBySlug(slug string) (Post, error) {
	p, err := services.Posts.BySlug(slug)
	return toPost(p), err
}

//rr:route GET /api/v1/posts/{pid}/comments/{cid} -- two auto-bound int params
func (pa *PostsApi) GetComment(pid, cid int) map[string]int {
	return map[string]int{"post": pid, "comment": cid}
}

func postNotFound(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusNotFound)
}
