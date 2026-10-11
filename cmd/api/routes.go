package main

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/auth"
	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/invoice"
	"github.com/kemalthoriq/healthcare_be/internal/platform/clientip"
	"github.com/kemalthoriq/healthcare_be/internal/platform/health"
	"github.com/kemalthoriq/healthcare_be/internal/platform/middleware"
	"github.com/kemalthoriq/healthcare_be/internal/platform/ratelimit"
	"github.com/kemalthoriq/healthcare_be/internal/promo"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

type rateLimits struct {
	bookingCreate *ratelimit.Limiter
	bookingCode   *ratelimit.Limiter
	login         *ratelimit.Limiter
}

func newRateLimits() rateLimits {
	return rateLimits{
		bookingCreate: ratelimit.New(5, 10*time.Minute),
		bookingCode:   ratelimit.New(30, time.Minute),
		login:         ratelimit.New(5, 15*time.Minute),
	}
}

type routerDeps struct {
	log              *slog.Logger
	db               health.Pinger
	frontendOrigin   string
	trustedProxyHops int
	limits           rateLimits
	auth             *auth.Handler
	treatment        *treatment.Handler
	therapist        *therapist.Handler
	account          *therapist.AccountHandler
	setting          *setting.Handler
	booking          *booking.Handler
	patient          *booking.PatientHandler
	invoice          *invoice.Handler
	promo            *promo.Handler
}

func newRouter(d routerDeps) http.Handler {
	admin := func(h http.HandlerFunc) http.Handler {
		return d.auth.RequireAuth(auth.RequireRole(auth.RoleAdmin)(h))
	}
	staff := func(h http.HandlerFunc) http.Handler {
		return d.auth.RequireAuth(auth.RequireRole(auth.RoleAdmin, auth.RoleTherapist)(h))
	}

	api := http.NewServeMux()
	api.Handle("POST /api/v1/admin/auth/login", d.limits.login.PerIP(http.HandlerFunc(d.auth.Login)))
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
	api.Handle("GET /api/v1/admin/therapists/{id}/schedules", admin(d.therapist.ListSchedules))
	api.Handle("PUT /api/v1/admin/therapists/{id}/schedules", admin(d.therapist.SetSchedules))
	api.Handle("GET /api/v1/admin/therapists/{id}/time-off", admin(d.therapist.ListTimeOff))
	api.Handle("POST /api/v1/admin/therapists/{id}/time-off", admin(d.therapist.CreateTimeOff))
	api.Handle("DELETE /api/v1/admin/therapists/{id}/time-off/{timeOffId}", admin(d.therapist.DeleteTimeOff))
	api.Handle("POST /api/v1/admin/therapists/{id}/account", admin(d.account.Create))

	api.HandleFunc("GET /api/v1/clinic", d.setting.GetClinicPublic)
	api.Handle("GET /api/v1/admin/settings", admin(d.setting.Get))
	api.Handle("PUT /api/v1/admin/settings", admin(d.setting.Update))
	api.Handle("GET /api/v1/admin/clinic", admin(d.setting.GetClinic))
	api.Handle("PUT /api/v1/admin/clinic", admin(d.setting.UpdateClinic))

	api.HandleFunc("GET /api/v1/availability/slots", d.booking.Slots)
	api.HandleFunc("GET /api/v1/availability/dates", d.booking.Dates)
	api.Handle("POST /api/v1/bookings", d.limits.bookingCreate.PerIP(http.HandlerFunc(d.booking.Create)))
	api.Handle("GET /api/v1/bookings/{code}", d.limits.bookingCode.PerIP(http.HandlerFunc(d.booking.GetByCode)))
	api.Handle("POST /api/v1/bookings/{code}/cancel", d.limits.bookingCode.PerIP(http.HandlerFunc(d.booking.Cancel)))
	api.Handle("GET /api/v1/admin/bookings", staff(d.booking.ListAdmin))
	api.Handle("GET /api/v1/admin/bookings/{id}", staff(d.booking.GetAdmin))
	api.Handle("POST /api/v1/admin/bookings/{id}/complete", staff(d.booking.Complete))
	api.Handle("PUT /api/v1/admin/bookings/{id}/note", staff(d.booking.SaveNote))
	api.Handle("POST /api/v1/admin/bookings/{id}/confirm", admin(d.booking.Confirm))
	api.Handle("POST /api/v1/admin/bookings/{id}/cancel", admin(d.booking.CancelAdmin))
	api.Handle("POST /api/v1/admin/bookings/{id}/no-show", admin(d.booking.NoShow))
	api.Handle("POST /api/v1/admin/bookings/{id}/reschedule", admin(d.booking.Reschedule))

	api.Handle("GET /api/v1/admin/patients", admin(d.patient.List))
	api.Handle("GET /api/v1/admin/patients/{id}", admin(d.patient.Get))

	api.Handle("POST /api/v1/admin/bookings/{id}/invoice", admin(d.invoice.Create))
	api.Handle("GET /api/v1/admin/invoices", admin(d.invoice.List))
	api.Handle("GET /api/v1/admin/invoices/{id}", admin(d.invoice.Get))
	api.Handle("PATCH /api/v1/admin/invoices/{id}", admin(d.invoice.Update))
	api.Handle("POST /api/v1/admin/invoices/{id}/pay", admin(d.invoice.Pay))

	api.HandleFunc("GET /api/v1/promos", d.promo.ListPublic)
	api.HandleFunc("GET /api/v1/promos/{slug}", d.promo.GetPublic)
	api.Handle("GET /api/v1/admin/promos", admin(d.promo.List))
	api.Handle("POST /api/v1/admin/promos", admin(d.promo.Create))
	api.Handle("GET /api/v1/admin/promos/{id}", admin(d.promo.Get))
	api.Handle("PUT /api/v1/admin/promos/{id}", admin(d.promo.Update))
	api.Handle("DELETE /api/v1/admin/promos/{id}", admin(d.promo.Delete))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", health.Live)
	mux.HandleFunc("GET /readyz", health.Ready(d.db))
	mux.Handle("/api/", middleware.SameOrigin(d.frontendOrigin)(api))

	return middleware.Chain(mux,
		middleware.RequestID,
		middleware.Logger(d.log),
		middleware.Recover,
		middleware.SecureHeaders,
		clientip.Middleware(d.trustedProxyHops),
	)
}
