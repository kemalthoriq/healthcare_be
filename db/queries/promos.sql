-- name: ListRunningPromos :many
SELECT * FROM promos
WHERE is_active AND sqlc.arg(today)::date BETWEEN start_date AND end_date
ORDER BY end_date, id;

-- name: GetRunningPromoBySlug :one
SELECT * FROM promos
WHERE slug = sqlc.arg(slug) AND is_active AND sqlc.arg(today)::date BETWEEN start_date AND end_date;

-- name: ListPromos :many
SELECT * FROM promos
ORDER BY created_at DESC, id DESC
LIMIT $1 OFFSET $2;

-- name: CountPromos :one
SELECT count(*) FROM promos;

-- name: GetPromo :one
SELECT * FROM promos
WHERE id = $1;

-- name: CreatePromo :one
INSERT INTO promos (title, slug, description, image_url, start_date, end_date, is_active)
VALUES ($1, $2, $3, $4, $5, $6, $7)
RETURNING id;

-- name: UpdatePromo :execrows
UPDATE promos
SET title = $2,
    description = $3,
    image_url = $4,
    start_date = $5,
    end_date = $6,
    is_active = $7,
    updated_at = now()
WHERE id = $1;

-- name: DeletePromo :execrows
DELETE FROM promos
WHERE id = $1;

-- name: ListPromoTreatments :many
SELECT pt.promo_id, t.id, t.name, t.slug, t.is_active
FROM promo_treatments pt
JOIN treatments t ON t.id = pt.treatment_id
WHERE pt.promo_id = ANY(sqlc.arg(promo_ids)::bigint[])
ORDER BY t.name, t.id;

-- name: DeletePromoTreatments :exec
DELETE FROM promo_treatments
WHERE promo_id = $1;

-- name: AddPromoTreatments :exec
INSERT INTO promo_treatments (promo_id, treatment_id)
SELECT sqlc.arg(promo_id), unnest(sqlc.arg(treatment_ids)::bigint[]);
