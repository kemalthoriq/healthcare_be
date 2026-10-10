package auth

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
)

type userStore interface {
	CountAdmins(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, u NewUser) (User, error)
}

type Service struct {
	users userStore
	log   *slog.Logger
}

func NewService(users userStore, log *slog.Logger) *Service {
	return &Service{users: users, log: log}
}

type AdminSeed struct {
	Name     string
	Email    string
	Password string
}

func (s *Service) EnsureAdmin(ctx context.Context, seed AdminSeed) error {
	count, err := s.users.CountAdmins(ctx)
	if err != nil {
		return fmt.Errorf("count admins: %w", err)
	}
	if count > 0 {
		return nil
	}

	if seed.Email == "" {
		s.log.WarnContext(ctx, "no admin account exists; set SEED_ADMIN_EMAIL and SEED_ADMIN_PASSWORD to create one")
		return nil
	}

	user, err := s.users.CreateUser(ctx, NewUser{
		Name:         seed.Name,
		Email:        strings.ToLower(seed.Email),
		PasswordHash: HashPassword(seed.Password),
		Role:         RoleAdmin,
	})
	if err != nil {
		return fmt.Errorf("create initial admin: %w", err)
	}

	s.log.InfoContext(ctx, "initial admin created", "user_id", user.ID, "email", user.Email)
	return nil
}
