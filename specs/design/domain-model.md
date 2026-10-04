# Domain Model

The bookstore's entities span six services, each owning the rows below that
belong to its own in-memory store; a `*Id` field is a foreign key held by the
owning service but resolved by calling the service that actually owns it.

```mermaid
erDiagram
    BOOK ||--o{ STOCK_LEVEL : "tracked by"
    CUSTOMER ||--o{ ADDRESS : has
    CUSTOMER ||--o{ CART : owns
    CART ||--o{ CART_ITEM : contains
    CUSTOMER ||--o{ ORDER : places
    ORDER ||--o{ ORDER_ITEM : contains
    ORDER ||--o{ RESERVATION : "backed by"
    ORDER ||--o{ NOTIFICATION : triggers
    BOOK ||--o{ CART_ITEM : "referenced by"
    BOOK ||--o{ ORDER_ITEM : "referenced by"
    BOOK ||--o{ RESERVATION : "reserved as"

    BOOK {
        string id
        string title
        string author
        string isbn
        string description
        number price
        string[] tags
    }
    STOCK_LEVEL {
        string bookId
        number quantityOnHand
        number quantityReserved
    }
    RESERVATION {
        string id
        string bookId
        string orderId
        number quantity
        string status
    }
    CUSTOMER {
        string id
        string name
        string email
    }
    ADDRESS {
        string id
        string customerId
        string line1
        string line2
        string city
        string region
        string postalCode
        string country
        boolean isDefault
    }
    CART {
        string id
        string customerId
        string updatedAt
    }
    CART_ITEM {
        string bookId
        number quantity
    }
    ORDER {
        string id
        string customerId
        string addressId
        string status
        number total
        string placedAt
    }
    ORDER_ITEM {
        string bookId
        number quantity
        number unitPrice
    }
    NOTIFICATION {
        string id
        string customerId
        string orderId
        string message
        boolean read
        string createdAt
    }
```

- **catalog-service** owns `BOOK`.
- **inventory-service** owns `STOCK_LEVEL` and `RESERVATION`.
- **customers-service** owns `CUSTOMER` and `ADDRESS`.
- **cart-service** owns `CART`/`CART_ITEM`, one cart per signed-in customer.
- **orders-service** owns `ORDER`/`ORDER_ITEM`, created from a cart at checkout.
- **notifications-service** owns `NOTIFICATION`, one row per order status change.

