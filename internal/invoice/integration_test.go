package invoice

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"slices"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/migrations"
	"github.com/kemalthoriq/healthcare_be/internal/platform/database"
)

func TestIntegration_InvoiceNumbersAreSequentialUnderConcurrency(t *testing.T) {
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

	const attempts = 10
	suffix := strconv.FormatInt(time.Now().UnixNano(), 36)
	period := "T" + suffix
	userID, bookingIDs := seedCompletedBookings(t, ctx, pool, suffix, attempts+1)

	t.Cleanup(func() {
		pool.Exec(ctx, `DELETE FROM invoices WHERE booking_id = ANY($1)`, bookingIDs)
		pool.Exec(ctx, `DELETE FROM invoice_counters WHERE period = $1`, period)
		pool.Exec(ctx, `WITH b AS (DELETE FROM bookings WHERE id = ANY($1) RETURNING patient_id, treatment_id, therapist_id),
			p AS (DELETE FROM patients WHERE id IN (SELECT patient_id FROM b)),
			tr AS (DELETE FROM treatments WHERE id IN (SELECT treatment_id FROM b))
			DELETE FROM therapists WHERE id IN (SELECT therapist_id FROM b)`, bookingIDs)
		pool.Exec(ctx, `DELETE FROM users WHERE id = $1`, userID)
	})

	repo := NewRepository(pool)
	warmUp(t, ctx, pool, attempts)

	ids := make([]int64, attempts)
	errs := make([]error, attempts)
	gate := make(chan struct{})
	var wg sync.WaitGroup
	for i := range attempts {
		wg.Go(func() {
			<-gate
			ids[i], errs[i] = repo.Create(ctx, period, NewInvoice{BookingID: bookingIDs[i], Subtotal: 100000, CreatedBy: userID})
		})
	}
	close(gate)
	wg.Wait()

	var numbers, want []string
	for i, err := range errs {
		if err != nil {
			t.Fatalf("create %d: %v", i, err)
		}
		inv, err := repo.Get(ctx, ids[i])
		if err != nil {
			t.Fatalf("get %d: %v", i, err)
		}
		numbers = append(numbers, inv.Number)
		want = append(want, FormatNumber(period, int32(i+1)))
	}
	slices.Sort(numbers)
	if !slices.Equal(numbers, want) {
		t.Fatalf("numbers = %v\nwant      %v", numbers, want)
	}

	if _, err := repo.Create(ctx, period, NewInvoice{BookingID: bookingIDs[0], Subtotal: 100000, CreatedBy: userID}); !errors.Is(err, errInvoiceExists) {
		t.Fatalf("duplicate invoice: err = %v, want errInvoiceExists", err)
	}

	id, err := repo.Create(ctx, period, NewInvoice{BookingID: bookingIDs[attempts], Subtotal: 100000, CreatedBy: userID})
	if err != nil {
		t.Fatalf("create after failed attempt: %v", err)
	}
	inv, _ := repo.Get(ctx, id)
	if inv.Number != FormatNumber(period, attempts+1) {
		t.Errorf("number after rolled back attempt = %s, want %s (no gap)", inv.Number, FormatNumber(period, attempts+1))
	}
}

func seedCompletedBookings(t *testing.T, ctx context.Context, pool *pgxpool.Pool, suffix string, n int) (int64, []int64) {
	t.Helper()

	var userID, treatmentID, therapistID, patientID int64
	err := pool.QueryRow(ctx, `INSERT INTO users (name, email, password_hash, role) VALUES ('Integrasi', $1, 'x', 'admin') RETURNING id`,
		suffix+"@integrasi.test").Scan(&userID)
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO treatments (name, slug, price, duration_minutes) VALUES ('Integrasi', $1, 100000, 60) RETURNING id`,
			"integrasi-"+suffix).Scan(&treatmentID)
	}
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO therapists (name, slug) VALUES ('Integrasi', $1) RETURNING id`,
			"integrasi-"+suffix).Scan(&therapistID)
	}
	if err == nil {
		err = pool.QueryRow(ctx, `INSERT INTO patients (name, phone, consent_at) VALUES ('Integrasi', $1, now()) RETURNING id`,
			"62899"+suffix).Scan(&patientID)
	}
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	ids := make([]int64, n)
	start := time.Now().Add(-48 * time.Hour)
	for i := range n {
		err := pool.QueryRow(ctx, `INSERT INTO bookings (code, patient_id, treatment_id, therapist_id, patient_name, start_at, end_at, blocked_until, status, price)
			VALUES ($1, $2, $3, $4, 'Integrasi', $5, $6, $6, 'completed', 100000) RETURNING id`,
			"INT-"+suffix+"-"+strconv.Itoa(i), patientID, treatmentID, therapistID, start, start.Add(time.Hour)).Scan(&ids[i])
		if err != nil {
			t.Fatalf("seed booking: %v", err)
		}
	}
	return userID, ids
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
