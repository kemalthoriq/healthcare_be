package therapist

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/slug"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

const (
	maxNameLength     = 100
	maxTitleLength    = 100
	maxBioLength      = 2000
	maxPhotoURLLength = 500
	maxSlugAttempts   = 50
)

var ErrNotFound = apperr.NotFound("Terapis tidak ditemukan.")

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
}

type treatmentFinder interface {
	Get(ctx context.Context, id int64) (treatment.Treatment, error)
	GetActiveBySlug(ctx context.Context, slug string) (treatment.Treatment, error)
}

type Service struct {
	store      store
	treatments treatmentFinder
}

func NewService(store store, treatments treatmentFinder) *Service {
	return &Service{store: store, treatments: treatments}
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

func (s *Service) Create(ctx context.Context, in Input) (Therapist, error) {
	in = normalize(in)
	if fields := validate(in); len(fields) > 0 {
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
	if fields := validate(in); len(fields) > 0 {
		return Therapist{}, apperr.Validation(fields)
	}

	t, err := s.store.Update(ctx, id, in)
	return t, mapNotFound(err)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapNotFound(s.store.Delete(ctx, id))
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

	err := s.store.ReplaceTreatmentTherapists(ctx, treatmentID, ids)
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

func validate(in Input) map[string]string {
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

	if in.PhotoURL != "" && !validPhotoURL(in.PhotoURL) {
		fields["photo_url"] = "URL foto harus diawali https:// dan maksimal 500 karakter."
	}

	return fields
}

func validPhotoURL(raw string) bool {
	if len(raw) > maxPhotoURLLength {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}
