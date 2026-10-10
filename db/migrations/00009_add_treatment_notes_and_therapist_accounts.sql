-- +goose Up
CREATE TABLE treatment_notes (
    booking_id BIGINT PRIMARY KEY REFERENCES bookings (id) ON DELETE CASCADE,
    author_id BIGINT NOT NULL REFERENCES users (id),
    content TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

ALTER TABLE therapists ADD COLUMN user_id BIGINT UNIQUE REFERENCES users (id) ON DELETE SET NULL;

-- +goose Down
ALTER TABLE therapists DROP COLUMN user_id;
DROP TABLE treatment_notes;
