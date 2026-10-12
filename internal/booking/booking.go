package booking

import (
	"crypto/rand"
	"strings"
	"time"
)

const (
	StatusPending   = "pending"
	StatusConfirmed = "confirmed"
	StatusCompleted = "completed"
	StatusNoShow    = "no_show"
	StatusCancelled = "cancelled"
)

const codeAlphabet = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"

type Booking struct {
	ID           int64
	Code         string
	PatientID    int64
	TreatmentID  int64
	TherapistID  int64
	PatientName  string
	StartAt      time.Time
	EndAt        time.Time
	BlockedUntil time.Time
	Status       string
	Price        int64
	PatientNote  string
	CancelReason string
	ConfirmedAt  *time.Time
	CancelledAt  *time.Time
	CompletedAt  *time.Time
	CreatedAt    time.Time
}

type NewBooking struct {
	PatientName  string
	Phone        string
	Email        string
	TreatmentID  int64
	TherapistID  int64
	StartAt      time.Time
	EndAt        time.Time
	BlockedUntil time.Time
	Price        int64
	PatientNote  string
}

func NewCode() string {
	b := make([]byte, 8)
	rand.Read(b)

	var sb strings.Builder
	sb.WriteString("LMR-")
	for i, v := range b {
		if i == 4 {
			sb.WriteByte('-')
		}
		sb.WriteByte(codeAlphabet[v%32])
	}
	return sb.String()
}
