BEGIN;

CREATE TABLE IF NOT EXISTS discount_codes (
    discount_id BIGSERIAL PRIMARY KEY,
    code VARCHAR NOT NULL,
    amount INT NOT NULL,
    ends_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE (discount_id)
);

INSERT INTO discount_codes (code, amount, ends_at) VALUES ('XSOLLA10', 10, '2026-09-15 15:00:00');
INSERT INTO discount_codes (code, amount, ends_at) VALUES ('XSOLLA20', 20, '2025-09-15 15:00:00');
INSERT INTO discount_codes (code, amount, ends_at) VALUES ('XSOLLA30', 30, '2026-09-15 15:00:00');

COMMIT;
