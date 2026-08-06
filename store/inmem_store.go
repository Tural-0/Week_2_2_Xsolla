package store

import (
	"checkout-api/models"
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
)

// InMemStore is an in-memory store for items and orders.
type InMemStore struct {
	items       map[int]*models.Item
	orders      map[int]*models.Order
	carts       map[int][]models.Cart
	nextOrderID int
	nextUserID  int
	users       map[string]models.User
}

// NewInMemStore creates a Store pre-loaded with seed data.
func NewInMemStore() *InMemStore {
	s := &InMemStore{
		items:       make(map[int]*models.Item),
		orders:      make(map[int]*models.Order),
		carts:       make(map[int][]models.Cart),
		nextOrderID: 1,
		nextUserID:  1,
		users:       make(map[string]models.User),
	}

	s.items[1] = &models.Item{ID: 1, Name: "Laptop", Description: "A fast laptop", Price: 120000, Stock: 10}
	s.items[2] = &models.Item{ID: 2, Name: "Mouse", Description: "Wireless mouse", Price: 2500, Stock: 50}
	s.items[3] = &models.Item{ID: 3, Name: "Keyboard", Description: "Mechanical keyboard", Price: 8000, Stock: 30}

	return s
}

func (s *InMemStore) GetItems(_ context.Context) ([]*models.Item, error) {
	items := make([]*models.Item, 0, len(s.items))
	for _, item := range s.items {
		items = append(items, item)
	}
	return items, nil
}

func (s *InMemStore) GetItem(_ context.Context, id int) (*models.Item, error) {
	return s.items[id], nil
}

func (s *InMemStore) CreateOrder(_ context.Context, userID int, items []models.LineItem, total int, status string) (*models.Order, error) {
	order := &models.Order{
		ID:     s.nextOrderID,
		UserID: userID,
		Items:  items,
		Total:  total,
		Status: status,
	}
	s.orders[order.ID] = order
	s.nextOrderID++
	return order, nil
}

func (s *InMemStore) UpdateOrderStatus(_ context.Context, orderID int, status string) error {
	order, ok := s.orders[orderID]
	if !ok {
		return fmt.Errorf("order %d not found", orderID)
	}
	order.Status = status
	return nil
}

func (s *InMemStore) UpsertCartItem(_ context.Context, userID int, itemID int, quantity int) error {
	for i := range s.carts[userID] {
		if s.carts[userID][i].ItemID == itemID {
			s.carts[userID][i].Quantity = quantity
			return nil
		}
	}
	s.carts[userID] = append(s.carts[userID], models.Cart{UserID: userID, ItemID: itemID, Quantity: quantity})
	return nil
}

func (s *InMemStore) GetUserCart(_ context.Context, userID int) ([]models.CartItem, error) {
	cartItems := make([]models.CartItem, 0, len(s.carts[userID]))
	for _, row := range s.carts[userID] {
		item, ok := s.items[row.ItemID]
		if !ok {
			continue
		}
		cartItems = append(cartItems, models.CartItem{
			ID:          item.ID,
			Name:        item.Name,
			Description: item.Description,
			Price:       item.Price,
			Stock:       item.Stock,
			CreatedAt:   item.CreatedAt,
			Quantity:    row.Quantity,
		})
	}
	return cartItems, nil
}

func (s *InMemStore) DeleteUserCart(_ context.Context, userID int) error {
	delete(s.carts, userID)
	return nil
}

func (s *InMemStore) RemoveCartItem(_ context.Context, userID int, itemID int) error {
	for i, item := range s.carts[userID] {
		if item.ItemID == itemID {
			s.carts[userID] = append(s.carts[userID][:i], s.carts[userID][i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("item %d not found in cart", itemID)
}

func (s *InMemStore) SaveUser(_ context.Context, email string, hash []byte) error {
	if _, exists := s.users[email]; exists {
		return fmt.Errorf("user %s already exists", email)
	}
	s.users[email] = models.User{ID: s.nextUserID, Email: email, Hash: hash}
	s.nextUserID++
	return nil
}

func (s *InMemStore) FindUserByEmail(_ context.Context, email string) (models.User, error) {
	user, ok := s.users[email]
	if !ok {
		return models.User{}, pgx.ErrNoRows
	}
	return user, nil
}
