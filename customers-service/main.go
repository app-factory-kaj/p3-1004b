// Command customers-service serves a signed-in Shopper's own account profile
// and shipping addresses, backed by an in-memory store with no database.
package main

import (
	"log"
	"net/http"

	"customers-service/internal/auth"
	"customers-service/internal/handlers"
	"customers-service/internal/store"
)

func main() {
	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		log.Fatalf("gateway assertion: %v", err)
	}

	handler := handlers.NewHandler(store.New(), verifier)

	port := "9090"
	log.Printf("customers-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatal(err)
	}
}
