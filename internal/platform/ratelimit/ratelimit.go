package ratelimit

import (
	"math"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/clientip"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

const sweepEvery = 5 * time.Minute

type bucket struct {
	tokens float64
	last   time.Time
}

type Limiter struct {
	mu        sync.Mutex
	burst     float64
	perSecond float64
	window    time.Duration
	buckets   map[string]*bucket
	lastSweep time.Time
	now       func() time.Time
}

func New(limit int, window time.Duration) *Limiter {
	return &Limiter{
		burst:     float64(limit),
		perSecond: float64(limit) / window.Seconds(),
		window:    window,
		buckets:   map[string]*bucket{},
		now:       time.Now,
	}
}

func (l *Limiter) Allow(key string) (bool, time.Duration) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := l.now()
	if now.Sub(l.lastSweep) >= sweepEvery {
		l.sweep(now)
	}

	b, ok := l.buckets[key]
	if !ok {
		b = &bucket{tokens: l.burst, last: now}
		l.buckets[key] = b
	}

	b.tokens = math.Min(l.burst, b.tokens+now.Sub(b.last).Seconds()*l.perSecond)
	b.last = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}
	wait := time.Duration((1 - b.tokens) / l.perSecond * float64(time.Second))
	return false, wait
}

func (l *Limiter) sweep(now time.Time) {
	for key, b := range l.buckets {
		if now.Sub(b.last) >= l.window {
			delete(l.buckets, key)
		}
	}
	l.lastSweep = now
}

func (l *Limiter) size() int {
	l.mu.Lock()
	defer l.mu.Unlock()
	return len(l.buckets)
}

func (l *Limiter) PerIP(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if ok, wait := l.Allow(clientip.FromContext(r.Context())); !ok {
			Reject(w, r, wait)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func Reject(w http.ResponseWriter, r *http.Request, wait time.Duration) {
	seconds := int(math.Ceil(wait.Seconds()))
	w.Header().Set("Retry-After", strconv.Itoa(max(seconds, 1)))
	httpx.Error(w, r, apperr.RateLimited())
}
