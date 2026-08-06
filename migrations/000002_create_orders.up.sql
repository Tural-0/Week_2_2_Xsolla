BEGIN;

CREATE TABLE orders (
    id         SERIAL      PRIMARY KEY,
    user_id    INTEGER     NOT NULL,
    total      INTEGER     NOT NULL,
    status     VARCHAR(20) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_orders_user_id    ON orders (user_id);
CREATE INDEX idx_orders_status     ON orders (status);
CREATE INDEX idx_orders_created_at ON orders (created_at DESC);

COMMIT;
