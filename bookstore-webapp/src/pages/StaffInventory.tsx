// screen StaffInventory "Store Staff manage stock levels and reservations"
// heading "Stock Levels", table "Book | On Hand | Reserved" -> StaffStockForm
// heading "Reservations", table "Order # | Book | Quantity | Status"
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import { Chip, ListingTable, PageContent, PageTitle, Typography } from "@wso2/oxygen-ui";
import { inventoryApi } from "../api";
import { fetchBookMap, type Book } from "../lib/books";
import type { components as InventoryComponents } from "../generated/inventory-service";

type StockLevel = InventoryComponents["schemas"]["StockLevel"];
type Reservation = InventoryComponents["schemas"]["Reservation"];

export function StaffInventoryPage(): ReactElement {
  const navigate = useNavigate();
  const [stock, setStock] = useState<StockLevel[] | null>(null);
  const [reservations, setReservations] = useState<Reservation[] | null>(null);
  const [books, setBooks] = useState<Map<string, Book>>(new Map());

  useEffect(() => {
    let live = true;
    void (async () => {
      const [{ data: stockData }, { data: reservationData }, bookMap] = await Promise.all([
        inventoryApi.GET("/stock-levels", { params: { query: { limit: 100 } } }),
        inventoryApi.GET("/reservations", { params: { query: { limit: 100 } } }),
        fetchBookMap(),
      ]);
      if (!live) return;
      setStock(stockData?.data ?? []);
      setReservations(reservationData?.data ?? []);
      setBooks(bookMap);
    })();
    return () => {
      live = false;
    };
  }, []);

  return (
    <PageContent>
      <PageTitle>
        <PageTitle.Header>Stock Levels</PageTitle.Header>
      </PageTitle>

      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Book</ListingTable.Cell>
              <ListingTable.Cell>On Hand</ListingTable.Cell>
              <ListingTable.Cell>Reserved</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {(stock ?? []).map((s) => (
              <ListingTable.Row key={s.bookId} clickable onClick={() => navigate(`/staff/inventory/${s.bookId}`)}>
                <ListingTable.Cell>{books.get(s.bookId)?.title ?? s.bookId}</ListingTable.Cell>
                <ListingTable.Cell>{s.quantityOnHand}</ListingTable.Cell>
                <ListingTable.Cell>{s.quantityReserved}</ListingTable.Cell>
              </ListingTable.Row>
            ))}
          </ListingTable.Body>
        </ListingTable>
        {stock && stock.length === 0 ? (
          <ListingTable.EmptyState title="No stock records yet" />
        ) : null}
      </ListingTable.Container>

      <Typography variant="h6" sx={{ mt: 4, mb: 2 }}>
        Reservations
      </Typography>
      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Order #</ListingTable.Cell>
              <ListingTable.Cell>Book</ListingTable.Cell>
              <ListingTable.Cell>Quantity</ListingTable.Cell>
              <ListingTable.Cell>Status</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {(reservations ?? []).map((r) => (
              <ListingTable.Row key={r.id}>
                <ListingTable.Cell>{r.orderId}</ListingTable.Cell>
                <ListingTable.Cell>{books.get(r.bookId)?.title ?? r.bookId}</ListingTable.Cell>
                <ListingTable.Cell>{r.quantity}</ListingTable.Cell>
                <ListingTable.Cell>
                  <Chip label={r.status} color={r.status === "held" ? "warning" : "default"} size="small" />
                </ListingTable.Cell>
              </ListingTable.Row>
            ))}
          </ListingTable.Body>
        </ListingTable>
        {reservations && reservations.length === 0 ? (
          <ListingTable.EmptyState title="No reservations" />
        ) : null}
      </ListingTable.Container>
    </PageContent>
  );
}
