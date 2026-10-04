package siblings

import (
	"context"
	"fmt"

	"orders-service/internal/auth"
	customersclient "orders-service/internal/clients/customers"
)

// Customers is orders-service's view of customers-service.
type Customers struct {
	api *customersclient.ClientWithResponses
}

// NewCustomers builds a client against the injected customers-service address.
func NewCustomers(baseURL string) (*Customers, error) {
	api, err := customersclient.NewClientWithResponses(baseURL, customersclient.WithRequestEditorFn(auth.ForwardAssertion))
	if err != nil {
		return nil, fmt.Errorf("customers client: %w", err)
	}
	return &Customers{api: api}, nil
}

// GetMyProfile returns the caller's own profile, or (nil, nil) when they have
// none yet.
func (c *Customers) GetMyProfile(ctx context.Context) (*customersclient.Customer, error) {
	resp, err := c.api.GetMyProfileWithResponse(ctx)
	if err != nil {
		return nil, fmt.Errorf("get profile: %w", err)
	}
	if resp.JSON404 != nil {
		return nil, nil
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("get profile: unexpected status %s", resp.Status())
	}
	return resp.JSON200, nil
}

// ListMyAddresses returns every shipping address the caller owns.
func (c *Customers) ListMyAddresses(ctx context.Context) ([]customersclient.Address, error) {
	resp, err := c.api.ListMyAddressesWithResponse(ctx, &customersclient.ListMyAddressesParams{Limit: 100})
	if err != nil {
		return nil, fmt.Errorf("list addresses: %w", err)
	}
	if resp.JSON200 == nil {
		return nil, fmt.Errorf("list addresses: unexpected status %s", resp.Status())
	}
	return resp.JSON200.Data, nil
}
