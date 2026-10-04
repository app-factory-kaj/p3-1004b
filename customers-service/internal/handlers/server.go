package handlers

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"

	"customers-service/internal/auth"
	"customers-service/internal/gen"
	"customers-service/internal/store"
)

// NewHandler wires the generated chi server, the gateway-assertion verifier
// middleware, and this service's error mapping into one http.Handler. main.go
// and the handler tests both call this, so the two can never drift.
func NewHandler(s *store.Store, verifier *auth.Verifier) http.Handler {
	srv := New(s)

	r := chi.NewRouter()
	r.Use(verifier.Middleware)

	return gen.HandlerWithOptions(
		gen.NewStrictHandlerWithOptions(srv, nil, gen.StrictHTTPServerOptions{
			RequestErrorHandlerFunc:  writeRequestError,
			ResponseErrorHandlerFunc: writeResponseError,
		}),
		gen.ChiServerOptions{BaseRouter: r},
	)
}

// writeRequestError answers a request oapi-codegen could not even decode
// (malformed JSON, a bad query param) with the shared Error schema rather
// than the library's default plain-text body.
func writeRequestError(w http.ResponseWriter, r *http.Request, err error) {
	writeError(w, http.StatusBadRequest, err.Error())
}

// writeResponseError is reached only when a handler returns a non-nil error
// instead of one of its generated response types. The one error every
// handler in this service can return is auth.ErrNoCaller — a request that
// reached a non-public operation with no verified identity on it — which is
// a 401, not the library's default 500. Some operations in openapi.yaml do
// not declare a 401 response (GET /me/profile, GET /me/addresses) because the
// gateway is expected to refuse an unauthenticated caller before this service
// ever sees the request; this is the defensive fallback for the case where
// that did not happen.
func writeResponseError(w http.ResponseWriter, r *http.Request, err error) {
	if errors.Is(err, auth.ErrNoCaller) {
		writeError(w, http.StatusUnauthorized, "not signed in")
		return
	}
	log.Printf("unhandled error: %v", err)
	writeError(w, http.StatusInternalServerError, "internal error")
}

func writeError(w http.ResponseWriter, code int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(gen.Error{Code: code, Message: message})
}
