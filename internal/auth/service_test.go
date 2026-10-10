package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
)

type fakeUserStore struct {
	adminCount int64
	countErr   error
	created    []NewUser
}

func (f *fakeUserStore) CountAdmins(ctx context.Context) (int64, error) {
	return f.adminCount, f.countErr
}

func (f *fakeUserStore) CreateUser(ctx context.Context, u NewUser) (User, error) {
	f.created = append(f.created, u)
	return User{ID: 1, Name: u.Name, Email: u.Email, Role: u.Role, IsActive: true}, nil
}

func newTestService(store *fakeUserStore) *Service {
	return NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)))
}

var testSeed = AdminSeed{Name: "Admin", Email: "Admin@Lumiere.Test", Password: "rahasia-sekali-123"}

func TestEnsureAdmin_CreatesAdminWhenNoneExists(t *testing.T) {
	store := &fakeUserStore{}

	if err := newTestService(store).EnsureAdmin(context.Background(), testSeed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(store.created) != 1 {
		t.Fatalf("created %d users, want 1", len(store.created))
	}
	got := store.created[0]
	if got.Role != RoleAdmin || got.Email != "admin@lumiere.test" {
		t.Errorf("got role %q email %q, want admin admin@lumiere.test", got.Role, got.Email)
	}
	if ok, _ := VerifyPassword(testSeed.Password, got.PasswordHash); !ok {
		t.Error("stored hash does not match the seed password")
	}
}

func TestEnsureAdmin_SkipsWhenAdminExists(t *testing.T) {
	store := &fakeUserStore{adminCount: 1}

	if err := newTestService(store).EnsureAdmin(context.Background(), testSeed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.created) != 0 {
		t.Errorf("created %d users, want 0", len(store.created))
	}
}

func TestEnsureAdmin_SkipsWhenSeedIsEmpty(t *testing.T) {
	store := &fakeUserStore{}

	if err := newTestService(store).EnsureAdmin(context.Background(), AdminSeed{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.created) != 0 {
		t.Errorf("created %d users, want 0", len(store.created))
	}
}

func TestEnsureAdmin_ReturnsStoreError(t *testing.T) {
	store := &fakeUserStore{countErr: errors.New("connection refused")}

	if err := newTestService(store).EnsureAdmin(context.Background(), testSeed); err == nil {
		t.Fatal("expected error, got nil")
	}
}
