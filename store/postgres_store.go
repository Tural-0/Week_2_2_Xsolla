package store

import (
	"checkout-api/models"
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PostgresStore is an in-memory store for items and orders.
type PostgresStore struct {
	conn *pgx.Conn
}

// NewPostgresStore creates a Store pre-loaded with seed data.
func NewPostgresStore(conn *pgx.Conn) *PostgresStore {
	s := &PostgresStore{
		conn: conn,
	}
	return s
}

func (s *PostgresStore) DB() *Query {
	return &Query{
		DBTX: s.conn,
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
		return nil, err
	}

	return items, nil
}

func (s *PostgresStore) GetItem(ctx context.Context, id int) (*models.Item, error) {
	var item models.Item
	err := s.DB().GetItemByID(ctx, id).Scan(&item.ID, &item.Name, &item.Description, &item.Price, &item.Stock, &item.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, err
	}
	return &item, nil
}

func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	tx, err := s.conn.Begin(ctx)
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
