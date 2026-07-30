package v2

import "net/http"

//go:generate go run github.com/sirkostya009/ggen/cli .
//go:generate go run ../../../../cmd $GOFILE

//rr:api onerror=@handleError on405=@on405 on404=@notFound on400=@badRequest
type Api struct {
	UsersApi
	PostsApi
	AdminApi
}

func badRequest(w http.ResponseWriter, err error) {
	msg := ""
	if err != nil {
		msg = err.Error()
	}
	http.Error(w, msg, http.StatusBadRequest)
}

func handleError(w http.ResponseWriter, err error) {
	http.Error(w, err.Error(), http.StatusInternalServerError)
}

func on405(w http.ResponseWriter) {
	http.Error(w, "", http.StatusMethodNotAllowed)
}

func notFound(w http.ResponseWriter) {
	http.Error(w, "", http.StatusNotFound)
}

func isUUID(s string) bool {
	if len(s) != 36 || s[8] != '-' || s[13] != '-' || s[18] != '-' || s[23] != '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		switch i {
		case 8, 13, 18, 23:
			continue
		}
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}
