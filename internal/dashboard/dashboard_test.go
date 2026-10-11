package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"
)

var wib = time.FixedZone("WIB", 7*60*60)

type fakeStore struct {
	today, month Range
	now          time.Time
	pending      []PendingBooking
	err          error
}

func (f *fakeStore) BookingCounts(ctx context.Context, today Range, now time.Time) (Counts, error) {
	f.today, f.now = today, now
	return Counts{Today: 4, Pending: 3, NeedsAction: 1}, f.err
}

func (f *fakeStore) PendingBookings(ctx context.Context) ([]PendingBooking, error) {
	return f.pending, nil
}

func (f *fakeStore) Revenue(ctx context.Context, month Range) (int64, error) {
	f.month = month
	return 750000, nil
}

func (f *fakeStore) TopTreatments(ctx context.Context, month Range) ([]TopTreatment, error) {
	return []TopTreatment{{Name: "Facial Glow", Count: 7}}, nil
}

func TestSummary_UsesClinicDayAndMonth(t *testing.T) {
	now := time.Date(2026, 10, 31, 18, 30, 0, 0, time.UTC)
	store := &fakeStore{pending: []PendingBooking{
		{ID: 1, StartAt: now.Add(-time.Hour)},
		{ID: 2, StartAt: now.Add(time.Hour)},
	}}
	svc := NewService(store, wib)
	svc.now = func() time.Time { return now }

	s, err := svc.Summary(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !store.today.Start.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, wib)) || !store.today.End.Equal(time.Date(2026, 11, 2, 0, 0, 0, 0, wib)) {
		t.Errorf("today = %v – %v, want 1 Nov WIB (31 Oct 18:30 UTC is 1 Nov in WIB)", store.today.Start, store.today.End)
	}
	if !store.month.Start.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, wib)) || !store.month.End.Equal(time.Date(2026, 12, 1, 0, 0, 0, 0, wib)) {
		t.Errorf("month = %v – %v, want November WIB", store.month.Start, store.month.End)
	}
	if !s.PendingUpcoming[0].NeedsAction || s.PendingUpcoming[1].NeedsAction {
		t.Errorf("needs_action flags = %v, %v", s.PendingUpcoming[0].NeedsAction, s.PendingUpcoming[1].NeedsAction)
	}
	if s.TodayCount != 4 || s.PendingCount != 3 || s.NeedsActionCount != 1 || s.RevenueMonth != 750000 || s.TopTreatments[0].Count != 7 {
		t.Errorf("summary = %+v", s)
	}
}

func TestSummary_PropagatesErrors(t *testing.T) {
	svc := NewService(&fakeStore{err: errors.New("db down")}, wib)
	if _, err := svc.Summary(context.Background()); err == nil {
		t.Error("expected error")
	}
}
