// Package handlers implements notifications-service's operations against the
// generated StrictServerInterface.
//
// Reach is the path, not a scope check: the gateway has already enforced each
// operation's declared scope (notifications:read / notifications:create)
// before a request reaches this process, so no handler here holds a copy of
// that table. What a handler DOES enforce is which rows an operation may
// reach: every /me/* handler resolves through auth.RequireCaller's sub, never
// through a client-supplied id.
package handlers

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"strings"

	"notifications-service/internal/auth"
	"notifications-service/internal/gen"
	"notifications-service/internal/store"
)

// unauthorizedResponse answers 401 with the shared Error schema for an
// operation whose openapi.yaml declares no 401 variant of its own (the
// gateway is expected to have refused an unauthenticated request before it
// ever reaches here; this is the defense-in-depth path for a request that
// somehow arrived with no verifiable assertion at all).
type unauthorizedResponse struct {
	body gen.Error
}

func (r unauthorizedResponse) write(w http.ResponseWriter) error {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(r.body); err != nil {
		return err
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnauthorized)
	_, err := buf.WriteTo(w)
	return err
}

type listMyNotificationsUnauthorized struct{ unauthorizedResponse }

func (r listMyNotificationsUnauthorized) VisitListMyNotificationsResponse(w http.ResponseWriter) error {
	return r.write(w)
}

// Server implements gen.StrictServerInterface.
type Server struct {
	store *store.Store
}

// New returns a Server backed by st.
func New(st *store.Store) *Server {
	return &Server{store: st}
}

// GetHealth is public (security: []) and reads no identity.
func (s *Server) GetHealth(ctx context.Context, _ gen.GetHealthRequestObject) (gen.GetHealthResponseObject, error) {
	return gen.GetHealth200Response{}, nil
}

// ListMyNotifications reaches only the signed-in caller's own notifications,
// resolved through the verified gateway assertion's sub — never a client
// supplied id.
func (s *Server) ListMyNotifications(ctx context.Context, request gen.ListMyNotificationsRequestObject) (gen.ListMyNotificationsResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return listMyNotificationsUnauthorized{unauthorizedResponse{gen.Error{
			Code:    401,
			Message: "not signed in",
		}}}, nil
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

	records, total := s.store.ListByCustomer(caller.UserID, limit, offset)
	data := make([]gen.Notification, 0, len(records))
	for _, rec := range records {
		data = append(data, rec.ToModel())
	}

	return gen.ListMyNotifications200JSONResponse{
		Count: total,
		Data:  data,
	}, nil
}

// MarkMyNotificationRead marks one of the CALLER's own notifications read.
// A notification that exists but belongs to someone else is indistinguishable
// from one that does not exist at all: both answer 404, never 403 — a caller
// who may not see a row may not learn it exists either.
func (s *Server) MarkMyNotificationRead(ctx context.Context, request gen.MarkMyNotificationReadRequestObject) (gen.MarkMyNotificationReadResponseObject, error) {
	caller, err := auth.RequireCaller(ctx)
	if err != nil {
		return gen.MarkMyNotificationRead401JSONResponse{
			Code:    401,
			Message: "not signed in",
		}, nil
	}

	rec, ok := s.store.MarkRead(caller.UserID, request.NotificationID)
	if !ok {
		return gen.MarkMyNotificationRead404JSONResponse{
			Code:    404,
			Message: "no such notification for the caller",
		}, nil
	}

	return gen.MarkMyNotificationRead200JSONResponse(rec.ToModel()), nil
}

// CreateNotification is called by orders-service (restricted to
// notifications:create at the gateway) to create a notification for ANY
// customer named in the body — the caller here is orders-service, not the
// recipient, so this is the one operation outside /me/ and it reaches every
// customer's collection, filtered on nothing.
func (s *Server) CreateNotification(ctx context.Context, request gen.CreateNotificationRequestObject) (gen.CreateNotificationResponseObject, error) {
	if request.Body == nil {
		return gen.CreateNotification400JSONResponse{
			Code:    400,
			Message: "request body required",
		}, nil
	}

	customerID := strings.TrimSpace(request.Body.CustomerID)
	orderID := strings.TrimSpace(request.Body.OrderID)
	message := strings.TrimSpace(request.Body.Message)
	if customerID == "" || orderID == "" || message == "" {
		return gen.CreateNotification400JSONResponse{
			Code:    400,
			Message: "customerId, orderId and message are all required",
		}, nil
	}

	rec := s.store.Create(customerID, orderID, message)
	return gen.CreateNotification201JSONResponse(rec.ToModel()), nil
}
