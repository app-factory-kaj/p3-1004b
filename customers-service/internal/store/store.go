// Package store is customers-service's in-memory, no-database persistence.
// Everything is keyed strictly by ownerID — the gateway assertion's verified
// `sub` — never by anything a client supplies, so one caller's rows are never
// reachable through another caller's request.
package store

import (
	"crypto/rand"
	"encoding/hex"
	"sync"

	"customers-service/internal/models"
)

// Store is safe for concurrent use.
type Store struct {
	mu        sync.RWMutex
	profiles  map[string]models.Customer  // ownerID -> profile
	addresses map[string][]models.Address // ownerID -> addresses, insertion order
}

// New returns an empty Store.
func New() *Store {
	return &Store{
		profiles:  make(map[string]models.Customer),
		addresses: make(map[string][]models.Address),
	}
}

// GetOrCreateProfile returns the caller's profile, auto-provisioning a blank
// one on first access. There is no separate signup step for a Shopper, so a
// new caller's first read finds a row (id set, name/email empty) rather than
// a 404.
func (s *Store) GetOrCreateProfile(ownerID string) models.Customer {
	s.mu.Lock()
	defer s.mu.Unlock()
	c, ok := s.profiles[ownerID]
	if !ok {
		c = models.Customer{ID: ownerID}
		s.profiles[ownerID] = c
	}
	return c
}

// PutProfile creates or updates the caller's profile and returns the saved
// row.
func (s *Store) PutProfile(ownerID, name, email string) models.Customer {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := models.Customer{ID: ownerID, Name: name, Email: email}
	s.profiles[ownerID] = c
	return c
}

// ListAddresses returns a copy of the caller's addresses in insertion order.
func (s *Store) ListAddresses(ownerID string) []models.Address {
	s.mu.RLock()
	defer s.mu.RUnlock()
	addrs := s.addresses[ownerID]
	out := make([]models.Address, len(addrs))
	copy(out, addrs)
	return out
}

// AddAddress appends a new address for the caller, assigning it a fresh id,
// and returns the saved row.
func (s *Store) AddAddress(ownerID string, in models.Address) models.Address {
	s.mu.Lock()
	defer s.mu.Unlock()
	in.ID = newID()
	s.addresses[ownerID] = append(s.addresses[ownerID], in)
	return in
}

// UpdateAddress updates one of the caller's own addresses by id. ok is false
// when no address with that id exists for THIS caller — only the caller's own
// slice is ever searched, so another caller's address id is indistinguishable
// from one that does not exist at all.
func (s *Store) UpdateAddress(ownerID, addressID string, in models.Address) (models.Address, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	addrs := s.addresses[ownerID]
	for i, a := range addrs {
		if a.ID == addressID {
			in.ID = addressID
			addrs[i] = in
			return in, true
		}
	}
	return models.Address{}, false
}

// RemoveAddress removes one of the caller's own addresses by id. ok is false
// when no address with that id exists for this caller.
func (s *Store) RemoveAddress(ownerID, addressID string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	addrs := s.addresses[ownerID]
	for i, a := range addrs {
		if a.ID == addressID {
			s.addresses[ownerID] = append(addrs[:i], addrs[i+1:]...)
			return true
		}
	}
	return false
}

// newID returns a fresh, unguessable address id.
func newID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b) // crypto/rand.Read does not fail on supported platforms
	return hex.EncodeToString(b)
}
