package setting

import (
	"context"
	"fmt"
	"net/http"
	"net/mail"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
	"github.com/kemalthoriq/healthcare_be/internal/platform/httpx"
	"github.com/kemalthoriq/healthcare_be/internal/platform/phone"
)

const (
	maxClinicNameLength   = 100
	maxAddressLength      = 300
	maxPhoneLength        = 30
	maxEmailLength        = 254
	maxOpeningHoursLength = 500
)

type Clinic struct {
	Name             string    `json:"clinic_name"`
	Address          string    `json:"address"`
	Phone            string    `json:"phone"`
	WhatsApp         string    `json:"whatsapp"`
	Email            string    `json:"email"`
	OpeningHoursText string    `json:"opening_hours_text"`
	UpdatedAt        time.Time `json:"updated_at"`
}

type PublicClinic struct {
	Name                  string `json:"clinic_name"`
	Address               string `json:"address"`
	Phone                 string `json:"phone"`
	WhatsApp              string `json:"whatsapp"`
	Email                 string `json:"email"`
	OpeningHoursText      string `json:"opening_hours_text"`
	MinBookingLeadMinutes int32  `json:"min_booking_lead_minutes"`
	MaxBookingDaysAhead   int32  `json:"max_booking_days_ahead"`
	CancelDeadlineHours   int32  `json:"cancel_deadline_hours"`
}

func (r *Repository) GetClinic(ctx context.Context) (Clinic, error) {
	row, err := r.q.GetClinicSettings(ctx)
	if err != nil {
		return Clinic{}, err
	}
	return toClinic(row), nil
}

func (r *Repository) UpdateClinic(ctx context.Context, c Clinic) (Clinic, error) {
	row, err := r.q.UpdateClinicProfile(ctx, sqlc.UpdateClinicProfileParams{
		ClinicName:       c.Name,
		Address:          c.Address,
		Phone:            c.Phone,
		Whatsapp:         c.WhatsApp,
		Email:            c.Email,
		OpeningHoursText: c.OpeningHoursText,
	})
	if err != nil {
		return Clinic{}, err
	}
	return toClinic(row), nil
}

func toClinic(row sqlc.ClinicSetting) Clinic {
	return Clinic{
		Name:             row.ClinicName,
		Address:          row.Address,
		Phone:            row.Phone,
		WhatsApp:         row.Whatsapp,
		Email:            row.Email,
		OpeningHoursText: row.OpeningHoursText,
		UpdatedAt:        row.UpdatedAt,
	}
}

func (s *Service) GetClinic(ctx context.Context) (Clinic, error) {
	return s.store.GetClinic(ctx)
}

func (s *Service) PublicClinic(ctx context.Context) (PublicClinic, error) {
	c, err := s.store.GetClinic(ctx)
	if err != nil {
		return PublicClinic{}, err
	}
	st, err := s.store.Get(ctx)
	if err != nil {
		return PublicClinic{}, err
	}
	return PublicClinic{
		Name:                  c.Name,
		Address:               c.Address,
		Phone:                 c.Phone,
		WhatsApp:              c.WhatsApp,
		Email:                 c.Email,
		OpeningHoursText:      c.OpeningHoursText,
		MinBookingLeadMinutes: st.MinBookingLeadMinutes,
		MaxBookingDaysAhead:   st.MaxBookingDaysAhead,
		CancelDeadlineHours:   st.CancelDeadlineHours,
	}, nil
}

func (s *Service) UpdateClinic(ctx context.Context, in Clinic) (Clinic, error) {
	in.Name = strings.TrimSpace(in.Name)
	in.Address = strings.TrimSpace(in.Address)
	in.Phone = strings.TrimSpace(in.Phone)
	in.WhatsApp = strings.TrimSpace(in.WhatsApp)
	in.Email = strings.TrimSpace(in.Email)
	in.OpeningHoursText = strings.TrimSpace(in.OpeningHoursText)

	fields := map[string]string{}
	switch {
	case in.Name == "":
		fields["clinic_name"] = "Nama klinik wajib diisi."
	case utf8.RuneCountInString(in.Name) > maxClinicNameLength:
		fields["clinic_name"] = fmt.Sprintf("Nama klinik maksimal %d karakter.", maxClinicNameLength)
	}
	if utf8.RuneCountInString(in.Address) > maxAddressLength {
		fields["address"] = fmt.Sprintf("Alamat maksimal %d karakter.", maxAddressLength)
	}
	if utf8.RuneCountInString(in.Phone) > maxPhoneLength {
		fields["phone"] = fmt.Sprintf("Nomor telepon maksimal %d karakter.", maxPhoneLength)
	}
	if in.WhatsApp != "" {
		normalized, ok := phone.Normalize(in.WhatsApp)
		if !ok {
			fields["whatsapp"] = "Nomor WhatsApp belum sesuai. Contoh: 081234567890."
		}
		in.WhatsApp = normalized
	}
	if in.Email != "" {
		addr, err := mail.ParseAddress(in.Email)
		if err != nil || addr.Address != in.Email || len(in.Email) > maxEmailLength {
			fields["email"] = "Format email belum sesuai."
		}
	}
	if utf8.RuneCountInString(in.OpeningHoursText) > maxOpeningHoursLength {
		fields["opening_hours_text"] = fmt.Sprintf("Jam operasional maksimal %d karakter.", maxOpeningHoursLength)
	}
	if len(fields) > 0 {
		return Clinic{}, apperr.Validation(fields)
	}

	return s.store.UpdateClinic(ctx, in)
}

func (h *Handler) GetClinicPublic(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.PublicClinic(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, c)
}

func (h *Handler) GetClinic(w http.ResponseWriter, r *http.Request) {
	c, err := h.service.GetClinic(r.Context())
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, c)
}

func (h *Handler) UpdateClinic(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Name             string `json:"clinic_name"`
		Address          string `json:"address"`
		Phone            string `json:"phone"`
		WhatsApp         string `json:"whatsapp"`
		Email            string `json:"email"`
		OpeningHoursText string `json:"opening_hours_text"`
	}
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Error(w, r, err)
		return
	}

	c, err := h.service.UpdateClinic(r.Context(), Clinic{
		Name:             req.Name,
		Address:          req.Address,
		Phone:            req.Phone,
		WhatsApp:         req.WhatsApp,
		Email:            req.Email,
		OpeningHoursText: req.OpeningHoursText,
	})
	if err != nil {
		httpx.Error(w, r, err)
		return
	}
	httpx.JSON(w, r, http.StatusOK, c)
}
