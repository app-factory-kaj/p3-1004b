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
	"fmt"
	"math/big"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"notifications-service/internal/auth"
	"notifications-service/internal/gen"
	"notifications-service/internal/handlers"
	"notifications-service/internal/store"
)

const (
	testIssuer = "aep-gateway-test-env"
	testHeader = "x-jwt-assertion"
)

// testGateway wires a throwaway RSA keypair, a verifier built from its
// self-signed certificate, and the real generated router (strict handler +
// verifier middleware) so tests drive the same path a real request takes.
type testGateway struct {
	key *rsa.PrivateKey
	srv http.Handler
}

func newTestGateway(t *testing.T) *testGateway {
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

	st := store.New()
	srv := handlers.New(st)

	r := chi.NewRouter()
	r.Use(verifier.Middleware)
	handler := gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{
		BaseRouter: r,
	})

	return &testGateway{key: key, srv: handler}
}

// otherKeyAssertion signs with a throwaway key that is NOT the one the
// verifier trusts — a forged assertion.
func (g *testGateway) otherKeyAssertion(t *testing.T, sub string, scopes []string) string {
	t.Helper()
	other, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate other key: %v", err)
	}
	return mintAssertion(t, other, testIssuer, sub, scopes, time.Now().Add(15*time.Minute))
}

func (g *testGateway) assertion(t *testing.T, sub string, scopes []string) string {
	t.Helper()
	return mintAssertion(t, g.key, testIssuer, sub, scopes, time.Now().Add(15*time.Minute))
}

func (g *testGateway) do(t *testing.T, method, path, assertion string, body []byte) *httptest.ResponseRecorder {
	t.Helper()
	var req *http.Request
	if body != nil {
		req = httptest.NewRequest(method, path, bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
	} else {
		req = httptest.NewRequest(method, path, nil)
	}
	if assertion != "" {
		req.Header.Set(testHeader, assertion)
	}
	rec := httptest.NewRecorder()
	g.srv.ServeHTTP(rec, req)
	return rec
}

type assertionClaims struct {
	Issuer  string `json:"iss"`
	Subject string `json:"sub"`
	Scope   string `json:"scope"`
	Exp     int64  `json:"exp"`
}

func mintAssertion(t *testing.T, key *rsa.PrivateKey, issuer, subject string, scopes []string, exp time.Time) string {
	t.Helper()

	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	headerJSON, err := json.Marshal(header)
	if err != nil {
		t.Fatalf("marshal header: %v", err)
	}

	scopeStr := ""
	for i, s := range scopes {
		if i > 0 {
			scopeStr += " "
		}
		scopeStr += s
	}
	claims := assertionClaims{
		Issuer:  issuer,
		Subject: subject,
		Scope:   scopeStr,
		Exp:     exp.Unix(),
	}
	payloadJSON, err := json.Marshal(claims)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	headerB64 := base64.RawURLEncoding.EncodeToString(headerJSON)
	payloadB64 := base64.RawURLEncoding.EncodeToString(payloadJSON)
	signingInput := headerB64 + "." + payloadB64

	digest := sha256.Sum256([]byte(signingInput))
	signature, err := rsa.SignPKCS1v15(rand.Reader, key, crypto.SHA256, digest[:])
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	sigB64 := base64.RawURLEncoding.EncodeToString(signature)

	return signingInput + "." + sigB64
}

// tamper edits the payload segment of an already-signed assertion, so the
// signature no longer matches.
func tamper(token string) string {
	parts := strings.SplitN(token, ".", 3)
	if len(parts) != 3 {
		return token
	}
	payload, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil || len(payload) == 0 {
		return token
	}
	var claims map[string]any
	if err := json.Unmarshal(payload, &claims); err != nil {
		return token
	}
	claims["sub"] = fmt.Sprintf("%v-tampered", claims["sub"])
	edited, err := json.Marshal(claims)
	if err != nil {
		return token
	}
	parts[1] = base64.RawURLEncoding.EncodeToString(edited)
	return parts[0] + "." + parts[1] + "." + parts[2]
}

func selfSignedCertPEM(t *testing.T, key *rsa.PrivateKey) string {
	t.Helper()
	template := x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "notifications-service-test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	var buf []byte
	buf = pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der})
	return string(buf)
}
