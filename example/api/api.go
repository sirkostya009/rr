package api

import "net/http"

//go:generate go run github.com/sirkostya009/ggen/cli .
//go:generate go run ../../cmd $GOFILE

//rr:api onerror=@handleError on405=@on405 on404=@notFound on400=@badRequest
type Api struct {
	UsersApi
	PostsApi
	AdminApi
	SearchApi
	FilesApi
	FormsApi
}

// error handlers bind params like handlers do: http.ResponseWriter is the only
// hard requirement (plus error for onerror). Here 400 also pulls a header and
// echoes it back — err is nil when the request is just plain bad.
func badRequest(
	w http.ResponseWriter,
	rid /* rr:header X-Request-Id */ string,
	err error,
) {
	w.Header().Set("X-Bad-Request", "1")
	if rid != "" {
		w.Header().Set("X-Request-Id", rid)
	}
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	http.Error(w, msg, http.StatusBadRequest)
}

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

// r optional: dropped entirely here
func on405(w http.ResponseWriter) {
	http.Error(w, "", http.StatusMethodNotAllowed)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, "", http.StatusNotFound)
}
