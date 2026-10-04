// screen BookDetail "A single book's details" — public, no sign-in required.
// card: text author, text description, text price, badge availability,
// button "Add to cart" primary -> Cart
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate, useParams } from "react-router-dom";
import { Button, Card, CardContent, CardHeader, Chip, PageContent, Stack, Typography } from "@wso2/oxygen-ui";
import { catalogApi, cartApi, inventoryApi } from "../api";
import { useAuthz } from "../authz/gates";
import { signIn } from "../authz/session";
import type { components as CatalogComponents } from "../generated/catalog-service";
import type { components as InventoryComponents } from "../generated/inventory-service";

type Book = CatalogComponents["schemas"]["Book"];
type StockLevel = InventoryComponents["schemas"]["StockLevel"];

function availability(stock: StockLevel | undefined): { label: string; color: "success" | "warning" | "error" } {
  if (!stock) return { label: "Unknown", color: "warning" };
  const available = stock.quantityOnHand - stock.quantityReserved;
  if (available <= 0) return { label: "Out of stock", color: "error" };
  if (available <= 5) return { label: "Low stock", color: "warning" };
  return { label: "In stock", color: "success" };
}

export function BookDetailPage(): ReactElement {
  const { bookId } = useParams<{ bookId: string }>();
  const navigate = useNavigate();
  const { signedIn } = useAuthz();
  const [book, setBook] = useState<Book | null | undefined>(undefined);
  const [stock, setStock] = useState<StockLevel | undefined>(undefined);
  const [adding, setAdding] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!bookId) return;
    let live = true;
    void (async () => {
      const { data, error: err } = await catalogApi.GET("/books/{bookId}", {
        params: { path: { bookId } },
      });
      if (!live) return;
      setBook(err ? null : (data ?? null));
      const { data: stockData } = await inventoryApi.GET("/stock-levels/{bookId}", {
        params: { path: { bookId } },
      });
      if (!live) return;
      setStock(stockData);
    })();
    return () => {
      live = false;
    };
  }, [bookId]);

  async function addToCart(): Promise<void> {
    if (!signedIn) {
      void signIn();
      return;
    }
    if (!bookId) return;
    setAdding(true);
    setError(null);
    const { error: err } = await cartApi.POST("/me/cart/items", {
      body: { bookId, quantity: 1 },
    });
    setAdding(false);
    if (err) {
      setError("Could not add that book to your cart.");
      return;
    }
    navigate("/cart");
  }

  if (book === undefined) return <PageContent>Loading…</PageContent>;
  if (book === null) return <PageContent>No such book.</PageContent>;

  const avail = availability(stock);

  return (
    <PageContent maxWidth={640}>
      <Card>
        <CardHeader title={book.title} />
        <CardContent>
          <Stack spacing={1.5}>
            <Typography color="text.secondary">by {book.author}</Typography>
            {book.description ? <Typography>{book.description}</Typography> : null}
            <Typography variant="h5">${book.price.toFixed(2)}</Typography>
            <Chip label={avail.label} color={avail.color} size="small" sx={{ width: "fit-content" }} />
            {error ? (
              <Typography color="error.main" variant="body2">
                {error}
              </Typography>
            ) : null}
            <Button variant="contained" disabled={adding} onClick={() => void addToCart()}>
              Add to cart
            </Button>
          </Stack>
        </CardContent>
      </Card>
    </PageContent>
  );
}
