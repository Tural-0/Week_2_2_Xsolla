BEGIN;

INSERT INTO items (name, description, price, stock) VALUES
    ('Xsolla T-Shirt',     'Classic cotton tee with Xsolla logo. Unisex fit.',         2500,  120),
    ('Developer Hoodie',   'Heavyweight pullover hoodie. Perfect for late-night PRs.',  6000,   45),
    ('Sticker Pack',       '10-pack of Xsolla and open-source themed stickers.',         500,  300),
    ('Mechanical Keyboard','Tenkeyless, Cherry MX Brown switches. USB-C.',             18000,   18),
    ('Laptop Stand',       'Aluminium adjustable stand. Folds flat for travel.',        4500,   60),
    ('USB-C Hub',          '7-in-1 hub: HDMI 4K, 3× USB-A, SD, microSD, PD.',         3200,   75),
    ('Notebook (A5)',      'Dot-grid, 200 pages, lay-flat binding.',                    1200,  200),
    ('Cable Organiser',    'Leather magnetic cable ties, pack of 6.',                    800,  150);

-- Order 1: paid — user 1 bought a T-shirt and a sticker pack
INSERT INTO orders (user_id, total, status) VALUES (1, 3000, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (1, 1, 2500, 1),
    (1, 3,  500, 1);

-- Order 2: pending — user 2 added a keyboard to their cart
INSERT INTO orders (user_id, total, status) VALUES (2, 18000, 'pending');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (2, 4, 18000, 1);

-- Order 3: failed — user 1 ordered a hoodie and a hub
INSERT INTO orders (user_id, total, status) VALUES (1, 9200, 'failed');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (3, 2, 6000, 1),
    (3, 6, 3200, 1);

-- Order 4: paid — user 3 bought three notebooks
INSERT INTO orders (user_id, total, status) VALUES (3, 3600, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (4, 7, 1200, 3);

-- Order 5: paid — user 2 bought a laptop stand and two cable organisers
INSERT INTO orders (user_id, total, status) VALUES (2, 6100, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (5, 5, 4500, 1),
    (5, 8,  800, 2);

-- Order 6: pending — user 1 adding a hoodie and a USB-C hub
INSERT INTO orders (user_id, total, status) VALUES (1, 9200, 'pending');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (6, 2, 6000, 1),
    (6, 6, 3200, 1);

-- Order 7: paid — user 3 bought a keyboard and a laptop stand
INSERT INTO orders (user_id, total, status) VALUES (3, 22500, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (7, 4, 18000, 1),
    (7, 5,  4500, 1);

-- Order 8: paid — user 1 bought five sticker packs
INSERT INTO orders (user_id, total, status) VALUES (1, 2500, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (8, 3, 500, 5);

-- Order 9: failed — user 2 tried to order a keyboard and hoodie
INSERT INTO orders (user_id, total, status) VALUES (2, 24000, 'failed');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (9, 4, 18000, 1),
    (9, 2,  6000, 1);

-- Order 10: paid — user 3 bought a T-shirt, notebook, and sticker pack
INSERT INTO orders (user_id, total, status) VALUES (3, 4200, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (10, 1, 2500, 1),
    (10, 7, 1200, 1),
    (10, 3,  500, 1);

-- Order 11: paid — user 1 bought two USB-C hubs
INSERT INTO orders (user_id, total, status) VALUES (1, 6400, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (11, 6, 3200, 2);

-- Order 12: pending — user 3 browsing a hoodie and cable organiser
INSERT INTO orders (user_id, total, status) VALUES (3, 6800, 'pending');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (12, 2, 6000, 1),
    (12, 8,  800, 1);

-- Order 13: paid — user 2 bought three T-shirts
INSERT INTO orders (user_id, total, status) VALUES (2, 7500, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (13, 1, 2500, 3);

-- Order 14: paid — user 1 bought a laptop stand, notebook, and sticker pack
INSERT INTO orders (user_id, total, status) VALUES (1, 6200, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (14, 5, 4500, 1),
    (14, 7, 1200, 1),
    (14, 3,  500, 1);

-- Order 15: failed — user 3 tried a keyboard
INSERT INTO orders (user_id, total, status) VALUES (3, 18000, 'failed');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (15, 4, 18000, 1);

-- Order 16: paid — user 2 bought a hoodie and two sticker packs
INSERT INTO orders (user_id, total, status) VALUES (2, 7000, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (16, 2, 6000, 1),
    (16, 3,  500, 2);

-- Order 17: paid — user 1 bought a USB-C hub, cable organiser, and notebook
INSERT INTO orders (user_id, total, status) VALUES (1, 5200, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (17, 6, 3200, 1),
    (17, 8,  800, 1),
    (17, 7, 1200, 1);

-- Order 18: pending — user 3 adding a T-shirt and laptop stand
INSERT INTO orders (user_id, total, status) VALUES (3, 7000, 'pending');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (18, 1, 2500, 1),
    (18, 5, 4500, 1);

-- Order 19: paid — user 2 bought four notebooks
INSERT INTO orders (user_id, total, status) VALUES (2, 4800, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (19, 7, 1200, 4);

-- Order 20: paid — user 1 bought a keyboard
INSERT INTO orders (user_id, total, status) VALUES (1, 18000, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (20, 4, 18000, 1);

-- Order 21: paid — user 3 bought three USB-C hubs
INSERT INTO orders (user_id, total, status) VALUES (3, 9600, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (21, 6, 3200, 3);

-- Order 22: failed — user 1 tried a hoodie and laptop stand
INSERT INTO orders (user_id, total, status) VALUES (1, 10500, 'failed');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (22, 2, 6000, 1),
    (22, 5, 4500, 1);

-- Order 23: paid — user 2 bought a T-shirt, sticker pack, and cable organiser
INSERT INTO orders (user_id, total, status) VALUES (2, 3800, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (23, 1, 2500, 1),
    (23, 3,  500, 1),
    (23, 8,  800, 1);

-- Order 24: pending — user 3 has a keyboard in cart
INSERT INTO orders (user_id, total, status) VALUES (3, 18000, 'pending');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (24, 4, 18000, 1);

-- Order 25: paid — user 1 bought two hoodies
INSERT INTO orders (user_id, total, status) VALUES (1, 12000, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (25, 2, 6000, 2);

-- Order 26: paid — user 2 bought a laptop stand and four sticker packs
INSERT INTO orders (user_id, total, status) VALUES (2, 6500, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (26, 5, 4500, 1),
    (26, 3,  500, 4);

-- Order 27: failed — user 3 tried a USB-C hub and two notebooks
INSERT INTO orders (user_id, total, status) VALUES (3, 5600, 'failed');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (27, 6, 3200, 1),
    (27, 7, 1200, 2);

-- Order 28: paid — user 1 bought a sticker pack, cable organiser, and T-shirt
INSERT INTO orders (user_id, total, status) VALUES (1, 3800, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (28, 3,  500, 1),
    (28, 8,  800, 1),
    (28, 1, 2500, 1);

-- Order 29: pending — user 2 has a hoodie and USB-C hub in cart
INSERT INTO orders (user_id, total, status) VALUES (2, 9200, 'pending');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (29, 2, 6000, 1),
    (29, 6, 3200, 1);

-- Order 30: paid — user 3 bought a keyboard, cable organiser, and sticker pack
INSERT INTO orders (user_id, total, status) VALUES (3, 19300, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (30, 4, 18000, 1),
    (30, 8,   800, 1),
    (30, 3,   500, 1);

-- Order 31: paid — user 1 bought a laptop stand and three cable organisers
INSERT INTO orders (user_id, total, status) VALUES (1, 6900, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (31, 5, 4500, 1),
    (31, 8,  800, 3);

-- Order 32: paid — user 2 bought five sticker packs and two notebooks
INSERT INTO orders (user_id, total, status) VALUES (2, 4900, 'paid');
INSERT INTO order_items (order_id, item_id, price, quantity) VALUES
    (32, 3,  500, 5),
    (32, 7, 1200, 2);

COMMIT;
