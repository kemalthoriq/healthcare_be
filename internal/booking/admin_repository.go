package booking

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

var errStatusChanged = errors.New("booking status changed concurrently")

func (r *Repository) ListAdmin(ctx context.Context, f AdminFilter, limit, offset int) ([]AdminBooking, int64, error) {
	rows, err := r.q.ListAdminBookings(ctx, sqlc.ListAdminBookingsParams{
		Status:      f.Status,
		TherapistID: f.TherapistID,
		DateFrom:    f.DateFrom,
		DateTo:      f.DateTo,
		PageLimit:   int32(limit),
		PageOffset:  int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountAdminBookings(ctx, sqlc.CountAdminBookingsParams{
		Status:      f.Status,
		TherapistID: f.TherapistID,
		DateFrom:    f.DateFrom,
		DateTo:      f.DateTo,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]AdminBooking, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminBooking{
			Booking: Booking{
				ID:          row.ID,
				Code:        row.Code,
				TreatmentID: row.TreatmentID,
				TherapistID: row.TherapistID,
				PatientName: row.PatientName,
				StartAt:     row.StartAt,
				EndAt:       row.EndAt,
				Status:      row.Status,
				Price:       row.Price,
				CreatedAt:   row.CreatedAt,
			},
			PatientPhone: row.PatientPhone,
		})
	}
	return out, total, nil
}

func (r *Repository) GetAdmin(ctx context.Context, id int64) (AdminBooking, error) {
	row, err := r.q.GetAdminBooking(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return AdminBooking{}, errNotFound
	}
	if err != nil {
		return AdminBooking{}, err
	}

	return AdminBooking{
		Booking: Booking{
			ID:           row.ID,
			Code:         row.Code,
			TreatmentID:  row.TreatmentID,
			TherapistID:  row.TherapistID,
			PatientName:  row.PatientName,
			StartAt:      row.StartAt,
			EndAt:        row.EndAt,
			Status:       row.Status,
			Price:        row.Price,
			PatientNote:  row.PatientNote,
			CancelReason: row.CancelReason,
			ConfirmedAt:  row.ConfirmedAt,
			CancelledAt:  row.CancelledAt,
			CompletedAt:  row.CompletedAt,
			CreatedAt:    row.CreatedAt,
		},
		PatientPhone: row.PatientPhone,
		PatientEmail: row.PatientEmail,
	}, nil
}

func (r *Repository) Transition(ctx context.Context, id int64, from, to, reason string) error {
	_, err := r.q.TransitionBooking(ctx, sqlc.TransitionBookingParams{
		ID:           id,
		FromStatus:   from,
		ToStatus:     to,
		CancelReason: reason,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return errStatusChanged
	}
	return err
}
