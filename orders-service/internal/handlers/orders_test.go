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
	"testing"
	"time"

	"github.com/go-chi/chi/v5"

	"orders-service/internal/auth"
	"orders-service/internal/gen"
	"orders-service/internal/handlers"
	"orders-service/internal/siblings"
	"orders-service/internal/store"
)

const (
	testIssuer = "test-gateway"
	testHeader = "x-jwt-assertion"
)

// --- throwaway RSA key + signed assertion helpers -------------------------

func genKeyAndCert(t *testing.T) (*rsa.PrivateKey, string) {
	t.Helper()
	priv, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate key: %v", err)
	}
	tmpl := &x509.Certificate{
		SerialNumber: big.NewInt(1),
		Subject:      pkix.Name{CommonName: "test"},
		NotBefore:    time.Now().Add(-time.Hour),
		NotAfter:     time.Now().Add(time.Hour),
	}
	der, err := x509.CreateCertificate(rand.Reader, tmpl, tmpl, &priv.PublicKey, priv)
	if err != nil {
		t.Fatalf("create certificate: %v", err)
	}
	certPEM := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	return priv, certPEM
}

func b64url(b []byte) string {
	return base64.RawURLEncoding.EncodeToString(b)
}

func mintAssertion(t *testing.T, priv *rsa.PrivateKey, issuer, sub string, scopes []string, expIn time.Duration) string {
	t.Helper()
	header := map[string]string{"alg": "RS256", "typ": "JWT"}
	headerJSON, _ := json.Marshal(header)
	scopeStr := ""
	for i, s := range scopes {
		if i > 0 {
			scopeStr += " "
		}
		scopeStr += s
	}
	claims := map[string]any{
		"iss":   issuer,
		"sub":   sub,
		"scope": scopeStr,
		"exp":   time.Now().Add(expIn).Unix(),
		"nbf":   time.Now().Add(-time.Minute).Unix(),
	}
	payloadJSON, _ := json.Marshal(claims)
	signingInput := b64url(headerJSON) + "." + b64url(payloadJSON)
	sum := sha256.Sum256([]byte(signingInput))
	sig, err := rsa.SignPKCS1v15(rand.Reader, priv, crypto.SHA256, sum[:])
	if err != nil {
		t.Fatalf("sign assertion: %v", err)
	}
	return signingInput + "." + b64url(sig)
}

// --- building the real, wired router --------------------------------------

type testDeps struct {
	cartSrv          *httptest.Server
	inventorySrv     *httptest.Server
	customersSrv     *httptest.Server
	notificationsSrv *httptest.Server
}

func buildRouter(t *testing.T, priv *rsa.PrivateKey, certPEM string, deps testDeps) http.Handler {
	t.Helper()
	t.Setenv("GATEWAY_ASSERTION_CERTIFICATE", certPEM)
	t.Setenv("GATEWAY_ASSERTION_ISSUER", testIssuer)
	t.Setenv("GATEWAY_ASSERTION_HEADER", testHeader)

	verifier, err := auth.NewVerifierFromEnv()
	if err != nil {
		t.Fatalf("new verifier: %v", err)
	}

	cartURL, inventoryURL, customersURL, notificationsURL := "http://127.0.0.1:0", "http://127.0.0.1:0", "http://127.0.0.1:0", "http://127.0.0.1:0"
	if deps.cartSrv != nil {
		cartURL = deps.cartSrv.URL
	}
	if deps.inventorySrv != nil {
		inventoryURL = deps.inventorySrv.URL
	}
	if deps.customersSrv != nil {
		customersURL = deps.customersSrv.URL
	}
	if deps.notificationsSrv != nil {
		notificationsURL = deps.notificationsSrv.URL
	}

	cartClient, err := siblings.NewCart(cartURL)
	if err != nil {
		t.Fatalf("cart client: %v", err)
	}
	inventoryClient, err := siblings.NewInventory(inventoryURL)
	if err != nil {
		t.Fatalf("inventory client: %v", err)
	}
	customersClient, err := siblings.NewCustomers(customersURL)
	if err != nil {
		t.Fatalf("customers client: %v", err)
	}
	notificationsClient, err := siblings.NewNotifications(notificationsURL)
	if err != nil {
		t.Fatalf("notifications client: %v", err)
	}

	srv := &handlers.Server{
		Store:         store.New(),
		Cart:          cartClient,
		Inventory:     inventoryClient,
		Customers:     customersClient,
		Notifications: notificationsClient,
	}

	r := chi.NewRouter()
	r.Use(verifier.Middleware)
	r.Use(auth.CaptureRawAssertion)
	return gen.HandlerWithOptions(gen.NewStrictHandler(srv, nil), gen.ChiServerOptions{BaseRouter: r})
}

func doRequest(h http.Handler, method, path, assertion string, body []byte) *httptest.ResponseRecorder {
	var r *http.Request
	if body != nil {
		r = httptest.NewRequest(method, path, bytes.NewReader(body))
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

// --- fake siblings ----------------------------------------------------------

// jsonHandler replies with the given status and JSON body for any request.
func jsonHandler(status int, body any) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_ = json.NewEncoder(w).Encode(body)
	}
}

// --- tests ------------------------------------------------------------------

func TestHealthIsPublic(t *testing.T) {
	priv, cert := genKeyAndCert(t)
	h := buildRouter(t, priv, cert, testDeps{})

	// No assertion, and forged X-User-* headers: the handler must read
	// neither and still answer 200.
	r := httptest.NewRequest(http.MethodGet, "/health", nil)
	r.Header.Set("x-user-id", "attacker")
	r.Header.Set("x-user-scopes", "orders:read orders:create")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Fatalf("GET /health = %d, want 200", w.Code)
	}
}

func TestForgedAssertionIs401(t *testing.T) {
	priv, cert := genKeyAndCert(t)
	h := buildRouter(t, priv, cert, testDeps{})

	otherPriv, _ := genKeyAndCert(t)
	forged := mintAssertion(t, otherPriv, testIssuer, "shopper-1", []string{"orders:read"}, time.Hour)
	w := doRequest(h, http.MethodGet, "/me/orders", forged, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("assertion signed by a different key: got %d, want 401", w.Code)
	}

	// Tamper with a validly-signed assertion's payload after signing.
	valid := mintAssertion(t, priv, testIssuer, "shopper-1", []string{"orders:read"}, time.Hour)
	tampered := valid[:len(valid)-4] + "abcd"
	w = doRequest(h, http.MethodGet, "/me/orders", tampered, nil)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("tampered assertion: got %d, want 401", w.Code)
	}
}

func TestOrdersListOwnershipAndWidening(t *testing.T) {
	priv, cert := genKeyAndCert(t)
	h := buildRouter(t, priv, cert, testDeps{
		cartSrv: httptest.NewServer(jsonHandler(http.StatusOK, map[string]any{
			"id":       "cart-1",
			"items":    []map[string]any{{"bookId": "book-1", "quantity": 2}},
			"subtotal": 20.0,
		})),
		inventorySrv: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPost && r.URL.Path == "/reservations" {
				jsonHandler(http.StatusCreated, map[string]any{
					"id": "res-1", "bookId": "book-1", "orderId": "whatever", "quantity": 2, "status": "held",
				})(w, r)
				return
			}
			w.WriteHeader(http.StatusOK)
		})),
		customersSrv: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch {
			case r.URL.Path == "/me/addresses":
				jsonHandler(http.StatusOK, map[string]any{
					"count": 1,
					"data":  []map[string]any{{"id": "addr-1", "line1": "1 Main St", "city": "Metropolis", "region": "NY", "postalCode": "10001", "country": "US"}},
				})(w, r)
			case r.URL.Path == "/me/profile":
				jsonHandler(http.StatusOK, map[string]any{"id": "cust-1", "name": "Alice", "email": "a@example.com"})(w, r)
			default:
				w.WriteHeader(http.StatusOK)
			}
		})),
		notificationsSrv: httptest.NewServer(jsonHandler(http.StatusCreated, map[string]any{
			"id": "notif-1", "orderId": "whatever", "message": "placed", "read": false, "createdAt": time.Now().Format(time.RFC3339),
		})),
	})

	aliceAssertion := mintAssertion(t, priv, testIssuer, "alice", []string{"orders:read", "orders:create"}, time.Hour)
	bobAssertion := mintAssertion(t, priv, testIssuer, "bob", []string{"orders:read", "orders:create"}, time.Hour)

	// Alice checks out, placing exactly one order of her own.
	checkoutBody, _ := json.Marshal(map[string]string{"addressId": "addr-1", "paymentMethodRef": "pm_sim_1"})
	w := doRequest(h, http.MethodPost, "/me/orders", aliceAssertion, checkoutBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("alice checkout = %d, body %s", w.Code, w.Body.String())
	}

	// Alice sees her own order.
	w = doRequest(h, http.MethodGet, "/me/orders", aliceAssertion, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("alice list = %d", w.Code)
	}
	var aliceList struct {
		Count int `json:"count"`
		Data  []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &aliceList); err != nil {
		t.Fatalf("decode alice list: %v", err)
	}
	if aliceList.Count != 1 {
		t.Fatalf("alice sees %d orders, want 1", aliceList.Count)
	}

	// Bob, who placed no order, sees none of Alice's.
	w = doRequest(h, http.MethodGet, "/me/orders", bobAssertion, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("bob list = %d", w.Code)
	}
	var bobList struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &bobList); err != nil {
		t.Fatalf("decode bob list: %v", err)
	}
	if bobList.Count != 0 {
		t.Fatalf("bob sees %d orders, want 0 (not alice's)", bobList.Count)
	}
}

func TestOrdersGetOwnershipWidening(t *testing.T) {
	priv, cert := genKeyAndCert(t)
	h := buildRouter(t, priv, cert, testDeps{
		cartSrv: httptest.NewServer(jsonHandler(http.StatusOK, map[string]any{
			"id":       "cart-1",
			"items":    []map[string]any{{"bookId": "book-1", "quantity": 1}},
			"subtotal": 10.0,
		})),
		inventorySrv: httptest.NewServer(jsonHandler(http.StatusCreated, map[string]any{
			"id": "res-1", "bookId": "book-1", "orderId": "whatever", "quantity": 1, "status": "held",
		})),
		customersSrv: httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/me/addresses":
				jsonHandler(http.StatusOK, map[string]any{
					"count": 1,
					"data":  []map[string]any{{"id": "addr-1", "line1": "1 Main St", "city": "Metropolis", "region": "NY", "postalCode": "10001", "country": "US"}},
				})(w, r)
			case "/me/profile":
				jsonHandler(http.StatusOK, map[string]any{"id": "cust-1", "name": "Alice", "email": "a@example.com"})(w, r)
			default:
				w.WriteHeader(http.StatusOK)
			}
		})),
		notificationsSrv: httptest.NewServer(jsonHandler(http.StatusCreated, map[string]any{
			"id": "notif-1", "orderId": "whatever", "message": "placed", "read": false, "createdAt": time.Now().Format(time.RFC3339),
		})),
	})

	aliceAssertion := mintAssertion(t, priv, testIssuer, "alice", []string{"orders:read", "orders:create"}, time.Hour)
	bobAssertion := mintAssertion(t, priv, testIssuer, "bob", []string{"orders:read"}, time.Hour)

	checkoutBody, _ := json.Marshal(map[string]string{"addressId": "addr-1", "paymentMethodRef": "pm_sim_1"})
	w := doRequest(h, http.MethodPost, "/me/orders", aliceAssertion, checkoutBody)
	if w.Code != http.StatusCreated {
		t.Fatalf("alice checkout = %d, body %s", w.Code, w.Body.String())
	}
	var order struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &order); err != nil {
		t.Fatalf("decode order: %v", err)
	}

	// The owner reads it fine.
	w = doRequest(h, http.MethodGet, "/me/orders/"+order.ID, aliceAssertion, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("owner get = %d", w.Code)
	}

	// Bob asking for Alice's order id gets 404, not 403 and not Alice's data.
	w = doRequest(h, http.MethodGet, "/me/orders/"+order.ID, bobAssertion, nil)
	if w.Code != http.StatusNotFound {
		t.Fatalf("other caller get = %d, want 404", w.Code)
	}
}

func TestCheckoutHappyPath(t *testing.T) {
	priv, cert := genKeyAndCert(t)

	var cartCleared bool
	var reserveCalls, releaseCalls int
	var notificationCreated bool

	cartSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/me/cart":
			jsonHandler(http.StatusOK, map[string]any{
				"id": "cart-1",
				"items": []map[string]any{
					{"bookId": "book-1", "quantity": 2},
					{"bookId": "book-2", "quantity": 1},
				},
				"subtotal": 30.0,
			})(w, r)
		case r.Method == http.MethodDelete && r.URL.Path == "/me/cart":
			cartCleared = true
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer cartSrv.Close()

	inventorySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/reservations" {
			reserveCalls++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			jsonHandler(http.StatusCreated, map[string]any{
				"id": fmt.Sprintf("res-%d", reserveCalls), "bookId": body["bookId"], "orderId": body["orderId"], "quantity": body["quantity"], "status": "held",
			})(w, r)
			return
		}
		if r.Method == http.MethodPost && len(r.URL.Path) > len("/reservations/") {
			releaseCalls++
			w.WriteHeader(http.StatusOK)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer inventorySrv.Close()

	customersSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/me/addresses":
			jsonHandler(http.StatusOK, map[string]any{
				"count": 1,
				"data":  []map[string]any{{"id": "addr-1", "line1": "1 Main St", "city": "Metropolis", "region": "NY", "postalCode": "10001", "country": "US"}},
			})(w, r)
		case "/me/profile":
			jsonHandler(http.StatusOK, map[string]any{"id": "cust-1", "name": "Alice", "email": "a@example.com"})(w, r)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer customersSrv.Close()

	notificationsSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/notifications" {
			notificationCreated = true
			jsonHandler(http.StatusCreated, map[string]any{
				"id": "notif-1", "orderId": "whatever", "message": "placed", "read": false, "createdAt": time.Now().Format(time.RFC3339),
			})(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer notificationsSrv.Close()

	h := buildRouter(t, priv, cert, testDeps{
		cartSrv:          cartSrv,
		inventorySrv:     inventorySrv,
		customersSrv:     customersSrv,
		notificationsSrv: notificationsSrv,
	})

	assertion := mintAssertion(t, priv, testIssuer, "alice", []string{"orders:read", "orders:create"}, time.Hour)
	body, _ := json.Marshal(map[string]string{"addressId": "addr-1", "paymentMethodRef": "pm_sim_1"})
	w := doRequest(h, http.MethodPost, "/me/orders", assertion, body)
	if w.Code != http.StatusCreated {
		t.Fatalf("checkout = %d, body %s", w.Code, w.Body.String())
	}

	var order gen.Order
	if err := json.Unmarshal(w.Body.Bytes(), &order); err != nil {
		t.Fatalf("decode order: %v", err)
	}
	if order.Total != 30.0 {
		t.Fatalf("order total = %v, want 30", order.Total)
	}
	if len(order.Items) != 2 {
		t.Fatalf("order items = %d, want 2", len(order.Items))
	}
	if order.Status != gen.Placed {
		t.Fatalf("order status = %v, want placed", order.Status)
	}
	if order.AddressID != "addr-1" {
		t.Fatalf("order addressId = %q, want addr-1", order.AddressID)
	}
	if reserveCalls != 2 {
		t.Fatalf("reserve calls = %d, want 2", reserveCalls)
	}
	if releaseCalls != 0 {
		t.Fatalf("release calls = %d, want 0 on the happy path", releaseCalls)
	}
	if !cartCleared {
		t.Fatal("cart was not cleared after checkout")
	}
	if !notificationCreated {
		t.Fatal("no notification was created")
	}
}

func TestCheckoutInsufficientStockRefusesLeavesCartIntactAndReleasesReservations(t *testing.T) {
	priv, cert := genKeyAndCert(t)

	var cartCleared bool
	var reserveCalls int
	var releasedReservationIDs []string

	cartSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/me/cart":
			jsonHandler(http.StatusOK, map[string]any{
				"id": "cart-1",
				"items": []map[string]any{
					{"bookId": "book-1", "quantity": 1},
					{"bookId": "book-2", "quantity": 5},
				},
				"subtotal": 50.0,
			})(w, r)
		case r.Method == http.MethodDelete && r.URL.Path == "/me/cart":
			cartCleared = true
			w.WriteHeader(http.StatusNoContent)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer cartSrv.Close()

	inventorySrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/reservations" {
			reserveCalls++
			var body map[string]any
			_ = json.NewDecoder(r.Body).Decode(&body)
			if body["bookId"] == "book-2" {
				jsonHandler(http.StatusBadRequest, map[string]any{"code": 400, "message": "insufficient stock"})(w, r)
				return
			}
			jsonHandler(http.StatusCreated, map[string]any{
				"id": "res-1", "bookId": body["bookId"], "orderId": body["orderId"], "quantity": body["quantity"], "status": "held",
			})(w, r)
			return
		}
		if r.Method == http.MethodPost {
			// POST /reservations/{id}/release
			parts := r.URL.Path
			releasedReservationIDs = append(releasedReservationIDs, parts)
			jsonHandler(http.StatusOK, map[string]any{"id": "res-1", "bookId": "book-1", "orderId": "x", "quantity": 1, "status": "released"})(w, r)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer inventorySrv.Close()

	customersSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/me/addresses":
			jsonHandler(http.StatusOK, map[string]any{
				"count": 1,
				"data":  []map[string]any{{"id": "addr-1", "line1": "1 Main St", "city": "Metropolis", "region": "NY", "postalCode": "10001", "country": "US"}},
			})(w, r)
		case "/me/profile":
			jsonHandler(http.StatusOK, map[string]any{"id": "cust-1", "name": "Alice", "email": "a@example.com"})(w, r)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	defer customersSrv.Close()

	notificationsSrv := httptest.NewServer(jsonHandler(http.StatusCreated, map[string]any{
		"id": "notif-1", "orderId": "whatever", "message": "placed", "read": false, "createdAt": time.Now().Format(time.RFC3339),
	}))
	defer notificationsSrv.Close()

	h := buildRouter(t, priv, cert, testDeps{
		cartSrv:          cartSrv,
		inventorySrv:     inventorySrv,
		customersSrv:     customersSrv,
		notificationsSrv: notificationsSrv,
	})

	assertion := mintAssertion(t, priv, testIssuer, "alice", []string{"orders:read", "orders:create"}, time.Hour)
	body, _ := json.Marshal(map[string]string{"addressId": "addr-1", "paymentMethodRef": "pm_sim_1"})
	w := doRequest(h, http.MethodPost, "/me/orders", assertion, body)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("checkout with out-of-stock item = %d, want 400, body %s", w.Code, w.Body.String())
	}
	if cartCleared {
		t.Fatal("cart was cleared despite checkout refusal")
	}
	if reserveCalls != 2 {
		t.Fatalf("reserve calls = %d, want 2 (book-1 held, book-2 refused)", reserveCalls)
	}
	if len(releasedReservationIDs) != 1 {
		t.Fatalf("released %d reservations, want 1 (book-1's)", len(releasedReservationIDs))
	}

	// No order was created.
	w = doRequest(h, http.MethodGet, "/me/orders", assertion, nil)
	if w.Code != http.StatusOK {
		t.Fatalf("list orders = %d", w.Code)
	}
	var list struct {
		Count int `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &list); err != nil {
		t.Fatalf("decode list: %v", err)
	}
	if list.Count != 0 {
		t.Fatalf("orders created despite refusal: count = %d", list.Count)
	}
}
