Feature: Catalog browsing and management

  @story-1
  Rule: Anyone may search and browse the book catalog without signing in

    Scenario: A visitor searches for a book by title
      Given the catalog has a book titled "Designing Data-Intensive Applications" by "Martin Kleppmann"
      When a visitor who is not signed in searches the catalog for "Data-Intensive"
      Then "Designing Data-Intensive Applications" appears in the search results

    Scenario: A visitor browses the catalog without searching
      Given the catalog has at least one book
      When a visitor who is not signed in browses the catalog
      Then the visitor sees the catalog's books

  @story-2
  Rule: Anyone may view a book's details, including stock availability

    Scenario: A visitor views a book's detail page
      Given the catalog has a book titled "Clean Architecture" by "Robert Martin" priced at "$29.99"
      When a visitor who is not signed in opens "Clean Architecture"
      Then the visitor sees its title, author, description, price and stock availability

  @story-13 @negative
  Rule: Only Store Staff may create, update or remove a book

    Scenario: Store Staff adds a new book
      Given Priya is signed in as Store Staff
      When Priya adds a book titled "The Pragmatic Programmer" by "David Thomas" priced at "$34.99"
      Then "The Pragmatic Programmer" appears in the catalog

    @negative
    Scenario: A shopper cannot add a book
      Given Marco is signed in as a Shopper
      When Marco tries to add a book titled "Unauthorized Entry" by "Nobody"
      Then the catalog still does not contain "Unauthorized Entry"

    @negative
    Scenario: A visitor who is not signed in cannot remove a book
      Given the catalog has a book titled "Refactoring" by "Martin Fowler"
      When a visitor who is not signed in tries to remove "Refactoring"
      Then the catalog still contains "Refactoring"
