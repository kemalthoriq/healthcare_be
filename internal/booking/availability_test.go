package booking

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/setting"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

type fakeTreatments struct{}

func (fakeTreatments) GetActiveBySlug(ctx context.Context, slug string) (treatment.Treatment, error) {
	if slug != "facial-glow" {
		return treatment.Treatment{}, treatment.ErrNotFound
	}
	return treatment.Treatment{ID: 1, Slug: slug, Price: 350000, DurationMinutes: 60, IsActive: true}, nil
}

type fakeTherapists struct {
	timeOffs []therapist.TimeOff
}

func (fakeTherapists) GetActiveBySlug(ctx context.Context, slug string) (therapist.Therapist, error) {
	switch slug {
	case "dr-sari":
		return therapist.Therapist{ID: 7, Slug: slug, IsActive: true}, nil
	case "rina":
		return therapist.Therapist{ID: 8, Slug: slug, IsActive: true}, nil
	}
	return therapist.Therapist{}, therapist.ErrNotFound
}

func (fakeTherapists) ListActiveForTreatment(ctx context.Context, treatmentSlug string) ([]therapist.Therapist, error) {
	return []therapist.Therapist{{ID: 7}}, nil
}

func (fakeTherapists) ListSchedules(ctx context.Context, therapistID int64) ([]therapist.Shift, error) {
	return []therapist.Shift{{DayOfWeek: int(time.Tuesday), Start: 600, End: 780}}, nil
}

func (f fakeTherapists) ListUpcomingTimeOff(ctx context.Context, therapistID int64) ([]therapist.TimeOff, error) {
	return f.timeOffs, nil
}

type fakeSettings struct{}

func (fakeSettings) Get(ctx context.Context) (setting.Settings, error) {
	return setting.Settings{BufferMinutes: 15, MinBookingLeadMinutes: 120, MaxBookingDaysAhead: 30}, nil
}

type fakeBookings struct {
	busy []Interval
}

func (f fakeBookings) ListActiveRanges(ctx context.Context, therapistID int64, from, to time.Time) ([]Interval, error) {
	return f.busy, nil
}

func newTestAvailability(therapists fakeTherapists, busy []Interval) *Availability {
	a := NewAvailability(fakeTreatments{}, therapists, fakeSettings{}, fakeBookings{busy: busy}, wib)
	a.now = func() time.Time { return time.Date(2026, 10, 20, 9, 0, 0, 0, wib) }
	return a
}

func day(s string) time.Time {
	t, _ := time.ParseInLocation("2006-01-02", s, wib)
	return t
}

func TestSlots(t *testing.T) {
	tests := []struct {
		name string
		date string
		want []string
	}{
		{"today respects minimum lead time", "2026-10-20", []string{"11:00", "11:30", "12:00"}},
		{"next tuesday has full shift", "2026-10-27", []string{"10:00", "10:30", "11:00", "11:30", "12:00"}},
		{"day without shift", "2026-10-21", nil},
		{"past day", "2026-10-13", nil},
		{"beyond max days ahead", "2026-11-24", nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			slots, err := newTestAvailability(fakeTherapists{}, nil).Slots(context.Background(), "facial-glow", "dr-sari", day(tt.date))
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got := clocks(slots); !slices.Equal(got, tt.want) && !(len(got) == 0 && len(tt.want) == 0) {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSlots_TimeOffBlocksSlots(t *testing.T) {
	off := therapist.TimeOff{
		StartAt: time.Date(2026, 10, 27, 0, 0, 0, 0, wib),
		EndAt:   time.Date(2026, 10, 28, 0, 0, 0, 0, wib),
	}

	slots, err := newTestAvailability(fakeTherapists{timeOffs: []therapist.TimeOff{off}}, nil).
		Slots(context.Background(), "facial-glow", "dr-sari", day("2026-10-27"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(slots) != 0 {
		t.Errorf("got %v, want no slots during time off", clocks(slots))
	}
}

func TestSlots_ExistingBookingBlocksSlots(t *testing.T) {
	existing := Interval{
		Start: time.Date(2026, 10, 27, 11, 0, 0, 0, wib),
		End:   time.Date(2026, 10, 27, 12, 15, 0, 0, wib),
	}

	slots, err := newTestAvailability(fakeTherapists{}, []Interval{existing}).
		Slots(context.Background(), "facial-glow", "dr-sari", day("2026-10-27"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := clocks(slots); len(got) != 0 {
		t.Errorf("got %v, want no slots: every start overlaps 11:00-12:15 once our buffer is included", got)
	}
}

func TestSlots_Errors(t *testing.T) {
	a := newTestAvailability(fakeTherapists{}, nil)
	ctx := context.Background()

	if _, err := a.Slots(ctx, "tidak-ada", "dr-sari", day("2026-10-27")); !errors.Is(err, treatment.ErrNotFound) {
		t.Errorf("unknown treatment: err = %v", err)
	}
	if _, err := a.Slots(ctx, "facial-glow", "tidak-ada", day("2026-10-27")); !errors.Is(err, therapist.ErrNotFound) {
		t.Errorf("unknown therapist: err = %v", err)
	}
	if _, err := a.Slots(ctx, "facial-glow", "rina", day("2026-10-27")); !errors.Is(err, ErrTherapistNotAssigned) {
		t.Errorf("unassigned therapist: err = %v", err)
	}
}

func TestDates(t *testing.T) {
	dates, err := newTestAvailability(fakeTherapists{}, nil).Dates(context.Background(), "facial-glow", "dr-sari", day("2026-10-01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := make([]string, 0, len(dates))
	for _, d := range dates {
		got = append(got, d.Format("2006-01-02"))
	}
	if want := []string{"2026-10-20", "2026-10-27"}; !slices.Equal(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
}
