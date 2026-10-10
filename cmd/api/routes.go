package main

import (
	"log/slog"
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/platform/health"
	"github.com/kemalthoriq/healthcare_be/internal/platform/middleware"
)

func newRouter(log *slog.Logger, db health.Pinger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready(db))

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger(log),
		middleware.Recover,
		middleware.SecureHeaders,
	)
}
