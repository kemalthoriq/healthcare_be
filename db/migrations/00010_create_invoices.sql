-- +goose Up
CREATE TABLE invoice_counters (
    period TEXT PRIMARY KEY,
    last_number INT NOT NULL
);

CREATE TABLE invoices (
    id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    number TEXT NOT NULL UNIQUE,
    booking_id BIGINT NOT NULL UNIQUE REFERENCES bookings (id),
    subtotal BIGINT NOT NULL CHECK (subtotal >= 0),
    discount BIGINT NOT NULL DEFAULT 0 CHECK (discount >= 0 AND discount <= subtotal),
    total BIGINT NOT NULL,
    status TEXT NOT NULL DEFAULT 'unpaid' CHECK (status IN ('unpaid', 'paid')),
    payment_method TEXT NOT NULL DEFAULT '' CHECK (payment_method IN ('', 'cash', 'transfer')),
    paid_at TIMESTAMPTZ,
    created_by BIGINT NOT NULL REFERENCES users (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CHECK (total = subtotal - discount),
    CHECK (
        (status = 'unpaid' AND payment_method = '' AND paid_at IS NULL)
        OR (status = 'paid' AND payment_method <> '' AND paid_at IS NOT NULL)
    )
);

CREATE INDEX invoices_created_at_idx ON invoices (created_at);

-- +goose Down
DROP TABLE invoices;
DROP TABLE invoice_counters;
