package booking

import (
	"net/http"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type Handler struct {
	availability *Availability
	loc          *time.Location
}

func NewHandler(availability *Availability, loc *time.Location) *Handler {
	return &Handler{availability: availability, loc: loc}
}

type slotResponse struct {
	StartAt time.Time `json:"start_at"`
	Time    string    `json:"time"`
}

func (h *Handler) Slots(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fields := requireParams(q.Get("treatment"), q.Get("therapist"))

	day, err := time.ParseInLocation("2006-01-02", q.Get("date"), h.loc)
	if err != nil {
		fields["date"] = "Tanggal wajib diisi dengan format YYYY-MM-DD."
	}
	if len(fields) > 0 {
		httpx.Error(w, r, apperr.Validation(fields))
		return
	}

	slots, err := h.availability.Slots(r.Context(), q.Get("treatment"), q.Get("therapist"), day)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]slotResponse, 0, len(slots))
	for _, s := range slots {
		out = append(out, slotResponse{StartAt: s, Time: s.Format("15:04")})
	}
	httpx.JSON(w, r, http.StatusOK, out)
}

func (h *Handler) Dates(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	fields := requireParams(q.Get("treatment"), q.Get("therapist"))

	month, err := time.ParseInLocation("2006-01", q.Get("month"), h.loc)
	if err != nil {
		fields["month"] = "Bulan wajib diisi dengan format YYYY-MM."
	}
	if len(fields) > 0 {
		httpx.Error(w, r, apperr.Validation(fields))
		return
	}

	dates, err := h.availability.Dates(r.Context(), q.Get("treatment"), q.Get("therapist"), month)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]string, 0, len(dates))
	for _, d := range dates {
		out = append(out, d.Format("2006-01-02"))
	}
	httpx.JSON(w, r, http.StatusOK, out)
}

func requireParams(treatmentSlug, therapistSlug string) map[string]string {
	fields := map[string]string{}
	if treatmentSlug == "" {
		fields["treatment"] = "Treatment wajib dipilih."
	}
	if therapistSlug == "" {
		fields["therapist"] = "Terapis wajib dipilih."
	}
	return fields
}
