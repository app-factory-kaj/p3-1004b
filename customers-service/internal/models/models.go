// Package models holds the plain domain types customers-service stores. They
// carry no JSON tags and no HTTP concerns — internal/handlers converts
// between these and the generated OpenAPI types.
package models

// Customer is a signed-in Shopper's own account profile. ID is always the
// gateway assertion's `sub` for the caller that owns it — never a client
// supplied value.
type Customer struct {
	ID    string
	Name  string
	Email string
}

// Address is one shipping address belonging to a Customer.
type Address struct {
	ID         string
	Line1      string
	Line2      string
	City       string
	Region     string
	PostalCode string
	Country    string
	IsDefault  bool
}
