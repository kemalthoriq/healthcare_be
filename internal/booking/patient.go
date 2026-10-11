package booking

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/like"
)

const (
	maxPatientQueryLength = 100
	minPhoneQueryDigits   = 3
)

var ErrPatientNotFound = apperr.NotFound("Pasien tidak ditemukan.")

type Patient struct {
	ID           int64
	Name         string
	Phone        string
	Email        string
	BookingCount int64
	LastVisitAt  *time.Time
	CreatedAt    time.Time
}

type PatientDetail struct {
	Patient
	Bookings []AdminBooking
}

type patientStore interface {
	ListPatients(ctx context.Context, namePattern, phonePattern string, limit, offset int) ([]Patient, int64, error)
	GetPatient(ctx context.Context, id int64) (Patient, error)
	ListPatientBookings(ctx context.Context, patientID int64) ([]AdminBooking, error)
}

type PatientService struct {
	store    patientStore
	bookings *Service
}

func NewPatientService(store patientStore, bookings *Service) *PatientService {
	return &PatientService{store: store, bookings: bookings}
}

func (s *PatientService) List(ctx context.Context, query string, limit, offset int) ([]Patient, int64, error) {
	query = strings.TrimSpace(query)
	if utf8.RuneCountInString(query) > maxPatientQueryLength {
		return nil, 0, apperr.Validation(map[string]string{"q": fmt.Sprintf("Kata kunci maksimal %d karakter.", maxPatientQueryLength)})
	}

	phonePattern := ""
	if digits, ok := phoneQuery(query); ok {
		phonePattern = like.Contains(digits)
	}

	return s.store.ListPatients(ctx, like.Contains(query), phonePattern, limit, offset)
}

func (s *PatientService) Get(ctx context.Context, id int64) (PatientDetail, error) {
	p, err := s.store.GetPatient(ctx, id)
	if errors.Is(err, errNotFound) {
		return PatientDetail{}, ErrPatientNotFound
	}
	if err != nil {
		return PatientDetail{}, err
	}

	bookings, err := s.store.ListPatientBookings(ctx, id)
	if err != nil {
		return PatientDetail{}, err
	}

	names := newNameCache(s.bookings)
	for i := range bookings {
		if err := names.fill(ctx, &bookings[i]); err != nil {
			return PatientDetail{}, err
		}
	}

	return PatientDetail{Patient: p, Bookings: bookings}, nil
}

func phoneQuery(query string) (string, bool) {
	digits := strings.Map(func(r rune) rune {
		switch {
		case r >= '0' && r <= '9':
			return r
		case r == ' ' || r == '-' || r == '+':
			return -1
		}
		return 'x'
	}, query)

	if strings.ContainsRune(digits, 'x') || len(digits) < minPhoneQueryDigits {
		return "", false
	}
	if strings.HasPrefix(digits, "0") {
		digits = "62" + digits[1:]
	}
	return digits, true
}
