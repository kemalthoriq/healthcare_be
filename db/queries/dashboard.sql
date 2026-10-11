-- name: DashboardBookingCounts :one
SELECT
    count(*) FILTER (WHERE status IN ('pending', 'confirmed', 'completed') AND start_at >= sqlc.arg(day_start) AND start_at < sqlc.arg(day_end)) AS today_count,
    count(*) FILTER (WHERE status = 'pending') AS pending_count,
    count(*) FILTER (WHERE status = 'pending' AND start_at < sqlc.arg(now)) AS needs_action_count
FROM bookings;

-- name: DashboardPendingBookings :many
SELECT b.id, b.code, b.patient_name, t.name AS treatment_name, th.name AS therapist_name, b.start_at
FROM bookings b
JOIN treatments t ON t.id = b.treatment_id
JOIN therapists th ON th.id = b.therapist_id
WHERE b.status = 'pending'
ORDER BY b.start_at, b.id
LIMIT 10;

-- name: DashboardRevenue :one
SELECT coalesce(sum(total), 0)::bigint AS revenue
FROM invoices
WHERE status = 'paid' AND paid_at >= sqlc.arg(month_start)::timestamptz AND paid_at < sqlc.arg(month_end)::timestamptz;

-- name: DashboardTopTreatments :many
SELECT t.name, count(*) AS count
FROM bookings b
JOIN treatments t ON t.id = b.treatment_id
WHERE b.status = 'completed' AND b.start_at >= sqlc.arg(month_start) AND b.start_at < sqlc.arg(month_end)
GROUP BY t.id, t.name
ORDER BY count DESC, t.name
LIMIT 5;
