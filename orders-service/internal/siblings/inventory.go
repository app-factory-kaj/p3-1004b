package siblings

import (
	"context"
	"fmt"

	"orders-service/internal/auth"
	inventoryclient "orders-service/internal/clients/inventory"
)

// Inventory is orders-service's view of inventory-service.
type Inventory struct {
	api *inventoryclient.ClientWithResponses
}

// NewInventory builds a client against the injected inventory-service address.
func NewInventory(baseURL string) (*Inventory, error) {
	api, err := inventoryclient.NewClientWithResponses(baseURL, inventoryclient.WithRequestEditorFn(auth.ForwardAssertion))
	if err != nil {
		return nil, fmt.Errorf("inventory client: %w", err)
	}
	return &Inventory{api: api}, nil
}

// ReserveResult is the outcome of one reservation attempt.
type ReserveResult struct {
	// Reservation is set only when the reservation succeeded.
	Reservation *inventoryclient.Reservation
	// InsufficientStock is true when inventory-service refused the
	// reservation for lack of stock (its 400) — distinct from a hard error.
	InsufficientStock bool
	// Message is the upstream's refusal message, when InsufficientStock.
	Message string
}

// Reserve asks inventory-service to hold stock for one line item of orderID.
func (i *Inventory) Reserve(ctx context.Context, bookID, orderID string, quantity int) (ReserveResult, error) {
	resp, err := i.api.ReserveStockWithResponse(ctx, inventoryclient.ReserveStockJSONRequestBody{
		BookID:   bookID,
		OrderID:  orderID,
		Quantity: quantity,
	})
	if err != nil {
		return ReserveResult{}, fmt.Errorf("reserve stock for %s: %w", bookID, err)
	}
	switch {
	case resp.JSON201 != nil:
		return ReserveResult{Reservation: resp.JSON201}, nil
	case resp.JSON400 != nil:
		return ReserveResult{InsufficientStock: true, Message: resp.JSON400.Message}, nil
	default:
		return ReserveResult{}, fmt.Errorf("reserve stock for %s: unexpected status %s", bookID, resp.Status())
	}
}

// Release gives a held reservation back, best-effort cleanup after a later
// line item in the same checkout failed.
func (i *Inventory) Release(ctx context.Context, reservationID string) error {
	resp, err := i.api.ReleaseReservationWithResponse(ctx, reservationID)
	if err != nil {
		return fmt.Errorf("release reservation %s: %w", reservationID, err)
	}
	if resp.JSON200 == nil {
		return fmt.Errorf("release reservation %s: unexpected status %s", reservationID, resp.Status())
	}
	return nil
}
