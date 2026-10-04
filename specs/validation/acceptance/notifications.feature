Feature: Order-status notifications

  @story-12
  Rule: A signed-in Shopper sees notifications about changes to their own order status

    Scenario: Marco is notified when his order is placed
      Given Marco just placed a new order
      When Marco opens his notifications
      Then he sees a notification that his order was placed

    Scenario: Marco is notified when his order ships
      Given Marco has a past order that just changed status to "shipped"
      When Marco opens his notifications
      Then he sees a notification that his order has shipped

    @negative
    Scenario: A shopper cannot see another shopper's notifications
      Given Marco just placed a new order
      And Elena is signed in as a different Shopper with no orders
      When Elena opens her own notifications
      Then Elena does not see a notification about Marco's order
