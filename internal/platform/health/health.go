package health

import (
	"context"
	"log/slog"
	"net/http"
	"time"
)

const readyTimeout = 2 * time.Second

type Pinger interface {
	Ping(ctx context.Context) error
}

func Live(w http.ResponseWriter, r *http.Request) {
	writeStatus(w, http.StatusOK, "ok")
}

func Ready(db Pinger) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), readyTimeout)
		defer cancel()

		if err := db.Ping(ctx); err != nil {
			slog.ErrorContext(ctx, "readiness check failed", "err", err)
			writeStatus(w, http.StatusServiceUnavailable, "unavailable")
			return
		}

		writeStatus(w, http.StatusOK, "ok")
	}
}

func writeStatus(w http.ResponseWriter, code int, status string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_, _ = w.Write([]byte(`{"status":"` + status + `"}`))
}
