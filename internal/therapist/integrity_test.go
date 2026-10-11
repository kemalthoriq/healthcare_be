package therapist

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

func wantActiveBookings(t *testing.T, err error) {
	t.Helper()
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Code != "HAS_ACTIVE_BOOKINGS" {
		t.Fatalf("err = %v, want HAS_ACTIVE_BOOKINGS", err)
	}
}

func activeBooking(therapistID int64, start time.Time) integrity.Booking {
	return integrity.Booking{Code: "LMR-AAAA-0001", TreatmentID: 1, TherapistID: therapistID, StartAt: start, EndAt: start.Add(time.Hour)}
}

func TestUpdate_DeactivateWithActiveBookings(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService()
	th, _ := svc.Create(ctx, validInput())
	svc.bookings = fakeBookings{activeBooking(th.ID, time.Now().Add(time.Hour))}

	in := validInput()
	in.IsActive = false
	_, err := svc.Update(ctx, th.ID, in)
	wantActiveBookings(t, err)
	if !store.items[th.ID].IsActive {
		t.Error("therapist must stay active when deactivation is rejected")
	}
}

func TestSetForTreatment_DetachWithActiveBookings(t *testing.T) {
	ctx := context.Background()
	svc, store := newTestService()
	a, _ := svc.Create(ctx, validInput())
	b, _ := svc.Create(ctx, Input{Name: "Rina", IsActive: true})
	svc.SetForTreatment(ctx, 1, []int64{a.ID, b.ID})
	svc.bookings = fakeBookings{activeBooking(a.ID, time.Now().Add(time.Hour))}

	if _, err := svc.SetForTreatment(ctx, 1, []int64{a.ID}); err != nil {
		t.Fatalf("detaching a therapist without bookings: %v", err)
	}

	_, err := svc.SetForTreatment(ctx, 1, []int64{})
	wantActiveBookings(t, err)
	if len(store.links[1]) != 1 {
		t.Errorf("links = %v, want unchanged", store.links[1])
	}
}

func TestSetSchedules_BookingOutsideNewShifts(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	th, _ := svc.Create(ctx, validInput())
	monday10 := time.Date(2026, 10, 12, 10, 0, 0, 0, integrity.Location)
	svc.bookings = fakeBookings{activeBooking(th.ID, monday10)}

	if _, err := svc.SetSchedules(ctx, th.ID, []Shift{{1, 9 * 60, 11 * 60}}); err != nil {
		t.Fatalf("booking fits exactly at shift end: %v", err)
	}

	tests := map[string][]Shift{
		"no shift that day":    {{2, 9 * 60, 17 * 60}},
		"shift ends too early": {{1, 9 * 60, 10*60 + 30}},
		"shift starts late":    {{1, 10*60 + 30, 17 * 60}},
		"empty schedule":       nil,
	}
	for name, shifts := range tests {
		t.Run(name, func(t *testing.T) {
			_, err := svc.SetSchedules(ctx, th.ID, shifts)
			wantActiveBookings(t, err)
		})
	}
}

func TestCreateTimeOff_OverlapsActiveBooking(t *testing.T) {
	ctx := context.Background()
	svc, _ := newTestService()
	th, _ := svc.Create(ctx, validInput())
	start := time.Now().Add(48 * time.Hour).Truncate(time.Hour)
	svc.bookings = fakeBookings{activeBooking(th.ID, start)}

	if _, err := svc.CreateTimeOff(ctx, th.ID, TimeOffInput{StartAt: start.Add(time.Hour), EndAt: start.Add(3 * time.Hour)}); err != nil {
		t.Fatalf("time off right after the booking: %v", err)
	}

	_, err := svc.CreateTimeOff(ctx, th.ID, TimeOffInput{StartAt: start.Add(30 * time.Minute), EndAt: start.Add(3 * time.Hour)})
	wantActiveBookings(t, err)
}
