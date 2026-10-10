package httpx

import (
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const fallbackBody = `{"error":{"code":"INTERNAL_ERROR","message":"Terjadi kesalahan pada server."}}`

var errInternal = apperr.New(apperr.KindInternal, "INTERNAL_ERROR", "Terjadi kesalahan pada server.")

type dataEnvelope struct {
	Data any `json:"data"`
}

type errorEnvelope struct {
	Error errorBody `json:"error"`
}

type errorBody struct {
	Code    string            `json:"code"`
	Message string            `json:"message"`
	Fields  map[string]string `json:"fields,omitempty"`
}

func JSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	write(w, r, status, dataEnvelope{Data: data})
}

func NoContent(w http.ResponseWriter) {
	w.WriteHeader(http.StatusNoContent)
}

func Error(w http.ResponseWriter, r *http.Request, err error) {
	var appErr *apperr.Error
	if !errors.As(err, &appErr) {
		appErr = errInternal
	}

	status := statusFor(appErr.Kind)
	if status >= http.StatusInternalServerError {
		slog.ErrorContext(r.Context(), "request failed", "err", err, "method", r.Method, "pattern", r.Pattern)
	}

	write(w, r, status, errorEnvelope{Error: errorBody{
		Code:    appErr.Code,
		Message: appErr.Message,
		Fields:  appErr.Fields,
	}})
}

func write(w http.ResponseWriter, r *http.Request, status int, body any) {
	b, err := json.Marshal(body)
	if err != nil {
		slog.ErrorContext(r.Context(), "encode response failed", "err", err)
		status = http.StatusInternalServerError
		b = []byte(fallbackBody)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_, _ = w.Write(b)
}

func statusFor(kind apperr.Kind) int {
	switch kind {
	case apperr.KindBadRequest:
		return http.StatusBadRequest
	case apperr.KindUnauthenticated:
		return http.StatusUnauthorized
	case apperr.KindForbidden:
		return http.StatusForbidden
	case apperr.KindNotFound:
		return http.StatusNotFound
	case apperr.KindConflict:
		return http.StatusConflict
	case apperr.KindTooLarge:
		return http.StatusRequestEntityTooLarge
	case apperr.KindUnsupportedMediaType:
		return http.StatusUnsupportedMediaType
	case apperr.KindUnprocessable:
		return http.StatusUnprocessableEntity
	case apperr.KindRateLimited:
		return http.StatusTooManyRequests
	default:
		return http.StatusInternalServerError
	}
}
