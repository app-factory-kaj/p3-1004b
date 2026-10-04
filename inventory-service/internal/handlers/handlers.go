// Package handlers implements inventory-service's StrictServerInterface
// against the in-memory store. Reach (which rows an operation returns) is
// fixed by its path per openapi-conventions: every operation here reaches
// every row — there is no `/me/...` operation in this contract — and the
// gateway has already enforced each operation's scope before a request
// reaches this process, so no handler holds a scope check of its own.
package handlers

import (
	"context"

	"inventory-service/internal/gen"
	"inventory-service/internal/store"
)

const (
	defaultLimit = 20
	maxLimit     = 100
)

// Server implements gen.StrictServerInterface.
type Server struct {
	store *store.Store
}

// New returns a Server backed by st.
func New(st *store.Store) *Server {
	return &Server{store: st}
}

func normalizePage(limit, offset int) (int, int) {
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	if offset < 0 {
		offset = 0
	}
	return limit, offset
}

// GetHealth is a public liveness check (security: []).
func (s *Server) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200Response{}, nil
}

// ListStockLevels is a public read (security: []) — every book's stock level.
func (s *Server) ListStockLevels(_ context.Context, request gen.ListStockLevelsRequestObject) (gen.ListStockLevelsResponseObject, error) {
	limit, offset := normalizePage(request.Params.Limit, request.Params.Offset)
	levels, total := s.store.ListStockLevels(limit, offset)

	data := make([]gen.StockLevel, 0, len(levels))
	for _, sl := range levels {
		data = append(data, toStockLevel(sl))
	}
	return gen.ListStockLevels200JSONResponse{
		Count: total,
		Data:  data,
	}, nil
}

// GetStockLevel is a public read (security: []) — one book's stock level.
func (s *Server) GetStockLevel(_ context.Context, request gen.GetStockLevelRequestObject) (gen.GetStockLevelResponseObject, error) {
	sl, err := s.store.GetStockLevel(request.BookID)
	if err != nil {
		return gen.GetStockLevel404JSONResponse{
			Code:    404,
			Message: "no stock record for this book",
		}, nil
	}
	return gen.GetStockLevel200JSONResponse(toStockLevel(sl)), nil
}

// SetStockLevel adjusts a book's on-hand quantity. Restricted to
// stock:manage at the gateway; this handler holds no copy of that check.
func (s *Server) SetStockLevel(_ context.Context, request gen.SetStockLevelRequestObject) (gen.SetStockLevelResponseObject, error) {
	if request.Body == nil {
		return gen.SetStockLevel400JSONResponse{
			Code:    400,
			Message: "quantityOnHand is required",
		}, nil
	}
	sl, err := s.store.SetStockLevel(request.BookID, request.Body.QuantityOnHand)
	if err != nil {
		return gen.SetStockLevel400JSONResponse{
			Code:    400,
			Message: "quantityOnHand must be zero or greater",
		}, nil
	}
	return gen.SetStockLevel200JSONResponse(toStockLevel(sl)), nil
}

// ListReservations is restricted to stock:view-reservations at the gateway —
// every current reservation, every row, since this operation's path is not
// under /me/.
func (s *Server) ListReservations(_ context.Context, request gen.ListReservationsRequestObject) (gen.ListReservationsResponseObject, error) {
	limit, offset := normalizePage(request.Params.Limit, request.Params.Offset)
	reservations, total := s.store.ListReservations(limit, offset)

	data := make([]gen.Reservation, 0, len(reservations))
	for _, r := range reservations {
		data = append(data, toReservation(r))
	}
	return gen.ListReservations200JSONResponse{
		Count: total,
		Data:  data,
	}, nil
}

// ReserveStock holds stock for an order being checked out. Restricted to
// stock:reserve at the gateway — called by orders-service during checkout. A
// reservation that would exceed on-hand quantity (on-hand minus already
// reserved) is refused with 400, per the acceptance criteria.
func (s *Server) ReserveStock(_ context.Context, request gen.ReserveStockRequestObject) (gen.ReserveStockResponseObject, error) {
	if request.Body == nil {
		return gen.ReserveStock400JSONResponse{
			Code:    400,
			Message: "bookId, orderId and quantity are required",
		}, nil
	}
	body := request.Body
	if body.BookID == "" || body.OrderID == "" || body.Quantity < 1 {
		return gen.ReserveStock400JSONResponse{
			Code:    400,
			Message: "bookId, orderId and a positive quantity are required",
		}, nil
	}

	res, err := s.store.Reserve(body.BookID, body.OrderID, body.Quantity)
	if err != nil {
		return gen.ReserveStock400JSONResponse{
			Code:    400,
			Message: "insufficient stock for this book",
		}, nil
	}
	return gen.ReserveStock201JSONResponse(toReservation(res)), nil
}

// ReleaseReservation releases held stock back into the pool. Restricted to
// stock:reserve at the gateway, same as ReserveStock — orders-service calls
// this to release a reservation an order no longer needs (e.g. a cancelled
// checkout).
func (s *Server) ReleaseReservation(_ context.Context, request gen.ReleaseReservationRequestObject) (gen.ReleaseReservationResponseObject, error) {
	res, err := s.store.Release(request.ReservationID)
	if err != nil {
		return gen.ReleaseReservation404JSONResponse{
			Code:    404,
			Message: "no such reservation",
		}, nil
	}
	return gen.ReleaseReservation200JSONResponse(toReservation(res)), nil
}

func toStockLevel(sl store.StockLevel) gen.StockLevel {
	return gen.StockLevel{
		BookID:           sl.BookID,
		QuantityOnHand:   sl.QuantityOnHand,
		QuantityReserved: sl.QuantityReserved,
	}
}

func toReservation(r store.Reservation) gen.Reservation {
	return gen.Reservation{
		ID:       r.ID,
		BookID:   r.BookID,
		OrderID:  r.OrderID,
		Quantity: r.Quantity,
		Status:   gen.ReservationStatus(r.Status),
	}
}
