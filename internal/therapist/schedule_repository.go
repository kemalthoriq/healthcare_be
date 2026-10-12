package therapist

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgtype"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var (
	errShiftOverlap    = errors.New("therapist shifts overlap")
	errTimeOffNotFound = errors.New("time off not found")
)

func (r *Repository) ListSchedules(ctx context.Context, therapistID int64) ([]Shift, error) {
	rows, err := r.q.ListTherapistSchedules(ctx, therapistID)
	if err != nil {
		return nil, err
	}

	shifts := make([]Shift, 0, len(rows))
	for _, row := range rows {
		shifts = append(shifts, Shift{
			DayOfWeek: int(row.DayOfWeek),
			Start:     toMinutes(row.StartTime),
			End:       toMinutes(row.EndTime),
		})
	}
	return shifts, nil
}

func (r *Repository) ReplaceSchedules(ctx context.Context, therapistID int64, shifts []Shift) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := r.q.WithTx(tx)

	if err := q.DeleteTherapistSchedules(ctx, therapistID); err != nil {
		return err
	}

	params := sqlc.AddTherapistSchedulesParams{TherapistID: therapistID}
	for _, s := range shifts {
		params.Days = append(params.Days, int16(s.DayOfWeek))
		params.StartTimes = append(params.StartTimes, toPgTime(s.Start))
		params.EndTimes = append(params.EndTimes, toPgTime(s.End))
	}

	err = q.AddTherapistSchedules(ctx, params)
	if pgErrorCode(err) == "23P01" {
		return errShiftOverlap
	}
	if err != nil {
		return err
	}

	return tx.Commit(ctx)
}

func (r *Repository) ListUpcomingTimeOff(ctx context.Context, therapistID int64) ([]TimeOff, error) {
	rows, err := r.q.ListUpcomingTimeOff(ctx, therapistID)
	if err != nil {
		return nil, err
	}

	out := make([]TimeOff, 0, len(rows))
	for _, row := range rows {
		out = append(out, toTimeOff(row))
	}
	return out, nil
}

func (r *Repository) CreateTimeOff(ctx context.Context, therapistID int64, in TimeOffInput) (TimeOff, error) {
	row, err := r.q.CreateTimeOff(ctx, sqlc.CreateTimeOffParams{
		TherapistID: therapistID,
		StartAt:     in.StartAt,
		EndAt:       in.EndAt,
		Reason:      in.Reason,
	})
	if err != nil {
		return TimeOff{}, err
	}
	return toTimeOff(row), nil
}

func (r *Repository) DeleteTimeOff(ctx context.Context, therapistID, timeOffID int64) error {
	n, err := r.q.DeleteTimeOff(ctx, sqlc.DeleteTimeOffParams{ID: timeOffID, TherapistID: therapistID})
	if err != nil {
		return err
	}
	if n == 0 {
		return errTimeOffNotFound
	}
	return nil
}

func toMinutes(t pgtype.Time) int {
	return int(time.Duration(t.Microseconds) * time.Microsecond / time.Minute)
}

func toPgTime(minutes int) pgtype.Time {
	return pgtype.Time{Microseconds: int64(time.Duration(minutes) * time.Minute / time.Microsecond), Valid: true}
}

func toTimeOff(row sqlc.TherapistTimeOff) TimeOff {
	return TimeOff{
		ID:          row.ID,
		TherapistID: row.TherapistID,
		StartAt:     row.StartAt,
		EndAt:       row.EndAt,
		Reason:      row.Reason,
		CreatedAt:   row.CreatedAt,
	}
}
