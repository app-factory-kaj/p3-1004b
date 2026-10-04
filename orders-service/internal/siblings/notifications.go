package siblings

import (
	"context"
	"fmt"

	"orders-service/internal/auth"
	notificationsclient "orders-service/internal/clients/notifications"
)

// Notifications is orders-service's view of notifications-service.
type Notifications struct {
	api *notificationsclient.ClientWithResponses
}

// NewNotifications builds a client against the injected notifications-service
// address.
func NewNotifications(baseURL string) (*Notifications, error) {
	api, err := notificationsclient.NewClientWithResponses(baseURL, notificationsclient.WithRequestEditorFn(auth.ForwardAssertion))
	if err != nil {
		return nil, fmt.Errorf("notifications client: %w", err)
	}
	return &Notifications{api: api}, nil
}

// Create asks notifications-service to notify customerID that orderID's
// status changed.
func (n *Notifications) Create(ctx context.Context, customerID, orderID, message string) error {
	resp, err := n.api.CreateNotificationWithResponse(ctx, notificationsclient.CreateNotificationJSONRequestBody{
		CustomerID: customerID,
		OrderID:    orderID,
		Message:    message,
	})
	if err != nil {
		return fmt.Errorf("create notification: %w", err)
	}
	if resp.JSON201 == nil {
		return fmt.Errorf("create notification: unexpected status %s", resp.Status())
	}
	return nil
}
