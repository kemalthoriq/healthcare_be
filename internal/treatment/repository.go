package treatment

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var (
	errNotFound  = errors.New("treatment not found")
	errSlugTaken = errors.New("treatment slug taken")
)

type Repository struct {
	q *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(pool)}
}

func (r *Repository) ListActive(ctx context.Context) ([]Treatment, error) {
	rows, err := r.q.ListActiveTreatments(ctx)
	if err != nil {
		return nil, err
	}
	return toTreatments(rows), nil
}

func (r *Repository) List(ctx context.Context) ([]Treatment, error) {
	rows, err := r.q.ListTreatments(ctx)
	if err != nil {
		return nil, err
	}
	return toTreatments(rows), nil
}

func (r *Repository) GetActiveBySlug(ctx context.Context, slug string) (Treatment, error) {
	row, err := r.q.GetActiveTreatmentBySlug(ctx, slug)
	return toTreatmentResult(row, err)
}

func (r *Repository) Get(ctx context.Context, id int64) (Treatment, error) {
	row, err := r.q.GetTreatment(ctx, id)
	return toTreatmentResult(row, err)
}

func (r *Repository) Create(ctx context.Context, slug string, in Input) (Treatment, error) {
	row, err := r.q.CreateTreatment(ctx, sqlc.CreateTreatmentParams{
		Name:            in.Name,
		Slug:            slug,
		Description:     in.Description,
		ImageUrl:        in.ImageURL,
		Price:           in.Price,
		DurationMinutes: in.DurationMinutes,
		IsActive:        in.IsActive,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Treatment{}, errSlugTaken
	}
	return toTreatmentResult(row, err)
}

func (r *Repository) Update(ctx context.Context, id int64, in Input) (Treatment, error) {
	row, err := r.q.UpdateTreatment(ctx, sqlc.UpdateTreatmentParams{
		ID:              id,
		Name:            in.Name,
		Description:     in.Description,
		ImageUrl:        in.ImageURL,
		Price:           in.Price,
		DurationMinutes: in.DurationMinutes,
		IsActive:        in.IsActive,
	})
	return toTreatmentResult(row, err)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	n, err := r.q.DeleteTreatment(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

func toTreatmentResult(row sqlc.Treatment, err error) (Treatment, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Treatment{}, errNotFound
	}
	if err != nil {
		return Treatment{}, err
	}
	return toTreatment(row), nil
}

func toTreatments(rows []sqlc.Treatment) []Treatment {
	out := make([]Treatment, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTreatment(row))
	}
	return out
}

func toTreatment(row sqlc.Treatment) Treatment {
	return Treatment{
		ID:              row.ID,
		Name:            row.Name,
		Slug:            row.Slug,
		Description:     row.Description,
		ImageURL:        row.ImageUrl,
		Price:           row.Price,
		DurationMinutes: row.DurationMinutes,
		IsActive:        row.IsActive,
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}
