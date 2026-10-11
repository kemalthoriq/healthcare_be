-- name: NextInvoiceNumber :one
INSERT INTO invoice_counters (period, last_number)
VALUES ($1, 1)
ON CONFLICT (period) DO UPDATE
SET last_number = invoice_counters.last_number + 1
RETURNING last_number;

-- name: CreateInvoice :one
INSERT INTO invoices (number, booking_id, subtotal, discount, total, created_by)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING id;

-- name: GetInvoice :one
SELECT i.id, i.number, i.booking_id, i.subtotal, i.discount, i.total, i.status, i.payment_method,
       i.paid_at, i.created_by, i.created_at, i.updated_at,
       b.code AS booking_code, b.patient_name, b.start_at AS booking_start_at, p.phone AS patient_phone,
       tr.name AS treatment_name, th.name AS therapist_name
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
JOIN patients p ON p.id = b.patient_id
JOIN treatments tr ON tr.id = b.treatment_id
JOIN therapists th ON th.id = b.therapist_id
WHERE i.id = $1;

-- name: ListInvoices :many
SELECT i.id, i.number, i.booking_id, i.subtotal, i.discount, i.total, i.status, i.payment_method,
       i.paid_at, i.created_by, i.created_at, i.updated_at,
       b.code AS booking_code, b.patient_name, b.start_at AS booking_start_at, p.phone AS patient_phone,
       tr.name AS treatment_name, th.name AS therapist_name
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
JOIN patients p ON p.id = b.patient_id
JOIN treatments tr ON tr.id = b.treatment_id
JOIN therapists th ON th.id = b.therapist_id
WHERE (sqlc.narg(status)::text IS NULL OR i.status = sqlc.narg(status)::text)
  AND (sqlc.narg(booking_id)::bigint IS NULL OR i.booking_id = sqlc.narg(booking_id)::bigint)
  AND (sqlc.arg(pattern)::text = '' OR i.number ILIKE sqlc.arg(pattern)::text OR b.patient_name ILIKE sqlc.arg(pattern)::text)
ORDER BY i.created_at DESC, i.id DESC
LIMIT sqlc.arg(page_limit) OFFSET sqlc.arg(page_offset);

-- name: CountInvoices :one
SELECT count(*)
FROM invoices i
JOIN bookings b ON b.id = i.booking_id
WHERE (sqlc.narg(status)::text IS NULL OR i.status = sqlc.narg(status)::text)
  AND (sqlc.narg(booking_id)::bigint IS NULL OR i.booking_id = sqlc.narg(booking_id)::bigint)
  AND (sqlc.arg(pattern)::text = '' OR i.number ILIKE sqlc.arg(pattern)::text OR b.patient_name ILIKE sqlc.arg(pattern)::text);

-- name: UpdateInvoiceDiscount :one
UPDATE invoices
SET discount = sqlc.arg(discount),
    total = subtotal - sqlc.arg(discount),
    updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'unpaid'
RETURNING id;

-- name: PayInvoice :one
UPDATE invoices
SET status = 'paid',
    payment_method = sqlc.arg(payment_method),
    paid_at = now(),
    updated_at = now()
WHERE id = sqlc.arg(id) AND status = 'unpaid'
RETURNING id;
