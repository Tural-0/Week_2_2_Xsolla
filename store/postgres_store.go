package store

import (
	"checkout-api/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore is an in-memory store for items and orders.
type PostgresStore struct {
	pool *pgxpool.Pool
}

// NewPostgresStore creates a Store pre-loaded with seed data.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	s := &PostgresStore{
		pool: pool,
	}
	return s
}

func (s *PostgresStore) DB() *Query {
	return &Query{
		DBTX: s.pool,
	}
}

func (s *PostgresStore) WithTx(tx pgx.Tx) *Query {
	return &Query{
		DBTX: tx,
	}
}

func (s *PostgresStore) GetItems(ctx context.Context) ([]*models.Item, error) {
	rows, err := s.DB().GetItems(ctx)
	if err != nil {
		return nil, fmt.Errorf("%w: failed to select all items", err)
	}
	defer rows.Close()

	items := make([]*models.Item, 0)
	for rows.Next() {
		var item models.Item
		err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("unable to scan row: %w", err)
		}
		items = append(items, &item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil
}

func (s *PostgresStore) GetItem(ctx context.Context, ID int) (*models.Item, error) {
	var item models.Item
	err := s.DB().GetItemByID(ctx, ID).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	q := s.WithTx(tx)

	// Pessimistic lock: acquire row-level lock on each item and verify sufficient stock.
	for _, lineItem := range items {
		var item models.Item
		err := q.GetItemByIDForUpdate(ctx, lineItem.ItemID).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, fmt.Errorf("item %d not found", lineItem.ItemID)
			}
			return nil, fmt.Errorf("failed to lock item %d: %w", lineItem.ItemID, err)
		}
		if item.Stock < lineItem.Quantity {
			return nil, fmt.Errorf("insufficient stock for item %d: have %d, need %d", lineItem.ItemID, item.Stock, lineItem.Quantity)
		}
	}

	var orderID int
	if err := q.InsertOrderReturning(ctx, userID, total, status).Scan(&orderID); err != nil {
		return nil, fmt.Errorf("failed to insert order: %w", err)
	}

	for _, lineItem := range items {
		if _, err := q.InsertLineItem(ctx, orderID, lineItem.ItemID, lineItem.Price, lineItem.Quantity); err != nil {
			return nil, fmt.Errorf("failed to insert line item for item %d: %w", lineItem.ItemID, err)
		}
		if _, err := q.DecrementItemStock(ctx, lineItem.ItemID, lineItem.Quantity); err != nil {
			return nil, fmt.Errorf("failed to decrement stock for item %d: %w", lineItem.ItemID, err)
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return &models.Order{
		ID:     orderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}, nil
}

func (s *PostgresStore) UpdateOrderStatus(ctx context.Context, orderID int, status string) error {
	_, err := s.DB().UpdateOrderStatus(ctx, orderID, status)
	return err
}

func (s *PostgresStore) CreateUserCart(ctx context.Context, cart *models.Cart) error {
	_, err := s.pool.Exec(ctx,
		"INSERT INTO carts (user_id, item_id, quantity) VALUES ($1, $2, $3)", cart.UserID, cart.ItemID, cart.Quantity)
	if err != nil {
		return fmt.Errorf("%w: failed to run query on CreateUserCart while INSERT", err)
	}

	return nil
}

func (s *PostgresStore) UpsertCartItem(ctx context.Context, userID int, itemID int, quantity int) error {
	_, err := s.DB().UpsertCart(ctx, userID, itemID, quantity)
	return err
}

func (s *PostgresStore) GetUserCart(ctx context.Context, userID int) ([]models.CartItem, error) {
	rows, err := s.DB().GetItemsFromUserCart(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get cart for user %d: %w", userID, err)
	}
	defer rows.Close()

	items := make([]models.CartItem, 0)
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt, &item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan cart row: %w", err)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil
}

func (s *PostgresStore) DeleteUserCart(ctx context.Context, userID int) error {
	_, err := s.DB().DeleteCartByUserID(ctx, userID)
	return err
}

func (s *PostgresStore) RemoveCartItem(ctx context.Context, userID int, itemID int) error {
	_, err := s.DB().DeleteItemFromUserCart(ctx, userID, itemID)
	return err
}

func (s *PostgresStore) FindUserByEmail(ctx context.Context, email string) (models.User, error) {
	var user models.User
	row := s.DB().GetUserByEmail(ctx, email)
	err := row.Scan(&user.ID, &user.Email, &user.Hash)
	if err != nil {
		return models.User{}, err
	}

	return user, nil
}

func (s *PostgresStore) SaveUser(ctx context.Context, email string, hash []byte) error {
	_, err := s.DB().InsertUser(ctx, email, hash)
	return err
}

/////////////////////  JWT TOKENS  ///////////////////////////

func (s *PostgresStore) GetRefreshToken(ctx context.Context, token string) (int, bool, error) {
	var userID int
	var active bool

	err := s.pool.QueryRow(
		ctx,
		`SELECT user_id, active
		 FROM refresh_tokens
		 WHERE token = $1`,
		token,
	).Scan(&userID, &active)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, err
		}
		return 0, false, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return userID, active, nil
}

func (s *PostgresStore) SaveRefreshToken(ctx context.Context, userID int, token string) error {

	_, err := s.pool.Exec(
		ctx,
		`INSERT INTO refresh_tokens
		(token, user_id, active)
		VALUES ($1, $2, TRUE)`,
		token,
		userID,
	)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

func (s *PostgresStore) DeactivateRefreshToken(ctx context.Context, token string) error {

	_, err := s.pool.Exec(
		ctx,
		`UPDATE refresh_tokens
		 SET active = FALSE
		 WHERE token = $1`,
		token,
	)

	if err != nil {
		return fmt.Errorf("failed to deactivate refresh token: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetItemQuantityByID(ctx context.Context, userID int, itemID int) (int, error) {
	var quantity int

	err := s.pool.QueryRow(
		ctx,
		`SELECT quantity
		 FROM carts
		 WHERE user_id = $1 AND item_id = $2`,
		userID, itemID,
	).Scan(&quantity)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, nil
		}
		return 0, fmt.Errorf("failed to get item quantity: %w", err)
	}

	return quantity, nil
}

///////////////////////

func (s *PostgresStore) GetOrderByID(ctx context.Context, orderID int) ([]models.CartItem, error) {
	rows, err := s.pool.Query(
		ctx,
		`select i.id, i.name, i.description, i.price, i.stock, i.created_at, oi.quantity
		from order_items oi
		inner join items i on i.id = oi.item_id
		where oi.order_id = $1`,
		orderID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get order with ID %d: %w", orderID, err)
	}

	defer rows.Close()

	items := make([]models.CartItem, 0)
	for rows.Next() {
		var item models.CartItem
		if err := rows.Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt, &item.Quantity); err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil

}

func (s *PostgresStore) GetOrderLineItemByID(ctx context.Context, orderID int) ([]models.LineItem, error) {
	rows, err := s.pool.Query(
		ctx,
		`select i.id, oi.quantity, i.price
		from order_items oi
		inner join items i on i.id = oi.item_id
		where oi.order_id = $1`,
		orderID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get order with ID %d: %w", orderID, err)
	}

	defer rows.Close()

	items := make([]models.LineItem, 0)
	for rows.Next() {
		var item models.LineItem
		if err := rows.Scan(&item.ItemID, &item.Quantity, &item.Price); err != nil {
			return nil, fmt.Errorf("failed to scan order row: %w", err)
		}
		items = append(items, item)
	}
	if rows.Err() != nil {
		return nil, rows.Err()
	}

	return items, nil

}

func (s *PostgresStore) GetUserOrders(ctx context.Context, userID int) ([]models.Order, error) {
	var total int
	var status string

	rows, err := s.pool.Query(
		ctx,
		`SELECT id
		FROM orders
		WHERE user_id = $1`,
		userID,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to find order IDs: %w", err)
	}

	defer rows.Close()

	var orderIDs = make([]int, 0)
	for rows.Next() {
		var ID int
		if err := rows.Scan(&ID); err != nil {
			return nil, fmt.Errorf("failed to scan order IDs: %w", err)
		}
		orderIDs = append(orderIDs, ID)
	}

	var orders = make([]models.Order, 0)
	for _, orderID := range orderIDs {
		items, err := s.GetOrderLineItemByID(ctx, orderID)

		if err != nil {
			return nil, fmt.Errorf("failed to scan order items: %w", err)
		}

		s.pool.QueryRow(
			ctx,
			`SELECT total, status
			FROM orders
			WHERE user_id = $1 AND id = $2`,
			userID, orderID,
		).Scan(&total, &status)

		var order = models.Order{
			ID:     orderID,
			UserID: userID,
			Items:  items,
			Total:  total,
			Status: status,
		}
		orders = append(orders, order)
	}

	return orders, nil

}
