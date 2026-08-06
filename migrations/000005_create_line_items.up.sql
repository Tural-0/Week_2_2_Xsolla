BEGIN;

CREATE TABLE IF NOT EXISTS line_items (
    id       SERIAL  PRIMARY KEY,
    order_id INTEGER NOT NULL REFERENCES orders (id),
    item_id  INTEGER NOT NULL REFERENCES items  (id),
    price    INTEGER NOT NULL CHECK (price >= 0),
    quantity INTEGER NOT NULL CHECK (quantity > 0)
);

CREATE INDEX idx_line_items_order_id ON line_items (order_id);
CREATE INDEX idx_line_items_item_id  ON line_items (item_id);

COMMIT;
