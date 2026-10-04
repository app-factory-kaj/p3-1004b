// Command inventory-service serves stock levels and reservations for every
// book, backed by an in-memory store with no database (design.json). Reading
// stock levels is public; adjusting on-hand quantity and viewing/making
// reservations are restricted by scope at the API gateway, which this service
// trusts via its signed assertion — see internal/auth.
package main

import (
	"log"
	"net/http"
	"os"

	"github.com/go-chi/chi/v5"

	"inventory-service/internal/auth"
	"inventory-service/internal/gen"
	"inventory-service/internal/handlers"
	"inventory-service/internal/store"
)

const defaultPort = "9090"

func main() {
	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		log.Fatalf("gateway assertion: %v", err)
	}

	st := store.New()
	srv := handlers.New(st)

	r := chi.NewRouter()
	r.Use(verifier.Middleware)
	handler := gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{BaseRouter: r})

	port := os.Getenv("PORT")
	if port == "" {
		port = defaultPort
	}

	log.Printf("inventory-service listening on :%s", port)
	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("server: %v", err)
	}
}
