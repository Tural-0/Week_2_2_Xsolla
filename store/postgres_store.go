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

func (s *PostgresStore) GetItemsOffset(
	ctx context.Context,
	limit int,
	offset int,
) ([]models.Item, error) {
	rows, err := s.DB().GetItemsOffset(ctx, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item

	for rows.Next() {
		var item models.Item

		if err := rows.Scan(
			&item.ID, &item.Name,
			&item.Description, &item.Price,
			&item.Stock, &item.CreatedAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
	}

	return items, nil
}

func (s *PostgresStore) GetItemsCursor(
	ctx context.Context,
	limit int,
	cursor *int,
) ([]models.Item, error) {

	rows, err := s.DB().GetItemsCursor(ctx, limit, cursor)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []models.Item

	for rows.Next() {
		var item models.Item

		if err := rows.Scan(
			&item.ID, &item.Name,
			&item.Description, &item.Price,
			&item.Stock, &item.CreatedAt); err != nil {
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		return nil, err
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

func (s *PostgresStore) CreateOrder(ctx context.Context, userID int, items []models.LineItem, total int, status string, discount int) (*models.Order, error) {
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
	total = total - (total / 100 * discount)
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
	_, err := s.DB().CreateUserCart(ctx, cart.UserID, cart.ItemID, cart.Quantity)
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

	err := s.DB().GetRefreshToken(ctx, token).Scan(&userID, &active)

	if err != nil {
		if err == pgx.ErrNoRows {
			return 0, false, err
		}
		return 0, false, fmt.Errorf("failed to get refresh token: %w", err)
	}

	return userID, active, nil
}

func (s *PostgresStore) SaveRefreshToken(ctx context.Context, userID int, token string) error {

	_, err := s.DB().SaveRefreshToken(ctx, userID, token)

	if err != nil {
		return fmt.Errorf("failed to save refresh token: %w", err)
	}

	return nil
}

func (s *PostgresStore) DeactivateRefreshToken(ctx context.Context, token string) error {

	_, err := s.DB().DeactivateRefreshToken(ctx, token)

	if err != nil {
		return fmt.Errorf("failed to deactivate refresh token: %w", err)
	}

	return nil
}

func (s *PostgresStore) GetItemQuantityByID(ctx context.Context, userID int, itemID int) (int, error) {
	var quantity int

	err := s.DB().GetItemQuantityByID(ctx, userID, itemID).Scan(&quantity)

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
	rows, err := s.DB().GetOrderByID(ctx, orderID)

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
	rows, err := s.DB().GetOrderLineItemByID(ctx, orderID)

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

	rows, err := s.DB().GetUserOrderIDs(ctx, userID)

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

		s.DB().GetOrderTotalStatus(ctx, userID, orderID).Scan(&total, &status)

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

func (s *PostgresStore) GetUserOrdersOffset(
	ctx context.Context,
	userID int,
	limit int,
	offset int,
) ([]models.Order, error) {
	var total int
	var status string

	rows, err := s.DB().GetUserOrdersOffset(ctx, userID, limit, offset)
	if err != nil {
		return nil, err
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

		s.DB().GetOrderTotalStatus(ctx, userID, orderID).Scan(&total, &status)

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

func (s *PostgresStore) GetUserOrdersCursor(
	ctx context.Context,
	userID int,
	limit int,
	cursor *int,
) ([]models.Order, error) {
	var total int
	var status string

	rows, err := s.DB().GetUserOrdersCursor(ctx, userID, limit, cursor)
	if err != nil {
		return nil, err
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

		s.DB().GetOrderTotalStatus(ctx, userID, orderID).Scan(&total, &status)

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

func (s *PostgresStore) GetDiscountDetails(ctx context.Context, discountCode string) (models.Discount, error) {
	var disc models.Discount

	err := s.DB().GetDiscountCode(ctx, discountCode).Scan(&disc.Code, &disc.Amount, &disc.Ends_at)
	if err != nil {
		if err == pgx.ErrNoRows {
			fmt.Print("no rows")
			return models.Discount{}, fmt.Errorf("this discount code is invalid")
		}
		fmt.Printf("err occured")
		fmt.Print(err)
		return models.Discount{}, fmt.Errorf("failed to get discount details: %w", err)
	}

	return disc, nil

}
