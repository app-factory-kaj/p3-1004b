// Package config reads every environment variable this service consumes, by
// name, in one place, at startup. Every setting here has a sensible default:
// the service starts with no required environment variables.
package config

import "os"

// Config is catalog-service's resolved runtime configuration.
type Config struct {
	// Port the HTTP server listens on. Fixed at 9090 per the platform
	// contract; not overridable by an env var.
	Port string

	// InventoryServiceURL is the injected address of the inventory-service
	// component dependency (design.json's `component` dependency,
	// envBindings.address: INVENTORY_SERVICE_URL). catalog-service's own
	// openapi.yaml Book/BookInput schemas carry no field for stock data, so
	// nothing in this service calls it today — see the deviation note in
	// the issue comments / final report. Read here anyway so the dependency
	// is wired in one place, ready for a handler that the design grows a
	// field for.
	InventoryServiceURL string
}

// Load reads configuration from the environment.
func Load() Config {
	return Config{
		Port:                "9090",
		InventoryServiceURL: os.Getenv("INVENTORY_SERVICE_URL"),
	}
}
