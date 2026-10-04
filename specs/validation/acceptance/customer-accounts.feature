Feature: Customer accounts and addresses

  @story-3
  Rule: A signed-in Shopper may create an account profile

    Scenario: Marco sets up his profile
      Given Marco is signed in as a Shopper for the first time
      When Marco saves his profile with name "Marco Diaz" and email "marco@example.com"
      Then Marco's profile shows name "Marco Diaz" and email "marco@example.com"

  @story-4 @negative
  Rule: Only a signed-in Shopper may manage their account

    @negative
    Scenario: A visitor who is not signed in cannot view a profile
      When a visitor who is not signed in tries to open the account page
      Then the visitor is sent to sign in

  @story-5
  Rule: A signed-in Shopper may manage their own shipping addresses

    Scenario: Marco adds a shipping address
      Given Marco is signed in as a Shopper
      When Marco adds a shipping address at "221B Baker Street, London"
      Then "221B Baker Street, London" appears among Marco's addresses

    Scenario: Marco edits an existing address
      Given Marco has a saved address at "221B Baker Street, London"
      When Marco edits that address to "10 Downing Street, London"
      Then Marco's address now reads "10 Downing Street, London"

    Scenario: Marco removes an address
      Given Marco has a saved address at "10 Downing Street, London"
      When Marco removes that address
      Then Marco has one fewer saved address than before

    @negative
    Scenario: A shopper cannot see another shopper's addresses
      Given Marco has a saved address at "221B Baker Street, London"
      And Elena is signed in as a different Shopper
      When Elena views her own saved addresses
      Then Elena does not see "221B Baker Street, London" among them
