package booking

import (
	"context"
	"testing"
	"time"
)

func TestNotify_Create(t *testing.T) {
	svc := newTestService(&fakeStore{})
	c, err := svc.Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := sentEvents(svc)
	if len(events) != 1 {
		t.Fatalf("events = %d, want 1", len(events))
	}
	e := events[0]
	if e.Kind != EventCreated || e.Email != "sari@example.com" || e.PatientName != "Sari Dewi" || e.TreatmentName != c.Treatment.Name || e.Code != c.Booking.Code {
		t.Errorf("event = %+v", e)
	}

	svc = newTestService(&fakeStore{})
	in := validCreateInput()
	in.Email = ""
	svc.Create(context.Background(), in)
	if len(sentEvents(svc)) != 0 {
		t.Error("booking without email must not notify")
	}
}

func TestNotify_Transition(t *testing.T) {
	tests := []struct {
		name     string
		booking  AdminBooking
		to       string
		reason   string
		wantKind string
	}{
		{"confirm", adminBooking(1, StatusPending, future), StatusConfirmed, "", EventConfirmed},
		{"cancel", adminBooking(1, StatusConfirmed, future), StatusCancelled, "Terapis sakit", EventCancelled},
		{"complete", adminBooking(1, StatusConfirmed, past), StatusCompleted, "", ""},
		{"no-show", adminBooking(1, StatusConfirmed, past), StatusNoShow, "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.booking.PatientEmail = "sari@example.com"
			svc := newTestService(adminStore(tt.booking))

			if _, err := svc.Transition(context.Background(), adminActor, 1, tt.to, tt.reason); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			events := sentEvents(svc)
			if tt.wantKind == "" {
				if len(events) != 0 {
					t.Errorf("events = %+v, want none", events)
				}
				return
			}
			if len(events) != 1 || events[0].Kind != tt.wantKind || events[0].Reason != tt.reason || events[0].TherapistName == "" {
				t.Errorf("events = %+v", events)
			}
		})
	}
}

func TestNotify_Reschedule(t *testing.T) {
	b := adminBooking(1, StatusConfirmed, future)
	b.PatientEmail = "sari@example.com"
	svc := newTestService(adminStore(b))
	newStart := future.Add(time.Hour)

	if _, err := svc.Reschedule(context.Background(), adminActor, 1, newStart); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := sentEvents(svc)
	if len(events) != 1 || events[0].Kind != EventRescheduled || !events[0].PreviousStartAt.Equal(future) || !events[0].StartAt.Equal(newStart) {
		t.Errorf("events = %+v", events)
	}
}

func TestNotify_CancelByPatient(t *testing.T) {
	b := farBooking
	b.ID = 1
	store := storeWith(b)
	store.admin = map[int64]AdminBooking{1: {Booking: b, PatientEmail: "sari@example.com"}}
	svc := newTestService(store)

	if _, err := svc.CancelByCode(context.Background(), b.Code, ""); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	events := sentEvents(svc)
	if len(events) != 1 || events[0].Kind != EventCancelled || !events[0].CancelledByPatient || events[0].Code != b.Code {
		t.Errorf("events = %+v", events)
	}
}
