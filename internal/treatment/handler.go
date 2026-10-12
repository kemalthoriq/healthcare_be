package treatment

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type treatmentRequest struct {
	Name            string `json:"name"`
	Description     string `json:"description"`
	ImageURL        string `json:"image_url"`
	Price           int64  `json:"price"`
	DurationMinutes int32  `json:"duration_minutes"`
	IsActive        *bool  `json:"is_active"`
}

type treatmentResponse struct {
	ID              int64     `json:"id"`
	Name            string    `json:"name"`
	Slug            string    `json:"slug"`
	Description     string    `json:"description"`
	ImageURL        string    `json:"image_url"`
	Price           int64     `json:"price"`
	DurationMinutes int32     `json:"duration_minutes"`
	IsActive        bool      `json:"is_active"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

func toResponse(t Treatment) treatmentResponse {
	return treatmentResponse{
		ID:              t.ID,
		Name:            t.Name,
		Slug:            t.Slug,
		Description:     t.Description,
		ImageURL:        t.ImageURL,
		Price:           t.Price,
		DurationMinutes: t.DurationMinutes,
		IsActive:        t.IsActive,
		CreatedAt:       t.CreatedAt,
		UpdatedAt:       t.UpdatedAt,
	}
}

func toResponses(ts []Treatment) []treatmentResponse {
	out := make([]treatmentResponse, 0, len(ts))
	for _, t := range ts {
		out = append(out, toResponse(t))
	}
	return out
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	ts, err := h.service.ListActive(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponses(ts))
}

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	t, err := h.service.GetActiveBySlug(r.Context(), r.PathValue("slug"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponse(t))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ts, err := h.service.List(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponses(ts))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	t, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponse(t))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req treatmentRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	t, err := h.service.Create(r.Context(), toInput(req, isActive))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, toResponse(t))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	var req treatmentRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.IsActive == nil {
		httpx.Error(w, r, apperr.Validation(map[string]string{"is_active": "Status aktif wajib diisi."}))
		return
	}

	t, err := h.service.Update(r.Context(), id, toInput(req, *req.IsActive))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponse(t))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	if err := h.service.Delete(r.Context(), id); err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.NoContent(w)
}

func toInput(req treatmentRequest, isActive bool) Input {
	return Input{
		Name:            req.Name,
		Description:     req.Description,
		ImageURL:        req.ImageURL,
		Price:           req.Price,
		DurationMinutes: req.DurationMinutes,
		IsActive:        isActive,
	}
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}
