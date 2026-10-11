package integrity

import (
	"errors"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

func TestConflict(t *testing.T) {
	if err := Conflict(nil); err != nil {
		t.Fatalf("no bookings: err = %v, want nil", err)
	}

	start := time.Date(2026, 10, 14, 3, 0, 0, 0, time.UTC)
	var bookings []Booking
	for i, code := range []string{"LMR-AAAA-0001", "LMR-AAAA-0002", "LMR-AAAA-0003", "LMR-AAAA-0004", "LMR-AAAA-0005"} {
		bookings = append(bookings, Booking{Code: code, StartAt: start.Add(time.Duration(i) * time.Hour)})
	}

	var appErr *apperr.Error
	if !errors.As(Conflict(bookings), &appErr) || appErr.Code != "HAS_ACTIVE_BOOKINGS" {
		t.Fatalf("err = %v, want HAS_ACTIVE_BOOKINGS", appErr)
	}

	want := "Masih ada 5 booking aktif: LMR-AAAA-0001 (14 Okt 10:00), LMR-AAAA-0002 (14 Okt 11:00), LMR-AAAA-0003 (14 Okt 12:00), dan 2 lainnya. Selesaikan, batalkan, atau reschedule dulu."
	if appErr.Message != want {
		t.Errorf("message = %q\nwant      %q", appErr.Message, want)
	}
}
