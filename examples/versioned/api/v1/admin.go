package v1

import (
	"net/http"

	"versioned/services"
)

// The middleware guards every admin route. Middleware bind params like
// handlers do and return bool (false = handled, stop) or error (goes to
// the error handler in scope).
//
//rr:pre @requireToken
type AdminApi struct{}

func requireToken(w http.ResponseWriter /* rr:header Authorization */, auth string) bool {
	if auth != "Bearer letmein" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

//rr:route GET /api/v1/admin/stats
func (ad *AdminApi) Stats() map[string]int {
	return map[string]int{"users": services.Users.Count(), "posts": services.Posts.Count()}
}

//rr:route /api/v1/admin/maintenance -- method catch-all
func (ad *AdminApi) Maintenance(w http.ResponseWriter) {
	w.WriteHeader(http.StatusAccepted)
}
