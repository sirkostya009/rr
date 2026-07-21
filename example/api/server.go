package api

import (
	"example/config"
	"log/slog"
	"net/http"
	"time"

	"github.com/getsentry/sentry-go"
)

// Server is API but wrapped in common logic like instrumentation: every
// request runs inside a sentry transaction whose context rides r, so any
// handler taking *http.Request can hang child spans off r.Context().
type Server struct {
	Api
	config.Config
}

// statusWriter captures what the router wrote so the span and the log line
// can report it.
type statusWriter struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (sw *statusWriter) WriteHeader(code int) {
	if !sw.wrote {
		sw.status = code
		sw.wrote = true
	}
	sw.ResponseWriter.WriteHeader(code)
}

func (sw *statusWriter) Write(b []byte) (int, error) {
	sw.wrote = true
	return sw.ResponseWriter.Write(b)
}

func (sw *statusWriter) Status() (int, bool) {
	return sw.status, sw.wrote
}

func (app *Server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	hub := sentry.GetHubFromContext(r.Context())
	if hub == nil {
		hub = sentry.CurrentHub().Clone()
	}
	hub.Scope().SetRequest(r)

	// pre-routing name; picks up trace headers from upstream callers
	span := sentry.StartTransaction(
		sentry.SetHubOnContext(r.Context(), hub),
		r.Method+" "+r.URL.Path,
		sentry.WithOpName("http.server"),
		sentry.ContinueFromRequest(r),
		sentry.WithTransactionSource(sentry.SourceURL),
	)

	r = r.WithContext(span.Context())
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}

	// one finalizer for both exits: the router stamps r.Pattern before it
	// invokes the handler, so the parameterized name is correct even when a
	// handler panics its way here
	defer func() {
		rec := recover()

		if r.Pattern != "" {
			span.Name = r.Pattern
			span.Source = sentry.SourceRoute
		}

		if rec != nil {
			hub.RecoverWithContext(span.Context(), rec)
			hub.Flush(2 * time.Second)
			span.Status = sentry.SpanStatusInternalError
			if !sw.wrote {
				http.Error(sw, "", http.StatusInternalServerError)
			}
			slog.Error("panic",
				slog.Any("err", rec),
				slog.String("pattern", r.Pattern),
				slog.String("path", r.URL.Path))
		} else {
			span.Status = sentry.HTTPtoSpanStatus(sw.status)
		}
		span.Finish()

		if app.LogLevel == config.LogLevelInfo {
			slog.Info("request",
				slog.String("pattern", r.Pattern),
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", sw.status),
				slog.Duration("dur", time.Since(start)))
		}
	}()

	app.Api.ServeHTTP(sw, r)
}
