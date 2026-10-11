package booking

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type patientResponse struct {
	ID           int64      `json:"id"`
	Name         string     `json:"name"`
	Phone        string     `json:"phone"`
	Email        string     `json:"email"`
	BookingCount int64      `json:"booking_count"`
	LastVisitAt  *time.Time `json:"last_visit_at"`
	CreatedAt    time.Time  `json:"created_at"`
}

type patientBookingResponse struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	Status        string    `json:"status"`
	TreatmentName string    `json:"treatment_name"`
	TherapistName string    `json:"therapist_name"`
	StartAt       time.Time `json:"start_at"`
	EndAt         time.Time `json:"end_at"`
	Price         int64     `json:"price"`
	TreatmentNote string    `json:"treatment_note"`
}

type patientDetailResponse struct {
	patientResponse
	Bookings []patientBookingResponse `json:"bookings"`
}

type PatientHandler struct {
	service *PatientService
	loc     *time.Location
}

func NewPatientHandler(service *PatientService, loc *time.Location) *PatientHandler {
	return &PatientHandler{service: service, loc: loc}
}

func (h *PatientHandler) List(w http.ResponseWriter, r *http.Request) {
	page := httpx.PageFromRequest(r)
	items, total, err := h.service.List(r.Context(), r.URL.Query().Get("q"), page.PerPage, page.Offset())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]patientResponse, 0, len(items))
	for _, p := range items {
		out = append(out, h.toResponse(p))
	}
	page.Total = total
	httpx.JSONPage(w, r, out, page)
}

func (h *PatientHandler) Get(w http.ResponseWriter, r *http.Request) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, ErrPatientNotFound)
		return
	}

	d, err := h.service.Get(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	bookings := make([]patientBookingResponse, 0, len(d.Bookings))
	for _, b := range d.Bookings {
		bookings = append(bookings, patientBookingResponse{
			ID:            b.ID,
			Code:          b.Code,
			Status:        b.Status,
			TreatmentName: b.TreatmentName,
			TherapistName: b.TherapistName,
			StartAt:       b.StartAt.In(h.loc),
			EndAt:         b.EndAt.In(h.loc),
			Price:         b.Price,
			TreatmentNote: b.TreatmentNote,
		})
	}

	httpx.JSON(w, r, http.StatusOK, patientDetailResponse{patientResponse: h.toResponse(d.Patient), Bookings: bookings})
}

func (h *PatientHandler) toResponse(p Patient) patientResponse {
	var lastVisit *time.Time
	if p.LastVisitAt != nil {
		t := p.LastVisitAt.In(h.loc)
		lastVisit = &t
	}

	return patientResponse{
		ID:           p.ID,
		Name:         p.Name,
		Phone:        p.Phone,
		Email:        p.Email,
		BookingCount: p.BookingCount,
		LastVisitAt:  lastVisit,
		CreatedAt:    p.CreatedAt,
	}
}
