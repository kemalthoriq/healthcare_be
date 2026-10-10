package booking

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type adminBookingResponse struct {
	ID            int64      `json:"id"`
	Code          string     `json:"code"`
	Status        string     `json:"status"`
	PatientName   string     `json:"patient_name"`
	PatientPhone  string     `json:"patient_phone"`
	PatientEmail  string     `json:"patient_email,omitempty"`
	TreatmentID   int64      `json:"treatment_id"`
	TreatmentName string     `json:"treatment_name"`
	TherapistID   int64      `json:"therapist_id"`
	TherapistName string     `json:"therapist_name"`
	StartAt       time.Time  `json:"start_at"`
	EndAt         time.Time  `json:"end_at"`
	Price         int64      `json:"price"`
	Note          string     `json:"note,omitempty"`
	CancelReason  string     `json:"cancel_reason,omitempty"`
	ConfirmedAt   *time.Time `json:"confirmed_at,omitempty"`
	CancelledAt   *time.Time `json:"cancelled_at,omitempty"`
	CompletedAt   *time.Time `json:"completed_at,omitempty"`
	NeedsAction   bool       `json:"needs_action"`
	CreatedAt     time.Time  `json:"created_at"`
}

func (h *Handler) toAdminResponse(b AdminBooking) adminBookingResponse {
	return adminBookingResponse{
		ID:            b.ID,
		Code:          b.Code,
		Status:        b.Status,
		PatientName:   b.PatientName,
		PatientPhone:  b.PatientPhone,
		PatientEmail:  b.PatientEmail,
		TreatmentID:   b.TreatmentID,
		TreatmentName: b.TreatmentName,
		TherapistID:   b.TherapistID,
		TherapistName: b.TherapistName,
		StartAt:       b.StartAt.In(h.loc),
		EndAt:         b.EndAt.In(h.loc),
		Price:         b.Price,
		Note:          b.PatientNote,
		CancelReason:  b.CancelReason,
		ConfirmedAt:   b.ConfirmedAt,
		CancelledAt:   b.CancelledAt,
		CompletedAt:   b.CompletedAt,
		NeedsAction:   b.NeedsAction,
		CreatedAt:     b.CreatedAt,
	}
}

func (h *Handler) ListAdmin(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	var f AdminFilter
	fields := map[string]string{}

	if s := q.Get("status"); s != "" {
		if !IsValidStatus(s) {
			fields["status"] = "Status tidak dikenal."
		}
		f.Status = &s
	}
	if s := q.Get("therapist_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			fields["therapist_id"] = "ID terapis tidak valid."
		}
		f.TherapistID = &id
	}
	if s := q.Get("date_from"); s != "" {
		d, err := time.ParseInLocation("2006-01-02", s, h.loc)
		if err != nil {
			fields["date_from"] = "Format tanggal harus YYYY-MM-DD."
		}
		f.DateFrom = &d
	}
	if s := q.Get("date_to"); s != "" {
		d, err := time.ParseInLocation("2006-01-02", s, h.loc)
		if err != nil {
			fields["date_to"] = "Format tanggal harus YYYY-MM-DD."
		}
		end := d.AddDate(0, 0, 1)
		f.DateTo = &end
	}
	if len(fields) > 0 {
		httpx.Error(w, r, apperr.Validation(fields))
		return
	}

	page := httpx.PageFromRequest(r)
	items, total, err := h.service.ListAdmin(r.Context(), f, page.PerPage, page.Offset())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]adminBookingResponse, 0, len(items))
	for _, b := range items {
		out = append(out, h.toAdminResponse(b))
	}
	page.Total = total
	httpx.JSONPage(w, r, out, page)
}

func (h *Handler) GetAdmin(w http.ResponseWriter, r *http.Request) {
	id, ok := bookingID(r)
	if !ok {
		httpx.Error(w, r, ErrBookingNotFound)
		return
	}

	b, err := h.service.GetAdmin(r.Context(), id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, h.toAdminResponse(b))
}

func (h *Handler) Confirm(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, StatusConfirmed, "")
}

func (h *Handler) Complete(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, StatusCompleted, "")
}

func (h *Handler) NoShow(w http.ResponseWriter, r *http.Request) {
	h.transition(w, r, StatusNoShow, "")
}

func (h *Handler) CancelAdmin(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Reason string `json:"reason"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	h.transition(w, r, StatusCancelled, req.Reason)
}

func (h *Handler) transition(w http.ResponseWriter, r *http.Request, to, reason string) {
	id, ok := bookingID(r)
	if !ok {
		httpx.Error(w, r, ErrBookingNotFound)
		return
	}

	b, err := h.service.Transition(r.Context(), id, to, reason)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, h.toAdminResponse(b))
}

func bookingID(r *http.Request) (int64, bool) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	return id, err == nil && id > 0
}
