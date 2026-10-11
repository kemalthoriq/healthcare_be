-- name: ListPatients :many
SELECT p.id, p.name, p.phone, p.email, p.created_at,
       count(b.id) AS booking_count,
       max(b.start_at) FILTER (WHERE b.status = 'completed') AS last_visit_at
FROM patients p
LEFT JOIN bookings b ON b.patient_id = p.id
WHERE sqlc.arg(name_pattern)::text = ''
   OR p.name ILIKE sqlc.arg(name_pattern)::text
   OR (sqlc.arg(phone_pattern)::text <> '' AND p.phone LIKE sqlc.arg(phone_pattern)::text)
GROUP BY p.id
ORDER BY p.name, p.id
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountPatients :one
SELECT count(*)
FROM patients p
WHERE sqlc.arg(name_pattern)::text = ''
   OR p.name ILIKE sqlc.arg(name_pattern)::text
   OR (sqlc.arg(phone_pattern)::text <> '' AND p.phone LIKE sqlc.arg(phone_pattern)::text);

-- name: GetPatient :one
SELECT p.id, p.name, p.phone, p.email, p.created_at,
       count(b.id) AS booking_count,
       max(b.start_at) FILTER (WHERE b.status = 'completed') AS last_visit_at
FROM patients p
LEFT JOIN bookings b ON b.patient_id = p.id
WHERE p.id = $1
GROUP BY p.id;

-- name: ListPatientBookings :many
SELECT b.id, b.code, b.treatment_id, b.therapist_id, b.patient_name, b.start_at, b.end_at,
       b.status, b.price, b.created_at, COALESCE(n.content, '') AS treatment_note
FROM bookings b
LEFT JOIN treatment_notes n ON n.booking_id = b.id
WHERE b.patient_id = $1
ORDER BY b.start_at DESC, b.id DESC;
