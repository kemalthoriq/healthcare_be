package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouter_Healthz(t *testing.T) {
	router := newRouter()

	tests := []struct {
		method     string
		wantStatus int
	}{
		{http.MethodGet, http.StatusOK},
		{http.MethodHead, http.StatusOK},
		{http.MethodPost, http.StatusMethodNotAllowed},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			rec := httptest.NewRecorder()
			req := httptest.NewRequest(tt.method, "/healthz", nil)

			router.ServeHTTP(rec, req)

			if rec.Code != tt.wantStatus {
				t.Errorf("%s /healthz status = %d, want %d", tt.method, rec.Code, tt.wantStatus)
			}
		})
	}
}

func TestRouter_UnknownPathReturnsNotFound(t *testing.T) {
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/does-not-exist", nil)

	newRouter().ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
