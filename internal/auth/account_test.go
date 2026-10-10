package auth

import (
	"context"
	"errors"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

type emailTakenStore struct {
	*fakeStore
}

func (emailTakenStore) CreateUser(ctx context.Context, u NewUser) (User, error) {
	return User{}, errEmailTaken
}

func TestCreateTherapistAccount(t *testing.T) {
	store := newFakeStore()

	id, err := newTestService(store).CreateTherapistAccount(context.Background(), "dr. Sari", " Sari@Lumiere.Test ", testPassword)
	if err != nil || id == 0 {
		t.Fatalf("got id %d, err %v", id, err)
	}

	got := store.created[0]
	if got.Role != RoleTherapist || got.Email != "sari@lumiere.test" || got.Name != "dr. Sari" {
		t.Errorf("unexpected user: %+v", got)
	}
	if ok, _ := VerifyPassword(testPassword, got.PasswordHash); !ok {
		t.Error("stored hash does not match the password")
	}
}

func TestCreateTherapistAccount_Validation(t *testing.T) {
	_, err := newTestService(newFakeStore()).CreateTherapistAccount(context.Background(), "dr. Sari", "bukan-email", "pendek")

	var appErr *apperr.Error
	if !errors.As(err, &appErr) || appErr.Fields["email"] == "" || appErr.Fields["password"] == "" {
		t.Errorf("expected validation errors on email and password, got %v", err)
	}
}

func TestCreateTherapistAccount_EmailTaken(t *testing.T) {
	svc := NewService(emailTakenStore{newFakeStore()}, newTestService(newFakeStore()).log, 0)

	if _, err := svc.CreateTherapistAccount(context.Background(), "dr. Sari", "sari@lumiere.test", testPassword); !errors.Is(err, ErrEmailTaken) {
		t.Errorf("err = %v, want ErrEmailTaken", err)
	}
}
