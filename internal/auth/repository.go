package auth

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var errNotFound = errors.New("not found")

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

func (r *Repository) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	row, err := r.q.GetUserByEmail(ctx, email)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, "", errNotFound
	}
	if err != nil {
		return User{}, "", err
	}

	return User{
		ID:        row.ID,
		Name:      row.Name,
		Email:     row.Email,
		Role:      row.Role,
		IsActive:  row.IsActive,
		CreatedAt: row.CreatedAt,
		UpdatedAt: row.UpdatedAt,
	}, row.PasswordHash, nil
}

func (r *Repository) CreateSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	return r.q.CreateSession(ctx, sqlc.CreateSessionParams{
		ID:        id,
		UserID:    userID,
		ExpiresAt: expiresAt,
	})
}

func (r *Repository) GetSessionUser(ctx context.Context, id string) (User, error) {
	row, err := r.q.GetSessionUser(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return User{}, errNotFound
	}
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

func (r *Repository) DeleteSession(ctx context.Context, id string) error {
	return r.q.DeleteSession(ctx, id)
}

func (r *Repository) DeleteExpiredSessions(ctx context.Context) error {
	return r.q.DeleteExpiredSessions(ctx)
}
