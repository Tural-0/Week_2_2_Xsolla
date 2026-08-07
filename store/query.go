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
