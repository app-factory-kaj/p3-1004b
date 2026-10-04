// Package config reads every setting this service needs from the environment,
// once, at startup, in one place. Nothing else in the service calls os.Getenv.
package config

import (
	"os"
	"strings"
)

// Config holds every runtime setting. Every field has a sensible default so
// the service starts with no required environment variables; a sibling base
// URL that is still its default never resolves to anything reachable in a
// real deployment, but the platform always injects the real one there.
type Config struct {
	CartServiceURL          string
	InventoryServiceURL     string
	CustomersServiceURL     string
	NotificationsServiceURL string
}

// Load reads the config from the environment.
func Load() Config {
	return Config{
		CartServiceURL:          getenv("CART_SERVICE_URL", "http://cart-service.invalid"),
		InventoryServiceURL:     getenv("INVENTORY_SERVICE_URL", "http://inventory-service.invalid"),
		CustomersServiceURL:     getenv("CUSTOMERS_SERVICE_URL", "http://customers-service.invalid"),
		NotificationsServiceURL: getenv("NOTIFICATIONS_SERVICE_URL", "http://notifications-service.invalid"),
	}
}

func getenv(key, def string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return def
}
