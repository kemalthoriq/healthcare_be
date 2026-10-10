-- +goose Up
CREATE TABLE clinic_settings (
    id SMALLINT PRIMARY KEY DEFAULT 1 CHECK (id = 1),
    buffer_minutes INTEGER NOT NULL DEFAULT 15 CHECK (buffer_minutes BETWEEN 0 AND 120),
    min_booking_lead_minutes INTEGER NOT NULL DEFAULT 120 CHECK (min_booking_lead_minutes BETWEEN 0 AND 10080),
    max_booking_days_ahead INTEGER NOT NULL DEFAULT 30 CHECK (max_booking_days_ahead BETWEEN 1 AND 365),
    cancel_deadline_hours INTEGER NOT NULL DEFAULT 24 CHECK (cancel_deadline_hours BETWEEN 0 AND 168),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

INSERT INTO clinic_settings DEFAULT VALUES;

-- +goose Down
DROP TABLE clinic_settings;
