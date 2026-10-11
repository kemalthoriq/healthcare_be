package booking

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/captcha"
	"github.com/kemalthoriq/healthcare_be/internal/platform/clientip"
)

type fakeCaptcha struct {
	token, ip string
}

func (f *fakeCaptcha) Verify(ctx context.Context, token, remoteIP string) error {
	f.token, f.ip = token, remoteIP
	return captcha.ErrFailed
}

func TestCreateHandler_VerifiesCaptchaBeforeBooking(t *testing.T) {
	fc := &fakeCaptcha{}
	h := clientip.Middleware(0)(http.HandlerFunc(NewHandler(nil, nil, fc, time.UTC).Create))

	req := httptest.NewRequest(http.MethodPost, "/api/v1/bookings", strings.NewReader(`{"captcha_token":"tok-123"}`))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "198.51.100.7:4000"
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnprocessableEntity || !strings.Contains(rec.Body.String(), "CAPTCHA_FAILED") {
		t.Errorf("got %d %s, want 422 CAPTCHA_FAILED", rec.Code, rec.Body.String())
	}
	if fc.token != "tok-123" || fc.ip != "198.51.100.7" {
		t.Errorf("verifier got token %q ip %q", fc.token, fc.ip)
	}
}
