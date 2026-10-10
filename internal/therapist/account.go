package therapist

import (
	"context"
	"errors"
	"net/http"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

var ErrAccountExists = apperr.Conflict("ACCOUNT_EXISTS", "Terapis ini sudah memiliki akun login.")

type accountCreator interface {
	CreateTherapistAccount(ctx context.Context, name, email, password string) (int64, error)
}

type AccountService struct {
	therapists *Service
	accounts   accountCreator
}

func NewAccountService(therapists *Service, accounts accountCreator) *AccountService {
	return &AccountService{therapists: therapists, accounts: accounts}
}

func (s *AccountService) Create(ctx context.Context, therapistID int64, email, password string) (Therapist, error) {
	th, err := s.therapists.Get(ctx, therapistID)
	if err != nil {
		return Therapist{}, err
	}
	if th.UserID != nil {
		return Therapist{}, ErrAccountExists
	}

	userID, err := s.accounts.CreateTherapistAccount(ctx, th.Name, email, password)
	if err != nil {
		return Therapist{}, err
	}

	err = s.therapists.store.SetUserID(ctx, therapistID, userID)
	if errors.Is(err, errAccountExists) {
		return Therapist{}, ErrAccountExists
	}
	if err != nil {
		return Therapist{}, err
	}

	return s.therapists.Get(ctx, therapistID)
}

type AccountHandler struct {
	service *AccountService
}

func NewAccountHandler(service *AccountService) *AccountHandler {
	return &AccountHandler{service: service}
}

func (h *AccountHandler) Create(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	var req struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	th, err := h.service.Create(r.Context(), id, req.Email, req.Password)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, toResponse(th))
}
