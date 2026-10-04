// Command orders-service owns checkout and order history for the bookstore
// app. It holds no database: every order lives in an in-memory store for the
// life of the process.
package main

import (
	"log"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"orders-service/internal/auth"
	"orders-service/internal/config"
	"orders-service/internal/gen"
	"orders-service/internal/handlers"
	"orders-service/internal/siblings"
	"orders-service/internal/store"
)

func main() {
	cfg := config.Load()

	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		log.Fatalf("gateway assertion: %v", err)
	}

	cartClient, err := siblings.NewCart(cfg.CartServiceURL)
	if err != nil {
		log.Fatalf("cart client: %v", err)
	}
	inventoryClient, err := siblings.NewInventory(cfg.InventoryServiceURL)
	if err != nil {
		log.Fatalf("inventory client: %v", err)
	}
	customersClient, err := siblings.NewCustomers(cfg.CustomersServiceURL)
	if err != nil {
		log.Fatalf("customers client: %v", err)
	}
	notificationsClient, err := siblings.NewNotifications(cfg.NotificationsServiceURL)
	if err != nil {
		log.Fatalf("notifications client: %v", err)
	}

	srv := &handlers.Server{
		Store:         store.New(),
		Cart:          cartClient,
		Inventory:     inventoryClient,
		Customers:     customersClient,
		Notifications: notificationsClient,
	}

	r := chi.NewRouter()
	r.Use(verifier.Middleware)
	r.Use(auth.CaptureRawAssertion)

	handler := gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{BaseRouter: r})

	slog.Info("orders-service listening", "port", 9090)
	if err := http.ListenAndServe(":9090", handler); err != nil {
		log.Fatalf("listen: %v", err)
	}
}
