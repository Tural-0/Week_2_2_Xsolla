BEGIN;

CREATE TABLE items (
    id          SERIAL PRIMARY KEY,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    price       INTEGER      NOT NULL,
    stock       INTEGER      NOT NULL DEFAULT 0,
    created_at  TIMESTAMP  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_items_price ON items (price);
CREATE INDEX idx_items_name  ON items (name);

COMMIT;
