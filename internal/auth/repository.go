package auth

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

type Repository struct {
	q *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(pool)}
}

func (r *Repository) CountAdmins(ctx context.Context) (int64, error) {
	return r.q.CountAdmins(ctx)
}

func (r *Repository) CreateUser(ctx context.Context, u NewUser) (User, error) {
	row, err := r.q.CreateUser(ctx, sqlc.CreateUserParams{
		Name:         u.Name,
		Email:        u.Email,
		PasswordHash: u.PasswordHash,
		Role:         u.Role,
	})
	if err != nil {
		return User{}, err
	}

	return User{
		ID:        row.ID,
		Name:      row.Name,
		Email:     row.Email,
		Role:      row.Role,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, nil
}
