// screen OrderConfirmation "The order was placed"
// card "Order placed": text "Order #… — Total $…", badge "placed" info
// button "View my orders" primary -> Orders
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button, Card, CardContent, CardHeader, Chip, PageContent, Stack, Typography } from "@wso2/oxygen-ui";
import { ordersApi } from "../api";
import type { components as OrdersComponents } from "../generated/orders-service";

type Order = OrdersComponents["schemas"]["Order"];

export function OrderConfirmationPage(): ReactElement {
  const { orderId } = useParams<{ orderId: string }>();
  const navigate = useNavigate();
  const [order, setOrder] = useState<Order | null | undefined>(undefined);

  useEffect(() => {
    if (!orderId) return;
    let live = true;
    void (async () => {
      const { data, error } = await ordersApi.GET("/me/orders/{orderId}", {
        params: { path: { orderId } },
      });
      if (!live) return;
      setOrder(error ? null : (data ?? null));
    })();
    return () => {
      live = false;
    };
  }, [orderId]);

  if (order === undefined) return <PageContent>Loading…</PageContent>;
  if (order === null) return <PageContent>No such order.</PageContent>;

  return (
    <PageContent maxWidth={560}>
      <Card>
        <CardHeader title="Order placed" />
        <CardContent>
          <Stack spacing={2}>
            <Typography>
              Order #{order.id} — Total ${order.total.toFixed(2)}
            </Typography>
            <Chip label={order.status} color="info" size="small" sx={{ width: "fit-content" }} />
            <Stack direction="row" justifyContent="flex-end">
              <Button variant="contained" onClick={() => navigate("/orders")}>
                View my orders
              </Button>
            </Stack>
          </Stack>
        </CardContent>
      </Card>
    </PageContent>
  );
}
