package dashboard

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/kemalthoriq/healthcare_be/db/sqlc"
)

type Repository struct {
	q *sqlc.Queries
}

func NewRepository(pool *pgxpool.Pool) *Repository {
	return &Repository{q: sqlc.New(pool)}
}

func (r *Repository) BookingCounts(ctx context.Context, today Range, now time.Time) (Counts, error) {
	row, err := r.q.DashboardBookingCounts(ctx, sqlc.DashboardBookingCountsParams{DayStart: today.Start, DayEnd: today.End, Now: now})
	if err != nil {
		return Counts{}, err
	}
	return Counts{Today: row.TodayCount, Pending: row.PendingCount, NeedsAction: row.NeedsActionCount}, nil
}

func (r *Repository) PendingBookings(ctx context.Context) ([]PendingBooking, error) {
	rows, err := r.q.DashboardPendingBookings(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]PendingBooking, 0, len(rows))
	for _, row := range rows {
		out = append(out, PendingBooking{
			ID:            row.ID,
			Code:          row.Code,
			PatientName:   row.PatientName,
			TreatmentName: row.TreatmentName,
			TherapistName: row.TherapistName,
			StartAt:       row.StartAt,
		})
	}
	return out, nil
}

func (r *Repository) Revenue(ctx context.Context, month Range) (int64, error) {
	return r.q.DashboardRevenue(ctx, sqlc.DashboardRevenueParams{MonthStart: month.Start, MonthEnd: month.End})
}

func (r *Repository) TopTreatments(ctx context.Context, month Range) ([]TopTreatment, error) {
	rows, err := r.q.DashboardTopTreatments(ctx, sqlc.DashboardTopTreatmentsParams{MonthStart: month.Start, MonthEnd: month.End})
	if err != nil {
		return nil, err
	}
	out := make([]TopTreatment, 0, len(rows))
	for _, row := range rows {
		out = append(out, TopTreatment{Name: row.Name, Count: row.Count})
	}
	return out, nil
}
