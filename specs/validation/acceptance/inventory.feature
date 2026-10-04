Feature: Stock levels and reservations

  @story-14 @negative
  Rule: Only Store Staff may adjust a book's stock level

    Scenario: Store Staff restocks a book
      Given "The Go Programming Language" has 10 units on hand
      And Priya is signed in as Store Staff
      When Priya sets the on-hand quantity of "The Go Programming Language" to 25
      Then "The Go Programming Language" shows 25 units on hand

    @negative
    Scenario: A shopper cannot adjust stock
      Given "The Go Programming Language" has 25 units on hand
      And Marco is signed in as a Shopper
      When Marco tries to set the on-hand quantity of "The Go Programming Language" to 1000
      Then "The Go Programming Language" still shows 25 units on hand

  @story-15 @negative
  Rule: Only Store Staff may view which stock is currently reserved against in-flight orders

    Scenario: Store Staff reviews reservations
      Given an order holds a reservation for 1 copy of "Clean Architecture"
      And Priya is signed in as Store Staff
      When Priya views current reservations
      Then she sees the reservation for 1 copy of "Clean Architecture"

    @negative
    Scenario: A shopper cannot view every reservation
      Given an order holds a reservation for 1 copy of "Clean Architecture"
      And Marco is signed in as a Shopper
      When Marco tries to view every current reservation
      Then Marco is refused
