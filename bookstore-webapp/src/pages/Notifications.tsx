// screen Notifications "The signed-in Shopper's order-status notifications"
// heading "Notifications"
// list "Order #10234 is now placed | Order #10198 was delivered"
import { useEffect, useState, type ReactElement } from "react";
import { List, ListItem, ListItemText, PageContent, PageTitle } from "@wso2/oxygen-ui";
import { notificationsApi } from "../api";
import type { components as NotificationsComponents } from "../generated/notifications-service";

type Notification = NotificationsComponents["schemas"]["Notification"];

export function NotificationsPage(): ReactElement {
  const [notifications, setNotifications] = useState<Notification[] | null>(null);

  useEffect(() => {
    let live = true;
    void (async () => {
      const { data } = await notificationsApi.GET("/me/notifications", { params: { query: {} } });
      if (!live) return;
      setNotifications(data?.data ?? []);
    })();
    return () => {
      live = false;
    };
  }, []);

  return (
    <PageContent maxWidth={640}>
      <PageTitle>
        <PageTitle.Header>Notifications</PageTitle.Header>
      </PageTitle>

      {notifications && notifications.length === 0 ? (
        <ListItemText primary="No notifications yet." />
      ) : (
        <List>
          {(notifications ?? []).map((n) => (
            <ListItem key={n.id} divider>
              <ListItemText
                primary={n.message}
                secondary={n.createdAt}
                slotProps={{
                  primary: { sx: { fontWeight: n.read ? 400 : 600 } },
                }}
              />
            </ListItem>
          ))}
        </List>
      )}
    </PageContent>
  );
}
