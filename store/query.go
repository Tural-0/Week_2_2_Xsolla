package store

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type DBTX interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
}

type Query struct {
	DBTX DBTX
}

func (q *Query) GetItems(ctx context.Context) (pgx.Rows, error) {
	return q.DBTX.Query(ctx, "select id, name, description, price, stock, created_at from items")
}

func (q *Query) GetItemsOffset(ctx context.Context, limit int, offset int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`SELECT id, name, description, price, stock, created_at
		FROM items
		ORDER BY id
		LIMIT $1 OFFSET $2`,
		limit,
		offset,
	)
}

func (q *Query) GetItemsCursor(ctx context.Context, limit int, cursor *int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`
		SELECT id, name, description, price, stock, created_at
		FROM items
		WHERE ($1::int IS NULL OR id > $1)
		ORDER BY id
		LIMIT $2
		`,
		cursor,
		limit,
	)
}

func (q *Query) GetItemByID(ctx context.Context, id int) pgx.Row {
	return q.DBTX.QueryRow(ctx, "select id, name, description, price, stock, created_at from items where id = $1", id)
}

func (q *Query) GetItemsFromUserCart(ctx context.Context, userID int) (pgx.Rows, error) {
	return q.DBTX.Query(ctx,
		`select i.id, i.name, i.description, i.price, i.stock, i.created_at, c.quantity
		from carts c
		inner join items i on i.id = c.item_id
		where c.user_id = $1`,
		userID,
	)
}

func (q *Query) GetItemByIDForUpdate(ctx context.Context, id int) pgx.Row {
	return q.DBTX.QueryRow(ctx, "select id, name, description, price, stock, created_at from items where id = $1 FOR UPDATE", id)
}

func (q *Query) InsertOrderReturning(ctx context.Context, userID int, total int, status string) pgx.Row {
	return q.DBTX.QueryRow(ctx, "insert into orders (user_id, total, status) values ($1, $2, $3) RETURNING id", userID, total, status)
}

func (q *Query) UpdateOrderStatus(ctx context.Context, orderID int, status string) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx, "update orders set status = $1 where id = $2", status, orderID)
}

func (q *Query) CreateUserCart(ctx context.Context, userID int, itemID int, quantity int) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(
		ctx,
		`INSERT INTO carts (user_id, item_id, quantity)
		VALUES ($1, $2, $3)", userID, itemID, quantity)`,
	)
}

func (q *Query) InsertLineItem(ctx context.Context, orderID int, itemID int, price int, quantity int) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx, "insert into order_items (order_id, item_id, price, quantity) values ($1, $2, $3, $4)", orderID, itemID, price, quantity)
}

func (q *Query) DecrementItemStock(ctx context.Context, itemID int, quantity int) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx, "update items set stock = stock - $1 where id = $2", quantity, itemID)
}

func (q *Query) UpsertCart(ctx context.Context, userID int, itemID int, quantity int) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx,
		"insert into carts (user_id, item_id, quantity) values ($1, $2, $3) on conflict (user_id, item_id) do update set quantity = excluded.quantity",
		userID, itemID, quantity,
	)
}

func (q *Query) GetCartByUserID(ctx context.Context, userID int) (pgx.Rows, error) {
	return q.DBTX.Query(ctx, "select item_id, quantity from carts where user_id = $1", userID)
}

func (q *Query) DeleteCartByUserID(ctx context.Context, userID int) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx, "delete from carts where user_id = $1", userID)
}

func (q *Query) DeleteItemFromUserCart(ctx context.Context, userID int, itemID int) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx, "delete from carts where user_id = $1 and item_id = $2", userID, itemID)
}

func (q *Query) InsertUser(ctx context.Context, email string, hash []byte) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(ctx, "insert into users (email, hash) values ($1, $2)", email, hash)
}

func (q *Query) GetUserByEmail(ctx context.Context, email string) pgx.Row {
	return q.DBTX.QueryRow(ctx, "select id, email, hash from users where email = $1", email)
}

func (q *Query) GetRefreshToken(ctx context.Context, token string) pgx.Row {
	return q.DBTX.QueryRow(
		ctx,
		`SELECT user_id, active
		 FROM refresh_tokens
		 WHERE token = $1`,
		token,
	)
}

func (q *Query) SaveRefreshToken(ctx context.Context, userID int, token string) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(
		ctx,
		`INSERT INTO refresh_tokens
		(token, user_id, active)
		VALUES ($1, $2, TRUE)`,
		token,
		userID,
	)
}

func (q *Query) DeactivateRefreshToken(ctx context.Context, token string) (pgconn.CommandTag, error) {
	return q.DBTX.Exec(
		ctx,
		`UPDATE refresh_tokens
		 SET active = FALSE
		 WHERE token = $1`,
		token,
	)
}

func (q *Query) GetItemQuantityByID(ctx context.Context, userID int, itemID int) pgx.Row {
	return q.DBTX.QueryRow(
		ctx,
		`SELECT quantity
		 FROM carts
		 WHERE user_id = $1 AND item_id = $2`,
		userID, itemID,
	)
}

func (q *Query) GetOrderByID(ctx context.Context, orderID int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`select i.id, i.name, i.description, i.price, i.stock, i.created_at, oi.quantity
		from order_items oi
		inner join items i on i.id = oi.item_id
		where oi.order_id = $1`,
		orderID,
	)
}

func (q *Query) GetOrderLineItemByID(ctx context.Context, orderID int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`select i.id, oi.quantity, i.price
		from order_items oi
		inner join items i on i.id = oi.item_id
		where oi.order_id = $1`,
		orderID,
	)
}

func (q *Query) GetUserOrderIDs(ctx context.Context, userID int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`SELECT id
		FROM orders
		WHERE user_id = $1`,
		userID,
	)
}

func (q *Query) GetOrderTotalStatus(ctx context.Context, userID int, orderID int) pgx.Row {
	return q.DBTX.QueryRow(
		ctx,
		`SELECT total, status
		FROM orders
		WHERE user_id = $1 AND id = $2`,
		userID, orderID,
	)
}

func (q *Query) GetUserOrdersOffset(ctx context.Context, userID int, limit int, offset int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`
		SELECT id
		FROM orders
		WHERE user_id = $1
		ORDER BY id DESC
		LIMIT $2 OFFSET $3
		`,
		userID,
		limit,
		offset,
	)
}

func (q *Query) GetUserOrdersCursor(ctx context.Context, userID int, limit int, cursor *int) (pgx.Rows, error) {
	return q.DBTX.Query(
		ctx,
		`
		SELECT id
		FROM orders
		WHERE user_id = $1
		AND ($2::int IS NULL OR id < $2)
		ORDER BY id DESC
		LIMIT $3
		`,
		userID,
		cursor,
		limit,
	)
}

func (q *Query) GetDiscountCode(ctx context.Context, discount string) pgx.Row {
	return q.DBTX.QueryRow(
		ctx,
		`SELECT code, amount, ends_at
		FROM discount_codes
		WHERE code = $1`,
		discount,
	)
}
