package booking

import (
	"context"
	"errors"
	"fmt"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

const (
	maxNameLength  = 100
	maxEmailLength = 254
	maxNoteLength  = 500
	maxCodeRetries = 5
)

var ErrSlotTaken = apperr.Conflict("SLOT_TAKEN", "Jam ini baru saja dipesan. Silakan pilih jam lain.")

type store interface {
	Create(ctx context.Context, code string, nb NewBooking) (Booking, error)
	GetByCode(ctx context.Context, code string) (Booking, error)
	CancelByCode(ctx context.Context, code, reason string, minStartAt time.Time) (Booking, error)
	ListAdmin(ctx context.Context, f AdminFilter, limit, offset int) ([]AdminBooking, int64, error)
	GetAdmin(ctx context.Context, id int64) (AdminBooking, error)
	Transition(ctx context.Context, id int64, from, to, reason string) error
	Reschedule(ctx context.Context, id int64, startAt, endAt, blockedUntil time.Time) error
	SaveNote(ctx context.Context, bookingID, authorID int64, content string) error
}

type Service struct {
	availability *Availability
	store        store
}

func NewService(availability *Availability, store store) *Service {
	return &Service{availability: availability, store: store}
}

type CreateInput struct {
	TreatmentSlug string
	TherapistSlug string
	StartAt       time.Time
	Name          string
	Phone         string
	Email         string
	Note          string
	Consent       bool
}

type Created struct {
	Booking   Booking
	Treatment treatment.Treatment
	Therapist therapist.Therapist
}

func (s *Service) Create(ctx context.Context, in CreateInput) (Created, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Email = strings.ToLower(strings.TrimSpace(in.Email))
	in.Note = strings.TrimSpace(in.Note)

	phone, fields := validateCreate(in)
	if len(fields) > 0 {
		return Created{}, apperr.Validation(fields)
	}

	p, err := s.availability.check(ctx, in.TreatmentSlug, in.TherapistSlug, in.StartAt, 0)
	if err != nil {
		return Created{}, err
	}

	endAt := in.StartAt.Add(p.duration)
	nb := NewBooking{
		PatientName:  in.Name,
		Phone:        phone,
		Email:        in.Email,
		TreatmentID:  p.treatment.ID,
		TherapistID:  p.therapist.ID,
		StartAt:      in.StartAt,
		EndAt:        endAt,
		BlockedUntil: endAt.Add(p.buffer),
		Price:        p.treatment.Price,
		PatientNote:  in.Note,
	}

	for range maxCodeRetries {
		b, err := s.store.Create(ctx, NewCode(), nb)
		switch {
		case errors.Is(err, errCodeTaken):
			continue
		case errors.Is(err, errSlotTaken):
			return Created{}, ErrSlotTaken
		case err != nil:
			return Created{}, err
		}
		return Created{Booking: b, Treatment: p.treatment, Therapist: p.therapist}, nil
	}

	return Created{}, fmt.Errorf("no unique booking code after %d attempts", maxCodeRetries)
}

func validateCreate(in CreateInput) (string, map[string]string) {
	fields := map[string]string{}

	if in.TreatmentSlug == "" {
		fields["treatment_slug"] = "Treatment wajib dipilih."
	}
	if in.TherapistSlug == "" {
		fields["therapist_slug"] = "Terapis wajib dipilih."
	}
	if in.StartAt.IsZero() {
		fields["start_at"] = "Jam booking wajib dipilih."
	}

	switch {
	case in.Name == "":
		fields["name"] = "Nama wajib diisi."
	case utf8.RuneCountInString(in.Name) > maxNameLength:
		fields["name"] = fmt.Sprintf("Nama maksimal %d karakter.", maxNameLength)
	}

	phone, ok := NormalizePhone(in.Phone)
	if !ok {
		fields["phone"] = "Nomor WhatsApp belum sesuai. Contoh: 081234567890."
	}

	if in.Email != "" {
		addr, err := mail.ParseAddress(in.Email)
		if err != nil || addr.Address != in.Email || len(in.Email) > maxEmailLength {
			fields["email"] = "Format email belum sesuai."
		}
	}

	if utf8.RuneCountInString(in.Note) > maxNoteLength {
		fields["note"] = fmt.Sprintf("Catatan maksimal %d karakter.", maxNoteLength)
	}

	if !in.Consent {
		fields["consent"] = "Persetujuan penggunaan data wajib dicentang."
	}

	return phone, fields
}
