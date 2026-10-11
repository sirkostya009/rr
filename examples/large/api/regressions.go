package api

import (
	"strconv"

	"large/api/user"
)

// Shapes that broke the generator once. Each has a test in regressions_test.go.

// equal-width literals next to a terminal param: the param forces a segment
// scan, and the literal arms must slice by that scan, not by the width

//rr:route GET /large/fw/{x}
func (a *Api) FwAny(x string) string { return "x:" + x }

//rr:route GET /large/fw/abc/{y}/q
func (a *Api) FwABC(y string) string { return "abc:" + y }

//rr:route GET /large/fw/cde/{y}/q
func (a *Api) FwCDE(y string) string { return "cde:" + y }

// a body of the same type as a path param: decoding it must not overwrite the param

//rr:route POST /large/counters/{id}
func (a *Api) SetCounter(id, body int) string {
	return strconv.Itoa(id) + ":" + strconv.Itoa(body)
}

type Job struct {
	ID     int    `json:"id"`
	Status string `json:"status"`
}

// queued keeps the body pointer, as a handler handing work to a queue would
var queued *Job

// a handler keeping its *T body: its own result must not land on that body

//rr:route POST /large/jobs
func (a *Api) Enqueue(body *Job) (Job, error) {
	queued = body
	return Job{ID: body.ID + 1000, Status: "accepted"}, nil
}

type Item struct {
	Name string `json:"name"`
}

// a body decoded by value with encoding/json: its address is taken, so it
// must stay per route, or every request heap-allocates it

//rr:route POST /large/items
func (a *Api) AddItem(body Item) string { return "item:" + body.Name }

//rr:route GET /large/ping
func (a *Api) Ping() {}

// results whose slot names would shadow the user package or the receiver

//rr:route GET /large/users/me
func (a *Api) Me() (user.User, error) { return user.User{Name: "me"}, nil }

// the body's new(user.Login) comes after the hoisted declarations, so a slot
// named user would turn it into a reference to the variable

//rr:route POST /large/users/login
func (a *Api) SetLogin(body *user.Login) (user.Login, error) { return *body, nil }

type A struct {
	N int `json:"n"`
}

//rr:route GET /large/a
func (a *Api) GetA() (A, error) { return A{N: 1}, nil }

// an unnamed catch-all next to a terminal route and a param sibling: the
// catch-all reads no rest-of-path, so none may be declared for it

//rr:route GET /large/star/{p}/a
func (a *Api) StarA(p string) string { return "a:" + p }

//rr:route GET /large/star/{p}/a/*
func (a *Api) StarAny(p string) string { return "any:" + p }

//rr:route GET /large/star/{p}/{q}/b
func (a *Api) StarB(p, q string) string { return "b:" + p + "/" + q }
