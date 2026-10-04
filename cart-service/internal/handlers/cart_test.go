package handlers_test

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"cart-service/internal/auth"
	"cart-service/internal/gen"
	"cart-service/internal/handlers"
	"cart-service/internal/store"
)

const (
	testIssuer = "test-gateway"
	testHeader = "x-jwt-assertion"
)

// --- test fixtures: a throwaway RSA keypair, a self-signed certificate for
// it, and a minimal JWT assertion minter that mirrors what the gateway signs.

func genKeyAndCert(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-gateway"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return key, certPEM
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

// mintAssertion signs a gateway-shaped assertion with key, the way the real
// gateway signs the caller it authenticated.
func mintAssertion(t *testing.T, key *rsa.PrivateKey, issuer, sub, scope string, exp time.Time) string {
	t.Helper()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   issuer,
		"sub":   sub,
		"scope": scope,
		"exp":   exp.Unix(),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	signingInput := b64url(headerJSON) + "." + b64url(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + b64url(sig)
}

// tamperPayload edits the token's payload segment after signing, so its
// signature no longer verifies — the "edited after signing" forged case.
func tamperPayload(t *testing.T, token string) string {
	t.Helper()
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		t.Fatalf("not a three-part JWT: %q", token)
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	claims["sub"] = "someone-else"
	edited, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal edited payload: %v", err)
	}
	return parts[0] + "." + b64url(edited) + "." + parts[2]
}

// testServer wires the real router — generated server, strict handler and
// gateway-assertion middleware — exactly as main.go does, against a fresh
// in-memory store.
type testServer struct {
	handler http.Handler
	key     *rsa.PrivateKey
}

func newTestServer(t *testing.T) testServer {
	t.Helper()
	key, certPEM := genKeyAndCert(t)
	t.Setenv("GATEWAY_ASSERTION_CERTIFICATE", certPEM)
	t.Setenv("GATEWAY_ASSERTION_ISSUER", testIssuer)
	t.Setenv("GATEWAY_ASSERTION_HEADER", testHeader)

	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		t.Fatalf("NewVerifierFromEnv: %v", err)
	}

	srv := handlers.New(store.New())
	h := handlers.Router(verifier, srv)
	return testServer{handler: h, key: key}
}

func (ts testServer) do(t *testing.T, method, path string, assertion string, body string, extraHeaders map[string]string) *httptest.ResponseRecorder {
	t.Helper()
	var reader *strings.Reader
	if body != "" {
		reader = strings.NewReader(body)
	} else {
		reader = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, reader)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if assertion != "" {
		req.Header.Set(testHeader, assertion)
	}
	for k, v := range extraHeaders {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	ts.handler.ServeHTTP(rec, req)
	return rec
}

// TestForgedAssertionIs401 proves the trust anchor is the signature, not the
// header: a request signed by a different key, and one whose payload was
// edited after signing with the right key, are both rejected — never
// downgraded to "anonymous".
func TestForgedAssertionIs401(t *testing.T) {
	ts := newTestServer(t)

	otherKey, _ := genKeyAndCert(t)
	wrongKeyToken := mintAssertion(t, otherKey, testIssuer, "attacker", "cart:read", time.Now().Add(time.Minute))
	rec := ts.do(t, http.MethodGet, "/me/cart", wrongKeyToken, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-key assertion: got status %d, want 401", rec.Code)
	}

	validToken := mintAssertion(t, ts.key, testIssuer, "marco", "cart:read", time.Now().Add(time.Minute))
	tampered := tamperPayload(t, validToken)
	rec = ts.do(t, http.MethodGet, "/me/cart", tampered, "", nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("tampered-payload assertion: got status %d, want 401", rec.Code)
	}
}

// TestHealthIsPublic proves the security:[] operation serves anonymously and
// reads no identity: it answers 200 with no assertion at all, and 200 with
// forged X-User-* headers that the handler must not trust or read.
func TestHealthIsPublic(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, http.MethodGet, "/health", "", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("no assertion: got status %d, want 200", rec.Code)
	}

	rec = ts.do(t, http.MethodGet, "/health", "", "", map[string]string{
		"x-user-id":     "forged-user",
		"x-user-scopes": "cart:read cart:manage",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("forged X-User-* headers: got status %d, want 200", rec.Code)
	}
}

// TestNoAssertionIs401OnEveryProtectedRoute proves an unauthenticated caller
// (no assertion at all, as happens if a request reaches this service without
// going through the gateway) is refused on every route except /health.
func TestNoAssertionIs401OnEveryProtectedRoute(t *testing.T) {
	ts := newTestServer(t)

	cases := []struct {
		method, path, body string
	}{
		{http.MethodGet, "/me/cart", ""},
		{http.MethodDelete, "/me/cart", ""},
		{http.MethodPost, "/me/cart/items", `{"bookId":"book-1","quantity":1}`},
		{http.MethodPut, "/me/cart/items/book-1", `{"quantity":1}`},
		{http.MethodDelete, "/me/cart/items/book-1", ""},
	}
	for _, c := range cases {
		rec := ts.do(t, c.method, c.path, "", c.body, nil)
		if rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s with no assertion: got status %d, want 401, body=%s", c.method, c.path, rec.Code, rec.Body.String())
		}
	}
}

// TestCartOwnershipAndWidening proves one Shopper's cart is never visible to
// another: Marco and Elena, two different callers resolved from their own
// verified assertions, each see only their own cart contents — adding to
// Marco's cart never appears in Elena's, and Elena starts (and stays) empty.
func TestCartOwnershipAndWidening(t *testing.T) {
	ts := newTestServer(t)
	marco := mintAssertion(t, ts.key, testIssuer, "marco", "cart:manage cart:read", time.Now().Add(time.Minute))
	elena := mintAssertion(t, ts.key, testIssuer, "elena", "cart:manage cart:read", time.Now().Add(time.Minute))

	rec := ts.do(t, http.MethodPost, "/me/cart/items", marco, `{"bookId":"clean-architecture","quantity":2}`, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("marco adds item: got status %d, want 200, body=%s", rec.Code, rec.Body.String())
	}

	rec = ts.do(t, http.MethodGet, "/me/cart", marco, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("marco views cart: got status %d, want 200", rec.Code)
	}
	var marcoCart gen.Cart
	if err := json.Unmarshal(rec.Body.Bytes(), &marcoCart); err != nil {
		t.Fatalf("decode marco cart: %v", err)
	}
	if len(marcoCart.Items) != 1 || marcoCart.Items[0].BookID != "clean-architecture" || marcoCart.Items[0].Quantity != 2 {
		t.Fatalf("marco's cart: got %+v, want one line of clean-architecture x2", marcoCart.Items)
	}

	rec = ts.do(t, http.MethodGet, "/me/cart", elena, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("elena views cart: got status %d, want 200", rec.Code)
	}
	var elenaCart gen.Cart
	if err := json.Unmarshal(rec.Body.Bytes(), &elenaCart); err != nil {
		t.Fatalf("decode elena cart: %v", err)
	}
	if len(elenaCart.Items) != 0 {
		t.Fatalf("elena's cart: got %+v, want empty — Marco's item must not widen into it", elenaCart.Items)
	}
	if marcoCart.ID == elenaCart.ID {
		t.Fatalf("marco and elena resolved to the same cart id %q", marcoCart.ID)
	}

	// Elena removing a book that was only ever in Marco's cart finds nothing
	// of her own to remove: 404, not a peek into Marco's cart.
	rec = ts.do(t, http.MethodDelete, "/me/cart/items/clean-architecture", elena, "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("elena removes marco's item: got status %d, want 404", rec.Code)
	}

	// Marco's cart is unaffected by anything Elena did.
	rec = ts.do(t, http.MethodGet, "/me/cart", marco, "", nil)
	var marcoCartAfter gen.Cart
	if err := json.Unmarshal(rec.Body.Bytes(), &marcoCartAfter); err != nil {
		t.Fatalf("decode marco cart after: %v", err)
	}
	if len(marcoCartAfter.Items) != 1 {
		t.Fatalf("marco's cart after elena's calls: got %+v, want still one item", marcoCartAfter.Items)
	}
}

// TestCartSubtotalCorrectness proves the subtotal reported alongside the
// cart's items stays arithmetically correct through an add, a quantity
// change and a removal.
func TestCartSubtotalCorrectness(t *testing.T) {
	ts := newTestServer(t)
	token := mintAssertion(t, ts.key, testIssuer, "marco", "cart:manage cart:read", time.Now().Add(time.Minute))

	rec := ts.do(t, http.MethodPost, "/me/cart/items", token, `{"bookId":"clean-architecture","quantity":2}`, nil)
	var cart gen.Cart
	mustDecode(t, rec, &cart)
	if cart.Subtotal != 2 {
		t.Fatalf("after adding 2: got subtotal %v, want 2", cart.Subtotal)
	}

	rec = ts.do(t, http.MethodPost, "/me/cart/items", token, `{"bookId":"domain-driven-design","quantity":3}`, nil)
	mustDecode(t, rec, &cart)
	if cart.Subtotal != 5 {
		t.Fatalf("after adding a second line of 3: got subtotal %v, want 5", cart.Subtotal)
	}

	rec = ts.do(t, http.MethodPut, "/me/cart/items/clean-architecture", token, `{"quantity":5}`, nil)
	mustDecode(t, rec, &cart)
	if cart.Subtotal != 8 {
		t.Fatalf("after updating first line to 5 (5+3): got subtotal %v, want 8", cart.Subtotal)
	}

	rec = ts.do(t, http.MethodDelete, "/me/cart/items/domain-driven-design", token, "", nil)
	mustDecode(t, rec, &cart)
	if cart.Subtotal != 5 {
		t.Fatalf("after removing the second line: got subtotal %v, want 5", cart.Subtotal)
	}

	rec = ts.do(t, http.MethodDelete, "/me/cart", token, "", nil)
	if rec.Code != http.StatusNoContent {
		t.Fatalf("clear cart: got status %d, want 204", rec.Code)
	}
	rec = ts.do(t, http.MethodGet, "/me/cart", token, "", nil)
	mustDecode(t, rec, &cart)
	if cart.Subtotal != 0 || len(cart.Items) != 0 {
		t.Fatalf("after clearing: got subtotal %v items %+v, want 0 and empty", cart.Subtotal, cart.Items)
	}
}

// TestAddAndUpdateValidation proves quantity and bookId validation answers
// 400, and an update/removal against a book not in the caller's cart answers
// 404.
func TestAddAndUpdateValidation(t *testing.T) {
	ts := newTestServer(t)
	token := mintAssertion(t, ts.key, testIssuer, "marco", "cart:manage cart:read", time.Now().Add(time.Minute))

	rec := ts.do(t, http.MethodPost, "/me/cart/items", token, `{"bookId":"","quantity":1}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("empty bookId: got status %d, want 400", rec.Code)
	}

	rec = ts.do(t, http.MethodPost, "/me/cart/items", token, `{"bookId":"book-1","quantity":0}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("zero quantity: got status %d, want 400", rec.Code)
	}

	rec = ts.do(t, http.MethodPut, "/me/cart/items/does-not-exist", token, `{"quantity":1}`, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("update unknown item: got status %d, want 404", rec.Code)
	}

	rec = ts.do(t, http.MethodDelete, "/me/cart/items/does-not-exist", token, "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("remove unknown item: got status %d, want 404", rec.Code)
	}
}

func mustDecode(t *testing.T, rec *httptest.ResponseRecorder, v any) {
	t.Helper()
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("got status %d, want 2xx, body=%s", rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), v); err != nil {
		t.Fatalf("decode response: %v, body=%s", err, rec.Body.String())
	}
}
