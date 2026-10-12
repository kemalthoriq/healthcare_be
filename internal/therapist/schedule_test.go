package therapist

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

func (f *fakeStore) ListSchedules(ctx context.Context, therapistID int64) ([]Shift, error) {
	return f.schedules[therapistID], nil
}

func (f *fakeStore) ReplaceSchedules(ctx context.Context, therapistID int64, shifts []Shift) error {
	f.schedules[therapistID] = shifts
	return nil
}

func (f *fakeStore) ListUpcomingTimeOff(ctx context.Context, therapistID int64) ([]TimeOff, error) {
	var out []TimeOff
	for _, t := range f.timeOffs {
		if t.TherapistID == therapistID {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) CreateTimeOff(ctx context.Context, therapistID int64, in TimeOffInput) (TimeOff, error) {
	f.nextID++
	t := TimeOff{ID: f.nextID, TherapistID: therapistID, StartAt: in.StartAt, EndAt: in.EndAt, Reason: in.Reason}
	f.timeOffs[t.ID] = t
	return t, nil
}

func (f *fakeStore) DeleteTimeOff(ctx context.Context, therapistID, timeOffID int64) error {
	t, ok := f.timeOffs[timeOffID]
	if !ok || t.TherapistID != therapistID {
		return errTimeOffNotFound
	}
	delete(f.timeOffs, timeOffID)
	return nil
}

func TestParseClock(t *testing.T) {
	valid := map[string]int{"00:00": 0, "09:30": 570, "23:59": 1439}
	for s, want := range valid {
		if got, ok := ParseClock(s); !ok || got != want {
			t.Errorf("ParseClock(%q) = %d, %v; want %d, true", s, got, ok, want)
		}
	}

	for _, s := range []string{"", "9:30", "24:00", "12:60", "12-30", "ab:cd", "12:300"} {
		if _, ok := ParseClock(s); ok {
			t.Errorf("ParseClock(%q) should fail", s)
		}
	}

	if got := FormatClock(570); got != "09:30" {
		t.Errorf("FormatClock(570) = %q, want 09:30", got)
	}
}

func TestValidateShifts(t *testing.T) {
	tests := []struct {
		name    string
		shifts  []Shift
		wantKey string
	}{
		{"valid split day", []Shift{{1, 600, 780}, {1, 840, 1080}}, ""},
		{"touching shifts are fine", []Shift{{1, 600, 780}, {1, 780, 900}}, ""},
		{"same time on different days", []Shift{{1, 600, 780}, {2, 600, 780}}, ""},
		{"day out of range", []Shift{{7, 600, 780}}, "shifts[0]"},
		{"not on half hour", []Shift{{1, 615, 780}}, "shifts[0]"},
		{"end before start", []Shift{{1, 780, 600}}, "shifts[0]"},
		{"overlap same day", []Shift{{1, 840, 1080}, {1, 600, 900}}, "shifts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fields := validateShifts(tt.shifts)
			if tt.wantKey == "" && len(fields) > 0 {
				t.Errorf("unexpected errors: %v", fields)
			}
			if tt.wantKey != "" && fields[tt.wantKey] == "" {
				t.Errorf("expected error on %q, got %v", tt.wantKey, fields)
			}
		})
	}
}

func TestSetSchedules_UnknownTherapist(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.SetSchedules(context.Background(), 99, nil); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}

func TestCreateTimeOff_Validation(t *testing.T) {
	now := time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC)
	svc, _ := newTestService()
	svc.now = func() time.Time { return now }
	th, _ := svc.Create(context.Background(), validInput())

	tests := []struct {
		name    string
		in      TimeOffInput
		wantKey string
	}{
		{"valid full day", TimeOffInput{StartAt: now.Add(24 * time.Hour), EndAt: now.Add(48 * time.Hour)}, ""},
		{"missing start", TimeOffInput{EndAt: now.Add(time.Hour)}, "start_at"},
		{"end before start", TimeOffInput{StartAt: now.Add(2 * time.Hour), EndAt: now.Add(time.Hour)}, "end_at"},
		{"already ended", TimeOffInput{StartAt: now.Add(-3 * time.Hour), EndAt: now.Add(-time.Hour)}, "end_at"},
		{"longer than 30 days", TimeOffInput{StartAt: now, EndAt: now.Add(31 * 24 * time.Hour)}, "end_at"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := svc.CreateTimeOff(context.Background(), th.ID, tt.in)

			if tt.wantKey == "" {
				if err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
				return
			}
			var appErr *apperr.Error
			if !errors.As(err, &appErr) || appErr.Fields[tt.wantKey] == "" {
				t.Errorf("expected validation error on %q, got %v", tt.wantKey, err)
			}
		})
	}
}

func TestDeleteTimeOff_WrongTherapist(t *testing.T) {
	svc, _ := newTestService()
	th, _ := svc.Create(context.Background(), validInput())
	off, _ := svc.CreateTimeOff(context.Background(), th.ID, TimeOffInput{StartAt: time.Now().Add(time.Hour), EndAt: time.Now().Add(2 * time.Hour)})

	if err := svc.DeleteTimeOff(context.Background(), th.ID+100, off.ID); !errors.Is(err, ErrTimeOffNotFound) {
		t.Errorf("err = %v, want ErrTimeOffNotFound", err)
	}
	if err := svc.DeleteTimeOff(context.Background(), th.ID, off.ID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}
