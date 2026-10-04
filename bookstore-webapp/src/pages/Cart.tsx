// screen Cart "The signed-in Shopper's cart"
// heading "My Cart"
// table "Book | Quantity | Unit Price | Line Total"
// row: right, text "Subtotal: $…"
// row: right, button "Checkout" primary -> Checkout
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import { Button, ListingTable, PageContent, PageTitle, Stack, Typography } from "@wso2/oxygen-ui";
import { cartApi } from "../api";
import { fetchBookMap, type Book } from "../lib/books";
import type { components as CartComponents } from "../generated/cart-service";

type Cart = CartComponents["schemas"]["Cart"];

export function CartPage(): ReactElement {
  const navigate = useNavigate();
  const [cart, setCart] = useState<Cart | null>(null);
  const [books, setBooks] = useState<Map<string, Book>>(new Map());
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    void (async () => {
      const [{ data, error: err }, bookMap] = await Promise.all([cartApi.GET("/me/cart", {}), fetchBookMap()]);
      if (!live) return;
      if (err) {
        setError("Could not load your cart.");
        return;
      }
      setCart(data ?? null);
      setBooks(bookMap);
    })();
    return () => {
      live = false;
    };
  }, []);

  return (
    <PageContent>
      <PageTitle>
        <PageTitle.Header>My Cart</PageTitle.Header>
      </PageTitle>

      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Book</ListingTable.Cell>
              <ListingTable.Cell>Quantity</ListingTable.Cell>
              <ListingTable.Cell>Unit Price</ListingTable.Cell>
              <ListingTable.Cell>Line Total</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {(cart?.items ?? []).map((item) => {
              const book = books.get(item.bookId);
              const unitPrice = book?.price ?? 0;
              return (
                <ListingTable.Row key={item.bookId}>
                  <ListingTable.Cell>{book?.title ?? item.bookId}</ListingTable.Cell>
                  <ListingTable.Cell>{item.quantity}</ListingTable.Cell>
                  <ListingTable.Cell>${unitPrice.toFixed(2)}</ListingTable.Cell>
                  <ListingTable.Cell>${(unitPrice * item.quantity).toFixed(2)}</ListingTable.Cell>
                </ListingTable.Row>
              );
            })}
          </ListingTable.Body>
        </ListingTable>
        {cart && cart.items.length === 0 ? (
          <ListingTable.EmptyState title="Your cart is empty" description="Add a book from the catalog." />
        ) : null}
        {error ? <ListingTable.EmptyState title="Something went wrong" description={error} /> : null}
      </ListingTable.Container>

      <Stack direction="row" justifyContent="flex-end" sx={{ mt: 2 }}>
        <Typography variant="h6">Subtotal: ${(cart?.subtotal ?? 0).toFixed(2)}</Typography>
      </Stack>
      <Stack direction="row" justifyContent="flex-end" sx={{ mt: 2 }}>
        <Button
          variant="contained"
          disabled={!cart || cart.items.length === 0}
          onClick={() => navigate("/checkout")}
        >
          Checkout
        </Button>
      </Stack>
    </PageContent>
  );
}
