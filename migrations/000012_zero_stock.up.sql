BEGIN;

UPDATE items
SET stock = 0
WHERE id = (SELECT id FROM items
ORDER BY price ASC LIMIT 1);

COMMIT;