package handlers

import (
	"encoding/json"
	"net/http"

	"orders-service/internal/gen"
)

// writeJSON writes an Error body with the given status. It backs the two
// 401 cases below: the spec's own default security requires a signed-in
// caller on every operation but /health, so the gateway always forwards an
// assertion here in a real deployment — this exists for the handler's own
// "no caller" rule (`go` skill) and for a test that exercises it directly.
func writeJSON(w http.ResponseWriter, status int, body gen.Error) error {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	return json.NewEncoder(w).Encode(body)
}

// listMyOrders401JSONResponse answers ListMyOrders when the request carries
// no verified caller.
type listMyOrders401JSONResponse gen.Error

func (r listMyOrders401JSONResponse) VisitListMyOrdersResponse(w http.ResponseWriter) error {
	return writeJSON(w, http.StatusUnauthorized, gen.Error(r))
}

// getMyOrder401JSONResponse answers GetMyOrder when the request carries no
// verified caller.
type getMyOrder401JSONResponse gen.Error

func (r getMyOrder401JSONResponse) VisitGetMyOrderResponse(w http.ResponseWriter) error {
	return writeJSON(w, http.StatusUnauthorized, gen.Error(r))
}
