-- name: ListAdminBookings :many
SELECT b.id, b.code, b.treatment_id, b.therapist_id, b.patient_name, b.start_at, b.end_at,
       b.status, b.price, b.created_at, p.phone AS patient_phone
FROM bookings b
JOIN patients p ON p.id = b.patient_id
WHERE (sqlc.narg(status)::text IS NULL OR b.status = sqlc.narg(status)::text)
  AND (sqlc.narg(therapist_id)::bigint IS NULL OR b.therapist_id = sqlc.narg(therapist_id)::bigint)
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR b.start_at >= sqlc.narg(date_from)::timestamptz)
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.start_at < sqlc.narg(date_to)::timestamptz)
ORDER BY b.start_at, b.id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountAdminBookings :one
SELECT count(*)
FROM bookings b
WHERE (sqlc.narg(status)::text IS NULL OR b.status = sqlc.narg(status)::text)
  AND (sqlc.narg(therapist_id)::bigint IS NULL OR b.therapist_id = sqlc.narg(therapist_id)::bigint)
  AND (sqlc.narg(date_from)::timestamptz IS NULL OR b.start_at >= sqlc.narg(date_from)::timestamptz)
  AND (sqlc.narg(date_to)::timestamptz IS NULL OR b.start_at < sqlc.narg(date_to)::timestamptz);

-- name: GetAdminBooking :one
SELECT b.id, b.code, b.treatment_id, b.therapist_id, b.patient_name, b.start_at, b.end_at,
       b.status, b.price, b.patient_note, b.cancel_reason, b.confirmed_at, b.cancelled_at,
       b.completed_at, b.created_at, p.phone AS patient_phone, p.email AS patient_email,
       COALESCE(n.content, '') AS treatment_note
FROM bookings b
JOIN patients p ON p.id = b.patient_id
LEFT JOIN treatment_notes n ON n.booking_id = b.id
WHERE b.id = $1;

-- name: TransitionBooking :one
UPDATE bookings
SET status = sqlc.arg(to_status)::text,
    cancel_reason = CASE WHEN sqlc.arg(to_status)::text = 'cancelled' THEN sqlc.arg(cancel_reason)::text ELSE cancel_reason END,
    confirmed_at = CASE WHEN sqlc.arg(to_status)::text = 'confirmed' THEN now() ELSE confirmed_at END,
    cancelled_at = CASE WHEN sqlc.arg(to_status)::text = 'cancelled' THEN now() ELSE cancelled_at END,
    completed_at = CASE WHEN sqlc.arg(to_status)::text = 'completed' THEN now() ELSE completed_at END,
    updated_at = now()
WHERE id = sqlc.arg(id) AND status = sqlc.arg(from_status)::text
RETURNING id;

-- name: RescheduleBooking :one
UPDATE bookings
SET start_at = $2,
    end_at = $3,
    blocked_until = $4,
    updated_at = now()
WHERE id = $1 AND status IN ('pending', 'confirmed')
RETURNING id;

-- name: UpsertTreatmentNote :exec
INSERT INTO treatment_notes (booking_id, author_id, content)
VALUES ($1, $2, $3)
ON CONFLICT (booking_id) DO UPDATE
SET content = EXCLUDED.content,
    author_id = EXCLUDED.author_id,
    updated_at = now();
