// Package store is cart-service's in-memory cart store — no database, per the
// design. One cart per customer, keyed by the gateway assertion's `sub`.
package store

import (
	"errors"
	"sync"
	"time"

	"cart-service/internal/models"
)

// ErrItemNotFound is returned when a caller updates or removes a book that is
// not in their own cart.
var ErrItemNotFound = errors.New("store: no such item in this cart")

// Store holds every customer's cart in memory, guarded by one mutex. A
// service with this little traffic does not need sharding, and in-memory
// means every cart is lost on restart — expected, per the design.
type Store struct {
	mu    sync.Mutex
	carts map[string]*models.Cart
}

// New returns an empty store.
func New() *Store {
	return &Store{carts: make(map[string]*models.Cart)}
}

// cartFor returns the caller's cart, creating an empty one on first use.
// Caller must hold s.mu.
func (s *Store) cartFor(customerID string) *models.Cart {
	c, ok := s.carts[customerID]
	if !ok {
		c = &models.Cart{
			ID:         "cart-" + customerID,
			CustomerID: customerID,
			Items:      []models.Item{},
			UpdatedAt:  time.Now().UTC(),
		}
		s.carts[customerID] = c
	}
	return c
}

// clone returns a value copy so a caller can never mutate the store's own
// slice through the returned Cart.
func clone(c *models.Cart) models.Cart {
	items := make([]models.Item, len(c.Items))
	copy(items, c.Items)
	return models.Cart{ID: c.ID, CustomerID: c.CustomerID, Items: items, UpdatedAt: c.UpdatedAt}
}

// Get returns the caller's own cart, creating an empty one on first view.
func (s *Store) Get(customerID string) models.Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	return clone(s.cartFor(customerID))
}

// Clear empties the caller's own cart.
func (s *Store) Clear(customerID string) models.Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cartFor(customerID)
	c.Items = []models.Item{}
	c.UpdatedAt = time.Now().UTC()
	return clone(c)
}

// AddItem adds a book to the caller's own cart. An existing line for the same
// book has its quantity increased by qty; otherwise a new line is appended.
func (s *Store) AddItem(customerID, bookID string, qty int) models.Cart {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cartFor(customerID)
	for i := range c.Items {
		if c.Items[i].BookID == bookID {
			c.Items[i].Quantity += qty
			c.UpdatedAt = time.Now().UTC()
			return clone(c)
		}
	}
	c.Items = append(c.Items, models.Item{BookID: bookID, Quantity: qty})
	c.UpdatedAt = time.Now().UTC()
	return clone(c)
}

// UpdateItem sets a book's quantity in the caller's own cart. Returns
// ErrItemNotFound when the book is not already a line in that cart.
func (s *Store) UpdateItem(customerID, bookID string, qty int) (models.Cart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cartFor(customerID)
	for i := range c.Items {
		if c.Items[i].BookID == bookID {
			c.Items[i].Quantity = qty
			c.UpdatedAt = time.Now().UTC()
			return clone(c), nil
		}
	}
	return models.Cart{}, ErrItemNotFound
}

// RemoveItem removes a book from the caller's own cart. Returns
// ErrItemNotFound when the book is not already a line in that cart.
func (s *Store) RemoveItem(customerID, bookID string) (models.Cart, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := s.cartFor(customerID)
	for i := range c.Items {
		if c.Items[i].BookID == bookID {
			c.Items = append(c.Items[:i], c.Items[i+1:]...)
			c.UpdatedAt = time.Now().UTC()
			return clone(c), nil
		}
	}
	return models.Cart{}, ErrItemNotFound
}
