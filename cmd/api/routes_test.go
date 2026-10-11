package main

import (
	"context"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/auth"
	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/invoice"
	"github.com/kemalthoriq/healthcare_be/internal/promo"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

const testOrigin = "http://localhost:3000"

type okPinger struct{}

func (okPinger) Ping(ctx context.Context) error {
	return nil
}

func newTestRouter() http.Handler {
	return newRouter(routerDeps{
		log:            slog.New(slog.NewTextHandler(io.Discard, nil)),
		db:             okPinger{},
		frontendOrigin: testOrigin,
		auth:           auth.NewHandler(nil, false),
		treatment:      treatment.NewHandler(nil),
		therapist:      therapist.NewHandler(nil),
		account:        therapist.NewAccountHandler(nil),
		setting:        setting.NewHandler(nil),
		booking:        booking.NewHandler(nil, nil, time.UTC),
		patient:        booking.NewPatientHandler(nil, time.UTC),
		invoice:        invoice.NewHandler(nil, time.UTC),
		promo:          promo.NewHandler(nil),
	})
}

func TestRouter_HealthEndpoints(t *testing.T) {
	router := newTestRouter()

	tests := []struct {
		method     string
		path       string
		wantStatus int
	}{
		{http.MethodGet, "/healthz", http.StatusOK},
		{http.MethodHead, "/healthz", http.StatusOK},
		{http.MethodPost, "/healthz", http.StatusMethodNotAllowed},
		{http.MethodGet, "/readyz", http.StatusOK},
		{http.MethodHead, "/readyz", http.StatusOK},
		{http.MethodPost, "/readyz", http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method+" "+tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, tt.path, nil)

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("%s %s status = %d, want %d", tt.method, tt.path, rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRouter_UnknownPathReturnsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)

	newTestRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}

func TestRouter_AppliesMiddleware(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	newTestRouter().ServeHTTP(rec, req)

	if rec.Header().Get("X-Request-ID") == "" {
		t.Error("missing X-Request-ID header")
	}
	if rec.Header().Get("X-Content-Type-Options") != "nosniff" {
		t.Error("missing security headers")
	}
}

func TestRouter_ProtectedRoutes(t *testing.T) {
	tests := []struct {
		name       string
		method     string
		path       string
		origin     string
		wantStatus int
	}{
		{"login rejects foreign origin", http.MethodPost, "/api/v1/admin/auth/login", "https://evil.example", http.StatusForbidden},
		{"login reaches handler", http.MethodPost, "/api/v1/admin/auth/login", testOrigin, http.StatusUnsupportedMediaType},
		{"me requires session", http.MethodGet, "/api/v1/admin/auth/me", "", http.StatusUnauthorized},
		{"logout requires session", http.MethodPost, "/api/v1/admin/auth/logout", testOrigin, http.StatusUnauthorized},
		{"admin treatment list requires session", http.MethodGet, "/api/v1/admin/treatments", "", http.StatusUnauthorized},
		{"admin treatment create requires session", http.MethodPost, "/api/v1/admin/treatments", testOrigin, http.StatusUnauthorized},
		{"admin treatment delete requires session", http.MethodDelete, "/api/v1/admin/treatments/1", testOrigin, http.StatusUnauthorized},
		{"admin therapist create requires session", http.MethodPost, "/api/v1/admin/therapists", testOrigin, http.StatusUnauthorized},
		{"admin treatment therapists requires session", http.MethodPut, "/api/v1/admin/treatments/1/therapists", testOrigin, http.StatusUnauthorized},
		{"admin schedules requires session", http.MethodPut, "/api/v1/admin/therapists/1/schedules", testOrigin, http.StatusUnauthorized},
		{"admin time off requires session", http.MethodPost, "/api/v1/admin/therapists/1/time-off", testOrigin, http.StatusUnauthorized},
		{"admin settings requires session", http.MethodPut, "/api/v1/admin/settings", testOrigin, http.StatusUnauthorized},
		{"slots validates query", http.MethodGet, "/api/v1/availability/slots", "", http.StatusUnprocessableEntity},
		{"dates validates query", http.MethodGet, "/api/v1/availability/dates?treatment=a&therapist=b&month=oktober", "", http.StatusUnprocessableEntity},
		{"create booking rejects foreign origin", http.MethodPost, "/api/v1/bookings", "https://evil.example", http.StatusForbidden},
		{"create booking requires json", http.MethodPost, "/api/v1/bookings", testOrigin, http.StatusUnsupportedMediaType},
		{"cancel booking rejects foreign origin", http.MethodPost, "/api/v1/bookings/LMR-AAAA-AAAA/cancel", "https://evil.example", http.StatusForbidden},
		{"cancel booking requires json", http.MethodPost, "/api/v1/bookings/LMR-AAAA-AAAA/cancel", testOrigin, http.StatusUnsupportedMediaType},
		{"admin booking list requires session", http.MethodGet, "/api/v1/admin/bookings", "", http.StatusUnauthorized},
		{"admin booking confirm requires session", http.MethodPost, "/api/v1/admin/bookings/1/confirm", testOrigin, http.StatusUnauthorized},
		{"booking note requires session", http.MethodPut, "/api/v1/admin/bookings/1/note", testOrigin, http.StatusUnauthorized},
		{"reschedule requires session", http.MethodPost, "/api/v1/admin/bookings/1/reschedule", testOrigin, http.StatusUnauthorized},
		{"therapist account requires session", http.MethodPost, "/api/v1/admin/therapists/1/account", testOrigin, http.StatusUnauthorized},
		{"patient list requires session", http.MethodGet, "/api/v1/admin/patients", "", http.StatusUnauthorized},
		{"patient detail requires session", http.MethodGet, "/api/v1/admin/patients/1", "", http.StatusUnauthorized},
		{"create invoice rejects foreign origin", http.MethodPost, "/api/v1/admin/bookings/1/invoice", "https://evil.example", http.StatusForbidden},
		{"create invoice requires session", http.MethodPost, "/api/v1/admin/bookings/1/invoice", testOrigin, http.StatusUnauthorized},
		{"invoice list requires session", http.MethodGet, "/api/v1/admin/invoices", "", http.StatusUnauthorized},
		{"invoice update requires session", http.MethodPatch, "/api/v1/admin/invoices/1", testOrigin, http.StatusUnauthorized},
		{"invoice pay requires session", http.MethodPost, "/api/v1/admin/invoices/1/pay", testOrigin, http.StatusUnauthorized},
		{"admin clinic requires session", http.MethodGet, "/api/v1/admin/clinic", "", http.StatusUnauthorized},
		{"admin clinic update requires session", http.MethodPut, "/api/v1/admin/clinic", testOrigin, http.StatusUnauthorized},
		{"admin promo list requires session", http.MethodGet, "/api/v1/admin/promos", "", http.StatusUnauthorized},
		{"admin promo create requires session", http.MethodPost, "/api/v1/admin/promos", testOrigin, http.StatusUnauthorized},
		{"admin promo update requires session", http.MethodPut, "/api/v1/admin/promos/1", testOrigin, http.StatusUnauthorized},
		{"admin promo delete requires session", http.MethodDelete, "/api/v1/admin/promos/1", testOrigin, http.StatusUnauthorized},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, tt.path, nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			rec := httptest.NewRecorder()

			newTestRouter().ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}
