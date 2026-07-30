package v2

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
	ID    string   `json:"id"`
	Slug  string   `json:"slug" pipe:"@isSlug"`
	Title string   `json:"title"`
	Tags  []string `json:"tags,omitempty"`
}

func toPost(p services.Post) Post {
	return Post{ID: p.ID, Slug: p.Slug, Title: p.Title, Tags: p.Tags}
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

//rr:route GET /api/v2/posts -- ?page=2&tag=go
func (pa *PostsApi) ListPosts(query PostsQuery) []Post {
	if query.Tag == "" {
		return toPosts(services.Posts.All())
	}
	return toPosts(services.Posts.ByTag(query.Tag))
}

//rr:route POST /api/v2/posts
func (pa *PostsApi) CreatePost(body *Post) (Post, error) {
	return toPost(services.Posts.Create(body.Slug, body.Title, body.Tags)), nil
}

// Both id and slug are strings now, so both routes carry a checker; a
// lowercase uuid is slug-shaped too, so the uuid route is declared first
// and wins the segment before the slug checker sees it.
//
//rr:route GET /api/v2/posts/{id=@isUUID}
func (pa *PostsApi) GetPost(id string) (Post, error) {
	p, err := services.Posts.ByID(id)
	return toPost(p), err
}

//rr:route GET /api/v2/posts/{slug=@isSlug}
func (pa *PostsApi) GetPostBySlug(slug string) (Post, error) {
	p, err := services.Posts.BySlug(slug)
	return toPost(p), err
}

//rr:route GET /api/v2/posts/{pid=@isUUID}/comments/{cid=@isUUID} -- uuid params bind as strings
func (pa *PostsApi) GetComment(pid, cid string) map[string]string {
	return map[string]string{"post": pid, "comment": cid}
}

func postNotFound(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusNotFound)
}
