package setting

import (
	"context"
	"errors"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type fakeStore struct {
	saved  Settings
	clinic Clinic
}

func (f *fakeStore) Get(ctx context.Context) (Settings, error) {
	return f.saved, nil
}

func (f *fakeStore) Update(ctx context.Context, s Settings) (Settings, error) {
	f.saved = s
	return s, nil
}

func (f *fakeStore) GetClinic(ctx context.Context) (Clinic, error) {
	return f.clinic, nil
}

func (f *fakeStore) UpdateClinic(ctx context.Context, c Clinic) (Clinic, error) {
	f.clinic = c
	return c, nil
}

func validSettings() Settings {
	return Settings{BufferMinutes: 15, MinBookingLeadMinutes: 120, MaxBookingDaysAhead: 30, CancelDeadlineHours: 24}
}

func TestUpdate_Valid(t *testing.T) {
	store := &fakeStore{}

	if _, err := NewService(store).Update(context.Background(), validSettings()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if store.saved != validSettings() {
		t.Errorf("saved = %+v", store.saved)
	}
}

func TestUpdate_Validation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Settings)
		field  string
	}{
		{"negative buffer", func(s *Settings) { s.BufferMinutes = -5 }, "buffer_minutes"},
		{"buffer not multiple of 5", func(s *Settings) { s.BufferMinutes = 12 }, "buffer_minutes"},
		{"lead too long", func(s *Settings) { s.MinBookingLeadMinutes = 10081 }, "min_booking_lead_minutes"},
		{"max days zero", func(s *Settings) { s.MaxBookingDaysAhead = 0 }, "max_booking_days_ahead"},
		{"cancel deadline too long", func(s *Settings) { s.CancelDeadlineHours = 169 }, "cancel_deadline_hours"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validSettings()
			tt.modify(&s)

			_, err := NewService(&fakeStore{}).Update(context.Background(), s)

			var appErr *apperr.Error
			if !errors.As(err, &appErr) || appErr.Fields[tt.field] == "" {
				t.Errorf("expected validation error on %q, got %v", tt.field, err)
			}
		})
	}
}
