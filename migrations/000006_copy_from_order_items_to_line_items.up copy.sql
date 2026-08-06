BEGIN;

INSERT INTO line_items 
SELECT * FROM order_items;

COMMIT;
