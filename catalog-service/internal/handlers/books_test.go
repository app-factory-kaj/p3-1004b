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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"catalog-service/internal/auth"
	"catalog-service/internal/gen"
	"catalog-service/internal/handlers"
	"catalog-service/internal/store"
)

const (
	testIssuer = "test-gateway"
	testHeader = "x-jwt-assertion"
)

// testEnv builds the real router — chi + the gateway-assertion verifier
// middleware + the generated strict server — wired exactly as main.go wires
// it, and returns a function that mints a signed assertion with the given
// subject and scopes.
func testEnv(t *testing.T) (http.Handler, func(sub string, scopes []string) string, *rsa.PrivateKey) {
	t.Helper()

	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	certPEM := selfSignedCertPEM(t, key)

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

	sign := func(sub string, scopes []string) string {
		return mintAssertion(t, key, testIssuer, sub, scopes, time.Now().Add(15*time.Minute))
	}
	return h, sign, key
}

func selfSignedCertPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	var buf bytes.Buffer
	if err := pem.Encode(&buf, &pem.Block{Type: "CERTIFICATE", Bytes: der}); err != nil {
		t.Fatalf("encode certificate: %v", err)
	}
	return buf.String()
}

func mintAssertion(t *testing.T, key *rsa.PrivateKey, iss, sub string, scopes []string, exp time.Time) string {
	t.Helper()
	header := map[string]any{"alg": "RS256", "typ": "JWT"}
	scopeStr := ""
	for i, s := range scopes {
		if i > 0 {
			scopeStr += " "
		}
		scopeStr += s
	}
	claims := map[string]any{
		"iss":   iss,
		"sub":   sub,
		"scope": scopeStr,
		"exp":   exp.Unix(),
	}
	return signJWT(t, key, header, claims)
}

func signJWT(t *testing.T, key *rsa.PrivateKey, header, claims map[string]any) string {
	t.Helper()
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}
	claimsJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal claims: %v", err)
	}
	signingInput := base64.RawURLEncoding.EncodeToString(headerJSON) + "." + base64.RawURLEncoding.EncodeToString(claimsJSON)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return signingInput + "." + base64.RawURLEncoding.EncodeToString(sig)
}

func TestHealthIsPublic(t *testing.T) {
	h, _, _ := testEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	// Forged, unsigned identity headers must be ignored entirely — health
	// reads no identity at all.
	req.Header.Set("X-User-Id", "attacker")
	req.Header.Set("X-User-Scopes", "books:manage")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}
}

func TestForgedAssertionIs401(t *testing.T) {
	h, sign, _ := testEnv(t)

	// A different key's valid-looking assertion: the signature was never
	// produced by this environment's certificate.
	otherKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	forged := mintAssertion(t, otherKey, testIssuer, "attacker", []string{"books:manage"}, time.Now().Add(15*time.Minute))

	req := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"title":"X","author":"Y","price":1}`))
	req.Header.Set(testHeader, forged)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("forged-key assertion: got %d, want 401; body=%s", rec.Code, rec.Body.String())
	}

	// A genuinely signed assertion whose payload is edited after signing.
	good := sign("staff-1", []string{"books:manage"})
	parts := bytes.SplitN([]byte(good), []byte("."), 3)
	tamperedPayload, err := base64.RawURLEncoding.DecodeString(string(parts[1]))
	if err != nil {
		t.Fatalf("decode payload: %v", err)
	}
	var claims map[string]any
	if err := json.Unmarshal(tamperedPayload, &claims); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	claims["sub"] = "someone-else"
	tampered, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal tampered claims: %v", err)
	}
	tamperedToken := string(parts[0]) + "." + base64.RawURLEncoding.EncodeToString(tampered) + "." + string(parts[2])

	req2 := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"title":"X","author":"Y","price":1}`))
	req2.Header.Set(testHeader, tamperedToken)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)
	if rec2.Code != http.StatusUnauthorized {
		t.Fatalf("tampered-payload assertion: got %d, want 401; body=%s", rec2.Code, rec2.Body.String())
	}
}

func TestListBooksIsPublicAndSeeded(t *testing.T) {
	h, _, _ := testEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/books", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /books = %d, want 200; body=%s", rec.Code, rec.Body.String())
	}

	var body struct {
		Count int        `json:"count"`
		Data  []gen.Book `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if body.Count == 0 || len(body.Data) == 0 {
		t.Fatalf("expected a seeded, non-empty catalog, got count=%d data=%d", body.Count, len(body.Data))
	}
}

func TestGetBookNotFound(t *testing.T) {
	h, _, _ := testEnv(t)

	req := httptest.NewRequest(http.MethodGet, "/books/does-not-exist", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("GET /books/does-not-exist = %d, want 404; body=%s", rec.Code, rec.Body.String())
	}
}

// TestBookManageLifecycle exercises create/update/delete with a verified
// assertion present. The operation's books:manage scope itself is enforced
// at the gateway (api-management, go skill) — this service holds no copy of
// that check — so the test asserts only what the handler itself is
// responsible for: validating input and round-tripping state.
func TestBookManageLifecycle(t *testing.T) {
	h, sign, _ := testEnv(t)
	token := sign("staff-1", []string{"books:manage"})

	// Invalid input is rejected with 400.
	badReq := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(`{"author":"Y","price":1}`))
	badReq.Header.Set(testHeader, token)
	badReq.Header.Set("Content-Type", "application/json")
	badRec := httptest.NewRecorder()
	h.ServeHTTP(badRec, badReq)
	if badRec.Code != http.StatusBadRequest {
		t.Fatalf("create with missing title: got %d, want 400; body=%s", badRec.Code, badRec.Body.String())
	}

	// Create.
	createReq := httptest.NewRequest(http.MethodPost, "/books", bytes.NewBufferString(
		`{"title":"New Book","author":"Someone","price":9.99,"tags":["new"]}`))
	createReq.Header.Set(testHeader, token)
	createReq.Header.Set("Content-Type", "application/json")
	createRec := httptest.NewRecorder()
	h.ServeHTTP(createRec, createReq)
	if createRec.Code != http.StatusCreated {
		t.Fatalf("create: got %d, want 201; body=%s", createRec.Code, createRec.Body.String())
	}
	var created gen.Book
	if err := json.Unmarshal(createRec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created book: %v", err)
	}
	if created.ID == "" || created.Title != "New Book" {
		t.Fatalf("unexpected created book: %+v", created)
	}

	// Update.
	updateReq := httptest.NewRequest(http.MethodPut, "/books/"+created.ID, bytes.NewBufferString(
		`{"title":"Updated Book","author":"Someone","price":12.50}`))
	updateReq.Header.Set(testHeader, token)
	updateReq.Header.Set("Content-Type", "application/json")
	updateRec := httptest.NewRecorder()
	h.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("update: got %d, want 200; body=%s", updateRec.Code, updateRec.Body.String())
	}

	// Delete.
	delReq := httptest.NewRequest(http.MethodDelete, "/books/"+created.ID, nil)
	delReq.Header.Set(testHeader, token)
	delRec := httptest.NewRecorder()
	h.ServeHTTP(delRec, delReq)
	if delRec.Code != http.StatusNoContent {
		t.Fatalf("delete: got %d, want 204; body=%s", delRec.Code, delRec.Body.String())
	}

	// Confirm it is gone.
	getReq := httptest.NewRequest(http.MethodGet, "/books/"+created.ID, nil)
	getRec := httptest.NewRecorder()
	h.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusNotFound {
		t.Fatalf("get after delete: got %d, want 404; body=%s", getRec.Code, getRec.Body.String())
	}
}
