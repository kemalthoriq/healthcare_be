package ratelimit

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/clientip"
)

func newTestLimiter(limit int, window time.Duration) (*Limiter, *time.Time) {
	now := time.Date(2026, 10, 11, 10, 0, 0, 0, time.UTC)
	l := New(limit, window)
	l.now = func() time.Time { return now }
	return l, &now
}

func TestAllow_BurstThenRefill(t *testing.T) {
	l, now := newTestLimiter(5, 10*time.Minute)

	for i := range 5 {
		if ok, _ := l.Allow("1.2.3.4"); !ok {
			t.Fatalf("request %d rejected within burst", i+1)
		}
	}

	ok, wait := l.Allow("1.2.3.4")
	if ok || wait != 2*time.Minute {
		t.Fatalf("6th request: ok = %v, wait = %v; want rejected, 2m", ok, wait)
	}
	if ok, _ := l.Allow("5.6.7.8"); !ok {
		t.Error("another key must have its own bucket")
	}

	*now = now.Add(2 * time.Minute)
	if ok, _ := l.Allow("1.2.3.4"); !ok {
		t.Error("one token should refill after 2 minutes")
	}
	if ok, _ := l.Allow("1.2.3.4"); ok {
		t.Error("only one token should have refilled")
	}
}

func TestSweep_RemovesIdleBuckets(t *testing.T) {
	l, now := newTestLimiter(30, time.Minute)
	l.Allow("a")
	l.Allow("b")

	*now = now.Add(sweepEvery)
	l.Allow("c")

	if l.size() != 1 {
		t.Errorf("buckets = %d, want 1 (idle a and b swept)", l.size())
	}
}

func TestReject(t *testing.T) {
	rec := httptest.NewRecorder()
	Reject(rec, httptest.NewRequest(http.MethodPost, "/", nil), 1500*time.Millisecond)

	if rec.Code != http.StatusTooManyRequests || rec.Header().Get("Retry-After") != "2" {
		t.Errorf("status = %d, Retry-After = %q", rec.Code, rec.Header().Get("Retry-After"))
	}
}

func TestPerIP(t *testing.T) {
	l, _ := newTestLimiter(2, time.Minute)
	h := clientip.Middleware(0)(l.PerIP(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	})))

	do := func(remote string) int {
		r := httptest.NewRequest(http.MethodPost, "/", nil)
		r.RemoteAddr = remote
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, r)
		return rec.Code
	}

	for range 2 {
		if code := do("203.0.113.9:1000"); code != http.StatusNoContent {
			t.Fatalf("status = %d, want 204", code)
		}
	}
	if code := do("203.0.113.9:2000"); code != http.StatusTooManyRequests {
		t.Errorf("3rd request from same IP (other port): status = %d, want 429", code)
	}
	if code := do("198.51.100.7:1000"); code != http.StatusNoContent {
		t.Errorf("other IP: status = %d, want 204", code)
	}
}
