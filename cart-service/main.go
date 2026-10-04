// Command cart-service serves a signed-in Shopper's own cart: add, update and
// remove line items, view the current cart, and clear it. One cart per
// customer, backed by an in-memory store — no database, per the design.
package main

import (
	"log"
	"net/http"

	"cart-service/internal/auth"
	"cart-service/internal/handlers"
	"cart-service/internal/store"
)

func main() {
	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		log.Fatalf("gateway assertion: %v", err)
	}

	srv := handlers.New(store.New())
	handler := handlers.Router(verifier, srv)

	log.Println("cart-service listening on :9090")
	if err := http.ListenAndServe(":9090", handler); err != nil {
		log.Fatalf("cart-service: %v", err)
	}
}
