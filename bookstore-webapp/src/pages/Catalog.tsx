// screen Catalog "Browse and search books" — public, no sign-in required.
// row: search "Search books..." + select "All tags"
// table "Title | Author | Price | Availability" -> BookDetail
import { useEffect, useMemo, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import {
  Chip,
  ListingTable,
  MenuItem,
  PageContent,
  SearchBar,
  Stack,
  TextField,
} from "@wso2/oxygen-ui";
import { catalogApi, inventoryApi } from "../api";
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

export function CatalogPage(): ReactElement {
  const navigate = useNavigate();
  const [books, setBooks] = useState<Book[] | null>(null);
  const [stock, setStock] = useState<Map<string, StockLevel>>(new Map());
  const [q, setQ] = useState("");
  const [tag, setTag] = useState("");
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    let live = true;
    void (async () => {
      setError(null);
      const { data, error: err } = await catalogApi.GET("/books", {
        params: { query: { q: q || undefined, tag: tag || undefined } },
      });
      if (!live) return;
      if (err) {
        setError("Could not load the catalog.");
        setBooks([]);
        return;
      }
      setBooks(data?.data ?? []);
      // Join availability from inventory-service's own list, one request total.
      const { data: stockData } = await inventoryApi.GET("/stock-levels", { params: { query: {} } });
      if (!live) return;
      setStock(new Map((stockData?.data ?? []).map((s) => [s.bookId, s])));
    })();
    return () => {
      live = false;
    };
  }, [q, tag]);

  const tags = useMemo(() => {
    const all = new Set<string>();
    for (const b of books ?? []) for (const t of b.tags ?? []) all.add(t);
    return [...all].sort();
  }, [books]);

  return (
    <PageContent>
      <Stack direction="row" spacing={2} sx={{ mb: 3 }}>
        <SearchBar
          placeholder="Search books..."
          value={q}
          onChange={(e) => setQ(e.target.value)}
          sx={{ flex: 1 }}
        />
        <TextField select label="Tag" value={tag} onChange={(e) => setTag(e.target.value)} sx={{ minWidth: 200 }}>
          <MenuItem value="">All tags</MenuItem>
          {tags.map((t) => (
            <MenuItem key={t} value={t}>
              {t}
            </MenuItem>
          ))}
        </TextField>
      </Stack>

      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Title</ListingTable.Cell>
              <ListingTable.Cell>Author</ListingTable.Cell>
              <ListingTable.Cell>Price</ListingTable.Cell>
              <ListingTable.Cell>Availability</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {(books ?? []).map((book) => {
              const avail = availability(stock.get(book.id));
              return (
                <ListingTable.Row key={book.id} clickable onClick={() => navigate(`/books/${book.id}`)}>
                  <ListingTable.Cell>{book.title}</ListingTable.Cell>
                  <ListingTable.Cell>{book.author}</ListingTable.Cell>
                  <ListingTable.Cell>${book.price.toFixed(2)}</ListingTable.Cell>
                  <ListingTable.Cell>
                    <Chip label={avail.label} color={avail.color} size="small" />
                  </ListingTable.Cell>
                </ListingTable.Row>
              );
            })}
          </ListingTable.Body>
        </ListingTable>
        {books !== null && books.length === 0 && !error ? (
          <ListingTable.EmptyState title="No books found" description="Try a different search or tag." />
        ) : null}
        {error ? <ListingTable.EmptyState title="Something went wrong" description={error} /> : null}
      </ListingTable.Container>
    </PageContent>
  );
}
