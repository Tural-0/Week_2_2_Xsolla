BEGIN;

CREATE TABLE IF NOT EXISTS order_items (
    id       SERIAL  PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders (id),
    item_id  INTEGER NOT NULL REFERENCES items  (id),
    price    INTEGER NOT NULL CHECK (price >= 0),
    quantity INTEGER NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_order_items_order_id ON order_items (order_id);
CREATE INDEX idx_order_items_item_id  ON order_items (item_id);

COMMIT;
