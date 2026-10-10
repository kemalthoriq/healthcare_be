package treatment

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const (
	maxNameLength        = 100
	maxDescriptionLength = 2000
	maxImageURLLength    = 500
	maxPrice             = 100_000_000
	minDuration          = 15
	maxDuration          = 480
	maxSlugAttempts      = 50
)

var ErrNotFound = apperr.NotFound("Treatment tidak ditemukan.")

type store interface {
	ListActive(ctx context.Context) ([]Treatment, error)
	List(ctx context.Context) ([]Treatment, error)
	GetActiveBySlug(ctx context.Context, slug string) (Treatment, error)
	Get(ctx context.Context, id int64) (Treatment, error)
	Create(ctx context.Context, slug string, in Input) (Treatment, error)
	Update(ctx context.Context, id int64, in Input) (Treatment, error)
	Delete(ctx context.Context, id int64) error
}

type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
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
	if fields := validate(in); len(fields) > 0 {
		return Treatment{}, apperr.Validation(fields)
	}

	base := Slugify(in.Name)
	for attempt := 1; attempt <= maxSlugAttempts; attempt++ {
		slug := base
		if attempt > 1 {
			slug = fmt.Sprintf("%s-%d", base, attempt)
		}

		t, err := s.store.Create(ctx, slug, in)
		if errors.Is(err, errSlugTaken) {
			continue
		}
		return t, err
	}

	return Treatment{}, fmt.Errorf("no free slug for %q after %d attempts", base, maxSlugAttempts)
}

func (s *Service) Update(ctx context.Context, id int64, in Input) (Treatment, error) {
	in = normalize(in)
	if fields := validate(in); len(fields) > 0 {
		return Treatment{}, apperr.Validation(fields)
	}

	t, err := s.store.Update(ctx, id, in)
	return t, mapNotFound(err)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapNotFound(s.store.Delete(ctx, id))
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

func validate(in Input) map[string]string {
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

	if in.ImageURL != "" && !validImageURL(in.ImageURL) {
		fields["image_url"] = "URL gambar harus diawali https:// dan maksimal 500 karakter."
	}

	if in.Price < 0 || in.Price > maxPrice {
		fields["price"] = "Harga harus antara Rp0 dan Rp100.000.000."
	}

	if in.DurationMinutes < minDuration || in.DurationMinutes > maxDuration || in.DurationMinutes%5 != 0 {
		fields["duration_minutes"] = "Durasi harus 15–480 menit dan kelipatan 5."
	}

	return fields
}

func validImageURL(raw string) bool {
	if len(raw) > maxImageURLLength {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}
