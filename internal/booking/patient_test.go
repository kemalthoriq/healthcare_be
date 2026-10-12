package booking

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type fakePatientStore struct {
	patients     map[int64]Patient
	bookings     map[int64][]AdminBooking
	namePattern  string
	phonePattern string
}

func (f *fakePatientStore) ListPatients(ctx context.Context, namePattern, phonePattern string, limit, offset int) ([]Patient, int64, error) {
	f.namePattern, f.phonePattern = namePattern, phonePattern
	return nil, 0, nil
}

func (f *fakePatientStore) GetPatient(ctx context.Context, id int64) (Patient, error) {
	p, ok := f.patients[id]
	if !ok {
		return Patient{}, errNotFound
	}
	return p, nil
}

func (f *fakePatientStore) ListPatientBookings(ctx context.Context, patientID int64) ([]AdminBooking, error) {
	return f.bookings[patientID], nil
}

func TestPatientList_Patterns(t *testing.T) {
	tests := []struct {
		query     string
		wantName  string
		wantPhone string
	}{
		{"", "", ""},
		{"  Sari ", "%Sari%", ""},
		{"50%_off\\", `%50\%\_off\\%`, ""},
		{"0812-345", "%0812-345%", "%62812345%"},
		{"+62 812", "%+62 812%", "%62812%"},
		{"81", "%81%", ""},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			store := &fakePatientStore{}
			if _, _, err := NewPatientService(store, nil).List(context.Background(), tt.query, 20, 0); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if store.namePattern != tt.wantName || store.phonePattern != tt.wantPhone {
				t.Errorf("patterns = %q, %q; want %q, %q", store.namePattern, store.phonePattern, tt.wantName, tt.wantPhone)
			}
		})
	}
}

func TestPatientList_QueryTooLong(t *testing.T) {
	_, _, err := NewPatientService(&fakePatientStore{}, nil).List(context.Background(), strings.Repeat("a", 101), 20, 0)

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Fields["q"] == "" {
		t.Errorf("err = %v, want validation error on q", err)
	}
}

func TestPatientGet(t *testing.T) {
	start := time.Date(2026, 10, 1, 10, 0, 0, 0, wib)
	store := &fakePatientStore{
		patients: map[int64]Patient{7: {ID: 7, Name: "Sari Dewi", Phone: "6281234567890"}},
		bookings: map[int64][]AdminBooking{7: {
			{Booking: Booking{ID: 1, TreatmentID: 1, TherapistID: 2, StartAt: start, Status: StatusCompleted}, TreatmentNote: "Kulit sensitif"},
		}},
	}
	svc := NewPatientService(store, newTestService(&fakeStore{}))

	d, err := svc.Get(context.Background(), 7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(d.Bookings) != 1 || d.Bookings[0].TreatmentName != "Facial Glow" || d.Bookings[0].TherapistName != "dr. Sari" {
		t.Errorf("bookings = %+v, want names filled", d.Bookings)
	}

	if _, err := svc.Get(context.Background(), 99); !errors.Is(err, ErrPatientNotFound) {
		t.Errorf("err = %v, want ErrPatientNotFound", err)
	}
}
