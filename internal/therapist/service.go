package therapist

import (
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/integrity"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/slug"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

const (
	maxNameLength   = 100
	maxTitleLength  = 100
	maxBioLength    = 2000
	maxSlugAttempts = 50
)

var (
	ErrNotFound    = apperr.NotFound("Terapis tidak ditemukan.")
	ErrHasBookings = apperr.Conflict("HAS_BOOKINGS", "Terapis sudah pernah dibooking sehingga tidak bisa dihapus. Nonaktifkan saja.")
)

type store interface {
	ListActive(ctx context.Context) ([]Therapist, error)
	List(ctx context.Context) ([]Therapist, error)
	GetActiveBySlug(ctx context.Context, slug string) (Therapist, error)
	Get(ctx context.Context, id int64) (Therapist, error)
	Create(ctx context.Context, slug string, in Input) (Therapist, error)
	Update(ctx context.Context, id int64, in Input) (Therapist, error)
	Delete(ctx context.Context, id int64) error
	ListByTreatment(ctx context.Context, treatmentID int64) ([]Therapist, error)
	ReplaceTreatmentTherapists(ctx context.Context, treatmentID int64, therapistIDs []int64) error
	ListSchedules(ctx context.Context, therapistID int64) ([]Shift, error)
	ReplaceSchedules(ctx context.Context, therapistID int64, shifts []Shift) error
	ListUpcomingTimeOff(ctx context.Context, therapistID int64) ([]TimeOff, error)
	CreateTimeOff(ctx context.Context, therapistID int64, in TimeOffInput) (TimeOff, error)
	DeleteTimeOff(ctx context.Context, therapistID, timeOffID int64) error
	GetByUserID(ctx context.Context, userID int64) (Therapist, error)
	SetUserID(ctx context.Context, id, userID int64) error
}

type treatmentFinder interface {
	Get(ctx context.Context, id int64) (treatment.Treatment, error)
	GetActiveBySlug(ctx context.Context, slug string) (treatment.Treatment, error)
}

type imageChecker interface {
	CheckImageURL(raw string) string
}

type Service struct {
	store      store
	treatments treatmentFinder
	bookings   integrity.Finder
	images     imageChecker
	now        func() time.Time
}

func NewService(store store, treatments treatmentFinder, bookings integrity.Finder, images imageChecker) *Service {
	return &Service{store: store, treatments: treatments, bookings: bookings, images: images, now: time.Now}
}

func (s *Service) ListActive(ctx context.Context) ([]Therapist, error) {
	return s.store.ListActive(ctx)
}

func (s *Service) List(ctx context.Context) ([]Therapist, error) {
	return s.store.List(ctx)
}

func (s *Service) GetActiveBySlug(ctx context.Context, slug string) (Therapist, error) {
	t, err := s.store.GetActiveBySlug(ctx, slug)
	return t, mapNotFound(err)
}

func (s *Service) Get(ctx context.Context, id int64) (Therapist, error) {
	t, err := s.store.Get(ctx, id)
	return t, mapNotFound(err)
}

func (s *Service) GetByUserID(ctx context.Context, userID int64) (Therapist, error) {
	t, err := s.store.GetByUserID(ctx, userID)
	return t, mapNotFound(err)
}

func (s *Service) Create(ctx context.Context, in Input) (Therapist, error) {
	in = normalize(in)
	if fields := validate(in, s.images); len(fields) > 0 {
		return Therapist{}, apperr.Validation(fields)
	}

	base := slug.Make(in.Name, "terapis")
	for attempt := 1; attempt <= maxSlugAttempts; attempt++ {
		candidate := base
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", base, attempt)
		}

		t, err := s.store.Create(ctx, candidate, in)
		if errors.Is(err, errSlugTaken) {
			continue
		}
		return t, err
	}

	return Therapist{}, fmt.Errorf("no free slug for %q after %d attempts", base, maxSlugAttempts)
}

func (s *Service) Update(ctx context.Context, id int64, in Input) (Therapist, error) {
	in = normalize(in)
	if fields := validate(in, s.images); len(fields) > 0 {
		return Therapist{}, apperr.Validation(fields)
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return Therapist{}, err
	}

	if current.IsActive && !in.IsActive {
		active, err := s.bookings.ActiveByTherapist(ctx, id)
		if err != nil {
			return Therapist{}, err
		}
		if err := integrity.Conflict(active); err != nil {
			return Therapist{}, err
		}
	}

	t, err := s.store.Update(ctx, id, in)
	return t, mapNotFound(err)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	err := s.store.Delete(ctx, id)
	if errors.Is(err, errInUse) {
		return ErrHasBookings
	}
	return mapNotFound(err)
}

func (s *Service) ListActiveForTreatment(ctx context.Context, treatmentSlug string) ([]Therapist, error) {
	t, err := s.treatments.GetActiveBySlug(ctx, treatmentSlug)
	if err != nil {
		return nil, err
	}

	all, err := s.store.ListByTreatment(ctx, t.ID)
	if err != nil {
		return nil, err
	}

	active := make([]Therapist, 0, len(all))
	for _, th := range all {
		if th.IsActive {
			active = append(active, th)
		}
	}
	return active, nil
}

func (s *Service) ListForTreatment(ctx context.Context, treatmentID int64) ([]Therapist, error) {
	if _, err := s.treatments.Get(ctx, treatmentID); err != nil {
		return nil, err
	}
	return s.store.ListByTreatment(ctx, treatmentID)
}

func (s *Service) SetForTreatment(ctx context.Context, treatmentID int64, therapistIDs []int64) ([]Therapist, error) {
	if _, err := s.treatments.Get(ctx, treatmentID); err != nil {
		return nil, err
	}

	ids := slices.Clone(therapistIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)

	active, err := s.bookings.ActiveByTreatment(ctx, treatmentID)
	if err != nil {
		return nil, err
	}
	var detached []integrity.Booking
	for _, b := range active {
		if !slices.Contains(ids, b.TherapistID) {
			detached = append(detached, b)
		}
	}
	if err := integrity.Conflict(detached); err != nil {
		return nil, err
	}

	err = s.store.ReplaceTreatmentTherapists(ctx, treatmentID, ids)
	if errors.Is(err, errUnknownTherapist) {
		return nil, apperr.Validation(map[string]string{"therapist_ids": "Ada terapis yang tidak ditemukan."})
	}
	if err != nil {
		return nil, err
	}

	return s.store.ListByTreatment(ctx, treatmentID)
}

func mapNotFound(err error) error {
	if errors.Is(err, errNotFound) {
		return ErrNotFound
	}
	return err
}

func normalize(in Input) Input {
	in.Name = strings.TrimSpace(in.Name)
	in.Title = strings.TrimSpace(in.Title)
	in.Bio = strings.TrimSpace(in.Bio)
	in.PhotoURL = strings.TrimSpace(in.PhotoURL)
	return in
}

func validate(in Input, images imageChecker) map[string]string {
	fields := map[string]string{}

	switch {
	case in.Name == "":
		fields["name"] = "Nama wajib diisi."
	case utf8.RuneCountInString(in.Name) > maxNameLength:
		fields["name"] = fmt.Sprintf("Nama maksimal %d karakter.", maxNameLength)
	}

	if utf8.RuneCountInString(in.Title) > maxTitleLength {
		fields["title"] = fmt.Sprintf("Gelar maksimal %d karakter.", maxTitleLength)
	}

	if utf8.RuneCountInString(in.Bio) > maxBioLength {
		fields["bio"] = fmt.Sprintf("Bio maksimal %d karakter.", maxBioLength)
	}

	if msg := images.CheckImageURL(in.PhotoURL); msg != "" {
		fields["photo_url"] = msg
	}

	return fields
}
