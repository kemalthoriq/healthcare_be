package main

import (
	"log/slog"
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/auth"
	"github.com/kemalthoriq/healthcare_be/internal/platform/health"
	"github.com/kemalthoriq/healthcare_be/internal/platform/middleware"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

type routerDeps struct {
	log            *slog.Logger
	db             health.Pinger
	frontendOrigin string
	auth           *auth.Handler
	treatment      *treatment.Handler
	therapist      *therapist.Handler
}

func newRouter(d routerDeps) http.Handler {
	admin := func(h http.HandlerFunc) http.Handler {
		return d.auth.RequireAuth(auth.RequireRole(auth.RoleAdmin)(h))
	}

	api := http.NewServeMux()
	api.HandleFunc("POST /api/v1/admin/auth/login", d.auth.Login)
	api.Handle("POST /api/v1/admin/auth/logout", d.auth.RequireAuth(http.HandlerFunc(d.auth.Logout)))
	api.Handle("GET /api/v1/admin/auth/me", d.auth.RequireAuth(http.HandlerFunc(d.auth.Me)))

	api.HandleFunc("GET /api/v1/treatments", d.treatment.ListPublic)
	api.HandleFunc("GET /api/v1/treatments/{slug}", d.treatment.GetPublic)
	api.Handle("GET /api/v1/admin/treatments", admin(d.treatment.List))
	api.Handle("POST /api/v1/admin/treatments", admin(d.treatment.Create))
	api.Handle("GET /api/v1/admin/treatments/{id}", admin(d.treatment.Get))
	api.Handle("PUT /api/v1/admin/treatments/{id}", admin(d.treatment.Update))
	api.Handle("DELETE /api/v1/admin/treatments/{id}", admin(d.treatment.Delete))

	api.HandleFunc("GET /api/v1/therapists", d.therapist.ListPublic)
	api.HandleFunc("GET /api/v1/therapists/{slug}", d.therapist.GetPublic)
	api.HandleFunc("GET /api/v1/treatments/{slug}/therapists", d.therapist.ListForTreatmentPublic)
	api.Handle("GET /api/v1/admin/therapists", admin(d.therapist.List))
	api.Handle("POST /api/v1/admin/therapists", admin(d.therapist.Create))
	api.Handle("GET /api/v1/admin/therapists/{id}", admin(d.therapist.Get))
	api.Handle("PUT /api/v1/admin/therapists/{id}", admin(d.therapist.Update))
	api.Handle("DELETE /api/v1/admin/therapists/{id}", admin(d.therapist.Delete))
	api.Handle("GET /api/v1/admin/treatments/{id}/therapists", admin(d.therapist.ListForTreatment))
	api.Handle("PUT /api/v1/admin/treatments/{id}/therapists", admin(d.therapist.SetForTreatment))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready(d.db))
	mux.Handle("/api/", middleware.SameOrigin(d.frontendOrigin)(api))

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger(d.log),
		middleware.Recover,
		middleware.SecureHeaders,
	)
}
