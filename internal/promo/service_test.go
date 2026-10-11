package promo

import (
	"context"
	"errors"
	"slices"
	"testing"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/storage"
)

var wib = time.FixedZone("WIB", 7*60*60)

type fakeStore struct {
	items     map[int64]Promo
	takenSlug map[string]bool
	today     time.Time
}

func newFakeStore() *fakeStore {
	return &fakeStore{items: map[int64]Promo{}, takenSlug: map[string]bool{}}
}

func (f *fakeStore) ListRunning(ctx context.Context, today time.Time) ([]Promo, error) {
	f.today = today
	var out []Promo
	for _, p := range f.items {
		out = append(out, p)
	}
	return out, nil
}

func (f *fakeStore) GetRunningBySlug(ctx context.Context, slug string, today time.Time) (Promo, error) {
	f.today = today
	for _, p := range f.items {
		if p.Slug == slug {
			return p, nil
		}
	}
	return Promo{}, errNotFound
}

func (f *fakeStore) List(ctx context.Context, limit, offset int) ([]Promo, int64, error) {
	return nil, 0, nil
}

func (f *fakeStore) Get(ctx context.Context, id int64) (Promo, error) {
	p, ok := f.items[id]
	if !ok {
		return Promo{}, errNotFound
	}
	return p, nil
}

func (f *fakeStore) Create(ctx context.Context, slug string, v Values) (int64, error) {
	if f.takenSlug[slug] {
		return 0, errSlugTaken
	}
	if slices.Contains(v.TreatmentIDs, 99) {
		return 0, errUnknownTreatment
	}
	f.takenSlug[slug] = true
	id := int64(len(f.items) + 1)
	f.items[id] = toPromo(id, slug, v)
	return id, nil
}

func (f *fakeStore) Update(ctx context.Context, id int64, v Values) error {
	p, ok := f.items[id]
	if !ok {
		return errNotFound
	}
	f.items[id] = toPromo(id, p.Slug, v)
	return nil
}

func (f *fakeStore) Delete(ctx context.Context, id int64) error {
	if _, ok := f.items[id]; !ok {
		return errNotFound
	}
	delete(f.items, id)
	return nil
}

func toPromo(id int64, slug string, v Values) Promo {
	var treatments []Treatment
	for _, tid := range v.TreatmentIDs {
		treatments = append(treatments, Treatment{ID: tid, IsActive: tid != 2})
	}
	return Promo{ID: id, Title: v.Title, Slug: slug, StartDate: v.StartDate, EndDate: v.EndDate, IsActive: v.IsActive, Treatments: treatments}
}

func newTestService() (*Service, *fakeStore) {
	store := newFakeStore()
	svc := NewService(store, storage.NewCloudinary("", "", ""), wib)
	svc.now = func() time.Time { return time.Date(2026, 10, 31, 18, 0, 0, 0, time.UTC) }
	return svc, store
}

func validInput() Input {
	return Input{
		Title:        "  Diskon Akhir Tahun ",
		StartDate:    "2026-11-01",
		EndDate:      "2026-12-31",
		IsActive:     true,
		TreatmentIDs: []int64{3, 1, 3, 2},
	}
}

func wantField(t *testing.T, err error, field string) {
	t.Helper()
	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Fields[field] == "" {
		t.Fatalf("expected validation error on %q, got %v", field, err)
	}
}

func TestCreate(t *testing.T) {
	svc, store := newTestService()
	store.takenSlug["diskon-akhir-tahun"] = true

	p, err := svc.Create(context.Background(), validInput())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Title != "Diskon Akhir Tahun" || p.Slug != "diskon-akhir-tahun-2" {
		t.Errorf("title = %q, slug = %q", p.Title, p.Slug)
	}
	if got := []int64{p.Treatments[0].ID, p.Treatments[1].ID, p.Treatments[2].ID}; len(p.Treatments) != 3 || !slices.Equal(got, []int64{1, 2, 3}) {
		t.Errorf("treatment ids = %v, want deduplicated and sorted", p.Treatments)
	}
	if !p.StartDate.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("start date = %v", p.StartDate)
	}
}

func TestCreate_Validation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Input)
		field  string
	}{
		{"empty title", func(in *Input) { in.Title = " " }, "title"},
		{"http image", func(in *Input) { in.ImageURL = "http://example.com/a.jpg" }, "image_url"},
		{"missing start", func(in *Input) { in.StartDate = "" }, "start_date"},
		{"bad end format", func(in *Input) { in.EndDate = "31-12-2026" }, "end_date"},
		{"end before start", func(in *Input) { in.EndDate = "2026-10-31" }, "end_date"},
		{"too many treatments", func(in *Input) {
			in.TreatmentIDs = nil
			for i := range 21 {
				in.TreatmentIDs = append(in.TreatmentIDs, int64(i+1))
			}
		}, "treatment_ids"},
		{"unknown treatment", func(in *Input) { in.TreatmentIDs = []int64{99} }, "treatment_ids"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, _ := newTestService()
			in := validInput()
			tt.modify(&in)

			_, err := svc.Create(context.Background(), in)
			wantField(t, err, tt.field)
		})
	}
}

func TestUpdateAndDelete(t *testing.T) {
	svc, _ := newTestService()
	ctx := context.Background()
	p, _ := svc.Create(ctx, validInput())

	in := validInput()
	in.Title = "Promo Baru"
	in.TreatmentIDs = nil
	updated, err := svc.Update(ctx, p.ID, in)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if updated.Title != "Promo Baru" || updated.Slug != p.Slug || len(updated.Treatments) != 0 {
		t.Errorf("updated = %+v", updated)
	}

	if _, err := svc.Update(ctx, 99, in); !errors.Is(err, ErrNotFound) {
		t.Errorf("update missing: err = %v, want ErrNotFound", err)
	}
	if err := svc.Delete(ctx, p.ID); err != nil {
		t.Fatalf("delete: %v", err)
	}
	if err := svc.Delete(ctx, p.ID); !errors.Is(err, ErrNotFound) {
		t.Errorf("delete twice: err = %v, want ErrNotFound", err)
	}
}

func TestPublic_UsesClinicDateAndHidesInactiveTreatments(t *testing.T) {
	svc, store := newTestService()
	ctx := context.Background()
	p, _ := svc.Create(ctx, validInput())

	promos, err := svc.ListPublic(ctx)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !store.today.Equal(time.Date(2026, 11, 1, 0, 0, 0, 0, time.UTC)) {
		t.Errorf("today = %v, want 2026-11-01 (31 Oct 18:00 UTC is 1 Nov in WIB)", store.today)
	}
	for _, tr := range promos[0].Treatments {
		if !tr.IsActive {
			t.Errorf("inactive treatment %d shown publicly", tr.ID)
		}
	}
	if len(promos[0].Treatments) != 2 {
		t.Errorf("public treatments = %d, want 2", len(promos[0].Treatments))
	}

	got, err := svc.GetPublic(ctx, p.Slug)
	if err != nil || len(got.Treatments) != 2 {
		t.Errorf("get public: %+v, %v", got, err)
	}
	if _, err := svc.GetPublic(ctx, "tidak-ada"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
