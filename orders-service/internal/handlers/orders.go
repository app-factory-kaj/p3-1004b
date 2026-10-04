// Package handlers implements orders-service's operations against an
// in-memory store, orchestrating cart-service, inventory-service,
// customers-service and notifications-service at checkout.
package handlers

import (
	"context"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"orders-service/internal/auth"
	"orders-service/internal/gen"
	"orders-service/internal/siblings"
	"orders-service/internal/store"
)

// Server implements gen.StrictServerInterface.
type Server struct {
	Store         *store.Store
	Cart          *siblings.Cart
	Inventory     *siblings.Inventory
	Customers     *siblings.Customers
	Notifications *siblings.Notifications
}

var _ gen.StrictServerInterface = (*Server)(nil)

// GetHealth is public: it reads no identity.
func (s *Server) GetHealth(ctx context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200Response{}, nil
}

// ListMyOrders returns the caller's own orders, most recent first.
func (s *Server) ListMyOrders(ctx context.Context, request gen.ListMyOrdersRequestObject) (gen.ListMyOrdersResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return listMyOrders401JSONResponse{Code: 401, Message: "not signed in"}, nil
	}

	limit := request.Params.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := request.Params.Offset
	if offset < 0 {
		offset = 0
	}

	all := s.Store.ListByOwner(caller.UserID)
	total := len(all)
	data := make([]gen.Order, 0)
	if offset < total {
		end := offset + limit
		if end > total {
			end = total
		}
		data = append(data, all[offset:end]...)
	}
	return gen.ListMyOrders200JSONResponse{Count: total, Data: data}, nil
}

// GetMyOrder returns one of the caller's own orders, 404 when it is not
// theirs (or does not exist at all — indistinguishable on purpose).
func (s *Server) GetMyOrder(ctx context.Context, request gen.GetMyOrderRequestObject) (gen.GetMyOrderResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return getMyOrder401JSONResponse{Code: 401, Message: "not signed in"}, nil
	}
	order, ok := s.Store.Get(caller.UserID, request.OrderID)
	if !ok {
		return gen.GetMyOrder404JSONResponse(gen.Error{Code: 404, Message: "no such order for the caller"}), nil
	}
	return gen.GetMyOrder200JSONResponse(order), nil
}

// Checkout turns the caller's cart into an order: reserve stock for every
// line item, attach their chosen shipping address, record a simulated
// payment-method reference, create the order, clear the cart and notify the
// caller. A reservation refused for insufficient stock releases every
// reservation already held for this checkout, leaves the cart untouched, and
// creates no order.
func (s *Server) Checkout(ctx context.Context, request gen.CheckoutRequestObject) (gen.CheckoutResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return gen.Checkout401JSONResponse(gen.Error{Code: 401, Message: "not signed in"}), nil
	}

	if request.Body == nil {
		return gen.Checkout400JSONResponse(gen.Error{Code: 400, Message: "addressId and paymentMethodRef are required"}), nil
	}
	addressID := strings.TrimSpace(request.Body.AddressID)
	paymentMethodRef := strings.TrimSpace(request.Body.PaymentMethodRef)
	if addressID == "" || paymentMethodRef == "" {
		return gen.Checkout400JSONResponse(gen.Error{Code: 400, Message: "addressId and paymentMethodRef are required"}), nil
	}

	cart, err := s.Cart.GetMyCart(ctx)
	if err != nil {
		return nil, fmt.Errorf("checkout: %w", err)
	}
	if len(cart.Items) == 0 {
		return gen.Checkout400JSONResponse(gen.Error{Code: 400, Message: "the cart is empty"}), nil
	}

	addresses, err := s.Customers.ListMyAddresses(ctx)
	if err != nil {
		return nil, fmt.Errorf("checkout: %w", err)
	}
	addressOwned := false
	for _, a := range addresses {
		if a.ID == addressID {
			addressOwned = true
			break
		}
	}
	if !addressOwned {
		return gen.Checkout400JSONResponse(gen.Error{Code: 400, Message: "no such address for the caller"}), nil
	}

	orderID := s.Store.NewOrderID()

	totalQuantity := 0
	for _, item := range cart.Items {
		totalQuantity += item.Quantity
	}
	// cart-service's contract exposes only an aggregate `subtotal`, never a
	// per-item price (CartItem carries bookId+quantity only) and
	// orders-service has no pricing dependency of its own. Average unit price
	// across the cart's units so the order's line items re-sum to the same
	// total cart-service already computed, which is the figure the design
	// requires to be correct.
	var averageUnitPrice float32
	if totalQuantity > 0 {
		averageUnitPrice = cart.Subtotal / float32(totalQuantity)
	}

	reservedIDs := make([]string, 0, len(cart.Items))
	items := make([]gen.OrderItem, 0, len(cart.Items))
	for _, cartItem := range cart.Items {
		result, err := s.Inventory.Reserve(ctx, cartItem.BookID, orderID, cartItem.Quantity)
		if err != nil {
			s.releaseAll(ctx, reservedIDs)
			return nil, fmt.Errorf("checkout: %w", err)
		}
		if result.InsufficientStock {
			s.releaseAll(ctx, reservedIDs)
			msg := result.Message
			if msg == "" {
				msg = fmt.Sprintf("insufficient stock for %s", cartItem.BookID)
			}
			return gen.Checkout400JSONResponse(gen.Error{Code: 400, Message: msg}), nil
		}
		reservedIDs = append(reservedIDs, result.Reservation.ID)
		items = append(items, gen.OrderItem{
			BookID:    cartItem.BookID,
			Quantity:  cartItem.Quantity,
			UnitPrice: averageUnitPrice,
		})
	}

	order := gen.Order{
		ID:        orderID,
		Status:    gen.Placed,
		Total:     cart.Subtotal,
		AddressID: addressID,
		Items:     items,
		PlacedAt:  time.Now().UTC().Format(time.RFC3339),
	}
	order = s.Store.Create(caller.UserID, order, paymentMethodRef)

	if err := s.Cart.ClearMyCart(ctx); err != nil {
		// The order is already placed and stock already held in the
		// caller's name; a failure to empty the cart afterward is logged,
		// never undoes the order.
		slog.Error("checkout: clear cart failed", "orderId", orderID, "error", err)
	}

	profile, err := s.Customers.GetMyProfile(ctx)
	if err != nil {
		slog.Error("checkout: fetch profile for notification failed", "orderId", orderID, "error", err)
	} else if profile != nil {
		if err := s.Notifications.Create(ctx, profile.ID, orderID, "Your order has been placed."); err != nil {
			// Best-effort: a notification failure never fails a checkout
			// that already placed the order.
			slog.Error("checkout: notification failed", "orderId", orderID, "error", err)
		}
	}

	return gen.Checkout201JSONResponse(order), nil
}

// releaseAll gives back every reservation already held for a checkout that is
// about to be refused, so a later line item's refusal never leaks a held
// reservation. Best-effort: a release failure is logged, not surfaced — the
// caller already sees the refusal that triggered it.
func (s *Server) releaseAll(ctx context.Context, reservationIDs []string) {
	for _, id := range reservationIDs {
		if err := s.Inventory.Release(ctx, id); err != nil {
			slog.Error("checkout: release reservation failed", "reservationId", id, "error", err)
		}
	}
}
