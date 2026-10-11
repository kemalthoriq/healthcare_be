package booking

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

func (f *fakeStore) GetByCode(ctx context.Context, code string) (Booking, error) {
	b, ok := f.byCode[code]
	if !ok {
		return Booking{}, errNotFound
	}
	return b, nil
}

func (f *fakeStore) CancelByCode(ctx context.Context, code, reason string, minStartAt time.Time) (Booking, error) {
	b, ok := f.byCode[code]
	if !ok || !isActive(b.Status) || !b.StartAt.After(minStartAt) {
		return Booking{}, errNotCancellable
	}
	b.Status = StatusCancelled
	b.CancelReason = reason
	f.byCode[code] = b
	return b, nil
}

func storeWith(bookings ...Booking) *fakeStore {
	f := &fakeStore{byCode: map[string]Booking{}}
	for _, b := range bookings {
		f.byCode[b.Code] = b
	}
	return f
}

var (
	farBooking  = Booking{Code: "LMR-AAAA-AAAA", Status: StatusPending, TreatmentID: 1, TherapistID: 7, StartAt: time.Date(2026, 10, 27, 10, 0, 0, 0, wib)}
	nearBooking = Booking{Code: "LMR-BBBB-BBBB", Status: StatusConfirmed, TreatmentID: 1, TherapistID: 7, StartAt: time.Date(2026, 10, 21, 8, 0, 0, 0, wib)}
	doneBooking = Booking{Code: "LMR-CCCC-CCCC", Status: StatusCompleted, TreatmentID: 1, TherapistID: 7, StartAt: time.Date(2026, 10, 27, 10, 0, 0, 0, wib)}
)

func TestGetByCode(t *testing.T) {
	svc := newTestService(storeWith(farBooking, nearBooking, doneBooking))

	d, err := svc.GetByCode(context.Background(), " lmr-aaaa-aaaa ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !d.CanCancel || d.Treatment.Name != "Facial Glow" || d.Therapist.Name != "dr. Sari" {
		t.Errorf("unexpected detail: %+v", d)
	}
	if want := time.Date(2026, 10, 26, 10, 0, 0, 0, wib); !d.CancelDeadline.Equal(want) {
		t.Errorf("CancelDeadline = %v, want %v", d.CancelDeadline, want)
	}

	if d, _ := svc.GetByCode(context.Background(), nearBooking.Code); d.CanCancel {
		t.Error("booking within the deadline must not be cancellable")
	}
	if d, _ := svc.GetByCode(context.Background(), doneBooking.Code); d.CanCancel {
		t.Error("completed booking must not be cancellable")
	}
	if _, err := svc.GetByCode(context.Background(), "LMR-ZZZZ-ZZZZ"); !errors.Is(err, ErrBookingNotFound) {
		t.Errorf("unknown code: err = %v, want ErrBookingNotFound", err)
	}
}

func TestCancelByCode(t *testing.T) {
	tests := []struct {
		name    string
		code    string
		reason  string
		wantErr error
	}{
		{"cancels before deadline", farBooking.Code, "Ada acara keluarga", nil},
		{"deadline passed", nearBooking.Code, "", ErrCancelDeadlinePassed},
		{"already completed", doneBooking.Code, "", ErrInvalidTransition},
		{"unknown code", "LMR-ZZZZ-ZZZZ", "", ErrBookingNotFound},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := storeWith(farBooking, nearBooking, doneBooking)

			d, err := newTestService(store).CancelByCode(context.Background(), tt.code, tt.reason)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil && (d.Booking.Status != StatusCancelled || d.CanCancel || d.Booking.CancelReason != tt.reason) {
				t.Errorf("unexpected result: %+v", d.Booking)
			}
		})
	}
}

func TestCancelByCode_ReasonTooLong(t *testing.T) {
	_, err := newTestService(storeWith(farBooking)).CancelByCode(context.Background(), farBooking.Code, strings.Repeat("a", 201))

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Fields["reason"] == "" {
		t.Errorf("expected validation error on reason, got %v", err)
	}
}

type racingStore struct {
	*fakeStore
}

func (racingStore) CancelByCode(ctx context.Context, code, reason string, minStartAt time.Time) (Booking, error) {
	return Booking{}, errNotCancellable
}

func TestCancelByCode_StatusChangedConcurrently(t *testing.T) {
	svc := NewService(newTestAvailability(fakeTherapists{}, nil), racingStore{storeWith(farBooking)}, &fakeNotifier{})

	if _, err := svc.CancelByCode(context.Background(), farBooking.Code, ""); !errors.Is(err, ErrInvalidTransition) {
		t.Errorf("err = %v, want ErrInvalidTransition when the update matches no row", err)
	}
}
