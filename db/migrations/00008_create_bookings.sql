-- +goose Up
CREATE TABLE patients (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    name TEXT NOT NULL,
    phone TEXT NOT NULL UNIQUE,
    email TEXT NOT NULL DEFAULT '',
    consent_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE TABLE bookings (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    code TEXT NOT NULL UNIQUE,
    patient_id BIGINT NOT NULL REFERENCES patients (id),
    treatment_id BIGINT NOT NULL REFERENCES treatments (id),
    therapist_id BIGINT NOT NULL REFERENCES therapists (id),
    patient_name TEXT NOT NULL,
    start_at TIMESTAMPTZ NOT NULL,
    end_at TIMESTAMPTZ NOT NULL,
    blocked_until TIMESTAMPTZ NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'confirmed', 'completed', 'no_show', 'cancelled')),
    price BIGINT NOT NULL CHECK (price >= 0),
    patient_note TEXT NOT NULL DEFAULT '',
    cancel_reason TEXT NOT NULL DEFAULT '',
    confirmed_at TIMESTAMPTZ,
    cancelled_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (start_at < end_at AND end_at <= blocked_until),
    EXCLUDE USING gist (
        therapist_id WITH =,
        tstzrange(start_at, blocked_until) WITH &&
    ) WHERE (status IN ('pending', 'confirmed'))
);

CREATE INDEX bookings_therapist_id_start_at_idx ON bookings (therapist_id, start_at);
CREATE INDEX bookings_status_start_at_idx ON bookings (status, start_at);
CREATE INDEX bookings_patient_id_idx ON bookings (patient_id);

-- +goose Down
DROP TABLE bookings;
DROP TABLE patients;
