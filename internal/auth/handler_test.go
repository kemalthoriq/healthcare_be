package auth

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/ratelimit"
)

func newTestHandler() *Handler {
	store := newFakeStore()
	store.addUser(User{ID: 7, Name: "Admin", Email: "admin@lumiere.test", Role: RoleAdmin, IsActive: true}, testPassword)
	return NewHandler(newTestService(store), true, ratelimit.New(10, 15*time.Minute))
}

func login(t *testing.T, h *Handler, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/api/v1/admin/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	h.Login(rec, req)
	return rec
}

func TestHandlerLogin_SetsSecureSessionCookie(t *testing.T) {
	rec := login(t, newTestHandler(), `{"email":"admin@lumiere.test","password":"`+testPassword+`"}`)

	if rec.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", rec.Code, rec.Body.String())
	}

	cookies := rec.Result().Cookies()
	if len(cookies) != 1 {
		t.Fatalf("got %d cookies, want 1", len(cookies))
	}
	c := cookies[0]
	if c.Name != sessionCookieName || c.Value == "" || !c.HttpOnly || !c.Secure || c.SameSite != http.SameSiteLaxMode {
		t.Errorf("unexpected cookie: %+v", c)
	}
	if strings.Contains(rec.Body.String(), "password") {
		t.Errorf("response must not contain password data: %s", rec.Body.String())
	}
}

func TestHandlerLogin_Rejects(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantStatus int
		wantCode   string
	}{
		{"missing fields", `{"email":"","password":""}`, http.StatusUnprocessableEntity, "VALIDATION_ERROR"},
		{"wrong password", `{"email":"admin@lumiere.test","password":"salah"}`, http.StatusUnauthorized, "INVALID_CREDENTIALS"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := login(t, newTestHandler(), tt.body)

			if rec.Code != tt.wantStatus || !strings.Contains(rec.Body.String(), tt.wantCode) {
				t.Errorf("got %d %s, want %d %s", rec.Code, rec.Body.String(), tt.wantStatus, tt.wantCode)
			}
			if len(rec.Result().Cookies()) != 0 {
				t.Error("no cookie should be set")
			}
		})
	}
}

func TestHandlerLogin_LimitsAttemptsPerEmail(t *testing.T) {
	h := newTestHandler()

	for i := range 10 {
		if rec := login(t, h, `{"email":"admin@lumiere.test","password":"salah"}`); rec.Code != http.StatusUnauthorized {
			t.Fatalf("attempt %d: status = %d, want 401", i+1, rec.Code)
		}
	}

	rec := login(t, h, `{"email":" ADMIN@lumiere.test ","password":"`+testPassword+`"}`)
	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") == "" {
		t.Errorf("11th attempt: status = %d, Retry-After = %q; want 429 with Retry-After", rec.Code, rec.Header().Get("Retry-After"))
	}

	if rec := login(t, h, `{"email":"lain@lumiere.test","password":"salah"}`); rec.Code != http.StatusUnauthorized {
		t.Errorf("other email: status = %d, want 401", rec.Code)
	}
}

func TestRequireAuth(t *testing.T) {
	h := newTestHandler()
	token := login(t, h, `{"email":"admin@lumiere.test","password":"`+testPassword+`"}`).Result().Cookies()[0].Value

	me := h.RequireAuth(http.HandlerFunc(h.Me))

	tests := []struct {
		name       string
		cookie     string
		wantStatus int
	}{
		{"no cookie", "", http.StatusUnauthorized},
		{"invalid cookie", "forged-token", http.StatusUnauthorized},
		{"valid cookie", token, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/api/v1/admin/auth/me", nil)
			if tt.cookie != "" {
				req.AddCookie(&http.Cookie{Name: sessionCookieName, Value: tt.cookie})
			}
			rec := httptest.NewRecorder()

			me.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d, body = %s", rec.Code, tt.wantStatus, rec.Body.String())
			}
		})
	}
}

func TestRequireRole(t *testing.T) {
	ok := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	adminOnly := RequireRole(RoleAdmin)(ok)

	tests := []struct {
		name       string
		user       *User
		wantStatus int
	}{
		{"no user", nil, http.StatusUnauthorized},
		{"therapist", &User{Role: RoleTherapist}, http.StatusForbidden},
		{"admin", &User{Role: RoleAdmin}, http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.user != nil {
				req = req.WithContext(withUser(req.Context(), *tt.user))
			}
			rec := httptest.NewRecorder()

			adminOnly.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", rec.Code, tt.wantStatus)
			}
		})
	}
}
