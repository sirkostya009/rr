package api

import (
	"net/http"

	"github.com/getsentry/sentry-go"
)

type FilesApi struct{}

//rr:route GET /api/files/{path...} -- trailing wildcard, read via PathValue
func (fa *FilesApi) StatFile(r *http.Request) map[string]string {
	// r carries the Server's transaction context: child spans just work
	span := sentry.StartSpan(r.Context(), "files.stat")
	defer span.Finish()
	return map[string]string{"path": r.PathValue("path")}
}
