package integrity

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const maxListed = 3

var Location = time.FixedZone("WIB", 7*60*60)

var monthNames = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

type Booking struct {
	Code        string
	TreatmentID int64
	TherapistID int64
	StartAt     time.Time
	EndAt       time.Time
}

type Finder interface {
	ActiveByTreatment(ctx context.Context, treatmentID int64) ([]Booking, error)
	ActiveByTherapist(ctx context.Context, therapistID int64) ([]Booking, error)
}

func Conflict(bookings []Booking) error {
	if len(bookings) == 0 {
		return nil
	}

	listed := make([]string, 0, maxListed)
	for _, b := range bookings[:min(len(bookings), maxListed)] {
		t := b.StartAt.In(Location)
		listed = append(listed, fmt.Sprintf("%s (%d %s %02d:%02d)", b.Code, t.Day(), monthNames[t.Month()-1], t.Hour(), t.Minute()))
	}

	msg := fmt.Sprintf("Masih ada %d booking aktif: %s", len(bookings), strings.Join(listed, ", "))
	if extra := len(bookings) - maxListed; extra > 0 {
		msg += fmt.Sprintf(", dan %d lainnya", extra)
	}

	return apperr.Conflict("HAS_ACTIVE_BOOKINGS", msg+". Selesaikan, batalkan, atau reschedule dulu.")
}
