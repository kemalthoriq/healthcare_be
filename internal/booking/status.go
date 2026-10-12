package booking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

const maxCancelReasonLength = 200

var (
	ErrBookingNotFound      = apperr.NotFound("Booking tidak ditemukan.")
	ErrCancelDeadlinePassed = apperr.Unprocessable("CANCEL_DEADLINE_PASSED", "Batas waktu pembatalan sudah lewat. Silakan hubungi klinik.")
	ErrInvalidTransition    = apperr.Conflict("INVALID_TRANSITION", "Booking ini sudah tidak bisa dibatalkan.")
)

type Detail struct {
	Booking        Booking
	Treatment      treatment.Treatment
	Therapist      therapist.Therapist
	CancelDeadline time.Time
	CanCancel      bool
}

func (s *Service) GetByCode(ctx context.Context, code string) (Detail, error) {
	b, err := s.store.GetByCode(ctx, normalizeCode(code))
	if errors.Is(err, errNotFound) {
		return Detail{}, ErrBookingNotFound
	}
	if err != nil {
		return Detail{}, err
	}
	return s.detail(ctx, b)
}

func (s *Service) CancelByCode(ctx context.Context, code, reason string) (Detail, error) {
	reason = strings.TrimSpace(reason)
	if utf8.RuneCountInString(reason) > maxCancelReasonLength {
		return Detail{}, apperr.Validation(map[string]string{
			"reason": fmt.Sprintf("Alasan maksimal %d karakter.", maxCancelReasonLength),
		})
	}

	current, err := s.GetByCode(ctx, code)
	if err != nil {
		return Detail{}, err
	}
	if !isActive(current.Booking.Status) {
		return Detail{}, ErrInvalidTransition
	}
	if !current.CanCancel {
		return Detail{}, ErrCancelDeadlinePassed
	}

	deadline, err := s.cancelDeadlineHours(ctx)
	if err != nil {
		return Detail{}, err
	}

	b, err := s.store.CancelByCode(ctx, current.Booking.Code, reason, s.availability.now().Add(deadline))
	if errors.Is(err, errNotCancellable) {
		return Detail{}, ErrInvalidTransition
	}
	if err != nil {
		return Detail{}, err
	}

	if ab, err := s.GetAdmin(ctx, Actor{}, b.ID); err == nil {
		e := eventFor(EventCancelled, ab)
		e.CancelledByPatient = true
		s.notify(ctx, e)
	}
	return s.detail(ctx, b)
}

func (s *Service) detail(ctx context.Context, b Booking) (Detail, error) {
	tr, err := s.availability.treatments.Get(ctx, b.TreatmentID)
	if err != nil {
		return Detail{}, fmt.Errorf("get treatment: %w", err)
	}
	th, err := s.availability.therapists.Get(ctx, b.TherapistID)
	if err != nil {
		return Detail{}, fmt.Errorf("get therapist: %w", err)
	}

	deadline, err := s.cancelDeadlineHours(ctx)
	if err != nil {
		return Detail{}, err
	}
	cancelDeadline := b.StartAt.Add(-deadline)

	return Detail{
		Booking:        b,
		Treatment:      tr,
		Therapist:      th,
		CancelDeadline: cancelDeadline,
		CanCancel:      isActive(b.Status) && s.availability.now().Before(cancelDeadline),
	}, nil
}

func (s *Service) cancelDeadlineHours(ctx context.Context) (time.Duration, error) {
	st, err := s.availability.settings.Get(ctx)
	if err != nil {
		return 0, fmt.Errorf("get settings: %w", err)
	}
	return time.Duration(st.CancelDeadlineHours) * time.Hour, nil
}

func isActive(status string) bool {
	return status == StatusPending || status == StatusConfirmed
}

func normalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}
