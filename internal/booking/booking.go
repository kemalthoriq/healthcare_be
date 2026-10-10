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

func NormalizePhone(raw string) (string, bool) {
	s := strings.Map(func(r rune) rune {
		switch r {
		case ' ', '-', '.', '(', ')':
			return -1
		}
		return r
	}, strings.TrimSpace(raw))
	s = strings.TrimPrefix(s, "+")

	switch {
	case strings.HasPrefix(s, "62"):
	case strings.HasPrefix(s, "0"):
		s = "62" + s[1:]
	case strings.HasPrefix(s, "8"):
		s = "62" + s
	default:
		return "", false
	}

	if !strings.HasPrefix(s, "628") || len(s) < 10 || len(s) > 15 {
		return "", false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return "", false
		}
	}
	return s, true
}
