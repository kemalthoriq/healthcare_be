-- +goose Up
CREATE TABLE promos (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    title TEXT NOT NULL,
    slug TEXT NOT NULL UNIQUE,
    description TEXT NOT NULL DEFAULT '',
    image_url TEXT NOT NULL DEFAULT '',
    start_date DATE NOT NULL,
    end_date DATE NOT NULL,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (end_date >= start_date)
);

CREATE TABLE promo_treatments (
    promo_id BIGINT NOT NULL REFERENCES promos (id) ON DELETE CASCADE,
    treatment_id BIGINT NOT NULL REFERENCES treatments (id) ON DELETE CASCADE,
    PRIMARY KEY (promo_id, treatment_id)
);

CREATE INDEX promo_treatments_treatment_id_idx ON promo_treatments (treatment_id);

-- +goose Down
DROP TABLE promo_treatments;
DROP TABLE promos;
