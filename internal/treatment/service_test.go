package treatment

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type fakeStore struct {
	items  map[int64]Treatment
	slugs  map[string]bool
	nextID int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: map[int64]Treatment{}, slugs: map[string]bool{}}
}

func (f *fakeStore) ListActive(ctx context.Context) ([]Treatment, error) {
	var out []Treatment
	for _, t := range f.items {
		if t.IsActive {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) List(ctx context.Context) ([]Treatment, error) {
	var out []Treatment
	for _, t := range f.items {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeStore) GetActiveBySlug(ctx context.Context, slug string) (Treatment, error) {
	for _, t := range f.items {
		if t.Slug == slug && t.IsActive {
			return t, nil
		}
	}
	return Treatment{}, errNotFound
}

func (f *fakeStore) Get(ctx context.Context, id int64) (Treatment, error) {
	t, ok := f.items[id]
	if !ok {
		return Treatment{}, errNotFound
	}
	return t, nil
}

func (f *fakeStore) Create(ctx context.Context, slug string, in Input) (Treatment, error) {
	if f.slugs[slug] {
		return Treatment{}, errSlugTaken
	}
	f.nextID++
	f.slugs[slug] = true
	t := Treatment{ID: f.nextID, Slug: slug, Name: in.Name, Description: in.Description, ImageURL: in.ImageURL,
		Price: in.Price, DurationMinutes: in.DurationMinutes, IsActive: in.IsActive}
	f.items[t.ID] = t
	return t, nil
}

func (f *fakeStore) Update(ctx context.Context, id int64, in Input) (Treatment, error) {
	t, ok := f.items[id]
	if !ok {
		return Treatment{}, errNotFound
	}
	t.Name, t.Description, t.ImageURL = in.Name, in.Description, in.ImageURL
	t.Price, t.DurationMinutes, t.IsActive = in.Price, in.DurationMinutes, in.IsActive
	f.items[id] = t
	return t, nil
}

func (f *fakeStore) Delete(ctx context.Context, id int64) error {
	if _, ok := f.items[id]; !ok {
		return errNotFound
	}
	delete(f.items, id)
	return nil
}

func validInput() Input {
	return Input{Name: "Facial Glow", Price: 350000, DurationMinutes: 60, IsActive: true}
}

func TestCreate_GeneratesUniqueSlug(t *testing.T) {
	svc := NewService(newFakeStore(), fakeBookings{})

	var slugs []string
	for range 3 {
		created, err := svc.Create(context.Background(), validInput())
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		slugs = append(slugs, created.Slug)
	}

	if got := strings.Join(slugs, ","); got != "facial-glow,facial-glow-2,facial-glow-3" {
		t.Errorf("slugs = %s", got)
	}
}

func TestCreate_TrimsInput(t *testing.T) {
	in := validInput()
	in.Name = "  Facial Glow  "

	created, err := NewService(newFakeStore(), fakeBookings{}).Create(context.Background(), in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if created.Name != "Facial Glow" {
		t.Errorf("Name = %q, want trimmed", created.Name)
	}
}

func TestCreate_Validation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Input)
		field  string
	}{
		{"empty name", func(in *Input) { in.Name = "  " }, "name"},
		{"name too long", func(in *Input) { in.Name = strings.Repeat("a", 101) }, "name"},
		{"description too long", func(in *Input) { in.Description = strings.Repeat("a", 2001) }, "description"},
		{"http image url", func(in *Input) { in.ImageURL = "http://res.cloudinary.com/x.jpg" }, "image_url"},
		{"negative price", func(in *Input) { in.Price = -1 }, "price"},
		{"price too high", func(in *Input) { in.Price = 100_000_001 }, "price"},
		{"duration too short", func(in *Input) { in.DurationMinutes = 10 }, "duration_minutes"},
		{"duration too long", func(in *Input) { in.DurationMinutes = 485 }, "duration_minutes"},
		{"duration not multiple of 5", func(in *Input) { in.DurationMinutes = 62 }, "duration_minutes"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := validInput()
			tt.modify(&in)

			_, err := NewService(newFakeStore(), fakeBookings{}).Create(context.Background(), in)

			var appErr *apperr.Error
			if !errors.As(err, &appErr) || appErr.Code != "VALIDATION_ERROR" {
				t.Fatalf("expected VALIDATION_ERROR, got %v", err)
			}
			if _, ok := appErr.Fields[tt.field]; !ok {
				t.Errorf("expected field %q in %v", tt.field, appErr.Fields)
			}
		})
	}
}

func TestUpdate_KeepsSlug(t *testing.T) {
	store := newFakeStore()
	svc := NewService(store, fakeBookings{})
	created, _ := svc.Create(context.Background(), validInput())

	in := validInput()
	in.Name = "Facial Glow Premium"
	updated, err := svc.Update(context.Background(), created.ID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Slug != "facial-glow" || updated.Name != "Facial Glow Premium" {
		t.Errorf("got slug %q name %q, want slug unchanged", updated.Slug, updated.Name)
	}
}

func TestNotFound(t *testing.T) {
	svc := NewService(newFakeStore(), fakeBookings{})
	ctx := context.Background()

	_, getErr := svc.Get(ctx, 99)
	_, slugErr := svc.GetActiveBySlug(ctx, "tidak-ada")
	_, updateErr := svc.Update(ctx, 99, validInput())
	deleteErr := svc.Delete(ctx, 99)

	for name, err := range map[string]error{"get": getErr, "slug": slugErr, "update": updateErr, "delete": deleteErr} {
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("%s: err = %v, want ErrNotFound", name, err)
		}
	}
}

func TestGetActiveBySlug_HidesInactive(t *testing.T) {
	svc := NewService(newFakeStore(), fakeBookings{})
	in := validInput()
	in.IsActive = false
	created, _ := svc.Create(context.Background(), in)

	if _, err := svc.GetActiveBySlug(context.Background(), created.Slug); !errors.Is(err, ErrNotFound) {
		t.Errorf("inactive treatment should not be visible publicly, err = %v", err)
	}
}
