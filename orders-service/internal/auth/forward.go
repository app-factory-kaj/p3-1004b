// Package auth: this file is NOT part of the copied gateway_assertion.go
// asset. gateway_assertion.go verifies who called THIS service; this file
// lets a handler forward that SAME, already-signed identity to a sibling
// service it calls on the caller's behalf, without ever minting or decoding a
// new assertion of its own.
//
// Mechanism: cart-service, inventory-service, customers-service and
// notifications-service are reached directly over `visibility: project` —
// pod-to-pod, never through the gateway (workload-and-wiring.md, "Provider
// endpoint visibility"). Each of them runs the same gateway_assertion verifier
// this service does, so the only way they learn who the caller is is the same
// signed assertion header the gateway attached to THIS inbound request. We
// capture its raw bytes and copy them onto the outgoing request, unmodified —
// the receiving sibling verifies the signature itself with the same shared
// certificate.
package auth

import (
	"context"
	"net/http"
	"os"
	"strings"
)

type rawAssertionCtxKey struct{}

// AssertionHeaderName is the header this environment's gateway signs the
// caller's assertion into. Mirrors NewVerifierFromEnv's own selection
// (GATEWAY_ASSERTION_HEADER, falling back to USER_TOKEN_HEADER or
// "x-forwarded-authorization" in the temporary unverified mode) so a
// forwarded call always lands on the header name the sibling's own verifier
// actually reads.
func AssertionHeaderName() string {
	if h := strings.TrimSpace(os.Getenv("GATEWAY_ASSERTION_HEADER")); h != "" {
		return h
	}
	if h := strings.TrimSpace(os.Getenv("USER_TOKEN_HEADER")); h != "" {
		return h
	}
	return "x-forwarded-authorization"
}

// CaptureRawAssertion stashes the raw, still-signed assertion header value
// (if any) on the request context, so a handler several calls deep can
// forward the same bytes to a sibling service. It does not decode or verify
// anything itself — Verifier.Middleware already did that for this request.
func CaptureRawAssertion(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw := strings.TrimSpace(r.Header.Get(AssertionHeaderName()))
		ctx := r.Context()
		if raw != "" {
			ctx = context.WithValue(ctx, rawAssertionCtxKey{}, raw)
		}
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

// RawAssertionFrom returns the raw assertion header value captured for this
// request, if any.
func RawAssertionFrom(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(rawAssertionCtxKey{}).(string)
	return v, ok
}

// ForwardAssertion is a request-editor callback, compatible with every
// generated sibling client's RequestEditorFn: it copies the caller's captured
// assertion onto the outgoing request so the sibling's own verifier can
// identify the SAME caller. It mints nothing and reads nothing from the
// response.
func ForwardAssertion(ctx context.Context, req *http.Request) error {
	if raw, ok := RawAssertionFrom(ctx); ok && raw != "" {
		req.Header.Set(AssertionHeaderName(), raw)
	}
	return nil
}
