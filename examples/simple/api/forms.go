package api

import (
	"mime/multipart"
	"net/url"
)

type FormsApi struct{}

// a param named headers binds request headers into a struct via `header:` tags
type ReqMeta struct {
	RequestID string `header:"X-Request-Id"`
	Trace     string `header:"X-Trace"`
}

//rr:route GET /api/meta
func (f *FormsApi) Meta(headers ReqMeta) map[string]string {
	return map[string]string{"rid": headers.RequestID, "trace": headers.Trace}
}

// body url.Values → application/x-www-form-urlencoded, parsed into r.PostForm
//
//rr:route POST /api/forms/login
func (f *FormsApi) Login(body url.Values) map[string]string {
	return map[string]string{"user": body.Get("user")}
}

// body *multipart.Form → multipart/form-data, parsed into r.MultipartForm
//
//rr:route POST /api/forms/upload
func (f *FormsApi) Upload(body *multipart.Form) map[string]int {
	return map[string]int{"fields": len(body.Value), "files": len(body.File)}
}
