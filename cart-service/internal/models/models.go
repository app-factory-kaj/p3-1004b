// Package models holds cart-service's own domain types — independent of the
// generated API types, so the store never imports the gen package.
package models

import "time"

// Item is one line in a cart: a book and how many copies of it.
type Item struct {
	BookID   string
	Quantity int
}

// Cart is one signed-in customer's own cart. CustomerID is the gateway
// assertion's `sub` — the only key for rows this service creates.
type Cart struct {
	ID         string
	CustomerID string
	Items      []Item
	UpdatedAt  time.Time
}

// Subtotal is the cart's running total.
//
// Design gap: cart-service's design.json declares no dependency that could
// supply a book's price (BOOK.price is owned by catalog-service, and neither
// cart-service nor orders-service depends on it), and CartItem in
// openapi.yaml carries no price field either. Absent a wired price source,
// Subtotal is the only value this component can compute correctly from data
// it owns: the sum of item quantities. Flagged on issue #5 for the design to
// wire a real price source rather than have this worked around.
func (c Cart) Subtotal() float64 {
	var total float64
	for _, it := range c.Items {
		total += float64(it.Quantity)
	}
	return total
}
