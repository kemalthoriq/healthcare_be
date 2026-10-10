-- name: UpsertPatient :one
INSERT INTO patients (name, phone, email, consent_at)
VALUES ($1, $2, $3, now())
ON CONFLICT (phone) DO UPDATE
SET email = CASE WHEN EXCLUDED.email <> '' THEN EXCLUDED.email ELSE patients.email END,
    updated_at = now()
RETURNING id;

-- name: CreateBooking :one
INSERT INTO bookings (
    code, patient_id, treatment_id, therapist_id, patient_name,
    start_at, end_at, blocked_until, price, patient_note
)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
RETURNING *;

-- name: ListActiveBookingRanges :many
SELECT start_at, blocked_until
FROM bookings
WHERE therapist_id = $1
  AND status IN ('pending', 'confirmed')
  AND blocked_until > sqlc.arg(range_start)
  AND start_at < sqlc.arg(range_end)
  AND id <> sqlc.arg(exclude_id)
ORDER BY start_at;

-- name: GetBookingByCode :one
SELECT * FROM bookings
WHERE code = $1;

-- name: CancelBookingByCode :one
UPDATE bookings
SET status = 'cancelled',
    cancel_reason = sqlc.arg(cancel_reason),
    cancelled_at = now(),
    updated_at = now()
WHERE code = sqlc.arg(code)
  AND status IN ('pending', 'confirmed')
  AND start_at > sqlc.arg(min_start_at)
RETURNING *;
