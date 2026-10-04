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

	"github.com/go-chi/chi/v5"

	"inventory-service/internal/auth"
	"inventory-service/internal/gen"
	"inventory-service/internal/handlers"
	"inventory-service/internal/store"
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
	r := chi.NewRouter()
	r.Use(verifier.Middleware)
	h := gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{BaseRouter: r})
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
	wrongKeyToken := mintAssertion(t, otherKey, testIssuer, "attacker", "stock:reserve", time.Now().Add(time.Minute))
	rec := ts.do(t, http.MethodPost, "/reservations", wrongKeyToken,
		`{"bookId":"book-001","orderId":"order-1","quantity":1}`, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("wrong-key assertion: got status %d, want 401", rec.Code)
	}

	validToken := mintAssertion(t, ts.key, testIssuer, "caller-1", "stock:reserve", time.Now().Add(time.Minute))
	tampered := tamperPayload(t, validToken)
	rec = ts.do(t, http.MethodPost, "/reservations", tampered,
		`{"bookId":"book-001","orderId":"order-1","quantity":1}`, nil)
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
		"x-user-scopes": "stock:manage stock:reserve stock:view-reservations",
	})
	if rec.Code != http.StatusOK {
		t.Fatalf("forged X-User-* headers: got status %d, want 200", rec.Code)
	}
}

// TestStockLevelsArePublicReads proves both read operations serve anonymous
// callers (security: []), as the storefront needs for availability display.
func TestStockLevelsArePublicReads(t *testing.T) {
	ts := newTestServer(t)

	rec := ts.do(t, http.MethodGet, "/stock-levels", "", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("list stock levels, no assertion: got status %d, want 200", rec.Code)
	}
	var list gen.ListStockLevels200JSONResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list response: %v", err)
	}
	if list.Count == 0 || len(list.Data) == 0 {
		t.Fatalf("expected seeded stock levels, got count=%d data=%v", list.Count, list.Data)
	}

	rec = ts.do(t, http.MethodGet, "/stock-levels/book-001", "", "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("get stock level, no assertion: got status %d, want 200", rec.Code)
	}
	var sl gen.StockLevel
	if err := json.Unmarshal(rec.Body.Bytes(), &sl); err != nil {
		t.Fatalf("decode stock level: %v", err)
	}
	if sl.BookID != "book-001" {
		t.Fatalf("got bookId %q, want book-001", sl.BookID)
	}

	rec = ts.do(t, http.MethodGet, "/stock-levels/does-not-exist", "", "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("get unknown stock level: got status %d, want 404", rec.Code)
	}
}

// TestReserveExceedingOnHandIsRefused proves a reservation that would exceed
// on-hand quantity (on-hand minus already-reserved) is refused with 400, per
// the acceptance criteria — not served partially and not panicking.
func TestReserveExceedingOnHandIsRefused(t *testing.T) {
	ts := newTestServer(t)
	token := mintAssertion(t, ts.key, testIssuer, "orders-service", "stock:reserve", time.Now().Add(time.Minute))

	// book-002 is seeded with on-hand 7; ask for more than that.
	rec := ts.do(t, http.MethodPost, "/reservations", token,
		`{"bookId":"book-002","orderId":"order-1","quantity":8}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("over-reservation: got status %d, want 400, body=%s", rec.Code, rec.Body.String())
	}

	// A reservation within the on-hand quantity still succeeds.
	rec = ts.do(t, http.MethodPost, "/reservations", token,
		`{"bookId":"book-002","orderId":"order-2","quantity":7}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("exact on-hand reservation: got status %d, want 201, body=%s", rec.Code, rec.Body.String())
	}

	// Now every unit of book-002 is reserved: even quantity 1 is refused.
	rec = ts.do(t, http.MethodPost, "/reservations", token,
		`{"bookId":"book-002","orderId":"order-3","quantity":1}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reservation against fully-reserved stock: got status %d, want 400", rec.Code)
	}
}

// TestReleaseThenReserveAgain proves a released reservation really does
// return its quantity to the available pool, rather than leaving the book
// stuck at zero availability.
func TestReleaseThenReserveAgain(t *testing.T) {
	ts := newTestServer(t)
	token := mintAssertion(t, ts.key, testIssuer, "orders-service", "stock:reserve", time.Now().Add(time.Minute))

	// book-005 is seeded with on-hand 3 — reserve all of it.
	rec := ts.do(t, http.MethodPost, "/reservations", token,
		`{"bookId":"book-005","orderId":"order-1","quantity":3}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("initial reservation: got status %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	var created gen.Reservation
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode reservation: %v", err)
	}

	// With nothing available, a second reservation is refused.
	rec = ts.do(t, http.MethodPost, "/reservations", token,
		`{"bookId":"book-005","orderId":"order-2","quantity":1}`, nil)
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("reservation against exhausted stock: got status %d, want 400", rec.Code)
	}

	// Release the first reservation, freeing its quantity back up.
	rec = ts.do(t, http.MethodPost, "/reservations/"+created.ID+"/release", token, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("release: got status %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var released gen.Reservation
	if err := json.Unmarshal(rec.Body.Bytes(), &released); err != nil {
		t.Fatalf("decode released reservation: %v", err)
	}
	if released.Status != gen.Released {
		t.Fatalf("got status %q, want released", released.Status)
	}

	// Now the freed quantity can be reserved again.
	rec = ts.do(t, http.MethodPost, "/reservations", token,
		`{"bookId":"book-005","orderId":"order-3","quantity":3}`, nil)
	if rec.Code != http.StatusCreated {
		t.Fatalf("re-reservation after release: got status %d, want 201, body=%s", rec.Code, rec.Body.String())
	}

	// Releasing an already-released reservation is idempotent, not an error.
	rec = ts.do(t, http.MethodPost, "/reservations/"+created.ID+"/release", token, "", nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("re-release of already-released reservation: got status %d, want 200", rec.Code)
	}

	// Releasing an unknown reservation id 404s.
	rec = ts.do(t, http.MethodPost, "/reservations/does-not-exist/release", token, "", nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("release of unknown reservation: got status %d, want 404", rec.Code)
	}
}
