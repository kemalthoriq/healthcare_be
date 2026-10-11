package setting

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

func validClinic() Clinic {
	return Clinic{
		Name:             " Lumière Clinic ",
		Address:          "Jl. Melati No. 1, Bandung",
		Phone:            "(022) 555-1234",
		WhatsApp:         "0812-3456-7890",
		Email:            "halo@lumiere.test",
		OpeningHoursText: "Senin–Sabtu 09.00–18.00",
	}
}

func TestUpdateClinic_Normalizes(t *testing.T) {
	store := &fakeStore{}

	c, err := NewService(store).UpdateClinic(context.Background(), validClinic())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Lumière Clinic" || c.WhatsApp != "6281234567890" {
		t.Errorf("saved = %+v", c)
	}

	empty := Clinic{Name: "Lumière Clinic"}
	if _, err := NewService(store).UpdateClinic(context.Background(), empty); err != nil {
		t.Errorf("optional fields empty: unexpected error %v", err)
	}
}

func TestUpdateClinic_Validation(t *testing.T) {
	tests := []struct {
		name   string
		modify func(*Clinic)
		field  string
	}{
		{"empty name", func(c *Clinic) { c.Name = "  " }, "clinic_name"},
		{"long address", func(c *Clinic) { c.Address = strings.Repeat("a", 301) }, "address"},
		{"long phone", func(c *Clinic) { c.Phone = strings.Repeat("1", 31) }, "phone"},
		{"bad whatsapp", func(c *Clinic) { c.WhatsApp = "0212345678" }, "whatsapp"},
		{"bad email", func(c *Clinic) { c.Email = "halo@" }, "email"},
		{"long opening hours", func(c *Clinic) { c.OpeningHoursText = strings.Repeat("a", 501) }, "opening_hours_text"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := validClinic()
			tt.modify(&c)

			_, err := NewService(&fakeStore{}).UpdateClinic(context.Background(), c)

			var appErr *apperr.Error
			if !errors.As(err, &appErr) || appErr.Fields[tt.field] == "" {
				t.Errorf("expected validation error on %q, got %v", tt.field, err)
			}
		})
	}
}

func TestPublicClinic(t *testing.T) {
	store := &fakeStore{saved: validSettings(), clinic: Clinic{Name: "Lumière Clinic", WhatsApp: "6281234567890"}}

	c, err := NewService(store).PublicClinic(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.Name != "Lumière Clinic" || c.WhatsApp != "6281234567890" || c.MaxBookingDaysAhead != 30 || c.CancelDeadlineHours != 24 {
		t.Errorf("public clinic = %+v", c)
	}
}
