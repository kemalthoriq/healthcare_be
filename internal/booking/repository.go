package booking

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var (
	errSlotTaken      = errors.New("booking slot taken")
	errCodeTaken      = errors.New("booking code taken")
	errNotFound       = errors.New("booking not found")
	errNotCancellable = errors.New("booking not cancellable")
)

type Repository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: sqlc.New(pool)}
}

func (r *Repository) Create(ctx context.Context, code string, nb NewBooking) (Booking, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return Booking{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	patientID, err := q.UpsertPatient(ctx, sqlc.UpsertPatientParams{
		Name:  nb.PatientName,
		Phone: nb.Phone,
		Email: nb.Email,
	})
	if err != nil {
		return Booking{}, fmt.Errorf("upsert patient: %w", err)
	}

	row, err := q.CreateBooking(ctx, sqlc.CreateBookingParams{
		Code:         code,
		PatientID:    patientID,
		TreatmentID:  nb.TreatmentID,
		TherapistID:  nb.TherapistID,
		PatientName:  nb.PatientName,
		StartAt:      nb.StartAt,
		EndAt:        nb.EndAt,
		BlockedUntil: nb.BlockedUntil,
		Price:        nb.Price,
		PatientNote:  nb.PatientNote,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		switch {
		case pgErr.Code == "23P01":
			return Booking{}, errSlotTaken
		case pgErr.Code == "23505" && pgErr.ConstraintName == "bookings_code_key":
			return Booking{}, errCodeTaken
		}
	}
	if err != nil {
		return Booking{}, fmt.Errorf("create booking: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Booking{}, fmt.Errorf("commit: %w", err)
	}

	return toBooking(row), nil
}

func (r *Repository) ListActiveRanges(ctx context.Context, therapistID int64, from, to time.Time) ([]Interval, error) {
	rows, err := r.q.ListActiveBookingRanges(ctx, sqlc.ListActiveBookingRangesParams{
		TherapistID: therapistID,
		RangeStart:  from,
		RangeEnd:    to,
	})
	if err != nil {
		return nil, err
	}

	out := make([]Interval, 0, len(rows))
	for _, row := range rows {
		out = append(out, Interval{Start: row.StartAt, End: row.BlockedUntil})
	}
	return out, nil
}

func (r *Repository) GetByCode(ctx context.Context, code string) (Booking, error) {
	row, err := r.q.GetBookingByCode(ctx, code)
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, errNotFound
	}
	if err != nil {
		return Booking{}, err
	}
	return toBooking(row), nil
}

func (r *Repository) CancelByCode(ctx context.Context, code, reason string, minStartAt time.Time) (Booking, error) {
	row, err := r.q.CancelBookingByCode(ctx, sqlc.CancelBookingByCodeParams{
		Code:         code,
		CancelReason: reason,
		MinStartAt:   minStartAt,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return Booking{}, errNotCancellable
	}
	if err != nil {
		return Booking{}, err
	}
	return toBooking(row), nil
}

func toBooking(row sqlc.Booking) Booking {
	return Booking{
		ID:           row.ID,
		Code:         row.Code,
		PatientID:    row.PatientID,
		TreatmentID:  row.TreatmentID,
		TherapistID:  row.TherapistID,
		PatientName:  row.PatientName,
		StartAt:      row.StartAt,
		EndAt:        row.EndAt,
		BlockedUntil: row.BlockedUntil,
		Status:       row.Status,
		Price:        row.Price,
		PatientNote:  row.PatientNote,
		CancelReason: row.CancelReason,
		ConfirmedAt:  row.ConfirmedAt,
		CancelledAt:  row.CancelledAt,
		CompletedAt:  row.CompletedAt,
		CreatedAt:    row.CreatedAt,
	}
}
