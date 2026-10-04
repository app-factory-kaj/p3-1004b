// Package siblings wraps the generated clients for the services orders-service
// consumes (cart-service, inventory-service, customers-service,
// notifications-service), each reached over its injected `project`-visibility
// address and each call carrying the caller's own gateway assertion forward
// (see internal/auth/forward.go) — never a minted or forged one.
package siblings

import (
	"context"
	"fmt"

	"orders-service/internal/auth"
	cartclient "orders-service/internal/clients/cart"
)

// Cart is orders-service's view of cart-service.
type Cart struct {
	api *cartclient.ClientWithResponses
}

// NewCart builds a client against the injected cart-service address.
func NewCart(baseURL string) (*Cart, error) {
	api, err := cartclient.NewClientWithResponses(baseURL, cartclient.WithRequestEditorFn(auth.ForwardAssertion))
	if err != nil {
		return nil, fmt.Errorf("cart client: %w", err)
	}
	return &Cart{api: api}, nil
}

// GetMyCart returns the caller's own cart.
func (c *Cart) GetMyCart(ctx context.Context) (*cartclient.Cart, error) {
	resp, err := c.api.GetMyCartWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get cart: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get cart: unexpected status %s", resp.Status())
	}
	return resp.JSON200, nil
}

// ClearMyCart empties the caller's own cart.
func (c *Cart) ClearMyCart(ctx context.Context) error {
	resp, err := c.api.ClearMyCartWithResponse(ctx)
	if err != nil {
		return fmt.Errorf("clear cart: %w", err)
	}
	if resp.StatusCode() != 204 {
		return fmt.Errorf("clear cart: unexpected status %s", resp.Status())
	}
	return nil
}
