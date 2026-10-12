package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

func TestJSON_WrapsDataInEnvelope(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(rec, req, http.StatusCreated, map[string]string{"code": "LMR-7K2Q-X9MD"})

	assertJSON(t, rec, http.StatusCreated, `{"data":{"code":"LMR-7K2Q-X9MD"}}`)
}

func TestJSON_FallsBackToInternalErrorWhenEncodingFails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	JSON(rec, req, http.StatusOK, map[string]any{"bad": make(chan int)})

	assertJSON(t, rec, http.StatusInternalServerError, fallbackBody)
}

func TestNoContent(t *testing.T) {
	rec := httptest.NewRecorder()

	NoContent(rec)

	if rec.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNoContent)
	}
	if rec.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", rec.Body.String())
	}
}

func TestError_MapsKindToStatus(t *testing.T) {
	tests := []struct {
		kind       apperr.Kind
		wantStatus int
	}{
		{apperr.KindBadRequest, http.StatusBadRequest},
		{apperr.KindUnauthenticated, http.StatusUnauthorized},
		{apperr.KindForbidden, http.StatusForbidden},
		{apperr.KindNotFound, http.StatusNotFound},
		{apperr.KindConflict, http.StatusConflict},
		{apperr.KindTooLarge, http.StatusRequestEntityTooLarge},
		{apperr.KindUnsupportedMediaType, http.StatusUnsupportedMediaType},
		{apperr.KindUnprocessable, http.StatusUnprocessableEntity},
		{apperr.KindRateLimited, http.StatusTooManyRequests},
		{apperr.KindUnavailable, http.StatusServiceUnavailable},
		{apperr.KindInternal, http.StatusInternalServerError},
	}

	for _, tt := range tests {
		t.Run(http.StatusText(tt.wantStatus), func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(http.MethodGet, "/", nil)

			Error(rec, req, apperr.New(tt.kind, "TEST_CODE", "Pesan uji."))

			assertJSON(t, rec, tt.wantStatus, `{"error":{"code":"TEST_CODE","message":"Pesan uji."}}`)
		})
	}
}

func TestError_IncludesValidationFields(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/", nil)

	Error(rec, req, apperr.Validation(map[string]string{"phone": "Nomor WhatsApp wajib diisi."}))

	assertJSON(t, rec, http.StatusUnprocessableEntity,
		`{"error":{"code":"VALIDATION_ERROR","message":"Data belum sesuai.","fields":{"phone":"Nomor WhatsApp wajib diisi."}}}`)
}

func TestError_UnwrapsWrappedAppError(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	err := fmt.Errorf("get treatment: %w", apperr.NotFound("Treatment tidak ditemukan."))
	Error(rec, req, err)

	assertJSON(t, rec, http.StatusNotFound, `{"error":{"code":"NOT_FOUND","message":"Treatment tidak ditemukan."}}`)
}

func TestError_HidesUnknownErrorDetails(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)

	Error(rec, req, errors.New(`pq: password authentication failed for user "postgres"`))

	assertJSON(t, rec, http.StatusInternalServerError, fallbackBody)
	if strings.Contains(rec.Body.String(), "postgres") {
		t.Errorf("response leaked internal error details: %s", rec.Body.String())
	}
}

func assertJSON(t *testing.T, rec *httptest.ResponseRecorder, wantStatus int, wantBody string) {
	t.Helper()

	if rec.Code != wantStatus {
		t.Errorf("status = %d, want %d", rec.Code, wantStatus)
	}
	if got := rec.Header().Get("Content-Type"); got != "application/json" {
		t.Errorf("Content-Type = %q, want %q", got, "application/json")
	}
	if got := rec.Body.String(); got != wantBody {
		t.Errorf("body = %s, want %s", got, wantBody)
	}
}
