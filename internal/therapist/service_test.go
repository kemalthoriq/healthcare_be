package therapist

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

type fakeStore struct {
	items     map[int64]Therapist
	slugs     map[string]bool
	links     map[int64][]int64
	schedules map[int64][]Shift
	timeOffs  map[int64]TimeOff
	nextID    int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		items:     map[int64]Therapist{},
		slugs:     map[string]bool{},
		links:     map[int64][]int64{},
		schedules: map[int64][]Shift{},
		timeOffs:  map[int64]TimeOff{},
	}
}

func (f *fakeStore) ListActive(ctx context.Context) ([]Therapist, error) {
	var out []Therapist
	for _, t := range f.items {
		if t.IsActive {
			out = append(out, t)
		}
	}
	return out, nil
}

func (f *fakeStore) List(ctx context.Context) ([]Therapist, error) {
	var out []Therapist
	for _, t := range f.items {
		out = append(out, t)
	}
	return out, nil
}

func (f *fakeStore) GetActiveBySlug(ctx context.Context, slug string) (Therapist, error) {
	for _, t := range f.items {
		if t.Slug == slug && t.IsActive {
			return t, nil
		}
	}
	return Therapist{}, errNotFound
}

func (f *fakeStore) Get(ctx context.Context, id int64) (Therapist, error) {
	t, ok := f.items[id]
	if !ok {
		return Therapist{}, errNotFound
	}
	return t, nil
}

func (f *fakeStore) Create(ctx context.Context, slug string, in Input) (Therapist, error) {
	if f.slugs[slug] {
		return Therapist{}, errSlugTaken
	}
	f.nextID++
	f.slugs[slug] = true
	t := Therapist{ID: f.nextID, Slug: slug, Name: in.Name, Title: in.Title, Bio: in.Bio, PhotoURL: in.PhotoURL, IsActive: in.IsActive}
	f.items[t.ID] = t
	return t, nil
}

func (f *fakeStore) Update(ctx context.Context, id int64, in Input) (Therapist, error) {
	t, ok := f.items[id]
	if !ok {
		return Therapist{}, errNotFound
	}
	t.Name, t.Title, t.Bio, t.PhotoURL, t.IsActive = in.Name, in.Title, in.Bio, in.PhotoURL, in.IsActive
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

func (f *fakeStore) ListByTreatment(ctx context.Context, treatmentID int64) ([]Therapist, error) {
	var out []Therapist
	for _, id := range f.links[treatmentID] {
		out = append(out, f.items[id])
	}
	return out, nil
}

func (f *fakeStore) ReplaceTreatmentTherapists(ctx context.Context, treatmentID int64, therapistIDs []int64) error {
	for _, id := range therapistIDs {
		if _, ok := f.items[id]; !ok {
			return errUnknownTherapist
		}
	}
	f.links[treatmentID] = therapistIDs
	return nil
}

type fakeTreatments struct{}

func (fakeTreatments) Get(ctx context.Context, id int64) (treatment.Treatment, error) {
	if id != 1 {
		return treatment.Treatment{}, treatment.ErrNotFound
	}
	return treatment.Treatment{ID: 1, Slug: "facial-glow", IsActive: true}, nil
}

func (fakeTreatments) GetActiveBySlug(ctx context.Context, slug string) (treatment.Treatment, error) {
	if slug != "facial-glow" {
		return treatment.Treatment{}, treatment.ErrNotFound
	}
	return treatment.Treatment{ID: 1, Slug: "facial-glow", IsActive: true}, nil
}

func newTestService() (*Service, *fakeStore) {
	store := newFakeStore()
	return NewService(store, fakeTreatments{}), store
}

func validInput() Input {
	return Input{Name: "dr. Sari", Title: "Dokter Estetika", IsActive: true}
}

func TestCreate_GeneratesUniqueSlug(t *testing.T) {
	svc, _ := newTestService()

	first, _ := svc.Create(context.Background(), validInput())
	second, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if first.Slug != "dr-sari" || second.Slug != "dr-sari-2" {
		t.Errorf("slugs = %q, %q", first.Slug, second.Slug)
	}
}

func TestCreate_Validation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Input)
		field  string
	}{
		{"empty name", func(in *Input) { in.Name = " " }, "name"},
		{"title too long", func(in *Input) { in.Title = strings.Repeat("a", 101) }, "title"},
		{"bio too long", func(in *Input) { in.Bio = strings.Repeat("a", 2001) }, "bio"},
		{"http photo url", func(in *Input) { in.PhotoURL = "http://example.com/a.jpg" }, "photo_url"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService()
			in := validInput()
			tt.modify(&in)

			_, err := svc.Create(context.Background(), in)

			var appErr *apperr.Error
			if !errors.As(err, &appErr) || appErr.Fields[tt.field] == "" {
				t.Fatalf("expected validation error on %q, got %v", tt.field, err)
			}
		})
	}
}

func TestSetForTreatment(t *testing.T) {
	svc, store := newTestService()
	a, _ := svc.Create(context.Background(), validInput())
	b, _ := svc.Create(context.Background(), Input{Name: "Rina", IsActive: true})

	got, err := svc.SetForTreatment(context.Background(), 1, []int64{b.ID, a.ID, b.ID})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 || !slices.Equal(store.links[1], []int64{a.ID, b.ID}) {
		t.Errorf("links = %v, want deduplicated [%d %d]", store.links[1], a.ID, b.ID)
	}
}

func TestSetForTreatment_Errors(t *testing.T) {
	svc, _ := newTestService()

	if _, err := svc.SetForTreatment(context.Background(), 99, []int64{}); !errors.Is(err, treatment.ErrNotFound) {
		t.Errorf("unknown treatment: err = %v, want treatment.ErrNotFound", err)
	}

	_, err := svc.SetForTreatment(context.Background(), 1, []int64{42})
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Fields["therapist_ids"] == "" {
		t.Errorf("unknown therapist: err = %v, want validation error on therapist_ids", err)
	}
}

func TestListActiveForTreatment_HidesInactiveTherapists(t *testing.T) {
	svc, _ := newTestService()
	active, _ := svc.Create(context.Background(), validInput())
	inactive, _ := svc.Create(context.Background(), Input{Name: "Rina", IsActive: false})
	svc.SetForTreatment(context.Background(), 1, []int64{active.ID, inactive.ID})

	got, err := svc.ListActiveForTreatment(context.Background(), "facial-glow")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].ID != active.ID {
		t.Errorf("got %v, want only the active therapist", got)
	}

	if _, err := svc.ListActiveForTreatment(context.Background(), "tidak-ada"); !errors.Is(err, treatment.ErrNotFound) {
		t.Errorf("unknown treatment slug: err = %v, want treatment.ErrNotFound", err)
	}
}
