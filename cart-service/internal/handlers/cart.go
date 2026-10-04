// Package handlers implements cart-service's generated StrictServerInterface.
// Every /me/... operation resolves the caller from the verified gateway
// assertion and reaches only that caller's own cart — never a header, never a
// client-supplied id.
package handlers

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"

	"cart-service/internal/auth"
	"cart-service/internal/gen"
	"cart-service/internal/models"
	"cart-service/internal/store"
)

// Server implements gen.StrictServerInterface against an in-memory Store.
type Server struct {
	store *store.Store
}

// New returns a Server backed by a fresh in-memory store.
func New(s *store.Store) *Server {
	return &Server{store: s}
}

// Router wires the verifier middleware and the generated strict handler
// exactly once, so main.go and the test suite can never drift apart on how a
// request is turned into a response.
func Router(verifier *auth.Verifier, srv *Server) http.Handler {
	r := chi.NewRouter()
	r.Use(verifier.Middleware)

	strict := gen.NewStrictHandlerWithOptions(srv, nil, gen.StrictHTTPServerOptions{
		RequestErrorHandlerFunc: writeError(http.StatusBadRequest),
		// GetMyCart is the one operation whose openapi.yaml carries no 401
		// response variant (a design gap, flagged on issue #5), so an
		// unauthenticated caller there surfaces as a plain error from the
		// handler rather than a typed response object. Every other
		// in-service error is a genuine 500.
		ResponseErrorHandlerFunc: func(w http.ResponseWriter, r *http.Request, err error) {
			if errors.Is(err, auth.ErrNoCaller) {
				writeError(http.StatusUnauthorized)(w, r, err)
				return
			}
			writeError(http.StatusInternalServerError)(w, r, err)
		},
	})

	return gen.HandlerWithOptions(strict, gen.ChiServerOptions{BaseRouter: r})
}

func writeError(status int) func(w http.ResponseWriter, r *http.Request, err error) {
	return func(w http.ResponseWriter, _ *http.Request, _ error) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(gen.Error{Code: status, Message: http.StatusText(status)})
	}
}

func toAPICart(c models.Cart) gen.Cart {
	items := make([]gen.CartItem, 0, len(c.Items))
	for _, it := range c.Items {
		items = append(items, gen.CartItem{BookID: it.BookID, Quantity: it.Quantity})
	}
	return gen.Cart{
		ID:        c.ID,
		Items:     items,
		Subtotal:  float32(c.Subtotal()),
		UpdatedAt: c.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}

func apiError(code int, message string) gen.Error {
	return gen.Error{Code: code, Message: message}
}

// GetHealth is public: it reads no identity.
func (s *Server) GetHealth(_ context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200Response{}, nil
}

// GetMyCart returns the caller's own cart, per the `sub` on the verified
// gateway assertion. openapi.yaml declares no 401 response for this
// operation (a design gap — the gateway's own 401 covers every ordinary
// case), so a missing caller here is surfaced through the strict handler's
// ResponseErrorHandlerFunc (wired in main.go) rather than a typed response
// variant that does not exist.
func (s *Server) GetMyCart(ctx context.Context, _ gen.GetMyCartRequestObject) (gen.GetMyCartResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return nil, err
	}
	return gen.GetMyCart200JSONResponse(toAPICart(s.store.Get(caller.UserID))), nil
}

// ClearMyCart empties the caller's own cart.
func (s *Server) ClearMyCart(ctx context.Context, _ gen.ClearMyCartRequestObject) (gen.ClearMyCartResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return gen.ClearMyCart401JSONResponse(apiError(401, "not signed in")), nil
	}
	s.store.Clear(caller.UserID)
	return gen.ClearMyCart204Response{}, nil
}

// AddMyCartItem adds a book to the caller's own cart.
func (s *Server) AddMyCartItem(ctx context.Context, request gen.AddMyCartItemRequestObject) (gen.AddMyCartItemResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return gen.AddMyCartItem401JSONResponse(apiError(401, "not signed in")), nil
	}
	if request.Body == nil || request.Body.BookID == "" {
		return gen.AddMyCartItem400JSONResponse(apiError(400, "bookId is required")), nil
	}
	if request.Body.Quantity < 1 {
		return gen.AddMyCartItem400JSONResponse(apiError(400, "quantity must be at least 1")), nil
	}
	cart := s.store.AddItem(caller.UserID, request.Body.BookID, request.Body.Quantity)
	return gen.AddMyCartItem200JSONResponse(toAPICart(cart)), nil
}

// UpdateMyCartItem changes the quantity of a book already in the caller's own
// cart.
func (s *Server) UpdateMyCartItem(ctx context.Context, request gen.UpdateMyCartItemRequestObject) (gen.UpdateMyCartItemResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return gen.UpdateMyCartItem401JSONResponse(apiError(401, "not signed in")), nil
	}
	if request.Body == nil || request.Body.Quantity < 1 {
		return gen.UpdateMyCartItem400JSONResponse(apiError(400, "quantity must be at least 1")), nil
	}
	cart, err := s.store.UpdateItem(caller.UserID, request.BookID, request.Body.Quantity)
	if errors.Is(err, store.ErrItemNotFound) {
		return gen.UpdateMyCartItem404JSONResponse(apiError(404, "no such item in your cart")), nil
	}
	if err != nil {
		return nil, err
	}
	return gen.UpdateMyCartItem200JSONResponse(toAPICart(cart)), nil
}

// RemoveMyCartItem removes a book from the caller's own cart.
func (s *Server) RemoveMyCartItem(ctx context.Context, request gen.RemoveMyCartItemRequestObject) (gen.RemoveMyCartItemResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return gen.RemoveMyCartItem401JSONResponse(apiError(401, "not signed in")), nil
	}
	cart, err := s.store.RemoveItem(caller.UserID, request.BookID)
	if errors.Is(err, store.ErrItemNotFound) {
		return gen.RemoveMyCartItem404JSONResponse(apiError(404, "no such item in your cart")), nil
	}
	if err != nil {
		return nil, err
	}
	return gen.RemoveMyCartItem200JSONResponse(toAPICart(cart)), nil
}
