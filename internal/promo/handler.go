package promo

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type promoRequest struct {
	Title        string  `json:"title"`
	Description  string  `json:"description"`
	ImageURL     string  `json:"image_url"`
	StartDate    string  `json:"start_date"`
	EndDate      string  `json:"end_date"`
	IsActive     *bool   `json:"is_active"`
	TreatmentIDs []int64 `json:"treatment_ids"`
}

type treatmentResponse struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	Slug     string `json:"slug"`
	IsActive bool   `json:"is_active"`
}

type promoResponse struct {
	ID          int64               `json:"id"`
	Title       string              `json:"title"`
	Slug        string              `json:"slug"`
	Description string              `json:"description"`
	ImageURL    string              `json:"image_url"`
	StartDate   string              `json:"start_date"`
	EndDate     string              `json:"end_date"`
	IsActive    bool                `json:"is_active"`
	Treatments  []treatmentResponse `json:"treatments"`
	CreatedAt   time.Time           `json:"created_at"`
	UpdatedAt   time.Time           `json:"updated_at"`
}

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) ListPublic(w http.ResponseWriter, r *http.Request) {
	promos, err := h.service.ListPublic(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponses(promos))
}

func (h *Handler) GetPublic(w http.ResponseWriter, r *http.Request) {
	p, err := h.service.GetPublic(r.Context(), r.PathValue("slug"))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponse(p))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r)
	promos, total, err := h.service.List(r.Context(), page.PerPage, page.Offset())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	page.Total = total
	httpx.JSONPage(w, r, toResponses(promos), page)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	p, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponse(p))
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	var req promoRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	p, err := h.service.Create(r.Context(), toInput(req, isActive))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, toResponse(p))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	id, ok := pathID(r)
	if !ok {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	var req promoRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.IsActive == nil {
		httpx.Error(w, r, apperr.Validation(map[string]string{"is_active": "Status aktif wajib diisi."}))
		return
	}

	p, err := h.service.Update(r.Context(), id, toInput(req, *req.IsActive))
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, toResponse(p))
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

func toInput(req promoRequest, isActive bool) Input {
	return Input{
		Title:        req.Title,
		Description:  req.Description,
		ImageURL:     req.ImageURL,
		StartDate:    req.StartDate,
		EndDate:      req.EndDate,
		IsActive:     isActive,
		TreatmentIDs: req.TreatmentIDs,
	}
}

func toResponse(p Promo) promoResponse {
	treatments := make([]treatmentResponse, 0, len(p.Treatments))
	for _, t := range p.Treatments {
		treatments = append(treatments, treatmentResponse{ID: t.ID, Name: t.Name, Slug: t.Slug, IsActive: t.IsActive})
	}

	return promoResponse{
		ID:          p.ID,
		Title:       p.Title,
		Slug:        p.Slug,
		Description: p.Description,
		ImageURL:    p.ImageURL,
		StartDate:   p.StartDate.Format(DateLayout),
		EndDate:     p.EndDate.Format(DateLayout),
		IsActive:    p.IsActive,
		Treatments:  treatments,
		CreatedAt:   p.CreatedAt,
		UpdatedAt:   p.UpdatedAt,
	}
}

func toResponses(promos []Promo) []promoResponse {
	out := make([]promoResponse, 0, len(promos))
	for _, p := range promos {
		out = append(out, toResponse(p))
	}
	return out
}

func pathID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}
