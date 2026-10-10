-- +goose Up
CREATE TABLE therapists (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    title TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    photo_url TEXT NOT NULL DEFAULT '',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE treatment_therapists (
    treatment_id BIGINT NOT NULL REFERENCES treatments (id) ON DELETE CASCADE,
    therapist_id BIGINT NOT NULL REFERENCES therapists (id) ON DELETE CASCADE,
    PRIMARY KEY (treatment_id, therapist_id)
);

CREATE INDEX treatment_therapists_therapist_id_idx ON treatment_therapists (therapist_id);

-- +goose Down
DROP TABLE treatment_therapists;
DROP TABLE therapists;
