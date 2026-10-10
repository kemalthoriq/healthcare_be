package main

import (
	"log/slog"
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/auth"
	"github.com/kemalthoriq/healthcare_be/internal/platform/health"
	"github.com/kemalthoriq/healthcare_be/internal/platform/middleware"
)

func newRouter(log *slog.Logger, db health.Pinger, authHandler *auth.Handler, frontendOrigin string) http.Handler {
	api := http.NewServeMux()
	api.HandleFunc("POST /api/v1/admin/auth/login", authHandler.Login)
	api.Handle("POST /api/v1/admin/auth/logout", authHandler.RequireAuth(http.HandlerFunc(authHandler.Logout)))
	api.Handle("GET /api/v1/admin/auth/me", authHandler.RequireAuth(http.HandlerFunc(authHandler.Me)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready(db))
	mux.Handle("/api/", middleware.SameOrigin(frontendOrigin)(api))

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger(log),
		middleware.Recover,
		middleware.SecureHeaders,
	)
}
