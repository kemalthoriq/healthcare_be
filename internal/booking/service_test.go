package booking

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type fakeStore struct {
	results []error
	created []NewBooking
	codes   []string
	byCode  map[string]Booking
}

func (f *fakeStore) Create(ctx context.Context, code string, nb NewBooking) (Booking, error) {
	f.codes = append(f.codes, code)
	if len(f.results) > 0 {
		err := f.results[0]
		f.results = f.results[1:]
		if err != nil {
			return Booking{}, err
		}
	}
	f.created = append(f.created, nb)
	return Booking{ID: 1, Code: code, Status: StatusPending, StartAt: nb.StartAt, EndAt: nb.EndAt, Price: nb.Price}, nil
}

func validCreateInput() CreateInput {
	return CreateInput{
		TreatmentSlug: "facial-glow",
		TherapistSlug: "dr-sari",
		StartAt:       time.Date(2026, 10, 27, 10, 0, 0, 0, wib),
		Name:          "  Sari Dewi ",
		Phone:         "0812-3456-7890",
		Email:         " Sari@Example.com ",
		Consent:       true,
	}
}

func newTestService(store *fakeStore) *Service {
	return NewService(newTestAvailability(fakeTherapists{}, nil), store)
}

func TestCreate_Success(t *testing.T) {
	store := &fakeStore{}

	c, err := newTestService(store).Create(context.Background(), validCreateInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	nb := store.created[0]
	if nb.Phone != "6281234567890" || nb.PatientName != "Sari Dewi" || nb.Email != "sari@example.com" {
		t.Errorf("unexpected normalized input: %+v", nb)
	}
	if nb.TreatmentID != 1 || nb.TherapistID != 7 || nb.Price != 350000 {
		t.Errorf("unexpected references or price: %+v", nb)
	}
	if want := time.Date(2026, 10, 27, 11, 0, 0, 0, wib); !nb.EndAt.Equal(want) {
		t.Errorf("EndAt = %v, want %v", nb.EndAt, want)
	}
	if want := time.Date(2026, 10, 27, 11, 15, 0, 0, wib); !nb.BlockedUntil.Equal(want) {
		t.Errorf("BlockedUntil = %v, want %v", nb.BlockedUntil, want)
	}
	if c.Treatment.Slug != "facial-glow" || c.Therapist.Slug != "dr-sari" {
		t.Errorf("unexpected refs: %+v %+v", c.Treatment, c.Therapist)
	}
}

func TestCreate_Validation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*CreateInput)
		field  string
	}{
		{"missing name", func(in *CreateInput) { in.Name = " " }, "name"},
		{"bad phone", func(in *CreateInput) { in.Phone = "12345" }, "phone"},
		{"bad email", func(in *CreateInput) { in.Email = "bukan-email" }, "email"},
		{"no consent", func(in *CreateInput) { in.Consent = false }, "consent"},
		{"missing start", func(in *CreateInput) { in.StartAt = time.Time{} }, "start_at"},
		{"missing treatment", func(in *CreateInput) { in.TreatmentSlug = "" }, "treatment_slug"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validCreateInput()
			tt.modify(&in)

			_, err := newTestService(&fakeStore{}).Create(context.Background(), in)

			var appErr *apperr.Error
			if !errors.As(err, &appErr) || appErr.Fields[tt.field] == "" {
				t.Errorf("expected validation error on %q, got %v", tt.field, err)
			}
		})
	}
}

func TestCreate_RejectsStartOutsideSlots(t *testing.T) {
	in := validCreateInput()
	in.StartAt = time.Date(2026, 10, 27, 10, 15, 0, 0, wib)

	if _, err := newTestService(&fakeStore{}).Create(context.Background(), in); !errors.Is(err, ErrSlotUnavailable) {
		t.Errorf("err = %v, want ErrSlotUnavailable", err)
	}
}

func TestCreate_MapsSlotTaken(t *testing.T) {
	store := &fakeStore{results: []error{errSlotTaken}}

	if _, err := newTestService(store).Create(context.Background(), validCreateInput()); !errors.Is(err, ErrSlotTaken) {
		t.Errorf("err = %v, want ErrSlotTaken", err)
	}
}

func TestCreate_RetriesOnCodeCollision(t *testing.T) {
	store := &fakeStore{results: []error{errCodeTaken, errCodeTaken, nil}}

	if _, err := newTestService(store).Create(context.Background(), validCreateInput()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.codes) != 3 {
		t.Errorf("attempts = %d, want 3", len(store.codes))
	}
}
