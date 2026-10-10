-- name: ListTherapistSchedules :many
SELECT day_of_week, start_time, end_time
FROM therapist_schedules
WHERE therapist_id = $1
ORDER BY day_of_week, start_time;

-- name: DeleteTherapistSchedules :exec
DELETE FROM therapist_schedules
WHERE therapist_id = $1;

-- name: AddTherapistSchedules :exec
INSERT INTO therapist_schedules (therapist_id, day_of_week, start_time, end_time)
SELECT sqlc.arg(therapist_id),
       unnest(sqlc.arg(days)::smallint[]),
       unnest(sqlc.arg(start_times)::time[]),
       unnest(sqlc.arg(end_times)::time[]);

-- name: ListUpcomingTimeOff :many
SELECT * FROM therapist_time_off
WHERE therapist_id = $1 AND end_at > now()
ORDER BY start_at;

-- name: CreateTimeOff :one
INSERT INTO therapist_time_off (therapist_id, start_at, end_at, reason)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: DeleteTimeOff :execrows
DELETE FROM therapist_time_off
WHERE id = $1 AND therapist_id = $2;
