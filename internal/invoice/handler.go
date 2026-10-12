package invoice

import (
	"net/http"
	"strconv"
	"time"

	"github.com/kemalthoriq/healthcare_be/internal/auth"
	"github.com/kemalthoriq/healthcare_be/internal/booking"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
)

type invoiceResponse struct {
	ID             int64      `json:"id"`
	Number         string     `json:"number"`
	BookingID      int64      `json:"booking_id"`
	BookingCode    string     `json:"booking_code"`
	BookingStartAt time.Time  `json:"booking_start_at"`
	PatientName    string     `json:"patient_name"`
	PatientPhone   string     `json:"patient_phone"`
	TreatmentName  string     `json:"treatment_name"`
	TherapistName  string     `json:"therapist_name"`
	Subtotal       int64      `json:"subtotal"`
	Discount       int64      `json:"discount"`
	Total          int64      `json:"total"`
	Status         string     `json:"status"`
	PaymentMethod  string     `json:"payment_method"`
	PaidAt         *time.Time `json:"paid_at"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type Handler struct {
	service *Service
	loc     *time.Location
}

func NewHandler(service *Service, loc *time.Location) *Handler {
	return &Handler{service: service, loc: loc}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	bookingID, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || bookingID <= 0 {
		httpx.Error(w, r, booking.ErrBookingNotFound)
		return
	}

	var req struct {
		Discount int64 `json:"discount"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	user, ok := auth.UserFromContext(r.Context())
	if !ok {
		httpx.Error(w, r, apperr.Unauthenticated())
		return
	}

	inv, err := h.service.Create(r.Context(), user.ID, bookingID, req.Discount)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusCreated, h.toResponse(inv))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	f := Filter{Query: q.Get("q")}
	fields := map[string]string{}

	if s := q.Get("status"); s != "" {
		if !IsValidStatus(s) {
			fields["status"] = "Status harus unpaid atau paid."
		}
		f.Status = &s
	}
	if s := q.Get("booking_id"); s != "" {
		id, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			fields["booking_id"] = "ID booking tidak valid."
		}
		f.BookingID = &id
	}
	if len(fields) > 0 {
		httpx.Error(w, r, apperr.Validation(fields))
		return
	}

	page := httpx.PageFromRequest(r)
	items, total, err := h.service.List(r.Context(), f, page.PerPage, page.Offset())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}

	out := make([]invoiceResponse, 0, len(items))
	for _, inv := range items {
		out = append(out, h.toResponse(inv))
	}
	page.Total = total
	httpx.JSONPage(w, r, out, page)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	h.withInvoice(w, r, func(id int64) (Invoice, error) {
		return h.service.Get(r.Context(), id)
	})
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Discount *int64 `json:"discount"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}
	if req.Discount == nil {
		httpx.Error(w, r, apperr.Validation(map[string]string{"discount": "Diskon wajib diisi."}))
		return
	}

	h.withInvoice(w, r, func(id int64) (Invoice, error) {
		return h.service.UpdateDiscount(r.Context(), id, *req.Discount)
	})
}

func (h *Handler) Pay(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PaymentMethod string `json:"payment_method"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	h.withInvoice(w, r, func(id int64) (Invoice, error) {
		return h.service.Pay(r.Context(), id, req.PaymentMethod)
	})
}

func (h *Handler) withInvoice(w http.ResponseWriter, r *http.Request, action func(id int64) (Invoice, error)) {
	id, err := strconv.ParseInt(r.PathValue("id"), 10, 64)
	if err != nil || id <= 0 {
		httpx.Error(w, r, ErrNotFound)
		return
	}

	inv, err := action(id)
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, h.toResponse(inv))
}

func (h *Handler) toResponse(inv Invoice) invoiceResponse {
	var paidAt *time.Time
	if inv.PaidAt != nil {
		t := inv.PaidAt.In(h.loc)
		paidAt = &t
	}

	return invoiceResponse{
		ID:             inv.ID,
		Number:         inv.Number,
		BookingID:      inv.BookingID,
		BookingCode:    inv.BookingCode,
		BookingStartAt: inv.BookingStartAt.In(h.loc),
		PatientName:    inv.PatientName,
		PatientPhone:   inv.PatientPhone,
		TreatmentName:  inv.TreatmentName,
		TherapistName:  inv.TherapistName,
		Subtotal:       inv.Subtotal,
		Discount:       inv.Discount,
		Total:          inv.Total,
		Status:         inv.Status,
		PaymentMethod:  inv.PaymentMethod,
		PaidAt:         paidAt,
		CreatedAt:      inv.CreatedAt,
		UpdatedAt:      inv.UpdatedAt,
	}
}
