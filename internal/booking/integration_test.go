package booking

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/migrations"
	"github.com/kemalthoriq/healthcare_be/internal/platform/database"
	"github.com/kemalthoriq/healthcare_be/internal/setting"
	"github.com/kemalthoriq/healthcare_be/internal/therapist"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

func TestIntegration_ExclusionConstraintPreventsDoubleBooking(t *testing.T) {
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		t.Skip("TEST_DATABASE_URL not set: run `make db-up` then `make test-integration`")
	}

	ctx := context.Background()
	pool, err := database.Connect(ctx, url, 20)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := database.Migrate(ctx, pool, migrations.FS, slog.New(slog.NewTextHandler(io.Discard, nil))); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	loc, err := time.LoadLocation("Asia/Jakarta")
	if err != nil {
		t.Fatalf("load location: %v", err)
	}

	repo := NewRepository(pool)
	treatments := treatment.NewService(treatment.NewRepository(pool), repo)
	therapists := therapist.NewService(therapist.NewRepository(pool), treatments, repo)
	settings := setting.NewService(setting.NewRepository(pool))
	svc := NewService(NewAvailability(treatments, therapists, settings, repo, loc), repo, &fakeNotifier{})

	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	tr, err := treatments.Create(ctx, treatment.Input{Name: "Integrasi " + suffix, Price: 100000, DurationMinutes: 60, IsActive: true})
	if err != nil {
		t.Fatalf("create treatment: %v", err)
	}
	th, err := therapists.Create(ctx, therapist.Input{Name: "Integrasi " + suffix, IsActive: true})
	if err != nil {
		t.Fatalf("create therapist: %v", err)
	}

	t.Cleanup(func() {
		pool.Exec(ctx, `WITH b AS (DELETE FROM bookings WHERE therapist_id = $1 RETURNING patient_id)
			DELETE FROM patients WHERE id IN (SELECT patient_id FROM b)`, th.ID)
		therapists.Delete(ctx, th.ID)
		treatments.Delete(ctx, tr.ID)
	})

	if _, err := therapists.SetForTreatment(ctx, tr.ID, []int64{th.ID}); err != nil {
		t.Fatalf("assign therapist: %v", err)
	}

	var week []therapist.Shift
	for d := range 7 {
		week = append(week, therapist.Shift{DayOfWeek: d, Start: 8 * 60, End: 20 * 60})
	}
	if _, err := therapists.SetSchedules(ctx, th.ID, week); err != nil {
		t.Fatalf("set schedules: %v", err)
	}

	start := StartOfDay(time.Now().In(loc)).AddDate(0, 0, 1).Add(10 * time.Hour)
	end := start.Add(time.Hour)

	const attempts = 10
	warmUp(t, ctx, pool, attempts)

	errs := make([]error, attempts)
	gate := make(chan struct{})
	var wg sync.WaitGroup

	for i := range attempts {
		wg.Go(func() {
			<-gate
			_, errs[i] = repo.Create(ctx, NewCode(), NewBooking{
				PatientName:  fmt.Sprintf("Pasien %d", i),
				Phone:        fmt.Sprintf("62899%08d", i),
				TreatmentID:  tr.ID,
				TherapistID:  th.ID,
				StartAt:      start,
				EndAt:        end,
				BlockedUntil: end.Add(15 * time.Minute),
				Price:        tr.Price,
			})
		})
	}
	close(gate)
	wg.Wait()

	succeeded, taken := 0, 0
	for _, err := range errs {
		switch {
		case err == nil:
			succeeded++
		case errors.Is(err, errSlotTaken):
			taken++
		default:
			t.Errorf("unexpected error: %v", err)
		}
	}
	t.Logf("succeeded=%d rejected_by_exclusion_constraint=%d", succeeded, taken)

	if succeeded != 1 || taken != attempts-1 {
		t.Fatalf("succeeded = %d, taken = %d; want 1 and %d", succeeded, taken, attempts-1)
	}

	_, err = svc.Create(ctx, CreateInput{
		TreatmentSlug: tr.Slug,
		TherapistSlug: th.Slug,
		StartAt:       start,
		Name:          "Pasien Terlambat",
		Phone:         "089900000099",
		Consent:       true,
	})
	if !errors.Is(err, ErrSlotUnavailable) {
		t.Errorf("booking a taken slot through the service: err = %v, want ErrSlotUnavailable", err)
	}
}

func warmUp(t *testing.T, ctx context.Context, pool *pgxpool.Pool, n int) {
	t.Helper()

	conns := make([]*pgxpool.Conn, 0, n)
	for range n {
		c, err := pool.Acquire(ctx)
		if err != nil {
			t.Fatalf("warm up pool: %v", err)
		}
		conns = append(conns, c)
	}
	for _, c := range conns {
		c.Release()
	}
}
