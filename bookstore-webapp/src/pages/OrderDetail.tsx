// screen OrderDetail "One order's items and status"
// heading "Order #…", badge status, table "Book | Quantity | Unit Price"
import { useEffect, useState, type ReactElement } from "react";
import { useParams } from "react-router-dom";
import { Chip, ListingTable, PageContent, PageTitle, Stack } from "@wso2/oxygen-ui";
import { ordersApi } from "../api";
import { fetchBookMap, type Book } from "../lib/books";
import type { components as OrdersComponents } from "../generated/orders-service";

type Order = OrdersComponents["schemas"]["Order"];

const STATUS_COLOR: Record<Order["status"], "info" | "success" | "warning" | "error"> = {
  placed: "info",
  confirmed: "info",
  shipped: "warning",
  delivered: "success",
  cancelled: "error",
};

export function OrderDetailPage(): ReactElement {
  const { orderId } = useParams<{ orderId: string }>();
  const [order, setOrder] = useState<Order | null | undefined>(undefined);
  const [books, setBooks] = useState<Map<string, Book>>(new Map());

  useEffect(() => {
    if (!orderId) return;
    let live = true;
    void (async () => {
      const [{ data, error }, bookMap] = await Promise.all([
        ordersApi.GET("/me/orders/{orderId}", { params: { path: { orderId } } }),
        fetchBookMap(),
      ]);
      if (!live) return;
      setOrder(error ? null : (data ?? null));
      setBooks(bookMap);
    })();
    return () => {
      live = false;
    };
  }, [orderId]);

  if (order === undefined) return <PageContent>Loading…</PageContent>;
  if (order === null) return <PageContent>No such order.</PageContent>;

  return (
    <PageContent>
      <PageTitle>
        <PageTitle.Header>Order #{order.id}</PageTitle.Header>
      </PageTitle>
      <Stack sx={{ mb: 2 }}>
        <Chip label={order.status} color={STATUS_COLOR[order.status]} size="small" sx={{ width: "fit-content" }} />
      </Stack>

      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Book</ListingTable.Cell>
              <ListingTable.Cell>Quantity</ListingTable.Cell>
              <ListingTable.Cell>Unit Price</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {order.items.map((item, i) => (
              <ListingTable.Row key={`${item.bookId}-${i}`}>
                <ListingTable.Cell>{books.get(item.bookId)?.title ?? item.bookId}</ListingTable.Cell>
                <ListingTable.Cell>{item.quantity}</ListingTable.Cell>
                <ListingTable.Cell>${item.unitPrice.toFixed(2)}</ListingTable.Cell>
              </ListingTable.Row>
            ))}
          </ListingTable.Body>
        </ListingTable>
      </ListingTable.Container>
    </PageContent>
  );
}
