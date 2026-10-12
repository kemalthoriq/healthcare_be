package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

var (
	ErrInvalidCredentials = apperr.New(apperr.KindUnauthenticated, "INVALID_CREDENTIALS", "Email atau password salah.")
	ErrAccountDisabled    = apperr.New(apperr.KindForbidden, "ACCOUNT_DISABLED", "Akun Anda dinonaktifkan.")
)

var dummyHash = HashPassword("dummy-password-for-constant-time")

type store interface {
	CountAdmins(ctx context.Context) (int64, error)
	CreateUser(ctx context.Context, u NewUser) (User, error)
	GetUserByEmail(ctx context.Context, email string) (User, string, error)
	CreateSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error
	GetSessionUser(ctx context.Context, id string) (User, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteExpiredSessions(ctx context.Context) error
}

type Service struct {
	store      store
	log        *slog.Logger
	sessionTTL time.Duration
	now        func() time.Time
}

func NewService(store store, log *slog.Logger, sessionTTL time.Duration) *Service {
	return &Service{store: store, log: log, sessionTTL: sessionTTL, now: time.Now}
}

type AdminSeed struct {
	Name     string
	Email    string
	Password string
}

func (s *Service) EnsureAdmin(ctx context.Context, seed AdminSeed) error {
	count, err := s.store.CountAdmins(ctx)
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

	user, err := s.store.CreateUser(ctx, NewUser{
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

type Session struct {
	Token     string
	ExpiresAt time.Time
}

func (s *Service) Login(ctx context.Context, email, password string) (User, Session, error) {
	user, hash, err := s.store.GetUserByEmail(ctx, email)
	if errors.Is(err, errNotFound) {
		_, _ = VerifyPassword(password, dummyHash)
		return User{}, Session{}, ErrInvalidCredentials
	}
	if err != nil {
		return User{}, Session{}, fmt.Errorf("get user: %w", err)
	}

	ok, err := VerifyPassword(password, hash)
	if err != nil {
		return User{}, Session{}, fmt.Errorf("verify password: %w", err)
	}
	if !ok {
		return User{}, Session{}, ErrInvalidCredentials
	}
	if !user.IsActive {
		return User{}, Session{}, ErrAccountDisabled
	}

	if err := s.store.DeleteExpiredSessions(ctx); err != nil {
		s.log.WarnContext(ctx, "delete expired sessions failed", "err", err)
	}

	session := Session{Token: rand.Text(), ExpiresAt: s.now().Add(s.sessionTTL)}
	if err := s.store.CreateSession(ctx, hashToken(session.Token), user.ID, session.ExpiresAt); err != nil {
		return User{}, Session{}, fmt.Errorf("create session: %w", err)
	}

	return user, session, nil
}

func (s *Service) Authenticate(ctx context.Context, token string) (User, error) {
	if token == "" {
		return User{}, apperr.Unauthenticated()
	}

	user, err := s.store.GetSessionUser(ctx, hashToken(token))
	if errors.Is(err, errNotFound) {
		return User{}, apperr.Unauthenticated()
	}
	if err != nil {
		return User{}, fmt.Errorf("get session: %w", err)
	}

	return user, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	if err := s.store.DeleteSession(ctx, hashToken(token)); err != nil {
		return fmt.Errorf("delete session: %w", err)
	}
	return nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}
