package invoice

import (
	"fmt"
	"time"
)

const (
	StatusUnpaid = "unpaid"
	StatusPaid   = "paid"

	MethodCash     = "cash"
	MethodTransfer = "transfer"
)

type Invoice struct {
	ID             int64
	Number         string
	BookingID      int64
	Subtotal       int64
	Discount       int64
	Total          int64
	Status         string
	PaymentMethod  string
	PaidAt         *time.Time
	CreatedBy      int64
	CreatedAt      time.Time
	UpdatedAt      time.Time
	BookingCode    string
	BookingStartAt time.Time
	PatientName    string
	PatientPhone   string
	TreatmentName  string
	TherapistName  string
}

type NewInvoice struct {
	BookingID int64
	Subtotal  int64
	Discount  int64
	CreatedBy int64
}

type Filter struct {
	Status    *string
	BookingID *int64
	Query     string
}

func FormatNumber(period string, seq int32) string {
	return fmt.Sprintf("INV-%s-%04d", period, seq)
}

func IsValidStatus(s string) bool {
	return s == StatusUnpaid || s == StatusPaid
}

func isValidMethod(s string) bool {
	return s == MethodCash || s == MethodTransfer
}
