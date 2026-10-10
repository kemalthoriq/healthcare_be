package auth

import (
	"context"
	"errors"
	"net/mail"
	"strings"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const minPasswordLength = 12

var ErrEmailTaken = apperr.Conflict("EMAIL_TAKEN", "Email ini sudah dipakai akun lain.")

func (s *Service) CreateTherapistAccount(ctx context.Context, name, email, password string) (int64, error) {
	email = strings.ToLower(strings.TrimSpace(email))

	fields := map[string]string{}
	if addr, err := mail.ParseAddress(email); err != nil || addr.Address != email {
		fields["email"] = "Format email belum sesuai."
	}
	if len(password) < minPasswordLength {
		fields["password"] = "Password minimal 12 karakter."
	}
	if len(fields) > 0 {
		return 0, apperr.Validation(fields)
	}

	user, err := s.store.CreateUser(ctx, NewUser{
		Name:         name,
		Email:        email,
		PasswordHash: HashPassword(password),
		Role:         RoleTherapist,
	})
	if errors.Is(err, errEmailTaken) {
		return 0, ErrEmailTaken
	}
	if err != nil {
		return 0, err
	}
	return user.ID, nil
}
