package booking

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

func (r *Repository) ListPatients(ctx context.Context, namePattern, phonePattern string, limit, offset int) ([]Patient, int64, error) {
	rows, err := r.q.ListPatients(ctx, sqlc.ListPatientsParams{
		NamePattern:  namePattern,
		PhonePattern: phonePattern,
		PageLimit:    int32(limit),
		PageOffset:   int32(offset),
	})
	if err != nil {
		return nil, 0, err
	}

	total, err := r.q.CountPatients(ctx, sqlc.CountPatientsParams{
		NamePattern:  namePattern,
		PhonePattern: phonePattern,
	})
	if err != nil {
		return nil, 0, err
	}

	out := make([]Patient, 0, len(rows))
	for _, row := range rows {
		out = append(out, Patient{
			ID:           row.ID,
			Name:         row.Name,
			Phone:        row.Phone,
			Email:        row.Email,
			BookingCount: row.BookingCount,
			LastVisitAt:  optionalTime(row.LastVisitAt),
			CreatedAt:    row.CreatedAt,
		})
	}
	return out, total, nil
}

func (r *Repository) GetPatient(ctx context.Context, id int64) (Patient, error) {
	row, err := r.q.GetPatient(ctx, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Patient{}, errNotFound
	}
	if err != nil {
		return Patient{}, err
	}

	return Patient{
		ID:           row.ID,
		Name:         row.Name,
		Phone:        row.Phone,
		Email:        row.Email,
		BookingCount: row.BookingCount,
		LastVisitAt:  optionalTime(row.LastVisitAt),
		CreatedAt:    row.CreatedAt,
	}, nil
}

func (r *Repository) ListPatientBookings(ctx context.Context, patientID int64) ([]AdminBooking, error) {
	rows, err := r.q.ListPatientBookings(ctx, patientID)
	if err != nil {
		return nil, err
	}

	out := make([]AdminBooking, 0, len(rows))
	for _, row := range rows {
		out = append(out, AdminBooking{
			Booking: Booking{
				ID:          row.ID,
				Code:        row.Code,
				PatientID:   patientID,
				TreatmentID: row.TreatmentID,
				TherapistID: row.TherapistID,
				PatientName: row.PatientName,
				StartAt:     row.StartAt,
				EndAt:       row.EndAt,
				Status:      row.Status,
				Price:       row.Price,
				CreatedAt:   row.CreatedAt,
			},
			TreatmentNote: row.TreatmentNote,
		})
	}
	return out, nil
}

func optionalTime(v any) *time.Time {
	t, ok := v.(time.Time)
	if !ok {
		return nil
	}
	return &t
}
