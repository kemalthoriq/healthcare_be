package therapist

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
	"github.com/kemalthoriq/healthcare_be/internal/treatment"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

type therapistRequest struct {
	Name     string `json:"name"`
	Title    string `json:"title"`
	Bio      string `json:"bio"`
	PhotoURL string `json:"photo_url"`
	IsActive *bool  `json:"is_active"`
}

type treatmentTherapistsRequest struct {
	TherapistIDs []int64 `json:"therapist_ids"`
}

type therapistResponse struct {
	ID        int64     `json:"id"`
	Name      string    `json:"name"`
	Slug      string    `json:"slug"`
	Title     string    `json:"title"`
	Bio       string    `json:"bio"`
	PhotoURL  string    `json:"photo_url"`
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func toResponse(t Therapist) therapistResponse {
	return therapistResponse{
		ID:        t.ID,
		Name:      t.Name,
		Slug:      t.Slug,
		Title:     t.Title,
		Bio:       t.Bio,
		PhotoURL:  t.PhotoURL,
		IsActive:  t.IsActive,
		CreatedAt: t.CreatedAt,
		UpdatedAt: t.UpdatedAt,
	}
}

func toResponses(ts []Therapist) []therapistResponse {
	out := make([]therapistResponse, 0, len(ts))
	for _, t := range ts {
		out = append(out, toResponse(t))
	}
	return out
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	ts, err := h.service.ListActive(r.Context())
	h.writeList(w, r, ts, err)
}

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	t, err := h.service.GetActiveBySlug(r.Context(), r.PathValue("slug"))
	h.writeOne(w, r, http.StatusOK, t, err)
}

func (h *Handler) ListForTreatmentPublic(w http.ResponseWriter, r *http.Request) {
	ts, err := h.service.ListActiveForTreatment(r.Context(), r.PathValue("slug"))
	h.writeList(w, r, ts, err)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	ts, err := h.service.List(r.Context())
	h.writeList(w, r, ts, err)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	t, err := h.service.Get(r.Context(), id)
	h.writeOne(w, r, http.StatusOK, t, err)
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req therapistRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	t, err := h.service.Create(r.Context(), toInput(req, isActive))
	h.writeOne(w, r, http.StatusCreated, t, err)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	var req therapistRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.IsActive == nil {
		httpx.Error(w, r, apperr.Validation(map[string]string{"is_active": "Status aktif wajib diisi."}))
		return
	}

	t, err := h.service.Update(r.Context(), id, toInput(req, *req.IsActive))
	h.writeOne(w, r, http.StatusOK, t, err)
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

func (h *Handler) ListForTreatment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, treatment.ErrNotFound)
		return
	}

	ts, err := h.service.ListForTreatment(r.Context(), id)
	h.writeList(w, r, ts, err)
}

func (h *Handler) SetForTreatment(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, treatment.ErrNotFound)
		return
	}

	var req treatmentTherapistsRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.TherapistIDs == nil {
		httpx.Error(w, r, apperr.Validation(map[string]string{"therapist_ids": "Daftar terapis wajib diisi (boleh kosong: [])."}))
		return
	}

	ts, err := h.service.SetForTreatment(r.Context(), id, req.TherapistIDs)
	h.writeList(w, r, ts, err)
}

func (h *Handler) writeOne(w http.ResponseWriter, r *http.Request, status int, t Therapist, err error) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, status, toResponse(t))
}

func (h *Handler) writeList(w http.ResponseWriter, r *http.Request, ts []Therapist, err error) {
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponses(ts))
}

func toInput(req therapistRequest, isActive bool) Input {
	return Input{
		Name:     req.Name,
		Title:    req.Title,
		Bio:      req.Bio,
		PhotoURL: req.PhotoURL,
		IsActive: isActive,
	}
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}
