BEGIN;

CREATE TABLE IF NOT EXISTS carts (
    id BIGSERIAL,
    user_id BIGINT NOT NULL,
    item_id BIGINT NOT NULL REFERENCES items(id),
    quantity BIGINT NOT NULL DEFAULT 1 CHECK (quantity>0),
    UNIQUE (user_id, item_id)
);

CREATE INDEX idx_user_id_item_id ON carts (user_id, item_id);

COMMIT;
