screen Catalog "Browse and search books"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  row
    search "Search books..."
    select "All tags"
  table "Title | Author | Price | Availability" -> BookDetail
    row "The Go Programming Language | Alan Donovan | $39.99 | In stock"
    row "Clean Architecture | Robert Martin | $29.99 | In stock"
    row "Designing Data-Intensive Applications | Martin Kleppmann | $44.99 | Low stock"

screen BookDetail "A single book's details"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  card "Designing Data-Intensive Applications"
    text "by Martin Kleppmann"
    text "A deep dive into the systems behind modern data infrastructure."
    text "$44.99"
    badge "Low stock" warning
    button "Add to cart" primary -> Cart

screen Cart "The signed-in Shopper's cart"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "My Cart"
  table "Book | Quantity | Unit Price | Line Total"
    row "Designing Data-Intensive Applications | 1 | $44.99 | $44.99"
    row "Clean Architecture | 2 | $29.99 | $59.98"
  row
    right
    text "Subtotal: $104.97"
  row
    right
    button "Checkout" primary -> Checkout

screen Checkout "Review shipping and place the order"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "Checkout"
  select "Shipping address"
  input "Payment method reference"
  row
    right
    button "Place order" primary -> OrderConfirmation

screen OrderConfirmation "The order was placed"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  card "Order placed"
    text "Order #10234 — Total $104.97"
    badge "placed" info
  button "View my orders" primary -> Orders

screen Orders "The signed-in Shopper's order history"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "My Orders"
  table "Order # | Placed | Status | Total" -> OrderDetail
    row "10234 | 2026-10-01 | placed | $104.97"
    row "10198 | 2026-09-20 | delivered | $69.98"

screen OrderDetail "One order's items and status"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "Order #10234"
  badge "placed" info
  table "Book | Quantity | Unit Price"
    row "Designing Data-Intensive Applications | 1 | $44.99"
    row "Clean Architecture | 2 | $29.98"

screen Notifications "The signed-in Shopper's order-status notifications"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "Notifications"
  list "Order #10234 is now placed | Order #10198 was delivered"

screen Account "The signed-in Shopper's profile and addresses"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "My Account"
  card "Profile"
    input "Full name"
    input "Email"
    button "Save profile"
  heading "Shipping Addresses"
  table "Address | City | Default"
    row "221B Baker Street | London | Yes"
  button "Add address" primary -> AddAddress

screen AddAddress "Add a new shipping address"
  navbar "Bookstore | My Cart -> Cart | My Orders -> Orders | Notifications -> Notifications | Sign out"
  heading "Add Shipping Address"
  input "Address line 1"
  input "Address line 2"
  input "City"
  input "Region"
  input "Postal code"
  input "Country"
  checkbox "Set as default"
  row
    right
    button "Save address" primary

screen StaffCatalog "Store Staff manage the book catalog"
  navbar "Bookstore Staff | Catalog -> StaffCatalog | Inventory -> StaffInventory | Sign out"
  row
    heading "Catalog"
    right
    button "Add book" primary -> StaffBookForm
  table "Title | Author | Price | Stock" -> StaffBookForm
    row "The Go Programming Language | Alan Donovan | $39.99 | 42"
    row "Clean Architecture | Robert Martin | $29.99 | 15"

screen StaffBookForm "Create or edit a book"
  navbar "Bookstore Staff | Catalog -> StaffCatalog | Inventory -> StaffInventory | Sign out"
  heading "Edit Book"
  input "Title"
  input "Author"
  input "ISBN"
  textarea "Description"
  input "Price"
  row
    right
    button "Cancel" -> StaffCatalog
    button "Save book" primary -> StaffCatalog

screen StaffInventory "Store Staff manage stock levels and reservations"
  navbar "Bookstore Staff | Catalog -> StaffCatalog | Inventory -> StaffInventory | Sign out"
  heading "Stock Levels"
  table "Book | On Hand | Reserved" -> StaffStockForm
    row "The Go Programming Language | 42 | 3"
    row "Clean Architecture | 15 | 2"
  heading "Reservations"
  table "Order # | Book | Quantity | Status"
    row "10234 | Designing Data-Intensive Applications | 1 | held"

screen StaffStockForm "Adjust a book's on-hand quantity"
  navbar "Bookstore Staff | Catalog -> StaffCatalog | Inventory -> StaffInventory | Sign out"
  heading "Adjust Stock — The Go Programming Language"
  input "On-hand quantity"
  row
    right
    button "Cancel" -> StaffInventory
    button "Save" primary -> StaffInventory

flow "Browse and buy"
  role "Shopper"
  description "A shopper browses the catalog, builds a cart, checks out, and tracks the order"
  Catalog
  BookDetail
  Cart
  Checkout
  OrderConfirmation
  Orders
  OrderDetail
  Notifications
  Account
  AddAddress

flow "Catalog and inventory management"
  role "Store Staff"
  description "Store staff keep the catalog and stock levels accurate"
  StaffCatalog
  StaffBookForm
  StaffInventory
  StaffStockForm
