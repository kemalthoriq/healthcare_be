package main

import (
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/platform/health"
)

func newRouter() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", health.Handler)

	return mux
}
