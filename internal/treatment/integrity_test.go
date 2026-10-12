package treatment

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/integrity"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type fakeBookings []integrity.Booking

func (f fakeBookings) ActiveByTreatment(ctx context.Context, treatmentID int64) ([]integrity.Booking, error) {
	var out []integrity.Booking
	for _, b := range f {
		if b.TreatmentID == treatmentID {
			out = append(out, b)
		}
	}
	return out, nil
}

func (f fakeBookings) ActiveByTherapist(ctx context.Context, therapistID int64) ([]integrity.Booking, error) {
	var out []integrity.Booking
	for _, b := range f {
		if b.TherapistID == therapistID {
			out = append(out, b)
		}
	}
	return out, nil
}

func TestUpdate_DeactivateWithActiveBookings(t *testing.T) {
	ctx := context.Background()
	store := newFakeStore()
	svc := NewService(store, fakeBookings{}, noCloudinary)
	created, _ := svc.Create(ctx, validInput())
	svc.bookings = fakeBookings{{Code: "LMR-AAAA-0001", TreatmentID: created.ID, StartAt: time.Now().Add(time.Hour)}}

	in := validInput()
	in.Name = "Facial Glow Baru"
	if _, err := svc.Update(ctx, created.ID, in); err != nil {
		t.Fatalf("editing an active treatment should still work: %v", err)
	}

	in.IsActive = false
	_, err := svc.Update(ctx, created.ID, in)
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Code != "HAS_ACTIVE_BOOKINGS" {
		t.Fatalf("err = %v, want HAS_ACTIVE_BOOKINGS", err)
	}
	if !store.items[created.ID].IsActive {
		t.Error("treatment must stay active when deactivation is rejected")
	}

	svc.bookings = fakeBookings{}
	if _, err := svc.Update(ctx, created.ID, in); err != nil {
		t.Errorf("deactivating without active bookings: %v", err)
	}
}
