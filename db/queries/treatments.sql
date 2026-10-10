-- name: ListActiveTreatments :many
SELECT * FROM treatments
WHERE is_active
ORDER BY name;

-- name: ListTreatments :many
SELECT * FROM treatments
ORDER BY name;

-- name: GetActiveTreatmentBySlug :one
SELECT * FROM treatments
WHERE slug = $1 AND is_active;

-- name: GetTreatment :one
SELECT * FROM treatments
WHERE id = $1;

-- name: CreateTreatment :one
INSERT INTO treatments (name, slug, description, image_url, price, duration_minutes, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING *;

-- name: UpdateTreatment :one
UPDATE treatments
SET name = $2,
    description = $3,
    image_url = $4,
    price = $5,
    duration_minutes = $6,
    is_active = $7,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTreatment :execrows
DELETE FROM treatments
WHERE id = $1;
