package httpx

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type bookingInput struct {
	Name  string `json:"name"`
	Phone string `json:"phone"`
	Age   int    `json:"age"`
}

func newJSONRequest(body, contentType string) *http.Request {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}
	return req
}

func TestDecodeJSON_Success(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
	}{
		{"plain content type", "application/json"},
		{"content type with charset", "application/json; charset=utf-8"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newJSONRequest(`{"name":"Sari","phone":"81234567890","age":28}`, tt.contentType)

			var got bookingInput
			if err := DecodeJSON(httptest.NewRecorder(), req, &got); err != nil {
				t.Fatalf("expected no error, got %v", err)
			}

			want := bookingInput{Name: "Sari", Phone: "81234567890", Age: 28}
			if got != want {
				t.Errorf("got %+v, want %+v", got, want)
			}
		})
	}
}

func TestDecodeJSON_Rejects(t *testing.T) {
	tests := []struct {
		name        string
		body        string
		contentType string
		wantKind    apperr.Kind
		wantCode    string
		wantField   string
	}{
		{"missing content type", `{"name":"Sari"}`, "", apperr.KindUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", ""},
		{"form content type", `name=Sari`, "application/x-www-form-urlencoded", apperr.KindUnsupportedMediaType, "UNSUPPORTED_MEDIA_TYPE", ""},
		{"empty body", ``, "application/json", apperr.KindBadRequest, "INVALID_JSON", ""},
		{"malformed json", `{"name":`, "application/json", apperr.KindBadRequest, "INVALID_JSON", ""},
		{"syntax error", `{"name" "Sari"}`, "application/json", apperr.KindBadRequest, "INVALID_JSON", ""},
		{"two json objects", `{"name":"Sari"}{"name":"Budi"}`, "application/json", apperr.KindBadRequest, "INVALID_JSON", ""},
		{"unknown field", `{"name":"Sari","phnoe":"812"}`, "application/json", apperr.KindUnprocessable, "VALIDATION_ERROR", "phnoe"},
		{"wrong field type", `{"name":"Sari","age":"dua puluh"}`, "application/json", apperr.KindUnprocessable, "VALIDATION_ERROR", "age"},
		{"body too large", `{"name":"` + strings.Repeat("a", maxBodyBytes) + `"}`, "application/json", apperr.KindTooLarge, "PAYLOAD_TOO_LARGE", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := newJSONRequest(tt.body, tt.contentType)

			var dst bookingInput
			err := DecodeJSON(httptest.NewRecorder(), req, &dst)

			var appErr *apperr.Error
			if !errors.As(err, &appErr) {
				t.Fatalf("expected *apperr.Error, got %v", err)
			}
			if appErr.Kind != tt.wantKind || appErr.Code != tt.wantCode {
				t.Errorf("got kind %v code %q, want kind %v code %q", appErr.Kind, appErr.Code, tt.wantKind, tt.wantCode)
			}
			if tt.wantField != "" {
				if _, ok := appErr.Fields[tt.wantField]; !ok {
					t.Errorf("expected field %q in %v", tt.wantField, appErr.Fields)
				}
			}
		})
	}
}
