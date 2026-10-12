-- +goose Up
CREATE TABLE therapist_schedules (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    therapist_id BIGINT NOT NULL REFERENCES therapists (id) ON DELETE CASCADE,
    day_of_week SMALLINT NOT NULL CHECK (day_of_week BETWEEN 0 AND 6),
    start_time TIME NOT NULL,
    end_time TIME NOT NULL,
    CHECK (end_time > start_time),
    EXCLUDE USING gist (
        therapist_id WITH =,
        day_of_week WITH =,
        tsrange('2000-01-01'::date + start_time, '2000-01-01'::date + end_time) WITH &&
    )
);

CREATE TABLE therapist_time_off (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    therapist_id BIGINT NOT NULL REFERENCES therapists (id) ON DELETE CASCADE,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    reason TEXT NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_at > start_at)
);

CREATE INDEX therapist_time_off_therapist_id_end_at_idx ON therapist_time_off (therapist_id, end_at);

-- +goose Down
DROP TABLE therapist_time_off;
DROP TABLE therapist_schedules;
