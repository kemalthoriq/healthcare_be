package main

import (
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/platform/health"
)

func newRouter(db health.Pinger) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready(db))

	return mux
}
