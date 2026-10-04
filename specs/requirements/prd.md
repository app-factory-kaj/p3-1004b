# Online Bookstore Platform — PRD

## Problem Statement

Independent bookstores and online booksellers need a way to sell books over the
web without taking on a complex, monolithic e-commerce platform. Today, running
an online storefront means either paying for an expensive all-in-one commerce
suite or stitching together disconnected spreadsheets and manual processes for
tracking stock, taking orders, and telling customers what happened to their
purchase. There is no lightweight, purpose-built system that lets a bookseller
list books, take orders, track stock, and keep customers informed, while
letting shoppers browse and buy with a simple, modern storefront experience.

## Solution

An online bookstore platform: a React storefront where shoppers search and buy
books, backed by a set of independently deployable services that each own one
part of the business — catalog, customer accounts, shopping cart, checkout and
order history, inventory, and order-status notifications. Shoppers can browse
the catalog freely, create an account to check out, and track their orders.
Store staff keep the catalog and stock levels accurate. Every service keeps
its own data in memory and exposes its own documented API, so each can be
built, deployed and scaled independently.

## Actors

- **Shopper** — browses and searches the catalog, creates an account with
shipping addresses, manages a cart, checks out into an order, views order
history and status, and receives order-status notifications.
- **Store Staff** — manages the book catalog (create, update, remove books) and
manages stock levels and reservations in inventory. Provisioned by an
administrator rather than signing up themselves *assumed*.

## User Stories

1. As a Shopper, I want to search and browse the book catalog without signing
 in, so that I can discover books before creating an account.
2. As a Shopper, I want to view a book's details — title, author, description,
 price, and stock availability — so that I can decide whether to buy it.
3. As a Shopper, I want to create an account with a profile and one or more
 shipping addresses, so that I can check out quickly.
4. As a Shopper, I want to sign in via single sign-on, so that my cart, orders
 and notifications are tied to my account.
5. As a Shopper, I want to add, edit and remove my saved shipping addresses, so
 that I can ship orders to the right place.
6. As a Shopper, I want to add books to my cart, change quantities, and remove
 items, so that I can prepare what I want to order.
7. As a Shopper, I want to view my current cart contents and subtotal, so that
 I can review my order before checking out.
8. As a Shopper, I want to check out my cart into an order, so that I can
 purchase the books I selected.
9. As a Shopper, I want checkout to tell me clearly when an item in my cart is
 no longer in stock, so that I can adjust my cart and still complete my
 purchase.
10. As a Shopper, I want to view my past orders, so that I can track what I've
 bought.
11. As a Shopper, I want to view the current status of each of my orders (for
 example placed, confirmed, shipped, delivered, or cancelled), so that I
 know where my purchase stands.
12. As a Shopper, I want to see notifications about changes to my order
 status in the storefront, so that I stay informed without checking each
 order manually.
13. As a Store Staff member, I want to create, update and remove books in the
 catalog, so that what shoppers see stays accurate.
14. As a Store Staff member, I want to view and adjust the stock level of any
 book, so that inventory reflects what is really on the shelf.
15. As a Store Staff member, I want to see which stock is currently reserved
 against in-flight orders, so that I can reconcile available inventory.

## Product Decisions

- **Sign-in**: single sign-on (SSO) for all accounts, per the organization's
standard. Catalog browsing and search are open to anonymous visitors; an
account is required to use the cart, check out, view order history, or see
notifications *assumed*.
- **Store Staff actor**: a dedicated staff role manages the catalog and
inventory; shoppers never manage either *assumed*.
- **Payment at checkout**: checkout is simulated — it records the order and a
payment method reference without calling a real payment gateway. No payment
provider is integrated *assumed*.
- **Order-status notifications**: delivered in-app only, viewed in the
storefront. No email or SMS channel is integrated *assumed*.
- **Service architecture**: the platform is built as six independent services
— catalog (books CRUD and search), customers (accounts and addresses), cart
(per-customer carts), orders (checkout and order history), inventory (stock
levels and reservations), and notifications (order status messages) — plus
one React storefront web app that uses all of them. This decomposition, and
that each service keeps its own in-memory store with no database, is a
stakeholder requirement, not an assumption.
- **Reference conventions**: service structure and conventions follow the
organization's `app-factory-kaj/e2e-reference` example. Stakeholder
requirement, not an assumption.

## Out of Scope

- Real payment processing or integration with a payment provider.
- Email or SMS delivery of notifications.
- Order cancellation, returns, or refunds.
- Multi-currency pricing or international shipping.
- Book reviews, ratings, or personalized recommendations.
- Data persistence across restarts — each service's in-memory store is reset
when the service restarts, by design.
- A separate administration console beyond the Store Staff capabilities
described above (e.g. staff account management, reporting/analytics).

## Open Questions

1. Should Store Staff sign in through the same SSO used by shoppers but with a
 different role/group, or does staff access need a separate mechanism?
 *(leaning toward same SSO with a distinct role — flagged here because it
 affects the design's security model)*

## Further Notes

The idea behind this project specified implementation-level constraints
directly — six separate Go microservices, each with its own OpenAPI spec and
an in-memory (non-database) store, following the conventions of the
`app-factory-kaj/e2e-reference` project. These are carried into the Product
Decisions above as explicit stakeholder requirements rather than inferred
choices, and the concrete architecture (component boundaries, contracts, and
persistence approach) is worked out at design time.