// Command catalog-service owns the book catalog: CRUD on books plus
// search/browse, backed by an in-memory store with no database. Book search
// and detail reads are public; creating, updating and removing books is
// restricted to Store Staff via the books:manage scope, enforced entirely at
// the API gateway.
package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"catalog-service/internal/auth"
	"catalog-service/internal/config"
	"catalog-service/internal/gen"
	"catalog-service/internal/handlers"
	"catalog-service/internal/store"
)

func main() {
	cfg := config.Load()

	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		log.Fatalf("gateway assertion: %v", err)
	}

	srv := handlers.New(store.New())

	r := chi.NewRouter()
	r.Use(verifier.Middleware)
	handler := gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{BaseRouter: r})

	slog.Info("catalog-service listening", "port", cfg.Port)
	if err := http.ListenAndServe(":"+cfg.Port, handler); err != nil {
		log.Fatal(err)
	}
}
