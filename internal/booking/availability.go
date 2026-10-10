package booking

import (
	"context"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

const slotStep = 30 * time.Minute

var ErrTherapistNotAssigned = apperr.Unprocessable("THERAPIST_NOT_ASSIGNED", "Terapis ini tidak menangani treatment yang dipilih.")

type treatmentFinder interface {
	GetActiveBySlug(ctx context.Context, slug string) (treatment.Treatment, error)
}

type therapistFinder interface {
	GetActiveBySlug(ctx context.Context, slug string) (therapist.Therapist, error)
	ListActiveForTreatment(ctx context.Context, treatmentSlug string) ([]therapist.Therapist, error)
	ListSchedules(ctx context.Context, therapistID int64) ([]therapist.Shift, error)
	ListUpcomingTimeOff(ctx context.Context, therapistID int64) ([]therapist.TimeOff, error)
}

type settingsGetter interface {
	Get(ctx context.Context) (setting.Settings, error)
}

type Availability struct {
	treatments treatmentFinder
	therapists therapistFinder
	settings   settingsGetter
	loc        *time.Location
	now        func() time.Time
}

func NewAvailability(treatments treatmentFinder, therapists therapistFinder, settings settingsGetter, loc *time.Location) *Availability {
	return &Availability{treatments: treatments, therapists: therapists, settings: settings, loc: loc, now: time.Now}
}

type plan struct {
	duration time.Duration
	buffer   time.Duration
	shifts   []therapist.Shift
	busy     []Interval
	earliest time.Time
	firstDay time.Time
	lastDay  time.Time
}

func (a *Availability) Slots(ctx context.Context, treatmentSlug, therapistSlug string, day time.Time) ([]time.Time, error) {
	p, err := a.load(ctx, treatmentSlug, therapistSlug)
	if err != nil {
		return nil, err
	}
	return p.slotsOn(day), nil
}

func (a *Availability) Dates(ctx context.Context, treatmentSlug, therapistSlug string, month time.Time) ([]time.Time, error) {
	p, err := a.load(ctx, treatmentSlug, therapistSlug)
	if err != nil {
		return nil, err
	}

	var dates []time.Time
	for day := month; day.Month() == month.Month(); day = day.AddDate(0, 0, 1) {
		if len(p.slotsOn(day)) > 0 {
			dates = append(dates, day)
		}
	}
	return dates, nil
}

func (a *Availability) load(ctx context.Context, treatmentSlug, therapistSlug string) (plan, error) {
	tr, err := a.treatments.GetActiveBySlug(ctx, treatmentSlug)
	if err != nil {
		return plan{}, err
	}

	th, err := a.therapists.GetActiveBySlug(ctx, therapistSlug)
	if err != nil {
		return plan{}, err
	}

	assigned, err := a.therapists.ListActiveForTreatment(ctx, treatmentSlug)
	if err != nil {
		return plan{}, err
	}
	if !containsTherapist(assigned, th.ID) {
		return plan{}, ErrTherapistNotAssigned
	}

	shifts, err := a.therapists.ListSchedules(ctx, th.ID)
	if err != nil {
		return plan{}, err
	}

	timeOffs, err := a.therapists.ListUpcomingTimeOff(ctx, th.ID)
	if err != nil {
		return plan{}, err
	}

	st, err := a.settings.Get(ctx)
	if err != nil {
		return plan{}, err
	}

	busy := make([]Interval, 0, len(timeOffs))
	for _, off := range timeOffs {
		busy = append(busy, Interval{Start: off.StartAt, End: off.EndAt})
	}

	now := a.now().In(a.loc)
	today := StartOfDay(now)

	return plan{
		duration: time.Duration(tr.DurationMinutes) * time.Minute,
		buffer:   time.Duration(st.BufferMinutes) * time.Minute,
		shifts:   shifts,
		busy:     busy,
		earliest: now.Add(time.Duration(st.MinBookingLeadMinutes) * time.Minute),
		firstDay: today,
		lastDay:  today.AddDate(0, 0, int(st.MaxBookingDaysAhead)),
	}, nil
}

func (p plan) slotsOn(day time.Time) []time.Time {
	if day.Before(p.firstDay) || day.After(p.lastDay) {
		return nil
	}

	var shifts []Interval
	for _, s := range p.shifts {
		if s.DayOfWeek == int(day.Weekday()) {
			shifts = append(shifts, Interval{
				Start: day.Add(time.Duration(s.Start) * time.Minute),
				End:   day.Add(time.Duration(s.End) * time.Minute),
			})
		}
	}

	return AvailableSlots(SlotInput{
		Shifts:        shifts,
		Busy:          p.busy,
		Duration:      p.duration,
		Buffer:        p.buffer,
		Step:          slotStep,
		EarliestStart: p.earliest,
	})
}

func StartOfDay(t time.Time) time.Time {
	y, m, d := t.Date()
	return time.Date(y, m, d, 0, 0, 0, 0, t.Location())
}

func containsTherapist(list []therapist.Therapist, id int64) bool {
	for _, th := range list {
		if th.ID == id {
			return true
		}
	}
	return false
}
