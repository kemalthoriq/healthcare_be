package treatment

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/integrity"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/slug"
)

const (
	maxNameLength        = 100
	maxDescriptionLength = 2000
	maxPrice             = 100_000_000
	minDuration          = 15
	maxDuration          = 480
	maxSlugAttempts      = 50
)

var (
	ErrNotFound    = apperr.NotFound("Treatment tidak ditemukan.")
	ErrHasBookings = apperr.Conflict("HAS_BOOKINGS", "Treatment sudah pernah dibooking sehingga tidak bisa dihapus. Nonaktifkan saja.")
)

type store interface {
	ListActive(ctx context.Context) ([]Treatment, error)
	List(ctx context.Context) ([]Treatment, error)
	GetActiveBySlug(ctx context.Context, slug string) (Treatment, error)
	Get(ctx context.Context, id int64) (Treatment, error)
	Create(ctx context.Context, slug string, in Input) (Treatment, error)
	Update(ctx context.Context, id int64, in Input) (Treatment, error)
	Delete(ctx context.Context, id int64) error
}

type imageChecker interface {
	CheckImageURL(raw string) string
}

type Service struct {
	store    store
	bookings integrity.Finder
	images   imageChecker
}

func NewService(store store, bookings integrity.Finder, images imageChecker) *Service {
	return &Service{store: store, bookings: bookings, images: images}
}

func (s *Service) ListActive(ctx context.Context) ([]Treatment, error) {
	return s.store.ListActive(ctx)
}

func (s *Service) List(ctx context.Context) ([]Treatment, error) {
	return s.store.List(ctx)
}

func (s *Service) GetActiveBySlug(ctx context.Context, slug string) (Treatment, error) {
	t, err := s.store.GetActiveBySlug(ctx, slug)
	return t, mapNotFound(err)
}

func (s *Service) Get(ctx context.Context, id int64) (Treatment, error) {
	t, err := s.store.Get(ctx, id)
	return t, mapNotFound(err)
}

func (s *Service) Create(ctx context.Context, in Input) (Treatment, error) {
	in = normalize(in)
	if fields := validate(in, s.images); len(fields) > 0 {
		return Treatment{}, apperr.Validation(fields)
	}

	base := slug.Make(in.Name, "treatment")
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

	return Treatment{}, fmt.Errorf("no free slug for %q after %d attempts", base, maxSlugAttempts)
}

func (s *Service) Update(ctx context.Context, id int64, in Input) (Treatment, error) {
	in = normalize(in)
	if fields := validate(in, s.images); len(fields) > 0 {
		return Treatment{}, apperr.Validation(fields)
	}

	current, err := s.Get(ctx, id)
	if err != nil {
		return Treatment{}, err
	}

	if current.IsActive && !in.IsActive {
		active, err := s.bookings.ActiveByTreatment(ctx, id)
		if err != nil {
			return Treatment{}, err
		}
		if err := integrity.Conflict(active); err != nil {
			return Treatment{}, err
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

func mapNotFound(err error) error {
	if errors.Is(err, errNotFound) {
		return ErrNotFound
	}
	return err
}

func normalize(in Input) Input {
	in.Name = strings.TrimSpace(in.Name)
	in.Description = strings.TrimSpace(in.Description)
	in.ImageURL = strings.TrimSpace(in.ImageURL)
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

	if utf8.RuneCountInString(in.Description) > maxDescriptionLength {
		fields["description"] = fmt.Sprintf("Deskripsi maksimal %d karakter.", maxDescriptionLength)
	}

	if msg := images.CheckImageURL(in.ImageURL); msg != "" {
		fields["image_url"] = msg
	}

	if in.Price < 0 || in.Price > maxPrice {
		fields["price"] = "Harga harus antara Rp0 dan Rp100.000.000."
	}

	if in.DurationMinutes < minDuration || in.DurationMinutes > maxDuration || in.DurationMinutes%5 != 0 {
		fields["duration_minutes"] = "Durasi harus 15–480 menit dan kelipatan 5."
	}

	return fields
}
