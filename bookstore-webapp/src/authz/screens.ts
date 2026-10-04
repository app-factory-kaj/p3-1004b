// THIS IS THE ONLY FILE THAT KNOWS ABOUT SCREENS. Each row names the one
// operation that screen exists to perform; the gate follows from whether the
// caller may call it. Nothing here names a scope or role literal in JSX.
//
// Order is rail order; the first reachable row is the landing screen.
//
// Two screens — Catalog and BookDetail — are `public: true`: the wireframe's
// "Browse and buy" flow carries a `role "Shopper"` line, but both screens'
// `loads` operations are `security: []` in catalog-service's contract (see
// operations.gen.ts: "GET /books" and "GET /books/{bookId}" are both `kind:
// "public"`), and the issue's acceptance criteria are explicit that both must
// be reachable with no sign-in. Routed above the sign-in guard per
// thunder-authentication's §6.
//
// StaffCatalog and StaffBookForm have no protected READ operation to gate on:
// catalog-service's listBooks/getBook are intentionally public, and the
// contract names no "list books for management" operation. The only
// books-resource operations that carry a scope at all are the writes
// (POST/PUT/DELETE /books...), all `books:manage`. Gating on "PUT
// /books/{bookId}" is the closest fit the committed contracts offer to
// "reachable only to the Store Staff role" — reported in the build notes as a
// design-contract gap rather than worked around with an invented endpoint.
import { canCall } from "./core";
import { OPERATIONS, isOperationKey, type OperationKey } from "./operations.gen";

export interface ScreenRoute {
  readonly key: string;
  readonly label: string;
  readonly path: string;
  readonly loads: OperationKey | null;
  readonly public?: boolean;
}

export const SCREEN_ROUTES: readonly ScreenRoute[] = [
  // ── Browse and buy (Shopper) ──────────────────────────────────────────
  { key: "catalog", label: "Catalog", path: "/", loads: "GET /books", public: true },
  { key: "bookdetail", label: "Book Detail", path: "/books/:bookId", loads: "GET /books/{bookId}", public: true },
  { key: "cart", label: "Cart", path: "/cart", loads: "GET /me/cart" },
  { key: "checkout", label: "Checkout", path: "/checkout", loads: "GET /me/addresses" },
  {
    key: "orderconfirmation",
    label: "Order Confirmation",
    path: "/orders/:orderId/confirmation",
    loads: "GET /me/orders/{orderId}",
  },
  { key: "orders", label: "Orders", path: "/orders", loads: "GET /me/orders" },
  { key: "orderdetail", label: "Order Detail", path: "/orders/:orderId", loads: "GET /me/orders/{orderId}" },
  { key: "notifications", label: "Notifications", path: "/notifications", loads: "GET /me/notifications" },
  { key: "account", label: "Account", path: "/account", loads: "GET /me/profile" },
  { key: "addaddress", label: "Add Address", path: "/account/addresses/new", loads: "POST /me/addresses" },

  // ── Catalog and inventory management (Store Staff) ────────────────────
  { key: "staffcatalog", label: "Staff Catalog", path: "/staff/catalog", loads: "PUT /books/{bookId}" },
  { key: "staffbookform", label: "Staff Book Form", path: "/staff/catalog/:bookId", loads: "PUT /books/{bookId}" },
  { key: "staffinventory", label: "Staff Inventory", path: "/staff/inventory", loads: "GET /reservations" },
  {
    key: "staffstockform",
    label: "Staff Stock Form",
    path: "/staff/inventory/:bookId",
    loads: "PUT /stock-levels/{bookId}",
  },
];

for (const screen of SCREEN_ROUTES) {
  if (screen.loads !== null && !isOperationKey(screen.loads)) {
    throw new Error(
      `src/authz/screens.ts: screen "${screen.label}" loads "${screen.loads}", which ` +
        `no contract declares. Re-run \`npm run gen\`, or name the operation the ` +
        `way openapi.yaml spells it.`,
    );
  }
}

export function reachableScreens(
  scopes: ReadonlySet<string>,
  signedIn: boolean,
): readonly ScreenRoute[] {
  return SCREEN_ROUTES.filter((screen) => {
    if (screen.public) return true;
    if (screen.loads === null) return signedIn;
    return canCall(OPERATIONS[screen.loads], scopes, signedIn);
  });
}

export function hasScopedReach(scopes: ReadonlySet<string>, signedIn: boolean): boolean {
  return reachableScreens(scopes, signedIn).some((screen) => !screen.public && screen.loads !== null);
}
