package v2

import (
	"context"
	"net/http"

	"versioned/services"
)

//rr:pre @requireToken
type AdminApi struct{}

func requireToken(ctx context.Context, w http.ResponseWriter /* rr:header Authorization */, auth string) bool {
	if ctx.Err() != nil || auth != "Bearer letmein" {
		w.WriteHeader(http.StatusUnauthorized)
		return false
	}
	return true
}

//rr:route GET /api/v2/admin/stats
func (ad *AdminApi) Stats() services.Stats {
	return services.Stats{Users: services.Users.Count(), Posts: services.Posts.Count()}
}

//rr:route /api/v2/admin/maintenance -- method catch-all
func (ad *AdminApi) Maintenance(w http.ResponseWriter) {
	w.WriteHeader(http.StatusAccepted)
}
