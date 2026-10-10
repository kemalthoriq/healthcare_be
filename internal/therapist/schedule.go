package therapist

import (
	"fmt"
	"time"
)

type Shift struct {
	DayOfWeek int
	Start     int
	End       int
}

type TimeOff struct {
	ID          int64
	TherapistID int64
	StartAt     time.Time
	EndAt       time.Time
	Reason      string
	CreatedAt   time.Time
}

type TimeOffInput struct {
	StartAt time.Time
	EndAt   time.Time
	Reason  string
}

func ParseClock(s string) (int, bool) {
	var h, m int
	if len(s) != 5 || s[2] != ':' {
		return 0, false
	}
	if _, err := fmt.Sscanf(s, "%02d:%02d", &h, &m); err != nil {
		return 0, false
	}
	if h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, false
	}
	return h*60 + m, true
}

func FormatClock(minutes int) string {
	return fmt.Sprintf("%02d:%02d", minutes/60, minutes%60)
}
