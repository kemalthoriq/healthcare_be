package dashboard

import (
	"net/http"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type pendingResponse struct {
	ID            int64     `json:"id"`
	Code          string    `json:"code"`
	PatientName   string    `json:"patient_name"`
	TreatmentName string    `json:"treatment_name"`
	TherapistName string    `json:"therapist_name"`
	StartAt       time.Time `json:"start_at"`
	NeedsAction   bool      `json:"needs_action"`
}

type topTreatmentResponse struct {
	Name  string `json:"name"`
	Count int64  `json:"count"`
}

type summaryResponse struct {
	TodayCount       int64                  `json:"today_count"`
	PendingCount     int64                  `json:"pending_count"`
	NeedsActionCount int64                  `json:"needs_action_count"`
	PendingUpcoming  []pendingResponse      `json:"pending_upcoming"`
	RevenueMonth     int64                  `json:"revenue_month"`
	TopTreatments    []topTreatmentResponse `json:"top_treatments"`
}

type Handler struct {
	service *Service
	loc     *time.Location
}

func NewHandler(service *Service, loc *time.Location) *Handler {
	return &Handler{service: service, loc: loc}
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	s, err := h.service.Summary(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	pending := make([]pendingResponse, 0, len(s.PendingUpcoming))
	for _, b := range s.PendingUpcoming {
		pending = append(pending, pendingResponse{
			ID:            b.ID,
			Code:          b.Code,
			PatientName:   b.PatientName,
			TreatmentName: b.TreatmentName,
			TherapistName: b.TherapistName,
			StartAt:       b.StartAt.In(h.loc),
			NeedsAction:   b.NeedsAction,
		})
	}
	top := make([]topTreatmentResponse, 0, len(s.TopTreatments))
	for _, t := range s.TopTreatments {
		top = append(top, topTreatmentResponse{Name: t.Name, Count: t.Count})
	}

	httpx.JSON(w, r, http.StatusOK, summaryResponse{
		TodayCount:       s.TodayCount,
		PendingCount:     s.PendingCount,
		NeedsActionCount: s.NeedsActionCount,
		PendingUpcoming:  pending,
		RevenueMonth:     s.RevenueMonth,
		TopTreatments:    top,
	})
}
