Feature: Checkout and order history

  @story-8
  Rule: A signed-in Shopper may check out their cart into an order

    Scenario: Marco checks out his cart
      Given Marco's cart contains 1 copy of "Clean Architecture" and he has a saved shipping address
      And "Clean Architecture" is in stock
      When Marco checks out
      Then a new order is placed for Marco containing 1 copy of "Clean Architecture"
      And Marco's cart is now empty

  @story-9 @negative
  Rule: Checkout refuses when an item in the cart is no longer in stock

    Scenario: An out-of-stock item blocks checkout
      Given Marco's cart contains 1 copy of "Designing Data-Intensive Applications" and he has a saved shipping address
      And "Designing Data-Intensive Applications" has zero stock remaining
      When Marco tries to check out
      Then no new order is placed for Marco
      And Marco's cart still contains "Designing Data-Intensive Applications"

  @story-10
  Rule: A signed-in Shopper may view their own past orders

    Scenario: Marco views his order history
      Given Marco has a past order for "Clean Architecture"
      When Marco opens his order history
      Then he sees that past order for "Clean Architecture"

    @negative
    Scenario: A shopper cannot see another shopper's orders
      Given Marco has a past order for "Clean Architecture"
      And Elena is signed in as a different Shopper with no past orders
      When Elena opens her own order history
      Then Elena does not see Marco's order for "Clean Architecture"

  @story-11
  Rule: A signed-in Shopper may view the current status of each of their orders

    Scenario: Marco checks an order's status
      Given Marco placed an order that is now "shipped"
      When Marco opens that order
      Then he sees its status is "shipped"
