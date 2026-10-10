-- name: GetClinicSettings :one
SELECT * FROM clinic_settings
WHERE id = 1;

-- name: UpdateClinicSettings :one
UPDATE clinic_settings
SET buffer_minutes = $1,
    min_booking_lead_minutes = $2,
    max_booking_days_ahead = $3,
    cancel_deadline_hours = $4,
    updated_at = now()
WHERE id = 1
RETURNING *;
