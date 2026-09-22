package api

import (
	"regexp"
	"strconv"
)

var hexRE = regexp.MustCompile(`^[0-9a-f]{8}$`)

// Four candidates at one position, each followed by its own terminal literal:
// the generator dispatches on the literal first and runs only that
// candidate's validator.

//rr:route GET /large/catalog/{n}/number
func (a *Api) CatalogNumber(n int) string { return "number:" + strconv.Itoa(n) }

//rr:route GET /large/catalog/{h=@hexRE}/hex
func (a *Api) CatalogHex(h string) string { return "hex:" + h }

//rr:route GET /large/catalog/{on}/toggle
func (a *Api) CatalogToggle(on bool) string { return "toggle:" + strconv.FormatBool(on) }

//rr:route GET /large/catalog/{name}/profile
func (a *Api) CatalogProfile(name string) string { return "profile:" + name }

// Two candidates sharing the literal: which one matches depends on the value,
// so these must stay an ordered chain, ints first.

//rr:route GET /large/docs/{id}/edit
func (a *Api) DocEditByID(id int) string { return "edit-id:" + strconv.Itoa(id) }

//rr:route GET /large/docs/{slug}/edit
func (a *Api) DocEditBySlug(slug string) string { return "edit-slug:" + slug }
