package promo

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var (
	errNotFound         = errors.New("promo not found")
	errSlugTaken        = errors.New("promo slug taken")
	errUnknownTreatment = errors.New("unknown treatment")
)

type Repository struct {
	pool *pgxpool.Pool
	q    *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{pool: pool, q: sqlc.New(pool)}
}

func (r *Repository) ListRunning(ctx context.Context, today time.Time) ([]Promo, error) {
	rows, err := r.q.ListRunningPromos(ctx, toDate(today))
	if err != nil {
		return nil, err
	}
	return r.withTreatments(ctx, rows)
}

func (r *Repository) GetRunningBySlug(ctx context.Context, slug string, today time.Time) (Promo, error) {
	row, err := r.q.GetRunningPromoBySlug(ctx, sqlc.GetRunningPromoBySlugParams{Slug: slug, Today: toDate(today)})
	return r.one(ctx, row, err)
}

func (r *Repository) List(ctx context.Context, limit, offset int) ([]Promo, int64, error) {
	rows, err := r.q.ListPromos(ctx, sqlc.ListPromosParams{Limit: int32(limit), Offset: int32(offset)})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.q.CountPromos(ctx)
	if err != nil {
		return nil, 0, err
	}
	promos, err := r.withTreatments(ctx, rows)
	return promos, total, err
}

func (r *Repository) Get(ctx context.Context, id int64) (Promo, error) {
	row, err := r.q.GetPromo(ctx, id)
	return r.one(ctx, row, err)
}

func (r *Repository) Create(ctx context.Context, slug string, v Values) (int64, error) {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	id, err := q.CreatePromo(ctx, sqlc.CreatePromoParams{
		Title:       v.Title,
		Slug:        slug,
		Description: v.Description,
		ImageUrl:    v.ImageURL,
		StartDate:   toDate(v.StartDate),
		EndDate:     toDate(v.EndDate),
		IsActive:    v.IsActive,
	})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "promos_slug_key" {
		return 0, errSlugTaken
	}
	if err != nil {
		return 0, err
	}

	if err := addTreatments(ctx, q, id, v.TreatmentIDs); err != nil {
		return 0, err
	}
	return id, tx.Commit(ctx)
}

func (r *Repository) Update(ctx context.Context, id int64, v Values) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	n, err := q.UpdatePromo(ctx, sqlc.UpdatePromoParams{
		ID:          id,
		Title:       v.Title,
		Description: v.Description,
		ImageUrl:    v.ImageURL,
		StartDate:   toDate(v.StartDate),
		EndDate:     toDate(v.EndDate),
		IsActive:    v.IsActive,
	})
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}

	if err := q.DeletePromoTreatments(ctx, id); err != nil {
		return err
	}
	if err := addTreatments(ctx, q, id, v.TreatmentIDs); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) Delete(ctx context.Context, id int64) error {
	n, err := r.q.DeletePromo(ctx, id)
	if err != nil {
		return err
	}
	if n == 0 {
		return errNotFound
	}
	return nil
}

func addTreatments(ctx context.Context, q *sqlc.Queries, promoID int64, ids []int64) error {
	if len(ids) == 0 {
		return nil
	}
	err := q.AddPromoTreatments(ctx, sqlc.AddPromoTreatmentsParams{PromoID: promoID, TreatmentIds: ids})
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return errUnknownTreatment
	}
	return err
}

func (r *Repository) one(ctx context.Context, row sqlc.Promo, err error) (Promo, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return Promo{}, errNotFound
	}
	if err != nil {
		return Promo{}, err
	}
	promos, err := r.withTreatments(ctx, []sqlc.Promo{row})
	if err != nil {
		return Promo{}, err
	}
	return promos[0], nil
}

func (r *Repository) withTreatments(ctx context.Context, rows []sqlc.Promo) ([]Promo, error) {
	out := make([]Promo, 0, len(rows))
	if len(rows) == 0 {
		return out, nil
	}

	ids := make([]int64, 0, len(rows))
	for _, row := range rows {
		ids = append(ids, row.ID)
	}
	links, err := r.q.ListPromoTreatments(ctx, ids)
	if err != nil {
		return nil, err
	}
	byPromo := map[int64][]Treatment{}
	for _, l := range links {
		byPromo[l.PromoID] = append(byPromo[l.PromoID], Treatment{ID: l.ID, Name: l.Name, Slug: l.Slug, IsActive: l.IsActive})
	}

	for _, row := range rows {
		treatments := byPromo[row.ID]
		if treatments == nil {
			treatments = []Treatment{}
		}
		out = append(out, Promo{
			ID:          row.ID,
			Title:       row.Title,
			Slug:        row.Slug,
			Description: row.Description,
			ImageURL:    row.ImageUrl,
			StartDate:   row.StartDate.Time,
			EndDate:     row.EndDate.Time,
			IsActive:    row.IsActive,
			Treatments:  treatments,
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
	}
	return out, nil
}

func toDate(t time.Time) pgtype.Date {
	return pgtype.Date{Time: t, Valid: true}
}
