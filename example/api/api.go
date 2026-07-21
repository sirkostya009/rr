package api

import "net/http"

//go:generate go run github.com/sirkostya009/ggen/cli .
//go:generate go run ../../cmd $GOFILE

//api:central response=json onerror=@handleError on405=@on405 on404=notFound on400=@badRequest
type Api struct {
	UsersApi
	PostsApi
	AdminApi
	SearchApi
	FilesApi
}

// err carries the decode/parse failure when there is one; nil means the
// request was just plain bad
func badRequest(w http.ResponseWriter, _ *http.Request, err error) {
	w.Header().Set("X-Bad-Request", "1")
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	http.Error(w, msg, http.StatusBadRequest)
}

func handleError(w http.ResponseWriter, _ *http.Request, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func on405(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "", http.StatusMethodNotAllowed)
}

func notFound(w http.ResponseWriter, _ *http.Request) {
	http.Error(w, "", http.StatusNotFound)
}
