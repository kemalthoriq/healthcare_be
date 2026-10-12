-- +goose Up
ALTER TABLE clinic_settings
    ADD COLUMN clinic_name TEXT NOT NULL DEFAULT 'Lumière Clinic',
    ADD COLUMN address TEXT NOT NULL DEFAULT '',
    ADD COLUMN phone TEXT NOT NULL DEFAULT '',
    ADD COLUMN whatsapp TEXT NOT NULL DEFAULT '',
    ADD COLUMN email TEXT NOT NULL DEFAULT '',
    ADD COLUMN opening_hours_text TEXT NOT NULL DEFAULT '';

-- +goose Down
ALTER TABLE clinic_settings
    DROP COLUMN clinic_name,
    DROP COLUMN address,
    DROP COLUMN phone,
    DROP COLUMN whatsapp,
    DROP COLUMN email,
    DROP COLUMN opening_hours_text;
