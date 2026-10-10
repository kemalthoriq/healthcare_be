package therapist

import (
	"cmp"
	"context"
	"errors"
	"fmt"
	"slices"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/kemalthoriq/healthcare_be/internal/platform/apperr"
)

const (
	slotStepMinutes    = 30
	maxShifts          = 50
	maxTimeOffDuration = 30 * 24 * time.Hour
	maxReasonLength    = 200
)

var ErrTimeOffNotFound = apperr.NotFound("Cuti tidak ditemukan.")

func (s *Service) ListSchedules(ctx context.Context, therapistID int64) ([]Shift, error) {
	if _, err := s.Get(ctx, therapistID); err != nil {
		return nil, err
	}
	return s.store.ListSchedules(ctx, therapistID)
}

func (s *Service) SetSchedules(ctx context.Context, therapistID int64, shifts []Shift) ([]Shift, error) {
	if _, err := s.Get(ctx, therapistID); err != nil {
		return nil, err
	}

	if fields := validateShifts(shifts); len(fields) > 0 {
		return nil, apperr.Validation(fields)
	}

	err := s.store.ReplaceSchedules(ctx, therapistID, shifts)
	if errors.Is(err, errShiftOverlap) {
		return nil, apperr.Validation(map[string]string{"shifts": "Ada shift yang bertumpuk di hari yang sama."})
	}
	if err != nil {
		return nil, err
	}

	return s.store.ListSchedules(ctx, therapistID)
}

func (s *Service) ListUpcomingTimeOff(ctx context.Context, therapistID int64) ([]TimeOff, error) {
	if _, err := s.Get(ctx, therapistID); err != nil {
		return nil, err
	}
	return s.store.ListUpcomingTimeOff(ctx, therapistID)
}

func (s *Service) CreateTimeOff(ctx context.Context, therapistID int64, in TimeOffInput) (TimeOff, error) {
	if _, err := s.Get(ctx, therapistID); err != nil {
		return TimeOff{}, err
	}

	in.Reason = strings.TrimSpace(in.Reason)
	if fields := s.validateTimeOff(in); len(fields) > 0 {
		return TimeOff{}, apperr.Validation(fields)
	}

	return s.store.CreateTimeOff(ctx, therapistID, in)
}

func (s *Service) DeleteTimeOff(ctx context.Context, therapistID, timeOffID int64) error {
	err := s.store.DeleteTimeOff(ctx, therapistID, timeOffID)
	if errors.Is(err, errTimeOffNotFound) {
		return ErrTimeOffNotFound
	}
	return err
}

func validateShifts(shifts []Shift) map[string]string {
	fields := map[string]string{}

	if len(shifts) > maxShifts {
		fields["shifts"] = fmt.Sprintf("Maksimal %d shift.", maxShifts)
		return fields
	}

	for i, sh := range shifts {
		key := fmt.Sprintf("shifts[%d]", i)
		switch {
		case sh.DayOfWeek < 0 || sh.DayOfWeek > 6:
			fields[key] = "Hari harus 0 (Minggu) sampai 6 (Sabtu)."
		case sh.Start%slotStepMinutes != 0 || sh.End%slotStepMinutes != 0:
			fields[key] = "Jam mulai dan selesai harus tepat :00 atau :30."
		case sh.End <= sh.Start:
			fields[key] = "Jam selesai harus setelah jam mulai."
		}
	}
	if len(fields) > 0 {
		return fields
	}

	sorted := slices.Clone(shifts)
	slices.SortFunc(sorted, func(a, b Shift) int {
		return cmp.Or(cmp.Compare(a.DayOfWeek, b.DayOfWeek), cmp.Compare(a.Start, b.Start))
	})
	for i := 1; i < len(sorted); i++ {
		prev, cur := sorted[i-1], sorted[i]
		if cur.DayOfWeek == prev.DayOfWeek && cur.Start < prev.End {
			fields["shifts"] = fmt.Sprintf("Shift %s–%s bertumpuk dengan %s–%s di hari yang sama.",
				FormatClock(prev.Start), FormatClock(prev.End), FormatClock(cur.Start), FormatClock(cur.End))
			break
		}
	}

	return fields
}

func (s *Service) validateTimeOff(in TimeOffInput) map[string]string {
	fields := map[string]string{}

	switch {
	case in.StartAt.IsZero():
		fields["start_at"] = "Waktu mulai wajib diisi."
	case in.EndAt.IsZero():
		fields["end_at"] = "Waktu selesai wajib diisi."
	case !in.EndAt.After(in.StartAt):
		fields["end_at"] = "Waktu selesai harus setelah waktu mulai."
	case !in.EndAt.After(s.now()):
		fields["end_at"] = "Cuti tidak boleh sudah berakhir."
	case in.EndAt.Sub(in.StartAt) > maxTimeOffDuration:
		fields["end_at"] = "Cuti maksimal 30 hari sekali input."
	}

	if utf8.RuneCountInString(in.Reason) > maxReasonLength {
		fields["reason"] = fmt.Sprintf("Alasan maksimal %d karakter.", maxReasonLength)
	}

	return fields
}
