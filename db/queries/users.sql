-- name: CountAdmins :one
SELECT count(*) FROM users WHERE role = 'admin';

-- name: CreateUser :one
INSERT INTO users (name, email, password_hash, role)
VALUES ($1, $2, $3, $4)
RETURNING id, name, email, role, is_active, created_at, updated_at;
-- name: GetUserByEmail :one
SELECT id, name, email, password_hash, role, is_active, created_at, updated_at
FROM users
WHERE lower(email) = lower(sqlc.arg(email));
