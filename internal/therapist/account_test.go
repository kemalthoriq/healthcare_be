package therapist

import (
	"context"
	"errors"
	"testing"
)

func (f *fakeStore) GetByUserID(ctx context.Context, userID int64) (Therapist, error) {
	for _, t := range f.items {
		if t.UserID != nil && *t.UserID == userID {
			return t, nil
		}
	}
	return Therapist{}, errNotFound
}

func (f *fakeStore) SetUserID(ctx context.Context, id, userID int64) error {
	t, ok := f.items[id]
	if !ok || t.UserID != nil {
		return errAccountExists
	}
	t.UserID = &userID
	f.items[id] = t
	return nil
}

type fakeAccounts struct {
	created []string
	err     error
}

func (f *fakeAccounts) CreateTherapistAccount(ctx context.Context, name, email, password string) (int64, error) {
	if f.err != nil {
		return 0, f.err
	}
	f.created = append(f.created, email)
	return 50, nil
}

func TestAccountService_Create(t *testing.T) {
	svc, _ := newTestService()
	th, _ := svc.Create(context.Background(), validInput())
	accounts := &fakeAccounts{}
	accountSvc := NewAccountService(svc, accounts)

	got, err := accountSvc.Create(context.Background(), th.ID, "sari@lumiere.test", "rahasia-sekali-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.UserID == nil || *got.UserID != 50 {
		t.Errorf("UserID = %v, want 50", got.UserID)
	}

	linked, err := svc.GetByUserID(context.Background(), 50)
	if err != nil || linked.ID != th.ID {
		t.Errorf("GetByUserID = %v, %v; want therapist %d", linked.ID, err, th.ID)
	}

	if _, err := accountSvc.Create(context.Background(), th.ID, "lain@lumiere.test", "rahasia-sekali-123"); !errors.Is(err, ErrAccountExists) {
		t.Errorf("second account: err = %v, want ErrAccountExists", err)
	}
	if len(accounts.created) != 1 {
		t.Errorf("accounts created = %d, want 1", len(accounts.created))
	}
}

func TestAccountService_UnknownTherapist(t *testing.T) {
	svc, _ := newTestService()

	if _, err := NewAccountService(svc, &fakeAccounts{}).Create(context.Background(), 99, "x@lumiere.test", "rahasia-sekali-123"); !errors.Is(err, ErrNotFound) {
		t.Errorf("err = %v, want ErrNotFound", err)
	}
}
