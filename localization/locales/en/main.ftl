register-title = Create account
register-username = Username
register-password = Password
register-password-confirm = Confirm password
register-tos = I have read and agree to the Terms of Service
register-submit = Create account
register-error-authenticated = You are already logged in.
register-error-username = Invalid username. Username must be between 3 and 16 characters long and may contain letters, numbers and '_'
register-error-password = Invalid password. Your password must be between 8 and 72 characters long.
register-error-passwords-do-not-match = Passwords do not match
register-error-tos = You must agree to the Terms of Service
register-error-username-taken = Username already taken
register-error-user = Could not create user
register-success = Account successfully created

login-title = Login
login-username = Username
login-password = Password
login-submit = Login
login-error-authenticated = You are already logged in.
login-error-username = Invalid username
login-error-password = Invalid password
login-error-invalid = Invalid username or password
login-error-internal = Login failed
login-success = Welcome back, { $username }

logout-unauthenticated = You are not logged in
logout-success = You have been logged out
logout-error-all = Failed to log out all devices

account-delete-success = { $username }'s account has been deleted
account-delete-success-self = Your account has been deleted
account-delete-error-generic = Failed to delete account
account-delete-error-missing = Missing account name
account-delete-error-permission = You do not have permissions to delete { $username }'s account

search-user-title = Search for users
search-user-submit = Search
search-user-form-placeholder = Username

navbar-label-toggle = Toggle menu
navbar-label-main = Main navigation
navbar-cards = Cards
navbar-inventory = Inventory
navbar-shop = Shop
navbar-profile = Your profile
navbar-logout = Log out

theme-switcher-light = Switch to light theme
theme-switcher-dark = Switch to dark theme

language-switcher = Choose language

cards-error-not-found = Card not found
cards-error-collection-not-found = Collection not found

paginator-label-nav = Pagination
paginator-label-previous = Previous page
paginator-label-next = Next page

card-rarity-common = Common
card-rarity-rare = Rare
card-rarity-legendary = Legendary


card-search-placeholder = Search for cards
card-search-empty = No cards found
card-search-submit = Search
card-search-error-internal = Search failed

card-error-not-found = Card not found

collection-title = { $collection } Collection
collection-booster = { $count ->
    [one] { $count } card booster
    *[other] { $count } cards booster
}
collection-count = { $count ->
    [one] { $count } CARD
    *[other] { $count } CARDS
}
collection-id = NO. { $id }

collection-search-placeholder = Search for collections
collection-search-empty = No collections found
collection-search-submit = Search
collection-search-error-internal = Search failed

collection-error-not-found = Collection not found

notification-title = Notifications
notification-unread = unread
notification-loading = Loading notifications...
notification-none = You're all caught up
notification-error = Could not load notifications
notification-view-all = View all notifications
notification-mark-all-read = Mark all as read
notification-mark-read = Mark as read
notification-mark-unread = Mark as unread
notification-delete = Dismiss
notification-tabs-label = Filter notifications
notification-tab-all = All
notification-tab-unread = Unread
notification-empty-all = No notifications yet
notification-empty-unread = No unread notifications

notification-friend-request-title = Friend Request
notification-friend-request-desc = { $username } sent you a friend request
notification-info-title = { $title }
notification-info-desc = { $description }

catalog-title = Card catalog
catalog-tabs-label = Catalog sections
catalog-tab-cards = Cards
catalog-tab-collections = Collections
catalog-collection-count = { $count ->
    [one] { $count } collection
    *[other] { $count } collections
}
catalog-filter-collection = Collection
catalog-filter-collection-all = All collections
catalog-sort-number = Collection order

inventory-title = Your inventory
inventory-title-other = { $username }'s inventory
inventory-unauthenticated = You must be authenticated to access inventories
inventory-message-empty = Nothing to show here
inventory-message-user-not-found = User not found
inventory-message-fetch-fail = Cannot fetch user inventory
inventory-message-no-match = No cards in this inventory match your search
inventory-search-placeholder = Search this inventory
inventory-search-submit = Search
inventory-search-clear = Clear search
inventory-card-acquired = Acquired on
inventory-card-close = Close
inventory-card-acquired-first = First acquired on
inventory-card-acquired-last = Last acquired on
inventory-card-copies = Copies
inventory-filter-rarity = Rarity
inventory-filter-rarity-all = All rarities
inventory-filter-effect = Effect
inventory-filter-effect-all = All effects
inventory-filter-effect-none = No effect
inventory-filter-effect-foil = Foil
inventory-sort = Sort by
inventory-sort-relevance = Relevance
inventory-sort-recent = Last acquired
inventory-sort-oldest = First acquired
inventory-sort-rarity = Rarity
inventory-sort-count = Count
inventory-sort-name = Name (A-Z)
inventory-sort-name-desc = Name (Z-A)
inventory-tabs-label = Inventory sections
inventory-tab-cards = Cards
inventory-tab-boosters = Boosters
inventory-message-no-boosters = No boosters yet
inventory-booster-open-confirm = Open booster { $name }?
inventory-booster-open-yes = Yes
inventory-booster-open-no = No
inventory-booster-count = { $count ->
    [one] { $count } booster
    *[other] { $count } boosters
}
inventory-count = { $count ->
    [one] { $count } card
    *[other] { $count } cards
}
inventory-search-count = { $count ->
    [one] { $count } card found
    *[other] { $count } cards found
}

pack-opening-title = Pack Opening
pack-opening-open = Open pack
pack-opening-open-another = Open another pack
pack-opening-error-collection = Collection not found
pack-opening-error-empty = This booster has no cards
pack-opening-error-internal = Failed to open the pack
pack-opening-obtained = Cards obtained
pack-opening-back = Back to inventory
pack-opening-return = Return
pack-opening-open-again = Open again
pack-opening-remaining = { $count ->
    [0] No boosters left
    [one] { $count } booster left
    *[other] { $count } boosters left
}
pack-opening-error-booster = This booster is not in your inventory

shop-title = Shop
shop-balance = Balance
shop-login-hint = Log in to get coins and buy boosters
shop-boosters-title = Boosters
shop-boosters-empty = No boosters are on sale right now
shop-booster-buy = Buy
shop-price = { $amount } coins
shop-topup-title = Get coins
shop-topup-note = Demo checkout: no real payment is taken. Coins are credited to your account immediately.
shop-topup-price = €{ $euros }.{ $cents }
shop-topup-open = Buy more coins
shop-topup-success = { $amount } coins have been added to your balance
shop-error-unauthenticated = You must be logged in to get coins
shop-error-pack = Unknown coin pack
shop-error-funds = You do not have enough coins for this purchase
shop-error-quantity = Enter a quantity between 1 and { $max }
shop-error-internal = Failed to update your balance
shop-buy-close = Close
shop-buy-success = { $count ->
    [one] A { $collection } booster has been added to your inventory
   *[other] { $count } { $collection } boosters have been added to your inventory
}
shop-buy-view-inventory = View inventory
shop-buy-quantity = Quantity
shop-buy-total = Total
shop-coins = coins
