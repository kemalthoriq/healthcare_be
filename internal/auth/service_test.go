package auth

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"
)

const testPassword = "rahasia-sekali-123"

type fakeStore struct {
	adminCount int64
	countErr   error
	created    []NewUser
	users      map[string]User
	hashes     map[string]string
	sessions   map[string]int64
}

func newFakeStore() *fakeStore {
	return &fakeStore{users: map[string]User{}, hashes: map[string]string{}, sessions: map[string]int64{}}
}

func (f *fakeStore) addUser(u User, password string) {
	f.users[u.Email] = u
	f.hashes[u.Email] = HashPassword(password)
}

func (f *fakeStore) CountAdmins(ctx context.Context) (int64, error) {
	return f.adminCount, f.countErr
}

func (f *fakeStore) CreateUser(ctx context.Context, u NewUser) (User, error) {
	f.created = append(f.created, u)
	return User{ID: 1, Name: u.Name, Email: u.Email, Role: u.Role, IsActive: true}, nil
}

func (f *fakeStore) GetUserByEmail(ctx context.Context, email string) (User, string, error) {
	u, ok := f.users[email]
	if !ok {
		return User{}, "", errNotFound
	}
	return u, f.hashes[email], nil
}

func (f *fakeStore) CreateSession(ctx context.Context, id string, userID int64, expiresAt time.Time) error {
	f.sessions[id] = userID
	return nil
}

func (f *fakeStore) GetSessionUser(ctx context.Context, id string) (User, error) {
	userID, ok := f.sessions[id]
	if !ok {
		return User{}, errNotFound
	}
	for _, u := range f.users {
		if u.ID == userID {
			return u, nil
		}
	}
	return User{}, errNotFound
}

func (f *fakeStore) DeleteSession(ctx context.Context, id string) error {
	delete(f.sessions, id)
	return nil
}

func (f *fakeStore) DeleteExpiredSessions(ctx context.Context) error {
	return nil
}

func newTestService(store *fakeStore) *Service {
	return NewService(store, slog.New(slog.NewTextHandler(io.Discard, nil)), 24*time.Hour)
}

var testSeed = AdminSeed{Name: "Admin", Email: "Admin@Lumiere.Test", Password: testPassword}

func TestEnsureAdmin_CreatesAdminWhenNoneExists(t *testing.T) {
	store := newFakeStore()

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
	store := newFakeStore()
	store.adminCount = 1

	if err := newTestService(store).EnsureAdmin(context.Background(), testSeed); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.created) != 0 {
		t.Errorf("created %d users, want 0", len(store.created))
	}
}

func TestEnsureAdmin_SkipsWhenSeedIsEmpty(t *testing.T) {
	store := newFakeStore()

	if err := newTestService(store).EnsureAdmin(context.Background(), AdminSeed{}); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(store.created) != 0 {
		t.Errorf("created %d users, want 0", len(store.created))
	}
}

func TestEnsureAdmin_ReturnsStoreError(t *testing.T) {
	store := newFakeStore()
	store.countErr = errors.New("connection refused")

	if err := newTestService(store).EnsureAdmin(context.Background(), testSeed); err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestLogin_Success(t *testing.T) {
	store := newFakeStore()
	store.addUser(User{ID: 7, Email: "admin@lumiere.test", Role: RoleAdmin, IsActive: true}, testPassword)
	svc := newTestService(store)
	svc.now = func() time.Time { return time.Date(2026, 10, 11, 9, 0, 0, 0, time.UTC) }

	user, session, err := svc.Login(context.Background(), "admin@lumiere.test", testPassword)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if user.ID != 7 {
		t.Errorf("user.ID = %d, want 7", user.ID)
	}
	if session.Token == "" {
		t.Fatal("empty session token")
	}
	if want := time.Date(2026, 10, 12, 9, 0, 0, 0, time.UTC); !session.ExpiresAt.Equal(want) {
		t.Errorf("ExpiresAt = %v, want %v", session.ExpiresAt, want)
	}
	if _, stored := store.sessions[session.Token]; stored {
		t.Error("raw token must not be stored")
	}
	if store.sessions[hashToken(session.Token)] != 7 {
		t.Error("hashed token not stored for the user")
	}
}

func TestLogin_Rejects(t *testing.T) {
	tests := []struct {
		name     string
		email    string
		password string
		active   bool
		wantErr  error
	}{
		{"unknown email", "nobody@lumiere.test", testPassword, true, ErrInvalidCredentials},
		{"wrong password", "admin@lumiere.test", "salah-password", true, ErrInvalidCredentials},
		{"disabled account", "admin@lumiere.test", testPassword, false, ErrAccountDisabled},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newFakeStore()
			store.addUser(User{ID: 7, Email: "admin@lumiere.test", Role: RoleAdmin, IsActive: tt.active}, testPassword)

			_, _, err := newTestService(store).Login(context.Background(), tt.email, tt.password)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
			if len(store.sessions) != 0 {
				t.Error("no session should be created")
			}
		})
	}
}

func TestAuthenticate(t *testing.T) {
	store := newFakeStore()
	store.addUser(User{ID: 7, Email: "admin@lumiere.test", Role: RoleAdmin, IsActive: true}, testPassword)
	svc := newTestService(store)

	_, session, err := svc.Login(context.Background(), "admin@lumiere.test", testPassword)
	if err != nil {
		t.Fatalf("login: %v", err)
	}

	user, err := svc.Authenticate(context.Background(), session.Token)
	if err != nil || user.ID != 7 {
		t.Fatalf("Authenticate(valid) = %v, %v; want user 7", user.ID, err)
	}

	for _, token := range []string{"", "unknown-token"} {
		if _, err := svc.Authenticate(context.Background(), token); err == nil {
			t.Errorf("Authenticate(%q) expected error", token)
		}
	}
}

func TestLogout_InvalidatesSession(t *testing.T) {
	store := newFakeStore()
	store.addUser(User{ID: 7, Email: "admin@lumiere.test", Role: RoleAdmin, IsActive: true}, testPassword)
	svc := newTestService(store)

	_, session, _ := svc.Login(context.Background(), "admin@lumiere.test", testPassword)
	if err := svc.Logout(context.Background(), session.Token); err != nil {
		t.Fatalf("logout: %v", err)
	}

	if _, err := svc.Authenticate(context.Background(), session.Token); err == nil {
		t.Error("session should be invalid after logout")
	}
}
