// Package store holds the in-memory notifications store. This service has no
// database per its design.json — every record lives only for the life of the
// process.
package store

import (
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"notifications-service/internal/gen"
)

// Record is one notification, owned by the customer it was created for.
type Record struct {
	ID         string
	CustomerID string
	OrderID    string
	Message    string
	Read       bool
	CreatedAt  time.Time
}

// ToModel projects a Record into the generated Notification response type.
// CustomerID is deliberately not exposed: the API never returns another
// caller's identifier, only the caller's own notification shape.
func (r Record) ToModel() gen.Notification {
	return gen.Notification{
		ID:        r.ID,
		OrderID:   r.OrderID,
		Message:   r.Message,
		Read:      r.Read,
		CreatedAt: r.CreatedAt.Format(time.RFC3339),
	}
}

// Store is an in-memory, concurrency-safe notification store.
type Store struct {
	mu            sync.RWMutex
	byID          map[string]*Record
	orderedByCust map[string][]string // customerID -> notification IDs, oldest first
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		byID:          make(map[string]*Record),
		orderedByCust: make(map[string][]string),
	}
}

// Create adds a notification for customerID and returns the stored record.
func (s *Store) Create(customerID, orderID, message string) Record {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec := &Record{
		ID:         uuid.NewString(),
		CustomerID: customerID,
		OrderID:    orderID,
		Message:    message,
		Read:       false,
		CreatedAt:  time.Now().UTC(),
	}
	s.byID[rec.ID] = rec
	s.orderedByCust[customerID] = append(s.orderedByCust[customerID], rec.ID)
	return *rec
}

// ListByCustomer returns customerID's notifications, newest first, with
// limit/offset applied, plus the total count matching (before paging).
func (s *Store) ListByCustomer(customerID string, limit, offset int) ([]Record, int) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	ids := s.orderedByCust[customerID]
	all := make([]Record, 0, len(ids))
	for _, id := range ids {
		if rec, ok := s.byID[id]; ok {
			all = append(all, *rec)
		}
	}
	// Newest first.
	sort.Slice(all, func(i, j int) bool { return all[i].CreatedAt.After(all[j].CreatedAt) })

	total := len(all)
	if offset > total {
		offset = total
	}
	end := offset + limit
	if end > total {
		end = total
	}
	return all[offset:end], total
}

// MarkRead marks notificationID as read, but ONLY when it belongs to
// customerID. Returns (record, true) on success; (zero, false) when no such
// notification exists for that customer — the caller answers 404, never a
// 403, whether the id does not exist at all or belongs to someone else.
func (s *Store) MarkRead(customerID, notificationID string) (Record, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()

	rec, ok := s.byID[notificationID]
	if !ok || rec.CustomerID != customerID {
		return Record{}, false
	}
	rec.Read = true
	return *rec, true
}
