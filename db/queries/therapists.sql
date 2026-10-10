-- name: ListActiveTherapists :many
SELECT * FROM therapists
WHERE is_active
ORDER BY name;

-- name: ListTherapists :many
SELECT * FROM therapists
ORDER BY name;

-- name: GetActiveTherapistBySlug :one
SELECT * FROM therapists
WHERE slug = $1 AND is_active;

-- name: GetTherapist :one
SELECT * FROM therapists
WHERE id = $1;

-- name: CreateTherapist :one
INSERT INTO therapists (name, slug, title, bio, photo_url, is_active)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: UpdateTherapist :one
UPDATE therapists
SET name = $2,
    title = $3,
    bio = $4,
    photo_url = $5,
    is_active = $6,
    updated_at = now()
WHERE id = $1
RETURNING *;

-- name: DeleteTherapist :execrows
DELETE FROM therapists
WHERE id = $1;

-- name: ListTherapistsByTreatment :many
SELECT th.id, th.name, th.slug, th.title, th.bio, th.photo_url, th.is_active, th.created_at, th.updated_at
FROM therapists th
JOIN treatment_therapists tt ON tt.therapist_id = th.id
WHERE tt.treatment_id = $1
ORDER BY th.name;

-- name: DeleteTreatmentTherapists :exec
DELETE FROM treatment_therapists
WHERE treatment_id = $1;

-- name: AddTreatmentTherapists :exec
INSERT INTO treatment_therapists (treatment_id, therapist_id)
SELECT sqlc.arg(treatment_id), unnest(sqlc.arg(therapist_ids)::bigint[]);
