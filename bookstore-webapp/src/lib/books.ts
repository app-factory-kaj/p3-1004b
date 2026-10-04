// Shared join helper: several screens show a book's title/price beside data
// that only carries its bookId (cart items, order items, stock rows,
// reservations). catalog-service's contract has no "get books by ids" bulk
// endpoint, so the join is ONE list call (the max page size, 100) rather than
// one request per row — exactly the wireframe-to-Oxygen table's rule for a
// joined column. Fine for this catalog's size; a catalog past 100 titles
// would need the contract to grow an ids filter.
import { catalogApi } from "../api";
import type { components as CatalogComponents } from "../generated/catalog-service";

export type Book = CatalogComponents["schemas"]["Book"];

export async function fetchBookMap(): Promise<Map<string, Book>> {
  const { data } = await catalogApi.GET("/books", { params: { query: { limit: 100 } } });
  return new Map((data?.data ?? []).map((b) => [b.id, b]));
}
