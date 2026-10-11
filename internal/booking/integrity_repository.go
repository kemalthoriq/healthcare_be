package booking

import (
	"context"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
	"github.com/kemalthoriq/healthcare_be/internal/integrity"
)

func (r *Repository) ActiveByTreatment(ctx context.Context, treatmentID int64) ([]integrity.Booking, error) {
	return r.listUpcomingActive(ctx, sqlc.ListUpcomingActiveBookingsParams{TreatmentID: &treatmentID})
}

func (r *Repository) ActiveByTherapist(ctx context.Context, therapistID int64) ([]integrity.Booking, error) {
	return r.listUpcomingActive(ctx, sqlc.ListUpcomingActiveBookingsParams{TherapistID: &therapistID})
}

func (r *Repository) listUpcomingActive(ctx context.Context, params sqlc.ListUpcomingActiveBookingsParams) ([]integrity.Booking, error) {
	rows, err := r.q.ListUpcomingActiveBookings(ctx, params)
	if err != nil {
		return nil, err
	}

	out := make([]integrity.Booking, 0, len(rows))
	for _, row := range rows {
		out = append(out, integrity.Booking{
			Code:        row.Code,
			TreatmentID: row.TreatmentID,
			TherapistID: row.TherapistID,
			StartAt:     row.StartAt,
			EndAt:       row.EndAt,
		})
	}
	return out, nil
}
