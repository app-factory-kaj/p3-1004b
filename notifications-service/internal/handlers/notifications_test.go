package handlers_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestForgedAssertionIs401 proves the trust anchor is the SIGNATURE, not the
// header: a request signed by a key the verifier does not trust, and one
// whose payload was edited after a real signature was applied, are both
// refused — never silently downgraded to "anonymous".
func TestForgedAssertionIs401(t *testing.T) {
	gw := newTestGateway(t)

	forged := gw.otherKeyAssertion(t, "shopper-1", []string{"notifications:read"})
	rec := gw.do(t, http.MethodGet, "/me/notifications", forged, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("assertion signed by an untrusted key: got %d, want 401", rec.Code)
	}

	genuine := gw.assertion(t, "shopper-1", []string{"notifications:read"})
	edited := tamper(genuine)
	rec = gw.do(t, http.MethodGet, "/me/notifications", edited, nil)
	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("assertion tampered after signing: got %d, want 401", rec.Code)
	}
}

// TestHealthIsPublic proves the security:[] operation answers 200 with no
// assertion at all, and even with forged X-User-* headers present, and that
// the handler reads none of it.
func TestHealthIsPublic(t *testing.T) {
	gw := newTestGateway(t)

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("X-User-Id", "someone-else")
	req.Header.Set("X-User-Scopes", "notifications:read notifications:create")
	rec := httptest.NewRecorder()
	gw.srv.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET /health: got %d, want 200", rec.Code)
	}
}

// TestNotificationsOwnershipAndWidening is the own/own pair: two different
// signed-in shoppers each see only their own notifications, and marking one
// read never reaches across that boundary — a notification that belongs to
// someone else answers 404 (not 403, and not success), exactly as a
// notification that does not exist at all would.
func TestNotificationsOwnershipAndWidening(t *testing.T) {
	gw := newTestGateway(t)

	// Seed one notification each for two different customers via the
	// create-for-any-customer path (the only way notifications enter the
	// store).
	creator := gw.assertion(t, "orders-service-client", []string{"notifications:create"})
	aliceID := createNotification(t, gw, creator, "alice", "order-1", "Your order shipped")
	bobID := createNotification(t, gw, creator, "bob", "order-2", "Your order was delivered")

	aliceAssertion := gw.assertion(t, "alice", []string{"notifications:read"})
	bobAssertion := gw.assertion(t, "bob", []string{"notifications:read"})

	// Alice sees only her own notification.
	rec := gw.do(t, http.MethodGet, "/me/notifications", aliceAssertion, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice GET /me/notifications: got %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var alicePage struct {
		Count int `json:"count"`
		Data  []struct {
			ID      string `json:"id"`
			OrderID string `json:"orderId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &alicePage); err != nil {
		t.Fatalf("decode alice page: %v", err)
	}
	if alicePage.Count != 1 || len(alicePage.Data) != 1 || alicePage.Data[0].ID != aliceID {
		t.Fatalf("alice's list leaked or missed rows: %+v", alicePage)
	}

	// Bob sees only his own, never alice's.
	rec = gw.do(t, http.MethodGet, "/me/notifications", bobAssertion, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("bob GET /me/notifications: got %d, want 200", rec.Code)
	}
	var bobPage struct {
		Count int `json:"count"`
		Data  []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &bobPage); err != nil {
		t.Fatalf("decode bob page: %v", err)
	}
	if bobPage.Count != 1 || len(bobPage.Data) != 1 || bobPage.Data[0].ID != bobID {
		t.Fatalf("bob's list leaked or missed rows: %+v", bobPage)
	}
	for _, n := range bobPage.Data {
		if n.ID == aliceID {
			t.Fatalf("bob's list contained alice's notification %s — ownership leak", aliceID)
		}
	}

	// Bob marking ALICE's notification read answers 404 — the row is not in
	// his collection, so it does not exist there; never 403, never success.
	rec = gw.do(t, http.MethodPost, "/me/notifications/"+aliceID+"/read", bobAssertion, nil)
	if rec.Code != http.StatusNotFound {
		t.Fatalf("bob marking alice's notification read: got %d, want 404", rec.Code)
	}

	// Alice marking her OWN notification read succeeds.
	rec = gw.do(t, http.MethodPost, "/me/notifications/"+aliceID+"/read", aliceAssertion, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("alice marking her own notification read: got %d, want 200, body=%s", rec.Code, rec.Body.String())
	}
	var updated struct {
		Read bool `json:"read"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &updated); err != nil {
		t.Fatalf("decode updated notification: %v", err)
	}
	if !updated.Read {
		t.Fatalf("alice's notification was not marked read")
	}

	// It did not leak back into bob's collection and alice's unread count did
	// not change for bob.
	rec = gw.do(t, http.MethodGet, "/me/notifications", bobAssertion, nil)
	json.Unmarshal(rec.Body.Bytes(), &bobPage)
	if bobPage.Count != 1 {
		t.Fatalf("bob's collection changed after alice's own mark-read: %+v", bobPage)
	}
}

// TestCreateNotificationForAnyCustomer proves orders-service (the caller
// here, never the recipient) can create a notification for ANY customer
// named in the body under notifications:create, and that it then shows up
// only in THAT customer's own /me/notifications.
func TestCreateNotificationForAnyCustomer(t *testing.T) {
	gw := newTestGateway(t)

	creator := gw.assertion(t, "orders-service-client", []string{"notifications:create"})
	body := []byte(`{"customerId":"carol","orderId":"order-9","message":"Your order was cancelled"}`)
	rec := gw.do(t, http.MethodPost, "/notifications", creator, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("POST /notifications: got %d, want 201, body=%s", rec.Code, rec.Body.String())
	}
	var created struct {
		ID      string `json:"id"`
		OrderID string `json:"orderId"`
		Message string `json:"message"`
		Read    bool   `json:"read"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created notification: %v", err)
	}
	if created.OrderID != "order-9" || created.Message != "Your order was cancelled" || created.Read {
		t.Fatalf("unexpected created notification: %+v", created)
	}

	carol := gw.assertion(t, "carol", []string{"notifications:read"})
	rec = gw.do(t, http.MethodGet, "/me/notifications", carol, nil)
	if rec.Code != http.StatusOK {
		t.Fatalf("carol GET /me/notifications: got %d, want 200", rec.Code)
	}
	var page struct {
		Count int `json:"count"`
		Data  []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &page); err != nil {
		t.Fatalf("decode carol page: %v", err)
	}
	if page.Count != 1 || page.Data[0].ID != created.ID {
		t.Fatalf("carol did not see the notification created for her: %+v", page)
	}

	// A missing required field is a 400, not a 201 or a 500.
	rec = gw.do(t, http.MethodPost, "/notifications", creator, []byte(`{"customerId":"dave","orderId":"","message":"x"}`))
	if rec.Code != http.StatusBadRequest {
		t.Fatalf("POST /notifications with empty orderId: got %d, want 400", rec.Code)
	}
}

func createNotification(t *testing.T, gw *testGateway, creatorAssertion, customerID, orderID, message string) string {
	t.Helper()
	body := []byte(`{"customerId":"` + customerID + `","orderId":"` + orderID + `","message":"` + message + `"}`)
	rec := gw.do(t, http.MethodPost, "/notifications", creatorAssertion, body)
	if rec.Code != http.StatusCreated {
		t.Fatalf("seed notification for %s: got %d, want 201, body=%s", customerID, rec.Code, rec.Body.String())
	}
	var created struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode seeded notification: %v", err)
	}
	return created.ID
}
