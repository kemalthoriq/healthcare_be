package setting

import (
	"context"
	"net/http"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type Settings struct {
	BufferMinutes         int32     `json:"buffer_minutes"`
	MinBookingLeadMinutes int32     `json:"min_booking_lead_minutes"`
	MaxBookingDaysAhead   int32     `json:"max_booking_days_ahead"`
	CancelDeadlineHours   int32     `json:"cancel_deadline_hours"`
	UpdatedAt             time.Time `json:"updated_at"`
}

type store interface {
	Get(ctx context.Context) (Settings, error)
	Update(ctx context.Context, s Settings) (Settings, error)
}

type Repository struct {
	q *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(pool)}
}

func (r *Repository) Get(ctx context.Context) (Settings, error) {
	row, err := r.q.GetClinicSettings(ctx)
	if err != nil {
		return Settings{}, err
	}
	return toSettings(row), nil
}

func (r *Repository) Update(ctx context.Context, s Settings) (Settings, error) {
	row, err := r.q.UpdateClinicSettings(ctx, sqlc.UpdateClinicSettingsParams{
		BufferMinutes:         s.BufferMinutes,
		MinBookingLeadMinutes: s.MinBookingLeadMinutes,
		MaxBookingDaysAhead:   s.MaxBookingDaysAhead,
		CancelDeadlineHours:   s.CancelDeadlineHours,
	})
	if err != nil {
		return Settings{}, err
	}
	return toSettings(row), nil
}

func toSettings(row sqlc.ClinicSetting) Settings {
	return Settings{
		BufferMinutes:         row.BufferMinutes,
		MinBookingLeadMinutes: row.MinBookingLeadMinutes,
		MaxBookingDaysAhead:   row.MaxBookingDaysAhead,
		CancelDeadlineHours:   row.CancelDeadlineHours,
		UpdatedAt:             row.UpdatedAt,
	}
}

type Service struct {
	store store
}

func NewService(store store) *Service {
	return &Service{store: store}
}

func (s *Service) Get(ctx context.Context) (Settings, error) {
	return s.store.Get(ctx)
}

func (s *Service) Update(ctx context.Context, in Settings) (Settings, error) {
	fields := map[string]string{}
	if in.BufferMinutes < 0 || in.BufferMinutes > 120 || in.BufferMinutes%5 != 0 {
		fields["buffer_minutes"] = "Jeda harus 0–120 menit dan kelipatan 5."
	}
	if in.MinBookingLeadMinutes < 0 || in.MinBookingLeadMinutes > 10080 {
		fields["min_booking_lead_minutes"] = "Batas minimal booking harus 0–10080 menit (7 hari)."
	}
	if in.MaxBookingDaysAhead < 1 || in.MaxBookingDaysAhead > 365 {
		fields["max_booking_days_ahead"] = "Batas maksimal booking harus 1–365 hari."
	}
	if in.CancelDeadlineHours < 0 || in.CancelDeadlineHours > 168 {
		fields["cancel_deadline_hours"] = "Batas pembatalan harus 0–168 jam."
	}
	if len(fields) > 0 {
		return Settings{}, apperr.Validation(fields)
	}

	return s.store.Update(ctx, in)
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.Get(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, s)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		BufferMinutes         int32 `json:"buffer_minutes"`
		MinBookingLeadMinutes int32 `json:"min_booking_lead_minutes"`
		MaxBookingDaysAhead   int32 `json:"max_booking_days_ahead"`
		CancelDeadlineHours   int32 `json:"cancel_deadline_hours"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	s, err := h.service.Update(r.Context(), Settings{
		BufferMinutes:         req.BufferMinutes,
		MinBookingLeadMinutes: req.MinBookingLeadMinutes,
		MaxBookingDaysAhead:   req.MaxBookingDaysAhead,
		CancelDeadlineHours:   req.CancelDeadlineHours,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, s)
}
