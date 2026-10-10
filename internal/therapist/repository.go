package therapist

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
	errNotFound         = errors.New("therapist not found")
	errSlugTaken        = errors.New("therapist slug taken")
	errUnknownTherapist = errors.New("unknown therapist id")
)

type Repository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: sqlc.New(pool)}
}

func (r *Repository) ListActive(ctx context.Context) ([]Therapist, error) {
	rows, err := r.q.ListActiveTherapists(ctx)
	if err != nil {
		return nil, err
	}
	return toTherapists(rows), nil
}

func (r *Repository) List(ctx context.Context) ([]Therapist, error) {
	rows, err := r.q.ListTherapists(ctx)
	if err != nil {
		return nil, err
	}
	return toTherapists(rows), nil
}

func (r *Repository) GetActiveBySlug(ctx context.Context, slug string) (Therapist, error) {
	row, err := r.q.GetActiveTherapistBySlug(ctx, slug)
	return toTherapistResult(row, err)
}

func (r *Repository) Get(ctx context.Context, id int64) (Therapist, error) {
	row, err := r.q.GetTherapist(ctx, id)
	return toTherapistResult(row, err)
}

func (r *Repository) Create(ctx context.Context, slug string, in Input) (Therapist, error) {
	row, err := r.q.CreateTherapist(ctx, sqlc.CreateTherapistParams{
		Name:     in.Name,
		Slug:     slug,
		Title:    in.Title,
		Bio:      in.Bio,
		PhotoUrl: in.PhotoURL,
		IsActive: in.IsActive,
	})
	if pgErrorCode(err) == "23505" {
		return Therapist{}, errSlugTaken
	}
	return toTherapistResult(row, err)
}

func (r *Repository) Update(ctx context.Context, id int64, in Input) (Therapist, error) {
	row, err := r.q.UpdateTherapist(ctx, sqlc.UpdateTherapistParams{
		ID:       id,
		Name:     in.Name,
		Title:    in.Title,
		Bio:      in.Bio,
		PhotoUrl: in.PhotoURL,
		IsActive: in.IsActive,
	})
	return toTherapistResult(row, err)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	n, err := r.q.DeleteTherapist(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

func (r *Repository) ListByTreatment(ctx context.Context, treatmentID int64) ([]Therapist, error) {
	rows, err := r.q.ListTherapistsByTreatment(ctx, treatmentID)
	if err != nil {
		return nil, err
	}
	return toTherapists(rows), nil
}

func (r *Repository) ReplaceTreatmentTherapists(ctx context.Context, treatmentID int64, therapistIDs []int64) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	if err := q.DeleteTreatmentTherapists(ctx, treatmentID); err != nil {
		return err
	}

	err = q.AddTreatmentTherapists(ctx, sqlc.AddTreatmentTherapistsParams{
		TreatmentID:  treatmentID,
		TherapistIds: therapistIDs,
	})
	if pgErrorCode(err) == "23503" {
		return errUnknownTherapist
	}
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func pgErrorCode(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code
	}
	return ""
}

func toTherapistResult(row sqlc.Therapist, err error) (Therapist, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Therapist{}, errNotFound
	}
	if err != nil {
		return Therapist{}, err
	}
	return toTherapist(row), nil
}

func toTherapists(rows []sqlc.Therapist) []Therapist {
	out := make([]Therapist, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTherapist(row))
	}
	return out
}

func toTherapist(row sqlc.Therapist) Therapist {
	return Therapist{
		ID:        row.ID,
		Name:      row.Name,
		Slug:      row.Slug,
		Title:     row.Title,
		Bio:       row.Bio,
		PhotoURL:  row.PhotoUrl,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}
}
