package booking

import (
	"context"
	"time"
)

const (
	EventCreated     = "created"
	EventConfirmed   = "confirmed"
	EventRescheduled = "rescheduled"
	EventCancelled   = "cancelled"
)

type Event struct {
	Kind               string
	Code               string
	PatientName        string
	Email              string
	TreatmentName      string
	TherapistName      string
	StartAt            time.Time
	EndAt              time.Time
	PreviousStartAt    time.Time
	Reason             string
	CancelledByPatient bool
}

type notifier interface {
	Notify(ctx context.Context, e Event)
}

func (s *Service) notify(ctx context.Context, e Event) {
	if e.Email == "" {
		return
	}
	s.notifier.Notify(ctx, e)
}

func eventFor(kind string, b AdminBooking) Event {
	return Event{
		Kind:          kind,
		Code:          b.Code,
		PatientName:   b.PatientName,
		Email:         b.PatientEmail,
		TreatmentName: b.TreatmentName,
		TherapistName: b.TherapistName,
		StartAt:       b.StartAt,
		EndAt:         b.EndAt,
		Reason:        b.CancelReason,
	}
}
