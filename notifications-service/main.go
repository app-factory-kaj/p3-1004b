// Command notifications-service serves in-app order-status notifications.
//
// Backed by an in-memory store only (no database, per design.json). Every
// read under /me/ is scoped to the caller resolved from the verified gateway
// assertion; POST /notifications is called by orders-service to create a
// notification for any customer, named in the request body.
package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"notifications-service/internal/auth"
	"notifications-service/internal/gen"
	"notifications-service/internal/handlers"
	"notifications-service/internal/store"
)

func main() {
	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		log.Fatalf("gateway assertion: %v", err)
	}

	st := store.New()
	srv := handlers.New(st)

	r := chi.NewRouter()
	r.Use(verifier.Middleware)

	handler := gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{
		BaseRouter: r,
	})

	addr := ":9090"
	log.Printf("notifications-service listening on %s", addr)
	if err := http.ListenAndServe(addr, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}
