package invoice

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var (
	errNotFound      = errors.New("invoice not found")
	errInvoiceExists = errors.New("booking already has an invoice")
	errAlreadyPaid   = errors.New("invoice already paid")
)

type Repository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: sqlc.New(pool)}
}

func (r *Repository) Create(ctx context.Context, period string, in NewInvoice) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	seq, err := q.NextInvoiceNumber(ctx, period)
	if err != nil {
		return 0, fmt.Errorf("next invoice number: %w", err)
	}

	id, err := q.CreateInvoice(ctx, sqlc.CreateInvoiceParams{
		Number:    FormatNumber(period, seq),
		BookingID: in.BookingID,
		Subtotal:  in.Subtotal,
		Discount:  in.Discount,
		Total:     in.Subtotal - in.Discount,
		CreatedBy: in.CreatedBy,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "invoices_booking_id_key" {
		return 0, errInvoiceExists
	}
	if err != nil {
		return 0, fmt.Errorf("create invoice: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit: %w", err)
	}
	return id, nil
}

func (r *Repository) Get(ctx context.Context, id int64) (Invoice, error) {
	row, err := r.q.GetInvoice(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Invoice{}, errNotFound
	}
	if err != nil {
		return Invoice{}, err
	}
	return toInvoice(row), nil
}

func (r *Repository) List(ctx context.Context, f Filter, pattern string, limit, offset int) ([]Invoice, int64, error) {
	rows, err := r.q.ListInvoices(ctx, sqlc.ListInvoicesParams{
		Status:     f.Status,
		BookingID:  f.BookingID,
		Pattern:    pattern,
		PageLimit:  int32(limit),
		PageOffset: int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountInvoices(ctx, sqlc.CountInvoicesParams{
		Status:    f.Status,
		BookingID: f.BookingID,
		Pattern:   pattern,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]Invoice, 0, len(rows))
	for _, row := range rows {
		out = append(out, toInvoice(sqlc.GetInvoiceRow(row)))
	}
	return out, total, nil
}

func (r *Repository) UpdateDiscount(ctx context.Context, id, discount int64) error {
	_, err := r.q.UpdateInvoiceDiscount(ctx, sqlc.UpdateInvoiceDiscountParams{ID: id, Discount: discount})
	if errors.Is(err, pgx.ErrNoRows) {
		return errAlreadyPaid
	}
	return err
}

func (r *Repository) Pay(ctx context.Context, id int64, method string) error {
	_, err := r.q.PayInvoice(ctx, sqlc.PayInvoiceParams{ID: id, PaymentMethod: method})
	if errors.Is(err, pgx.ErrNoRows) {
		return errAlreadyPaid
	}
	return err
}

func toInvoice(row sqlc.GetInvoiceRow) Invoice {
	return Invoice{
		ID:             row.ID,
		Number:         row.Number,
		BookingID:      row.BookingID,
		Subtotal:       row.Subtotal,
		Discount:       row.Discount,
		Total:          row.Total,
		Status:         row.Status,
		PaymentMethod:  row.PaymentMethod,
		PaidAt:         row.PaidAt,
		CreatedBy:      row.CreatedBy,
		CreatedAt:      row.CreatedAt,
		UpdatedAt:      row.UpdatedAt,
		BookingCode:    row.BookingCode,
		BookingStartAt: row.BookingStartAt,
		PatientName:    row.PatientName,
		PatientPhone:   row.PatientPhone,
		TreatmentName:  row.TreatmentName,
		TherapistName:  row.TherapistName,
	}
}
