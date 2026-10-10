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
ORDER BY start_at;
