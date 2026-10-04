package handlers_test

import (
	"bytes"
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

	"customers-service/internal/auth"
	"customers-service/internal/gen"
	"customers-service/internal/handlers"
	"customers-service/internal/store"
)

// --- test fixture: a throwaway RSA keypair, a self-signed certificate for
// it, and a JWT minter standing in for the gateway. Nothing here talks to a
// real gateway or IdP, per the go skill's testing section. ---

const (
	testIssuer = "test-gateway"
	testHeader = "x-jwt-assertion"
)

type fixture struct {
	key     *rsa.PrivateKey
	certPEM string
}

func newFixture(t *testing.T) fixture {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test-gateway"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPEM := pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return fixture{key: key, certPEM: string(certPEM)}
}

// newServer wires the real router — the generated chi server plus the
// verifier middleware — exactly as main.go does, pointed at the fixture's
// certificate. Every test drives requests through this, never the bare
// handlers.
func (f fixture) newServer(t *testing.T) http.Handler {
	t.Helper()
	t.Setenv("GATEWAY_ASSERTION_CERTIFICATE", f.certPEM)
	t.Setenv("GATEWAY_ASSERTION_ISSUER", testIssuer)
	t.Setenv("GATEWAY_ASSERTION_HEADER", testHeader)

	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		t.Fatalf("NewVerifierFromEnv: %v", err)
	}

	return handlers.NewHandler(store.New(), verifier)
}

// assertionFor mints an assertion for sub/scopes signed with signKey (usually
// f.key; a different key is how TestForgedAssertionIs401 forges one).
func assertionFor(t *testing.T, signKey *rsa.PrivateKey, sub string, scopes []string) string {
	t.Helper()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	claims := map[string]any{
		"iss":   testIssuer,
		"sub":   sub,
		"scope": strings.Join(scopes, " "),
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	signingInput := b64(headerJSON) + "." + b64(claimsJSON)
	digest := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, signKey, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + b64(sig)
}

func b64(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func doRequest(h http.Handler, method, path, assertion, body string) *httptest.ResponseRecorder {
	var r *http.Request
	if body != "" {
		r = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		r.Header.Set("Content-Type", "application/json")
	} else {
		r = httptest.NewRequest(method, path, nil)
	}
	if assertion != "" {
		r.Header.Set(testHeader, assertion)
	}
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	return w
}

// --- the four required tests ---

// TestForgedAssertionIs401 proves the trust anchor is the signature, not the
// header: an assertion signed by a different key, and one whose payload was
// edited after signing, are both rejected — never downgraded to anonymous.
func TestForgedAssertionIs401(t *testing.T) {
	f := newFixture(t)
	h := f.newServer(t)

	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	forged := assertionFor(t, otherKey, "intruder", []string{"account:read", "account:manage"})
	w := doRequest(h, http.MethodGet, "/me/profile", forged, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("assertion signed by a different key: got %d, want 401", w.Code)
	}

	// Same shape, but the payload is edited after a REAL signature was made.
	genuine := assertionFor(t, f.key, "marco", []string{"account:read", "account:manage"})
	parts := strings.Split(genuine, ".")
	tamperedClaims := map[string]any{
		"iss":   testIssuer,
		"sub":   "someone-else",
		"scope": "account:read account:manage",
		"exp":   time.Now().Add(15 * time.Minute).Unix(),
	}
	tamperedJSON, _ := json.Marshal(tamperedClaims)
	tampered := parts[0] + "." + b64(tamperedJSON) + "." + parts[2]
	w = doRequest(h, http.MethodGet, "/me/profile", tampered, "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("assertion with edited payload: got %d, want 401", w.Code)
	}

	// No assertion at all on a non-public operation is a 401 too (the
	// defensive fallback in main's ResponseErrorHandlerFunc — exercised here
	// directly since the handler always calls auth.RequireCaller).
	w = doRequest(h, http.MethodGet, "/me/profile", "", "")
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("no assertion at all: got %d, want 401", w.Code)
	}
}

// TestHealthIsPublic proves the security:[] operation answers 200 with no
// assertion, and ignores forged X-User-* headers — it reads no identity at
// all.
func TestHealthIsPublic(t *testing.T) {
	f := newFixture(t)
	h := f.newServer(t)

	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.Header.Set("X-User-Id", "someone")
	r.Header.Set("X-User-Scopes", "account:read account:manage")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health: got %d, want 200", w.Code)
	}
}

// TestProfileOwnershipAndWidening proves /me/profile resolves strictly
// through the caller's own `sub`: two callers get two distinct profiles, a
// profile auto-provisions rather than 404ing on a new caller's first access,
// and one caller's update never reaches the other's row.
func TestProfileOwnershipAndWidening(t *testing.T) {
	f := newFixture(t)
	h := f.newServer(t)
	scopes := []string{"account:read", "account:manage"}

	marco := assertionFor(t, f.key, "marco", scopes)
	elena := assertionFor(t, f.key, "elena", scopes)

	// First access for a brand-new caller: 200 with an auto-provisioned row,
	// never 404 — there is no separate signup step.
	w := doRequest(h, http.MethodGet, "/me/profile", marco, "")
	if w.Code != http.StatusOK {
		t.Fatalf("first GET /me/profile for a new caller: got %d, want 200", w.Code)
	}
	var marcoProfile gen.Customer
	mustDecode(t, w, &marcoProfile)
	if marcoProfile.ID != "marco" {
		t.Fatalf("auto-provisioned profile id = %q, want %q", marcoProfile.ID, "marco")
	}

	// Marco sets his profile.
	w = doRequest(h, http.MethodPut, "/me/profile", marco,
		`{"name":"Marco Diaz","email":"marco@example.com"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("PUT /me/profile for marco: got %d, want 200", w.Code)
	}
	mustDecode(t, w, &marcoProfile)
	if marcoProfile.Name != "Marco Diaz" || marcoProfile.Email != "marco@example.com" {
		t.Fatalf("marco's saved profile = %+v", marcoProfile)
	}

	// Elena, a different caller, never sees Marco's row.
	w = doRequest(h, http.MethodGet, "/me/profile", elena, "")
	if w.Code != http.StatusOK {
		t.Fatalf("GET /me/profile for elena: got %d, want 200", w.Code)
	}
	var elenaProfile gen.Customer
	mustDecode(t, w, &elenaProfile)
	if elenaProfile.ID == marcoProfile.ID || elenaProfile.Name == marcoProfile.Name {
		t.Fatalf("elena's profile widened into marco's: %+v", elenaProfile)
	}
	if elenaProfile.Name != "" || elenaProfile.Email != "" {
		t.Fatalf("elena's untouched profile should still be blank, got %+v", elenaProfile)
	}

	// Marco's row is unaffected by Elena's read.
	w = doRequest(h, http.MethodGet, "/me/profile", marco, "")
	mustDecode(t, w, &marcoProfile)
	if marcoProfile.Name != "Marco Diaz" {
		t.Fatalf("marco's profile changed after elena's request: %+v", marcoProfile)
	}
}

// TestAddressListOwnershipAndWidening proves GET /me/addresses returns only
// the caller's own rows: two callers each add an address, and neither sees
// the other's in their list.
func TestAddressListOwnershipAndWidening(t *testing.T) {
	f := newFixture(t)
	h := f.newServer(t)
	scopes := []string{"account:read", "account:manage"}
	marco := assertionFor(t, f.key, "marco", scopes)
	elena := assertionFor(t, f.key, "elena", scopes)

	addr := `{"line1":"221B Baker Street","city":"London","region":"London","postalCode":"NW1 6XE","country":"UK"}`
	w := doRequest(h, http.MethodPost, "/me/addresses", marco, addr)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /me/addresses for marco: got %d, want 201, body=%s", w.Code, w.Body.String())
	}

	elenaAddr := `{"line1":"10 Downing Street","city":"London","region":"London","postalCode":"SW1A 2AA","country":"UK"}`
	w = doRequest(h, http.MethodPost, "/me/addresses", elena, elenaAddr)
	if w.Code != http.StatusCreated {
		t.Fatalf("POST /me/addresses for elena: got %d, want 201", w.Code)
	}

	w = doRequest(h, http.MethodGet, "/me/addresses", marco, "")
	var marcoList struct {
		Count int           `json:"count"`
		Data  []gen.Address `json:"data"`
	}
	mustDecode(t, w, &marcoList)
	if marcoList.Count != 1 || len(marcoList.Data) != 1 {
		t.Fatalf("marco's address list = %+v, want exactly his own one row", marcoList)
	}
	if marcoList.Data[0].Line1 != "221B Baker Street" {
		t.Fatalf("marco's address list leaked another caller's row: %+v", marcoList.Data)
	}

	w = doRequest(h, http.MethodGet, "/me/addresses", elena, "")
	var elenaList struct {
		Count int           `json:"count"`
		Data  []gen.Address `json:"data"`
	}
	mustDecode(t, w, &elenaList)
	if elenaList.Count != 1 || len(elenaList.Data) != 1 {
		t.Fatalf("elena's address list = %+v, want exactly her own one row", elenaList)
	}
	if elenaList.Data[0].Line1 != "10 Downing Street" {
		t.Fatalf("elena's address list leaked another caller's row: %+v", elenaList.Data)
	}
}

// TestAddressGetOwnershipWidening proves a single address is reachable only
// by the caller who owns it: another caller's update or delete against that
// id is a 404, not a 403 and not a silent success — the row is simply not in
// their own collection.
func TestAddressGetOwnershipWidening(t *testing.T) {
	f := newFixture(t)
	h := f.newServer(t)
	scopes := []string{"account:read", "account:manage"}
	marco := assertionFor(t, f.key, "marco", scopes)
	elena := assertionFor(t, f.key, "elena", scopes)

	addr := `{"line1":"221B Baker Street","city":"London","region":"London","postalCode":"NW1 6XE","country":"UK"}`
	w := doRequest(h, http.MethodPost, "/me/addresses", marco, addr)
	var created gen.Address
	mustDecode(t, w, &created)

	update := `{"line1":"10 Downing Street","city":"London","region":"London","postalCode":"SW1A 2AA","country":"UK"}`
	w = doRequest(h, http.MethodPut, "/me/addresses/"+created.ID, elena, update)
	if w.Code != http.StatusNotFound {
		t.Fatalf("elena updating marco's address: got %d, want 404", w.Code)
	}

	w = doRequest(h, http.MethodDelete, "/me/addresses/"+created.ID, elena, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("elena deleting marco's address: got %d, want 404", w.Code)
	}

	// Marco's own address is untouched and still reachable by him.
	w = doRequest(h, http.MethodPut, "/me/addresses/"+created.ID, marco, update)
	if w.Code != http.StatusOK {
		t.Fatalf("marco updating his own address: got %d, want 200, body=%s", w.Code, w.Body.String())
	}

	// An id that never existed is also a 404 for its would-be owner.
	w = doRequest(h, http.MethodDelete, "/me/addresses/does-not-exist", marco, "")
	if w.Code != http.StatusNotFound {
		t.Fatalf("deleting an unknown address id: got %d, want 404", w.Code)
	}
}

func mustDecode(t *testing.T, w *httptest.ResponseRecorder, v any) {
	t.Helper()
	if err := json.Unmarshal(w.Body.Bytes(), v); err != nil {
		t.Fatalf("decode response body %q: %v", w.Body.String(), err)
	}
}
