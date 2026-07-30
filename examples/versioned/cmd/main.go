package main

import (
	"net/http"
	"versioned/api"
	v1 "versioned/api/v1"
	"versioned/config"
)

func main() {
	cfg := config.ParseConfig()

	a := &api.Api{
		V1: v1.Api{
			UsersApi: v1.UsersApi{Foo: cfg.Foo},
		},
	}

	http.ListenAndServe(cfg.Addr, a)
}
