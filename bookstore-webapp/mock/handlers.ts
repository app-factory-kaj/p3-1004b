// Mock mode's stand-in for all six sibling services. One handler per
// operation in each service's openapi.yaml, paths prefixed exactly as the app
// calls them through nginx (/api for catalog-service, /api/<name>/ for the
// other five). State lives in module scope, reset on every full page load —
// see react-webapp's mock-mode.md.
//
// NO scope check here: mock/authz/gateway.ts is the gateway layer and answers
// every refusal from the contract before a request reaches these handlers.
// What a handler owes is its PATH's reach — a /me/… handler answers the
// caller's own rows — and since none of these six services' response schemas
// expose an owner id, there is exactly one mock "self" and nothing to filter.
import { http, HttpResponse } from "msw";
import type { components as Catalog } from "../src/generated/catalog-service";
import type { components as Customers } from "../src/generated/customers-service";
import type { components as CartTypes } from "../src/generated/cart-service";
import type { components as Orders } from "../src/generated/orders-service";
import type { components as Inventory } from "../src/generated/inventory-service";
import type { components as Notifications } from "../src/generated/notifications-service";

type Book = Catalog["schemas"]["Book"];
type Customer = Customers["schemas"]["Customer"];
type Address = Customers["schemas"]["Address"];
type Cart = CartTypes["schemas"]["Cart"];
type Order = Orders["schemas"]["Order"];
type StockLevel = Inventory["schemas"]["StockLevel"];
type Reservation = Inventory["schemas"]["Reservation"];
type Notification = Notifications["schemas"]["Notification"];

// ---------------------------------------------------------------------------
// Seed data — the wireframe's own rows (node scripts/seed.mjs wireframes.dsl),
// cross-linked so Cart/Orders/StaffInventory agree with one another.
// ---------------------------------------------------------------------------

let books: Book[] = [
  {
    id: "book-1",
    title: "The Go Programming Language",
    author: "Alan Donovan",
    isbn: "978-0134190440",
    description: "The authoritative book on the Go programming language.",
    price: 39.99,
    tags: ["go", "programming"],
  },
  {
    id: "book-2",
    title: "Clean Architecture",
    author: "Robert Martin",
    isbn: "978-0134494166",
    description: "A craftsman's guide to software structure and design.",
    price: 29.99,
    tags: ["architecture", "software"],
  },
  {
    id: "book-3",
    title: "Designing Data-Intensive Applications",
    author: "Martin Kleppmann",
    isbn: "978-1449373320",
    description: "A deep dive into the systems behind modern data infrastructure.",
    price: 44.99,
    tags: ["data", "systems"],
  },
];

let stockLevels: StockLevel[] = [
  { bookId: "book-1", quantityOnHand: 42, quantityReserved: 3 },
  { bookId: "book-2", quantityOnHand: 15, quantityReserved: 2 },
  { bookId: "book-3", quantityOnHand: 6, quantityReserved: 1 },
];

let reservations: Reservation[] = [
  { id: "res-1", bookId: "book-3", orderId: "10234", quantity: 1, status: "held" },
];

let cart: Cart = {
  id: "cart-1",
  items: [
    { bookId: "book-3", quantity: 1 },
    { bookId: "book-2", quantity: 2 },
  ],
  subtotal: 104.97,
  updatedAt: "2026-10-01T09:00:00Z",
};

let orders: Order[] = [
  {
    id: "10234",
    status: "placed",
    total: 104.97,
    addressId: "addr-1",
    items: [
      { bookId: "book-3", quantity: 1, unitPrice: 44.99 },
      { bookId: "book-2", quantity: 2, unitPrice: 29.99 },
    ],
    placedAt: "2026-10-01",
  },
  {
    id: "10198",
    status: "delivered",
    total: 69.98,
    addressId: "addr-1",
    items: [
      { bookId: "book-1", quantity: 1, unitPrice: 39.99 },
      { bookId: "book-2", quantity: 1, unitPrice: 29.99 },
    ],
    placedAt: "2026-09-20",
  },
];
let nextOrderSeq = 10235;

let profile: Customer = { id: "cust-1", name: "Jane Shopper", email: "jane@example.test" };

let addresses: Address[] = [
  {
    id: "addr-1",
    line1: "221B Baker Street",
    line2: undefined,
    city: "London",
    region: "Greater London",
    postalCode: "NW1 6XE",
    country: "UK",
    isDefault: true,
  },
];
let nextAddressSeq = 2;

let notifications: Notification[] = [
  { id: "notif-1", orderId: "10234", message: "Order #10234 is now placed", read: false, createdAt: "2026-10-01T10:00:00Z" },
  { id: "notif-2", orderId: "10198", message: "Order #10198 was delivered", read: true, createdAt: "2026-09-25T10:00:00Z" },
];

function err(code: number, message: string) {
  return HttpResponse.json({ code, message }, { status: code });
}

export const handlers = [
  // ---- catalog-service (primary, /api) -----------------------------------
  http.get("/api/books", ({ request }) => {
    const url = new URL(request.url);
    const q = url.searchParams.get("q")?.toLowerCase();
    const tag = url.searchParams.get("tag");
    let rows = books;
    if (q) {
      rows = rows.filter(
        (b) =>
          b.title.toLowerCase().includes(q) ||
          b.author.toLowerCase().includes(q) ||
          (b.description ?? "").toLowerCase().includes(q),
      );
    }
    if (tag) rows = rows.filter((b) => (b.tags ?? []).includes(tag));
    return HttpResponse.json({ count: rows.length, next: null, previous: null, data: rows });
  }),
  http.post("/api/books", async ({ request }) => {
    const input = (await request.json()) as Catalog["schemas"]["BookInput"];
    if (!input?.title || !input?.author || typeof input?.price !== "number") {
      return err(400, "title, author and price are required");
    }
    const created: Book = { id: `book-${books.length + 1}-${Date.now()}`, ...input };
    books = [...books, created];
    return HttpResponse.json(created, { status: 201 });
  }),
  http.get("/api/books/:bookId", ({ params }) => {
    const book = books.find((b) => b.id === params.bookId);
    return book ? HttpResponse.json(book) : err(404, "No such book");
  }),
  http.put("/api/books/:bookId", async ({ params, request }) => {
    const input = (await request.json()) as Catalog["schemas"]["BookInput"];
    const idx = books.findIndex((b) => b.id === params.bookId);
    if (idx === -1) return err(404, "No such book");
    if (!input?.title || !input?.author || typeof input?.price !== "number") {
      return err(400, "title, author and price are required");
    }
    const updated: Book = { ...books[idx], ...input, id: books[idx].id };
    books = books.map((b, i) => (i === idx ? updated : b));
    return HttpResponse.json(updated);
  }),
  http.delete("/api/books/:bookId", ({ params }) => {
    const before = books.length;
    books = books.filter((b) => b.id !== params.bookId);
    return before === books.length ? err(404, "No such book") : new HttpResponse(null, { status: 204 });
  }),

  // ---- customers-service (/api/customers-service/) -----------------------
  http.get("/api/customers-service/me/profile", () =>
    profile ? HttpResponse.json(profile) : err(404, "No profile yet"),
  ),
  http.put("/api/customers-service/me/profile", async ({ request }) => {
    const input = (await request.json()) as Customers["schemas"]["CustomerInput"];
    if (!input?.name || !input?.email) return err(400, "name and email are required");
    profile = { id: profile.id, ...input };
    return HttpResponse.json(profile);
  }),
  http.get("/api/customers-service/me/addresses", () =>
    HttpResponse.json({ count: addresses.length, next: null, previous: null, data: addresses }),
  ),
  http.post("/api/customers-service/me/addresses", async ({ request }) => {
    const input = (await request.json()) as Customers["schemas"]["AddressInput"];
    if (!input?.line1 || !input?.city || !input?.region || !input?.postalCode || !input?.country) {
      return err(400, "line1, city, region, postalCode and country are required");
    }
    const created: Address = { id: `addr-${nextAddressSeq++}`, ...input };
    if (created.isDefault) addresses = addresses.map((a) => ({ ...a, isDefault: false }));
    addresses = [...addresses, created];
    return HttpResponse.json(created, { status: 201 });
  }),
  http.put("/api/customers-service/me/addresses/:addressId", async ({ params, request }) => {
    const idx = addresses.findIndex((a) => a.id === params.addressId);
    if (idx === -1) return err(404, "No such address");
    const input = (await request.json()) as Customers["schemas"]["AddressInput"];
    const updated: Address = { ...addresses[idx], ...input };
    addresses = addresses.map((a, i) => (i === idx ? updated : a));
    return HttpResponse.json(updated);
  }),
  http.delete("/api/customers-service/me/addresses/:addressId", ({ params }) => {
    const before = addresses.length;
    addresses = addresses.filter((a) => a.id !== params.addressId);
    return before === addresses.length ? err(404, "No such address") : new HttpResponse(null, { status: 204 });
  }),

  // ---- cart-service (/api/cart-service/) ----------------------------------
  http.get("/api/cart-service/me/cart", () => HttpResponse.json(cart)),
  http.delete("/api/cart-service/me/cart", () => {
    cart = { ...cart, items: [], subtotal: 0, updatedAt: new Date().toISOString() };
    return new HttpResponse(null, { status: 204 });
  }),
  http.post("/api/cart-service/me/cart/items", async ({ request }) => {
    const input = (await request.json()) as { bookId?: string; quantity?: number };
    if (!input?.bookId || !input?.quantity || input.quantity < 1) return err(400, "bookId and quantity are required");
    const book = books.find((b) => b.id === input.bookId);
    const existing = cart.items.find((i) => i.bookId === input.bookId);
    const items = existing
      ? cart.items.map((i) => (i.bookId === input.bookId ? { ...i, quantity: i.quantity + (input.quantity ?? 1) } : i))
      : [...cart.items, { bookId: input.bookId, quantity: input.quantity }];
    cart = { ...cart, items, subtotal: subtotalFor(items), updatedAt: new Date().toISOString() };
    void book;
    return HttpResponse.json(cart);
  }),
  http.put("/api/cart-service/me/cart/items/:bookId", async ({ params, request }) => {
    const input = (await request.json()) as { quantity?: number };
    if (!input?.quantity || input.quantity < 1) return err(400, "quantity must be at least 1");
    if (!cart.items.some((i) => i.bookId === params.bookId)) return err(404, "No such item in cart");
    const items = cart.items.map((i) => (i.bookId === params.bookId ? { ...i, quantity: input.quantity! } : i));
    cart = { ...cart, items, subtotal: subtotalFor(items), updatedAt: new Date().toISOString() };
    return HttpResponse.json(cart);
  }),
  http.delete("/api/cart-service/me/cart/items/:bookId", ({ params }) => {
    if (!cart.items.some((i) => i.bookId === params.bookId)) return err(404, "No such item in cart");
    const items = cart.items.filter((i) => i.bookId !== params.bookId);
    cart = { ...cart, items, subtotal: subtotalFor(items), updatedAt: new Date().toISOString() };
    return HttpResponse.json(cart);
  }),

  // ---- orders-service (/api/orders-service/) ------------------------------
  http.get("/api/orders-service/me/orders", () =>
    HttpResponse.json({ count: orders.length, next: null, previous: null, data: orders }),
  ),
  http.post("/api/orders-service/me/orders", async ({ request }) => {
    const input = (await request.json()) as { addressId?: string; paymentMethodRef?: string };
    if (!input?.addressId || !input?.paymentMethodRef) {
      return err(400, "addressId and paymentMethodRef are required");
    }
    if (cart.items.length === 0) return err(400, "Your cart is empty");
    // Story 9: refuse when a cart line exceeds available stock (onHand - reserved).
    for (const item of cart.items) {
      const stock = stockLevels.find((s) => s.bookId === item.bookId);
      const available = stock ? stock.quantityOnHand - stock.quantityReserved : 0;
      if (item.quantity > available) {
        const title = books.find((b) => b.id === item.bookId)?.title ?? item.bookId;
        return err(400, `Insufficient stock for "${title}": ${available} available, ${item.quantity} requested.`);
      }
    }
    const id = String(nextOrderSeq++);
    const items = cart.items.map((i) => ({
      bookId: i.bookId,
      quantity: i.quantity,
      unitPrice: books.find((b) => b.id === i.bookId)?.price ?? 0,
    }));
    const created: Order = {
      id,
      status: "placed",
      total: cart.subtotal,
      addressId: input.addressId,
      items,
      placedAt: new Date().toISOString().slice(0, 10),
    };
    orders = [created, ...orders];
    // Reserve stock for every line and clear the cart, mirroring
    // browse-and-purchase.md's "orders reserves stock" step.
    for (const item of items) {
      stockLevels = stockLevels.map((s) =>
        s.bookId === item.bookId ? { ...s, quantityReserved: s.quantityReserved + item.quantity } : s,
      );
      reservations = [
        ...reservations,
        { id: `res-${reservations.length + 1}`, bookId: item.bookId, orderId: id, quantity: item.quantity, status: "held" },
      ];
    }
    notifications = [
      { id: `notif-${notifications.length + 1}`, orderId: id, message: `Order #${id} is now placed`, read: false, createdAt: new Date().toISOString() },
      ...notifications,
    ];
    cart = { ...cart, items: [], subtotal: 0, updatedAt: new Date().toISOString() };
    return HttpResponse.json(created, { status: 201 });
  }),
  http.get("/api/orders-service/me/orders/:orderId", ({ params }) => {
    const order = orders.find((o) => o.id === params.orderId);
    return order ? HttpResponse.json(order) : err(404, "No such order");
  }),

  // ---- inventory-service (/api/inventory-service/) ------------------------
  http.get("/api/inventory-service/stock-levels", () =>
    HttpResponse.json({ count: stockLevels.length, next: null, previous: null, data: stockLevels }),
  ),
  http.get("/api/inventory-service/stock-levels/:bookId", ({ params }) => {
    const level = stockLevels.find((s) => s.bookId === params.bookId);
    return level ? HttpResponse.json(level) : err(404, "No stock record for this book");
  }),
  http.put("/api/inventory-service/stock-levels/:bookId", async ({ params, request }) => {
    const input = (await request.json()) as { quantityOnHand?: number };
    if (typeof input?.quantityOnHand !== "number" || input.quantityOnHand < 0) {
      return err(400, "quantityOnHand must be a non-negative number");
    }
    const idx = stockLevels.findIndex((s) => s.bookId === params.bookId);
    const updated: StockLevel = idx === -1
      ? { bookId: String(params.bookId), quantityOnHand: input.quantityOnHand, quantityReserved: 0 }
      : { ...stockLevels[idx], quantityOnHand: input.quantityOnHand };
    stockLevels = idx === -1 ? [...stockLevels, updated] : stockLevels.map((s, i) => (i === idx ? updated : s));
    return HttpResponse.json(updated);
  }),
  http.get("/api/inventory-service/reservations", () =>
    HttpResponse.json({ count: reservations.length, next: null, previous: null, data: reservations }),
  ),
  http.post("/api/inventory-service/reservations", async ({ request }) => {
    const input = (await request.json()) as { bookId?: string; orderId?: string; quantity?: number };
    if (!input?.bookId || !input?.orderId || !input?.quantity || input.quantity < 1) {
      return err(400, "bookId, orderId and quantity are required");
    }
    const stock = stockLevels.find((s) => s.bookId === input.bookId);
    const available = stock ? stock.quantityOnHand - stock.quantityReserved : 0;
    if (input.quantity > available) return err(400, "Insufficient stock");
    const created: Reservation = {
      id: `res-${reservations.length + 1}`,
      bookId: input.bookId,
      orderId: input.orderId,
      quantity: input.quantity,
      status: "held",
    };
    reservations = [...reservations, created];
    stockLevels = stockLevels.map((s) =>
      s.bookId === input.bookId ? { ...s, quantityReserved: s.quantityReserved + input.quantity! } : s,
    );
    return HttpResponse.json(created, { status: 201 });
  }),
  http.post("/api/inventory-service/reservations/:reservationId/release", ({ params }) => {
    const idx = reservations.findIndex((r) => r.id === params.reservationId);
    if (idx === -1) return err(404, "No such reservation");
    const released: Reservation = { ...reservations[idx], status: "released" };
    reservations = reservations.map((r, i) => (i === idx ? released : r));
    stockLevels = stockLevels.map((s) =>
      s.bookId === released.bookId ? { ...s, quantityReserved: Math.max(0, s.quantityReserved - released.quantity) } : s,
    );
    return HttpResponse.json(released);
  }),

  // ---- notifications-service (/api/notifications-service/) ---------------
  http.get("/api/notifications-service/me/notifications", () =>
    HttpResponse.json({ count: notifications.length, next: null, previous: null, data: notifications }),
  ),
  http.post("/api/notifications-service/me/notifications/:notificationId/read", ({ params }) => {
    const idx = notifications.findIndex((n) => n.id === params.notificationId);
    if (idx === -1) return err(404, "No such notification");
    const updated = { ...notifications[idx], read: true };
    notifications = notifications.map((n, i) => (i === idx ? updated : n));
    return HttpResponse.json(updated);
  }),
  http.post("/api/notifications-service/notifications", async ({ request }) => {
    const input = (await request.json()) as { customerId?: string; orderId?: string; message?: string };
    if (!input?.customerId || !input?.orderId || !input?.message) {
      return err(400, "customerId, orderId and message are required");
    }
    const created: Notification = {
      id: `notif-${notifications.length + 1}`,
      orderId: input.orderId,
      message: input.message,
      read: false,
      createdAt: new Date().toISOString(),
    };
    notifications = [created, ...notifications];
    return HttpResponse.json(created, { status: 201 });
  }),
];

function subtotalFor(items: { bookId: string; quantity: number }[]): number {
  return items.reduce((sum, i) => sum + (books.find((b) => b.id === i.bookId)?.price ?? 0) * i.quantity, 0);
}
