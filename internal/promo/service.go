package promo

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/slug"
)

const (
	maxTitleLength       = 100
	maxDescriptionLength = 2000
	maxImageURLLength    = 500
	maxTreatments        = 20
	maxSlugAttempts      = 50
)

var ErrNotFound = apperr.NotFound("Promo tidak ditemukan.")

type store interface {
	ListRunning(ctx context.Context, today time.Time) ([]Promo, error)
	GetRunningBySlug(ctx context.Context, slug string, today time.Time) (Promo, error)
	List(ctx context.Context, limit, offset int) ([]Promo, int64, error)
	Get(ctx context.Context, id int64) (Promo, error)
	Create(ctx context.Context, slug string, v Values) (int64, error)
	Update(ctx context.Context, id int64, v Values) error
	Delete(ctx context.Context, id int64) error
}

type Service struct {
	store store
	loc   *time.Location
	now   func() time.Time
}

func NewService(store store, loc *time.Location) *Service {
	return &Service{store: store, loc: loc, now: time.Now}
}

func (s *Service) ListPublic(ctx context.Context) ([]Promo, error) {
	promos, err := s.store.ListRunning(ctx, s.today())
	if err != nil {
		return nil, err
	}
	for i := range promos {
		promos[i] = publicOnly(promos[i])
	}
	return promos, nil
}

func (s *Service) GetPublic(ctx context.Context, slug string) (Promo, error) {
	p, err := s.store.GetRunningBySlug(ctx, slug, s.today())
	if err != nil {
		return Promo{}, mapNotFound(err)
	}
	return publicOnly(p), nil
}

func (s *Service) List(ctx context.Context, limit, offset int) ([]Promo, int64, error) {
	return s.store.List(ctx, limit, offset)
}

func (s *Service) Get(ctx context.Context, id int64) (Promo, error) {
	p, err := s.store.Get(ctx, id)
	return p, mapNotFound(err)
}

func (s *Service) Create(ctx context.Context, in Input) (Promo, error) {
	v, err := validate(in)
	if err != nil {
		return Promo{}, err
	}

	base := slug.Make(v.Title, "promo")
	for attempt := 1; attempt <= maxSlugAttempts; attempt++ {
		candidate := base
		if attempt > 1 {
			candidate = fmt.Sprintf("%s-%d", base, attempt)
		}

		id, err := s.store.Create(ctx, candidate, v)
		if errors.Is(err, errSlugTaken) {
			continue
		}
		if err != nil {
			return Promo{}, mapStoreError(err)
		}
		return s.Get(ctx, id)
	}

	return Promo{}, fmt.Errorf("no free slug for %q after %d attempts", base, maxSlugAttempts)
}

func (s *Service) Update(ctx context.Context, id int64, in Input) (Promo, error) {
	v, err := validate(in)
	if err != nil {
		return Promo{}, err
	}

	if err := s.store.Update(ctx, id, v); err != nil {
		return Promo{}, mapStoreError(err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	return mapNotFound(s.store.Delete(ctx, id))
}

func (s *Service) today() time.Time {
	n := s.now().In(s.loc)
	return time.Date(n.Year(), n.Month(), n.Day(), 0, 0, 0, 0, time.UTC)
}

func publicOnly(p Promo) Promo {
	active := make([]Treatment, 0, len(p.Treatments))
	for _, t := range p.Treatments {
		if t.IsActive {
			active = append(active, t)
		}
	}
	p.Treatments = active
	return p
}

func mapNotFound(err error) error {
	if errors.Is(err, errNotFound) {
		return ErrNotFound
	}
	return err
}

func mapStoreError(err error) error {
	if errors.Is(err, errUnknownTreatment) {
		return apperr.Validation(map[string]string{"treatment_ids": "Ada treatment yang tidak ditemukan."})
	}
	return mapNotFound(err)
}

func validate(in Input) (Values, error) {
	v := Values{
		Title:       strings.TrimSpace(in.Title),
		Description: strings.TrimSpace(in.Description),
		ImageURL:    strings.TrimSpace(in.ImageURL),
		IsActive:    in.IsActive,
	}
	fields := map[string]string{}

	switch {
	case v.Title == "":
		fields["title"] = "Judul wajib diisi."
	case utf8.RuneCountInString(v.Title) > maxTitleLength:
		fields["title"] = fmt.Sprintf("Judul maksimal %d karakter.", maxTitleLength)
	}

	if utf8.RuneCountInString(v.Description) > maxDescriptionLength {
		fields["description"] = fmt.Sprintf("Deskripsi maksimal %d karakter.", maxDescriptionLength)
	}

	if v.ImageURL != "" && !validImageURL(v.ImageURL) {
		fields["image_url"] = "URL gambar harus diawali https:// dan maksimal 500 karakter."
	}

	start, startErr := time.Parse(DateLayout, strings.TrimSpace(in.StartDate))
	if startErr != nil {
		fields["start_date"] = "Tanggal mulai wajib diisi dengan format YYYY-MM-DD."
	}
	end, endErr := time.Parse(DateLayout, strings.TrimSpace(in.EndDate))
	if endErr != nil {
		fields["end_date"] = "Tanggal selesai wajib diisi dengan format YYYY-MM-DD."
	}
	if startErr == nil && endErr == nil && end.Before(start) {
		fields["end_date"] = "Tanggal selesai tidak boleh sebelum tanggal mulai."
	}
	v.StartDate, v.EndDate = start, end

	ids := slices.Clone(in.TreatmentIDs)
	slices.Sort(ids)
	ids = slices.Compact(ids)
	if len(ids) > maxTreatments {
		fields["treatment_ids"] = fmt.Sprintf("Maksimal %d treatment per promo.", maxTreatments)
	}
	v.TreatmentIDs = ids

	if len(fields) > 0 {
		return Values{}, apperr.Validation(fields)
	}
	return v, nil
}

func validImageURL(raw string) bool {
	if len(raw) > maxImageURLLength {
		return false
	}
	u, err := url.Parse(raw)
	return err == nil && u.Scheme == "https" && u.Host != ""
}
