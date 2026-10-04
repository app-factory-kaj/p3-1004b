// screen Orders "The signed-in Shopper's order history"
// heading "My Orders"
// table "Order # | Placed | Status | Total" -> OrderDetail
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import { Chip, ListingTable, PageContent, PageTitle } from "@wso2/oxygen-ui";
import { ordersApi } from "../api";
import type { components as OrdersComponents } from "../generated/orders-service";

type Order = OrdersComponents["schemas"]["Order"];

const STATUS_COLOR: Record<Order["status"], "info" | "success" | "warning" | "error"> = {
  placed: "info",
  confirmed: "info",
  shipped: "warning",
  delivered: "success",
  cancelled: "error",
};

export function OrdersPage(): ReactElement {
  const navigate = useNavigate();
  const [orders, setOrders] = useState<Order[] | null>(null);

  useEffect(() => {
    let live = true;
    void (async () => {
      const { data } = await ordersApi.GET("/me/orders", { params: { query: {} } });
      if (!live) return;
      setOrders(data?.data ?? []);
    })();
    return () => {
      live = false;
    };
  }, []);

  return (
    <PageContent>
      <PageTitle>
        <PageTitle.Header>My Orders</PageTitle.Header>
      </PageTitle>

      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Order #</ListingTable.Cell>
              <ListingTable.Cell>Placed</ListingTable.Cell>
              <ListingTable.Cell>Status</ListingTable.Cell>
              <ListingTable.Cell>Total</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {(orders ?? []).map((order) => (
              <ListingTable.Row key={order.id} clickable onClick={() => navigate(`/orders/${order.id}`)}>
                <ListingTable.Cell>{order.id}</ListingTable.Cell>
                <ListingTable.Cell>{order.placedAt}</ListingTable.Cell>
                <ListingTable.Cell>
                  <Chip label={order.status} color={STATUS_COLOR[order.status]} size="small" />
                </ListingTable.Cell>
                <ListingTable.Cell>${order.total.toFixed(2)}</ListingTable.Cell>
              </ListingTable.Row>
            ))}
          </ListingTable.Body>
        </ListingTable>
        {orders && orders.length === 0 ? (
          <ListingTable.EmptyState title="No orders yet" description="Place your first order from the catalog." />
        ) : null}
      </ListingTable.Container>
    </PageContent>
  );
}
