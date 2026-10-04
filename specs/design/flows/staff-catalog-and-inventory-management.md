# Staff catalog and inventory management

A Store Staff member keeps the catalog and stock levels accurate.

```mermaid
sequenceDiagram
    actor Staff as Store Staff
    participant web as bookstore-webapp
    participant catalog as catalog-service
    participant inventory as inventory-service

    Staff->>web: sign in
    Staff->>web: create or edit a book
    web->>catalog: create/update book
    Staff->>web: adjust stock level
    web->>inventory: set stock quantity
    Staff->>web: view reservations
    web->>inventory: list reservations
```

