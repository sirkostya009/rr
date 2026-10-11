package main

import (
	"log"
	"net/http"
	"simple/api"
	"simple/config"
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
		Foo:      cfg.Foo,
		PostsApi: api.PostsApi{},
	}

	app := &api.Server{Api: a, Config: cfg}

	srv := http.Server{
		Addr:    cfg.Addr,
		Handler: app,
	}

	srv.ListenAndServe()
}
