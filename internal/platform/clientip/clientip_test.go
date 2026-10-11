package clientip

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolve(t *testing.T) {
	tests := []struct {
		name   string
		hops   int
		remote string
		xff    []string
		want   string
	}{
		{"no proxy ignores spoofed header", 0, "203.0.113.9:5000", []string{"1.1.1.1"}, "203.0.113.9"},
		{"vercel then render", 2, "10.0.0.5:5000", []string{"198.51.100.7, 76.76.21.1"}, "198.51.100.7"},
		{"spoofed entry left of real client", 2, "10.0.0.5:5000", []string{"1.1.1.1, 198.51.100.7, 76.76.21.1"}, "198.51.100.7"},
		{"multiple header lines", 2, "10.0.0.5:5000", []string{"198.51.100.7", "76.76.21.1"}, "198.51.100.7"},
		{"short chain falls back to leftmost", 2, "10.0.0.5:5000", []string{"76.76.21.1"}, "76.76.21.1"},
		{"no header falls back to remote", 2, "10.0.0.5:5000", nil, "10.0.0.5"},
		{"garbage falls back to remote", 1, "10.0.0.5:5000", []string{"bukan-ip"}, "10.0.0.5"},
		{"ipv6", 1, "[2001:db8::1]:443", []string{"2001:db8::7"}, "2001:db8::7"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, "/", nil)
			r.RemoteAddr = tt.remote
			for _, v := range tt.xff {
				r.Header.Add("X-Forwarded-For", v)
			}
			if got := Resolve(r, tt.hops); got != tt.want {
				t.Errorf("Resolve = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestMiddleware(t *testing.T) {
	var got string
	h := Middleware(0)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		got = FromContext(r.Context())
	}))

	r := httptest.NewRequest(http.MethodGet, "/", nil)
	r.RemoteAddr = "203.0.113.9:5000"
	h.ServeHTTP(httptest.NewRecorder(), r)

	if got != "203.0.113.9" {
		t.Errorf("ip in context = %q", got)
	}
}
