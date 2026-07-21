package main

import (
	"example/api"
	"example/config"
	"log"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

func main() {
	cfg := config.ParseConfig()

	// an empty DSN leaves the SDK disabled: spans become cheap no-ops
	if err := sentry.Init(sentry.ClientOptions{
		Dsn:              cfg.SentryDSN,
		EnableTracing:    true,
		TracesSampleRate: 1.0,
	}); err != nil {
		log.Fatal(err)
	}
	defer sentry.Flush(2 * time.Second)

	a := api.Api{
		UsersApi: api.UsersApi{Foo: cfg.Foo},
		PostsApi: api.PostsApi{},
	}

	app := &api.Server{Api: a, Config: cfg}

	srv := http.Server{
		Addr:    cfg.Addr,
		Handler: app,
	}

	srv.ListenAndServe()
}
