package middleware

import (
	"bytes"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/logger"
)

func TestChain_FirstMiddlewareIsOutermost(t *testing.T) {
	var order []string
	mark := func(name string) Middleware {
		return func(next http.Handler) http.Handler {
			return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				order = append(order, name)
				next.ServeHTTP(w, r)
			})
		}
	}
	final := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
	})

	Chain(final, mark("a"), mark("b")).ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if got := strings.Join(order, ","); got != "a,b,handler" {
		t.Errorf("order = %s, want a,b,handler", got)
	}
}

func TestRequestID(t *testing.T) {
	tests := []struct {
		name     string
		incoming string
		wantSame bool
	}{
		{"generates when missing", "", false},
		{"keeps valid incoming id", "abc-123_XYZ", true},
		{"replaces id with invalid characters", "abc 123<script>", false},
		{"replaces id that is too long", strings.Repeat("a", 65), false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var ctxID string
			handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				ctxID = logger.RequestID(r.Context())
			}))

			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.incoming != "" {
				req.Header.Set(HeaderRequestID, tt.incoming)
			}
			rec := httptest.NewRecorder()
			handler.ServeHTTP(rec, req)

			headerID := rec.Header().Get(HeaderRequestID)
			if headerID == "" || headerID != ctxID {
				t.Fatalf("header id %q and context id %q must be equal and non-empty", headerID, ctxID)
			}
			if (headerID == tt.incoming) != tt.wantSame {
				t.Errorf("id = %q, incoming = %q, wantSame = %v", headerID, tt.incoming, tt.wantSame)
			}
		})
	}
}

func TestLogger_RecordsRequest(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(logger.ContextHandler(slog.NewJSONHandler(&buf, nil)))

	mux := http.NewServeMux()
	mux.HandleFunc("GET /bookings/{code}", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
		_, _ = w.Write([]byte("hello"))
	})
	handler := Chain(mux, RequestID, Logger(log))

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/bookings/LMR-SECRET", nil))

	out := buf.String()
	for _, want := range []string{`"pattern":"GET /bookings/{code}"`, `"status":418`, `"bytes":5`, `"request_id":`} {
		if !strings.Contains(out, want) {
			t.Errorf("log missing %s: %s", want, out)
		}
	}
	if strings.Contains(out, "LMR-SECRET") {
		t.Errorf("log leaked raw path: %s", out)
	}
}

func TestLogger_DefaultsToStatusOK(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))

	if !strings.Contains(buf.String(), `"status":200`) {
		t.Errorf("log should record status 200: %s", buf.String())
	}
}

func TestLogger_SkipsHealthz(t *testing.T) {
	var buf bytes.Buffer
	log := slog.New(slog.NewJSONHandler(&buf, nil))

	handler := Logger(log)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/healthz", nil))

	if buf.Len() != 0 {
		t.Errorf("healthz should not be logged: %s", buf.String())
	}
}

func TestRecover_ReturnsInternalError(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("boom")
	}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusInternalServerError)
	}
	if !strings.Contains(rec.Body.String(), `"code":"INTERNAL_ERROR"`) {
		t.Errorf("body = %s, want INTERNAL_ERROR", rec.Body.String())
	}
	if strings.Contains(rec.Body.String(), "boom") {
		t.Errorf("panic value leaked to client: %s", rec.Body.String())
	}
}

func TestRecover_RepanicsOnAbortHandler(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic(http.ErrAbortHandler)
	}))

	defer func() {
		if v := recover(); v != http.ErrAbortHandler {
			t.Errorf("recovered %v, want http.ErrAbortHandler", v)
		}
	}()

	handler.ServeHTTP(httptest.NewRecorder(), httptest.NewRequest(http.MethodGet, "/", nil))
}

func TestSecureHeaders(t *testing.T) {
	handler := SecureHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))

	want := map[string]string{
		"X-Content-Type-Options":  "nosniff",
		"X-Frame-Options":         "DENY",
		"Referrer-Policy":         "no-referrer",
		"Content-Security-Policy": "default-src 'none'; frame-ancestors 'none'",
	}
	for k, v := range want {
		if got := rec.Header().Get(k); got != v {
			t.Errorf("%s = %q, want %q", k, got, v)
		}
	}
}
