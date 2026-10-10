package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
)

type okPinger struct{}

func (okPinger) Ping(ctx context.Context) error {
	return nil
}

func TestRouter_HealthEndpoints(t *testing.T) {
	router := newRouter(okPinger{})

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

	newRouter(okPinger{}).ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", rec.Code, http.StatusNotFound)
	}
}
