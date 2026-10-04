Feature: Shopping cart

  @story-6
  Rule: A signed-in Shopper may add, change the quantity of, and remove books in their own cart

    Scenario: Marco adds a book to his cart
      Given Marco is signed in as a Shopper with an empty cart
      When Marco adds 2 copies of "Clean Architecture" to his cart
      Then Marco's cart contains 2 copies of "Clean Architecture"

    Scenario: Marco changes a cart item's quantity
      Given Marco's cart contains 2 copies of "Clean Architecture"
      When Marco changes the quantity of "Clean Architecture" in his cart to 3
      Then Marco's cart contains 3 copies of "Clean Architecture"

    Scenario: Marco removes a book from his cart
      Given Marco's cart contains 3 copies of "Clean Architecture"
      When Marco removes "Clean Architecture" from his cart
      Then Marco's cart no longer contains "Clean Architecture"

  @story-7
  Rule: A signed-in Shopper may view their current cart contents and subtotal

    Scenario: Marco reviews his cart before checkout
      Given Marco's cart contains 1 copy of "Clean Architecture" priced at "$29.99"
      When Marco views his cart
      Then Marco sees "Clean Architecture" with quantity 1 and a subtotal of "$29.99"

    @negative
    Scenario: A shopper cannot see another shopper's cart
      Given Marco's cart contains 1 copy of "Clean Architecture"
      And Elena is signed in as a different Shopper with an empty cart
      When Elena views her own cart
      Then Elena's cart does not contain "Clean Architecture"
