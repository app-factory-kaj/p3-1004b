// Package store is orders-service's in-memory order store. No database — the
// design declares none, and an order lives only as long as the process does.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"sort"
	"sync"
	"time"

	"orders-service/internal/gen"
)

// record is one order plus the facts the public Order schema does not carry:
// who owns it, and the simulated payment-method reference checkout recorded.
type record struct {
	order            gen.Order
	ownerID          string
	paymentMethodRef string
}

// Store is safe for concurrent use.
type Store struct {
	mu     sync.Mutex
	orders map[string]record
}

// New returns an empty store.
func New() *Store {
	return &Store{orders: make(map[string]record)}
}

// NewOrderID returns an id for a new order, unique within this process.
func (s *Store) NewOrderID() string {
	buf := make([]byte, 8)
	_, _ = rand.Read(buf)
	return fmt.Sprintf("order_%d_%s", time.Now().UnixNano(), hex.EncodeToString(buf))
}

// Create stores a new order owned by ownerID. The order is expected to
// already carry its id (from NewOrderID).
func (s *Store) Create(ownerID string, order gen.Order, paymentMethodRef string) gen.Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.orders[order.ID] = record{order: order, ownerID: ownerID, paymentMethodRef: paymentMethodRef}
	return order
}

// Get returns the order if it exists AND belongs to ownerID. A row that
// exists but belongs to someone else is reported exactly like one that does
// not exist — the caller never learns it's there.
func (s *Store) Get(ownerID, orderID string) (gen.Order, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.orders[orderID]
	if !ok || rec.ownerID != ownerID {
		return gen.Order{}, false
	}
	return rec.order, true
}

// ListByOwner returns every order owned by ownerID, most recently placed
// first.
func (s *Store) ListByOwner(ownerID string) []gen.Order {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]gen.Order, 0)
	for _, rec := range s.orders {
		if rec.ownerID == ownerID {
			out = append(out, rec.order)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].PlacedAt > out[j].PlacedAt })
	return out
}
