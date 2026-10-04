// screen StaffCatalog "Store Staff manage the book catalog"
// row: heading "Catalog", right, button "Add book" primary -> StaffBookForm
// table "Title | Author | Price | Stock" -> StaffBookForm
import { useEffect, useState, type ReactElement } from "react";
import { useNavigate } from "react-router-dom";
import { Button, ListingTable, PageContent, PageTitle } from "@wso2/oxygen-ui";
import { Plus } from "@wso2/oxygen-ui-icons-react";
import { catalogApi, inventoryApi } from "../api";
import { Can } from "../authz/gates";
import type { components as CatalogComponents } from "../generated/catalog-service";
import type { components as InventoryComponents } from "../generated/inventory-service";

type Book = CatalogComponents["schemas"]["Book"];
type StockLevel = InventoryComponents["schemas"]["StockLevel"];

export function StaffCatalogPage(): ReactElement {
  const navigate = useNavigate();
  const [books, setBooks] = useState<Book[] | null>(null);
  const [stock, setStock] = useState<Map<string, StockLevel>>(new Map());

  useEffect(() => {
    let live = true;
    void (async () => {
      const [{ data: bookData }, { data: stockData }] = await Promise.all([
        catalogApi.GET("/books", { params: { query: { limit: 100 } } }),
        inventoryApi.GET("/stock-levels", { params: { query: { limit: 100 } } }),
      ]);
      if (!live) return;
      setBooks(bookData?.data ?? []);
      setStock(new Map((stockData?.data ?? []).map((s) => [s.bookId, s])));
    })();
    return () => {
      live = false;
    };
  }, []);

  return (
    <PageContent>
      <PageTitle>
        <PageTitle.Header>Catalog</PageTitle.Header>
        <PageTitle.Actions>
          <Can op="POST /books">
            <Button variant="contained" startIcon={<Plus size={16} />} onClick={() => navigate("/staff/catalog/new")}>
              Add book
            </Button>
          </Can>
        </PageTitle.Actions>
      </PageTitle>

      <ListingTable.Container>
        <ListingTable>
          <ListingTable.Head>
            <ListingTable.Row>
              <ListingTable.Cell>Title</ListingTable.Cell>
              <ListingTable.Cell>Author</ListingTable.Cell>
              <ListingTable.Cell>Price</ListingTable.Cell>
              <ListingTable.Cell>Stock</ListingTable.Cell>
            </ListingTable.Row>
          </ListingTable.Head>
          <ListingTable.Body>
            {(books ?? []).map((book) => (
              <ListingTable.Row key={book.id} clickable onClick={() => navigate(`/staff/catalog/${book.id}`)}>
                <ListingTable.Cell>{book.title}</ListingTable.Cell>
                <ListingTable.Cell>{book.author}</ListingTable.Cell>
                <ListingTable.Cell>${book.price.toFixed(2)}</ListingTable.Cell>
                <ListingTable.Cell>{stock.get(book.id)?.quantityOnHand ?? "—"}</ListingTable.Cell>
              </ListingTable.Row>
            ))}
          </ListingTable.Body>
        </ListingTable>
        {books && books.length === 0 ? (
          <ListingTable.EmptyState title="No books yet" description="Add the first book to the catalog." />
        ) : null}
      </ListingTable.Container>
    </PageContent>
  );
}
