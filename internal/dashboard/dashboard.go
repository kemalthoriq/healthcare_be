package dashboard

import (
	"context"
	"time"
)

type PendingBooking struct {
	ID            int64
	Code          string
	PatientName   string
	TreatmentName string
	TherapistName string
	StartAt       time.Time
	NeedsAction   bool
}

type TopTreatment struct {
	Name  string
	Count int64
}

type Summary struct {
	TodayCount       int64
	PendingCount     int64
	NeedsActionCount int64
	PendingUpcoming  []PendingBooking
	RevenueMonth     int64
	TopTreatments    []TopTreatment
}

type Range struct {
	Start time.Time
	End   time.Time
}

type Counts struct {
	Today       int64
	Pending     int64
	NeedsAction int64
}

type store interface {
	BookingCounts(ctx context.Context, today Range, now time.Time) (Counts, error)
	PendingBookings(ctx context.Context) ([]PendingBooking, error)
	Revenue(ctx context.Context, month Range) (int64, error)
	TopTreatments(ctx context.Context, month Range) ([]TopTreatment, error)
}

type Service struct {
	store store
	loc   *time.Location
	now   func() time.Time
}

func NewService(store store, loc *time.Location) *Service {
	return &Service{store: store, loc: loc, now: time.Now}
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	now := s.now()
	local := now.In(s.loc)
	dayStart := time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, s.loc)
	monthStart := time.Date(local.Year(), local.Month(), 1, 0, 0, 0, 0, s.loc)
	today := Range{Start: dayStart, End: dayStart.AddDate(0, 0, 1)}
	month := Range{Start: monthStart, End: monthStart.AddDate(0, 1, 0)}

	counts, err := s.store.BookingCounts(ctx, today, now)
	if err != nil {
		return Summary{}, err
	}
	pending, err := s.store.PendingBookings(ctx)
	if err != nil {
		return Summary{}, err
	}
	for i := range pending {
		pending[i].NeedsAction = pending[i].StartAt.Before(now)
	}
	revenue, err := s.store.Revenue(ctx, month)
	if err != nil {
		return Summary{}, err
	}
	top, err := s.store.TopTreatments(ctx, month)
	if err != nil {
		return Summary{}, err
	}

	return Summary{
		TodayCount:       counts.Today,
		PendingCount:     counts.Pending,
		NeedsActionCount: counts.NeedsAction,
		PendingUpcoming:  pending,
		RevenueMonth:     revenue,
		TopTreatments:    top,
	}, nil
}
