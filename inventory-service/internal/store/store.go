// Package store is the in-memory backing store for stock levels and
// reservations. This service owns no database — see design.json — so every
// record lives only as long as the process does.
package store

import (
	"errors"
	"fmt"
	"sort"
	"sync"
)

// StockLevel is one book's on-hand and reserved quantity.
type StockLevel struct {
	BookID           string
	QuantityOnHand   int
	QuantityReserved int
}

// ReservationStatus is the lifecycle of a Reservation.
type ReservationStatus string

const (
	StatusHeld     ReservationStatus = "held"
	StatusReleased ReservationStatus = "released"
)

// Reservation is stock held against an order being checked out.
type Reservation struct {
	ID       string
	BookID   string
	OrderID  string
	Quantity int
	Status   ReservationStatus
}

// ErrNotFound is returned when a bookId or reservationId has no record.
var ErrNotFound = errors.New("store: not found")

// ErrInsufficientStock is returned when a reservation would exceed the
// available (on-hand minus already-reserved) quantity for a book.
var ErrInsufficientStock = errors.New("store: insufficient stock")

// ErrInvalidQuantity is returned for a non-positive quantity.
var ErrInvalidQuantity = errors.New("store: invalid quantity")

// Store is a thread-safe in-memory store for stock levels and reservations.
type Store struct {
	mu           sync.Mutex
	stockLevels  map[string]*StockLevel
	reservations map[string]*Reservation
	nextResID    int
}

// New returns a Store seeded with a handful of plausible book ids so there is
// something to read from a fresh process.
func New() *Store {
	s := &Store{
		stockLevels:  make(map[string]*StockLevel),
		reservations: make(map[string]*Reservation),
	}
	seed := []StockLevel{
		{BookID: "book-001", QuantityOnHand: 42},
		{BookID: "book-002", QuantityOnHand: 7},
		{BookID: "book-003", QuantityOnHand: 0},
		{BookID: "book-004", QuantityOnHand: 150},
		{BookID: "book-005", QuantityOnHand: 3},
	}
	for _, sl := range seed {
		v := sl
		s.stockLevels[sl.BookID] = &v
	}
	return s
}

// ListStockLevels returns a page of stock levels, sorted by bookId for a
// stable, deterministic ordering, plus the total count.
func (s *Store) ListStockLevels(limit, offset int) ([]StockLevel, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.stockLevels))
	for id := range s.stockLevels {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	total := len(ids)
	out := make([]StockLevel, 0, limit)
	for i := offset; i < total && len(out) < limit; i++ {
		out = append(out, *s.stockLevels[ids[i]])
	}
	return out, total
}

// GetStockLevel returns the stock level for one book.
func (s *Store) GetStockLevel(bookID string) (StockLevel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sl, ok := s.stockLevels[bookID]
	if !ok {
		return StockLevel{}, ErrNotFound
	}
	return *sl, nil
}

// SetStockLevel sets the on-hand quantity for a book, creating the record if
// it does not yet exist. The reserved quantity is left untouched.
func (s *Store) SetStockLevel(bookID string, quantityOnHand int) (StockLevel, error) {
	if quantityOnHand < 0 {
		return StockLevel{}, ErrInvalidQuantity
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	sl, ok := s.stockLevels[bookID]
	if !ok {
		sl = &StockLevel{BookID: bookID}
		s.stockLevels[bookID] = sl
	}
	sl.QuantityOnHand = quantityOnHand
	return *sl, nil
}

// ListReservations returns a page of reservations, sorted by id for a stable,
// deterministic ordering, plus the total count.
func (s *Store) ListReservations(limit, offset int) ([]Reservation, int) {
	s.mu.Lock()
	defer s.mu.Unlock()

	ids := make([]string, 0, len(s.reservations))
	for id := range s.reservations {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	total := len(ids)
	out := make([]Reservation, 0, limit)
	for i := offset; i < total && len(out) < limit; i++ {
		out = append(out, *s.reservations[ids[i]])
	}
	return out, total
}

// Reserve holds `quantity` of `bookId` against `orderId`. A book with no
// stock-level record at all is treated as zero on-hand, so it is refused the
// same as any other insufficient-stock case rather than crashing or silently
// creating stock out of nothing.
func (s *Store) Reserve(bookID, orderID string, quantity int) (Reservation, error) {
	if quantity < 1 {
		return Reservation{}, ErrInvalidQuantity
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	sl, ok := s.stockLevels[bookID]
	if !ok {
		sl = &StockLevel{BookID: bookID}
		s.stockLevels[bookID] = sl
	}
	available := sl.QuantityOnHand - sl.QuantityReserved
	if quantity > available {
		return Reservation{}, ErrInsufficientStock
	}

	sl.QuantityReserved += quantity
	s.nextResID++
	res := Reservation{
		ID:       fmt.Sprintf("res-%06d", s.nextResID),
		BookID:   bookID,
		OrderID:  orderID,
		Quantity: quantity,
		Status:   StatusHeld,
	}
	s.reservations[res.ID] = &res
	return res, nil
}

// Release returns a held reservation's quantity back to the book's available
// stock. Releasing an already-released reservation is idempotent: it returns
// the reservation unchanged rather than double-crediting the stock level.
func (s *Store) Release(reservationID string) (Reservation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, ok := s.reservations[reservationID]
	if !ok {
		return Reservation{}, ErrNotFound
	}
	if res.Status == StatusReleased {
		return *res, nil
	}
	if sl, ok := s.stockLevels[res.BookID]; ok {
		sl.QuantityReserved -= res.Quantity
		if sl.QuantityReserved < 0 {
			sl.QuantityReserved = 0
		}
	}
	res.Status = StatusReleased
	return *res, nil
}
