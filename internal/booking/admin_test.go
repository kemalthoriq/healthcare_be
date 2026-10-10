package booking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

func (f *fakeStore) ListAdmin(ctx context.Context, filter AdminFilter, limit, offset int) ([]AdminBooking, int64, error) {
	var out []AdminBooking
	for _, b := range f.admin {
		if filter.Status == nil || b.Status == *filter.Status {
			out = append(out, b)
		}
	}
	return out, int64(len(out)), nil
}

func (f *fakeStore) GetAdmin(ctx context.Context, id int64) (AdminBooking, error) {
	b, ok := f.admin[id]
	if !ok {
		return AdminBooking{}, errNotFound
	}
	return b, nil
}

func (f *fakeStore) Transition(ctx context.Context, id int64, from, to, reason string) error {
	b, ok := f.admin[id]
	if !ok || b.Status != from {
		return errStatusChanged
	}
	b.Status = to
	if to == StatusCancelled {
		b.CancelReason = reason
	}
	f.admin[id] = b
	return nil
}

func (f *fakeStore) Reschedule(ctx context.Context, id int64, startAt, endAt, blockedUntil time.Time) error {
	b, ok := f.admin[id]
	if !ok || !isActive(b.Status) {
		return errStatusChanged
	}
	b.StartAt, b.EndAt, b.BlockedUntil = startAt, endAt, blockedUntil
	f.admin[id] = b
	return nil
}

func (f *fakeStore) SaveNote(ctx context.Context, bookingID, authorID int64, content string) error {
	b := f.admin[bookingID]
	b.TreatmentNote = content
	f.admin[bookingID] = b
	return nil
}

var (
	adminActor     = Actor{UserID: 1}
	therapistActor = Actor{UserID: 50, TherapistID: 7}
	otherTherapist = Actor{UserID: 51, TherapistID: 8}
)

func adminStore(bookings ...AdminBooking) *fakeStore {
	f := &fakeStore{admin: map[int64]AdminBooking{}}
	for _, b := range bookings {
		f.admin[b.ID] = b
	}
	return f
}

func adminBooking(id int64, status string, start time.Time) AdminBooking {
	return AdminBooking{Booking: Booking{ID: id, Status: status, TreatmentID: 1, TherapistID: 7, StartAt: start}}
}

var (
	past   = time.Date(2026, 10, 20, 8, 0, 0, 0, wib)
	future = time.Date(2026, 10, 27, 10, 0, 0, 0, wib)
)

func TestCanTransition(t *testing.T) {
	allowed := [][2]string{
		{StatusPending, StatusConfirmed}, {StatusPending, StatusCancelled},
		{StatusConfirmed, StatusCompleted}, {StatusConfirmed, StatusNoShow}, {StatusConfirmed, StatusCancelled},
	}
	for _, p := range allowed {
		if !canTransition(p[0], p[1]) {
			t.Errorf("%s → %s should be allowed", p[0], p[1])
		}
	}

	denied := [][2]string{
		{StatusPending, StatusCompleted}, {StatusPending, StatusNoShow},
		{StatusCompleted, StatusCancelled}, {StatusCancelled, StatusConfirmed}, {StatusNoShow, StatusCompleted},
	}
	for _, p := range denied {
		if canTransition(p[0], p[1]) {
			t.Errorf("%s → %s should be denied", p[0], p[1])
		}
	}
}

func TestTransition(t *testing.T) {
	tests := []struct {
		name     string
		booking  AdminBooking
		to       string
		reason   string
		wantErr  error
		wantCode string
	}{
		{"confirm pending", adminBooking(1, StatusPending, future), StatusConfirmed, "", nil, ""},
		{"complete after start", adminBooking(1, StatusConfirmed, past), StatusCompleted, "", nil, ""},
		{"no-show after start", adminBooking(1, StatusConfirmed, past), StatusNoShow, "", nil, ""},
		{"cancel with reason", adminBooking(1, StatusConfirmed, future), StatusCancelled, "Terapis sakit", nil, ""},
		{"complete before start", adminBooking(1, StatusConfirmed, future), StatusCompleted, "", ErrBookingNotStarted, ""},
		{"complete pending", adminBooking(1, StatusPending, past), StatusCompleted, "", nil, "INVALID_TRANSITION"},
		{"cancel without reason", adminBooking(1, StatusPending, future), StatusCancelled, " ", nil, "VALIDATION_ERROR"},
		{"unknown booking", adminBooking(2, StatusPending, future), StatusConfirmed, "", ErrBookingNotFound, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := adminStore(tt.booking)
			svc := newTestService(store)

			got, err := svc.Transition(context.Background(), adminActor, 1, tt.to, tt.reason)

			switch {
			case tt.wantErr != nil:
				if !errors.Is(err, tt.wantErr) {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
			case tt.wantCode != "":
				var appErr *apperr.Error
				if !errors.As(err, &appErr) || appErr.Code != tt.wantCode {
					t.Fatalf("err = %v, want code %s", err, tt.wantCode)
				}
			default:
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				if got.Status != tt.to || got.TreatmentName != "Facial Glow" {
					t.Errorf("got status %q treatment %q", got.Status, got.TreatmentName)
				}
			}
		})
	}
}

func TestTransition_StatusChangedConcurrently(t *testing.T) {
	store := adminStore(adminBooking(1, StatusPending, future))
	svc := newTestService(store)

	racing := racingTransitionStore{store}
	svc.store = racing

	if _, err := svc.Transition(context.Background(), adminActor, 1, StatusConfirmed, ""); !errors.Is(err, ErrStatusChanged) {
		t.Errorf("err = %v, want ErrStatusChanged", err)
	}
}

type racingTransitionStore struct {
	*fakeStore
}

func (racingTransitionStore) Transition(ctx context.Context, id int64, from, to, reason string) error {
	return errStatusChanged
}

func TestListAdmin_FlagsOverduePending(t *testing.T) {
	store := adminStore(adminBooking(1, StatusPending, past), adminBooking(2, StatusPending, future), adminBooking(3, StatusConfirmed, past))

	items, total, err := newTestService(store).ListAdmin(context.Background(), adminActor, AdminFilter{}, 20, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 3 {
		t.Fatalf("total = %d, want 3", total)
	}

	for _, b := range items {
		want := b.ID == 1
		if b.NeedsAction != want {
			t.Errorf("booking %d NeedsAction = %v, want %v", b.ID, b.NeedsAction, want)
		}
		if b.TherapistName != "dr. Sari" {
			t.Errorf("booking %d TherapistName = %q", b.ID, b.TherapistName)
		}
	}
}

func TestTherapistActor(t *testing.T) {
	svc := newTestService(adminStore())

	actor, err := svc.TherapistActor(context.Background(), 50)
	if err != nil || actor.TherapistID != 7 {
		t.Fatalf("TherapistActor(50) = %+v, %v; want therapist 7", actor, err)
	}

	var appErr *apperr.Error
	if _, err := svc.TherapistActor(context.Background(), 99); !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Errorf("unlinked user: err = %v, want FORBIDDEN", err)
	}
}

func TestTherapistScope(t *testing.T) {
	store := adminStore(adminBooking(1, StatusConfirmed, past))
	svc := newTestService(store)
	ctx := context.Background()

	if _, err := svc.GetAdmin(ctx, therapistActor, 1); err != nil {
		t.Errorf("own booking: unexpected error %v", err)
	}

	var appErr *apperr.Error
	if _, err := svc.GetAdmin(ctx, otherTherapist, 1); !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Errorf("other therapist's booking: err = %v, want FORBIDDEN", err)
	}
	if _, err := svc.Transition(ctx, otherTherapist, 1, StatusCompleted, ""); !errors.As(err, &appErr) || appErr.Code != "FORBIDDEN" {
		t.Errorf("complete other therapist's booking: err = %v, want FORBIDDEN", err)
	}
	if got, err := svc.Transition(ctx, therapistActor, 1, StatusCompleted, ""); err != nil || got.Status != StatusCompleted {
		t.Errorf("complete own booking: %v, %v", got.Status, err)
	}
}

func TestReschedule(t *testing.T) {
	newStart := time.Date(2026, 10, 27, 11, 0, 0, 0, wib)

	tests := []struct {
		name    string
		booking AdminBooking
		start   time.Time
		wantErr error
	}{
		{"moves to a free slot", adminBooking(1, StatusConfirmed, future), newStart, nil},
		{"rejects time outside slots", adminBooking(1, StatusConfirmed, future), newStart.Add(15 * time.Minute), ErrSlotUnavailable},
		{"rejects completed booking", adminBooking(1, StatusCompleted, past), newStart, ErrNotRescheduleable},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := adminStore(tt.booking)

			got, err := newTestService(store).Reschedule(context.Background(), adminActor, 1, tt.start)

			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if tt.wantErr == nil {
				if !got.StartAt.Equal(newStart) || !got.EndAt.Equal(newStart.Add(time.Hour)) || !got.BlockedUntil.Equal(newStart.Add(75*time.Minute)) {
					t.Errorf("got %v–%v blocked until %v", got.StartAt, got.EndAt, got.BlockedUntil)
				}
			}
		})
	}
}

func TestSaveNote(t *testing.T) {
	store := adminStore(adminBooking(1, StatusConfirmed, past))
	svc := newTestService(store)

	got, err := svc.SaveNote(context.Background(), therapistActor, 1, "  Kulit sensitif, pakai serum ringan.  ")
	if err != nil || got.TreatmentNote != "Kulit sensitif, pakai serum ringan." {
		t.Fatalf("got %q, %v", got.TreatmentNote, err)
	}

	var appErr *apperr.Error
	if _, err := svc.SaveNote(context.Background(), therapistActor, 1, " "); !errors.As(err, &appErr) || appErr.Fields["content"] == "" {
		t.Errorf("empty note: err = %v, want validation error", err)
	}
}
