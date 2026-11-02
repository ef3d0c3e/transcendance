register-title = Create account
register-username = Username
register-password = Password
register-password-confirm = Confirm password
register-password-hint = 8 to 72 characters.
register-tos = I have read and agree to the
register-tos-link = Terms of Service
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
login-type = Log in using
login-username = Username
login-password = Password
login-submit = Login
login-error-authenticated = You are already logged in.
login-error-type = Choose username or email
login-error-username = Invalid username
login-error-password = Invalid password
login-error-invalid = Invalid login credentials
login-error-internal = Login failed
login-error-rate-limited = Too many failed attempts, try again in { $seconds ->
    [one] { $seconds } second
    *[other] { $seconds } seconds
}.
login-success = Welcome back, { $username }
login-popup-loading = Loading the form…
login-popup-error = Unable to load the form. Check your connection and try again.
login-popup-close = Close

logout-unauthenticated = You are not logged in
logout-success = You have been logged out
logout-error-all = Failed to log out all devices

account-delete-error-generic = Failed to delete account
account-delete-error-last-admin = You are the only administrator: make someone else an administrator before deleting your account.
account-delete-title = Delete account
account-email = Email
account-error-email = A valid email address is required, e.g. name@example.org (254 characters maximum)
account-error-email-taken = Email already in use

search-user-title = Search for users
search-user-submit = Search
search-user-form-placeholder = Username
search-user-count = { $count ->
    [one] { $count } user
    *[other] { $count } users
}
search-user-empty = No users found
search-user-error-internal = Search failed
search-user-filter-joined = Joined
search-user-filter-joined-all = Any time
search-user-filter-joined-day = Past day
search-user-filter-joined-week = Past week
search-user-filter-joined-month = Past month
search-user-filter-joined-year = Past year
search-user-sort-relevance = Relevance
search-user-sort-name = Name (A-Z)
search-user-sort-name-desc = Name (Z-A)
search-user-sort-newest = Newest members
search-user-sort-oldest = Oldest members
search-user-sort-cards = Most cards
search-user-joined = Joined:
search-user-cards = Cards:
search-user-filter-status = Status
search-user-filter-status-all = Anyone
search-user-filter-status-online = Online
user-status-online = Online
user-status-offline = Offline
user-status-last-seen = Last seen:

friends-title = Friends list
friends-add = Add Friend
friends-cancel = Cancel
friends-accept = Accept
friends-deny = Ignore
friends-unfriend = Unfriend
friends-block = Block user
friends-unblock = Unblock
friends-unknown-error = Something went wrong
friends-blocked-error = Can't add this user as a friend
friends-unblock-error = Unblock this user first
friends-not-found-error = This user doesn't exist
friends-yourself-error = You can't add yourself as a friend
friends-list = Show
friends-list-friends = Friends
friends-list-sent = Sent requests
friends-list-received = Received requests
friends-empty-friends = You have no friends yet
friends-empty-sent = You haven't sent any friend request
friends-empty-received = You have no friend request

navbar-label-toggle = Toggle menu
navbar-label-close = Close menu
navbar-label-main = Main navigation
navbar-skip = Skip to content
navbar-users = Users
navbar-inventory = Inventory
navbar-friends = Friends
navbar-profile = Your profile
navbar-logout = Log out
header-balance = Balance: { $amount }

theme-switcher-light = Switch to light theme
theme-switcher-dark = Switch to dark theme

language-switcher = Choose language
language-switcher-current = Language: English

cards-error-not-found = Card not found
cards-error-collection-not-found = Collection not found

paginator-label-nav = Pagination
paginator-label-nav-bottom = Pagination, bottom of the list
paginator-label-previous = Previous page
paginator-label-next = Next page

card-rarity-common = Common
card-rarity-rare = Rare
card-rarity-legendary = Legendary

card-search-placeholder = Search for cards
card-filter-placeholder = Filter by card or collection name
card-filter-no-match = No card matches this filter
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

profile-username = Username
profile-bio = Bio
profile-birthdate = Birth date
profile-contact = Contact information
profile-new-password = New password
profile-confirm-password = Confirm password
profile-current-password = Current password
profile-username-hint = 3–16 characters: letters, digits or underscores.
profile-title = Profile
profile-error-user = User doesn't exist
profile-error-username = Invalid username. Username must be between 3 and 16 characters long and may contain letters, numbers and '_'
profile-error-username-taken = Username already taken
profile-error-bio = Bio must be no more than 244 bytes long
profile-error-birthdate = Birthdate must be between year 0 and the present day
profile-error-contactinfo = Contact must be no more than 256 bytes long
profile-error-password = Password does not match the current password
profile-error-new_password = Invalid password. Your password must be between 8 and 72 characters long
profile-error-passwords-do-not-match = Passwords do not match
profile-error-password-missing = Must provide this field to change password
profile-empty-field = Not specified
profile-submit = Save
profile-saved = Profile saved
profile-error-network = Unable to contact the server

error-unexpected = An error occurred
error-unauthorized = You must be logged in to access this resource
error-not-found = This page does not exist

error-title = Error

profile-avatar-label = Profile picture
profile-avatar-alt = Profile picture of { $username }
profile-avatar-help = JPEG, PNG or WebP. Maximum 5 MiB, 1024 by 1024 pixels.
profile-avatar-upload = Upload picture
profile-avatar-delete = Delete picture
profile-avatar-invalid = Invalid image. JPEG, PNG or WebP image up to 5 MiB, 1024 by 1024 pixels.
profile-avatar-view = Open full-size profile picture (new tab)
profile-avatar-delete-title = Delete profile picture?
profile-avatar-delete-description = This profile’s picture will be removed and replaced with the default avatar.
profile-avatar-cancel = Cancel
profile-inventory = View inventory
profile-message = Send a message

profile-rank = Rank
profile-rank-user = User
profile-rank-moderator = Moderator
profile-rank-admin = Administrator
profile-error-rank = Rank must be 0, 1 or 2.
profile-error-rank-self = You cannot change your own rank.
account-delete-profile-text = Permanently delete { $username } and all associated data. This cannot be undone.
account-delete-profile-confirm = Permanently delete { $username } and all associated data?
account-delete-profile-submit = Delete account

admin-users-title = Manage accounts
admin-users-caption = Accounts, their role and the actions available on them
admin-users-search = Username or email
admin-users-role = Role
admin-users-role-all = All roles
admin-users-apply = Apply
admin-users-col-user = User
admin-users-col-joined = Joined
admin-users-col-cards = Cards
admin-users-col-actions = Actions
admin-users-online = online
admin-users-own = Your account, managed from your profile
admin-users-rank-of = Role of { $username }
admin-users-rank-submit = Change role
admin-users-delete = Delete…
admin-users-done-rank = The role was changed.
admin-users-done-delete = The account was deleted.
admin-users-error-rank = This role does not exist.
admin-users-error-self = Manage your own account from your profile.
admin-users-error-missing = This account no longer exists.
admin-users-error-internal = The action failed, try again.

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
notification-friend-accepted-title = Friend request accepted
notification-friend-accepted-desc = { $username } accepted your friend request
notification-info-title = { $title }
notification-info-desc = { $description }
notification-trade-title = Trade offer
notification-trade-desc = { $username } sent you a trade offer
notification-trade-counter-title = Trade offer countered
notification-trade-counter-desc = { $username } countered your trade offer
notification-private-message-title = { $count ->
    [one] New message from { $username }
   *[other] { $count } new messages from { $username }
}
notification-private-message-desc = { $preview }

catalog-title = Card catalog
catalog-collection-count = { $count ->
    [one] { $count } collection
    *[other] { $count } collections
}
catalog-filter-collection = Collection
catalog-filter-collection-all = All collections
catalog-sort-number = Collection order
catalog-filter-status = Collected
catalog-filter-status-all = All cards
catalog-filter-status-owned = Collected
catalog-filter-status-missing = Missing
catalog-sort-missing = Missing first

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
inventory-card-value = Value
inventory-card-sell = Sell
inventory-sell-quantity = Quantity
inventory-sell-total = Total
inventory-sell-confirm = Confirm sale
inventory-sold = { $count ->
    [one] You sold { $count } card for { $amount } coins
   *[other] You sold { $count } cards for { $amount } coins
}
inventory-sell-error-card = This card cannot be sold
inventory-sell-error-quantity = Choose a quantity of at least 1
inventory-sell-error-missing = You do not have that many copies to sell
inventory-sell-error-locked = Some of these copies are up in trades
inventory-sell-error-none = You have no duplicate cards to sell
inventory-sell-error-internal = Failed to sell your cards
inventory-sell-duplicates = Sell duplicates
inventory-sell-duplicates-title = Sell duplicate cards
inventory-sell-duplicates-text = { $count ->
    [one] Sell { $count } duplicate card for { $amount } coins?
   *[other] Sell { $count } duplicate cards for { $amount } coins?
}
inventory-sell-duplicates-note = One copy of each card is kept, for each foil. Copies up in trades are neither sold nor counted as the copy kept.
inventory-sell-duplicates-confirm = Sell for { $amount } coins
inventory-sell-cancel = Cancel
inventory-card-acquired-first = First acquired on
inventory-card-acquired-last = Last acquired on
inventory-card-copies = Copies
inventory-filter-rarity = Rarity
inventory-filter-rarity-all = All rarities
inventory-show = Show
inventory-show-owned = Owned cards
inventory-show-all = All cards
inventory-filter-effect = Effect
inventory-filter-effect-all = All effects
inventory-filter-effect-none = No effect
inventory-filter-effect-foil = Foil
inventory-filter-collection = Collection
inventory-filter-collection-all = All collections
inventory-sort = Sort by
inventory-sort-relevance = Relevance
inventory-sort-recent = Last acquired
inventory-sort-oldest = First acquired
inventory-sort-rarity = Rarity
inventory-sort-count = Count
inventory-sort-name = Name (A-Z)
inventory-sort-name-desc = Name (Z-A)
inventory-filters-apply = Apply
inventory-tabs-label = Inventory sections
inventory-tab-cards = Cards
inventory-tab-boosters = Boosters
inventory-tab-collections = Collections
inventory-tab-tradeup = Trade up
inventory-message-no-boosters = No boosters yet
inventory-booster-open = Open booster { $name }
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
inventory-locked-badge = { $count } in trades

booster-pile-title = Default booster
booster-pile-desc = { $count } random cards from all collections
daily-streak-title = Daily streak
daily-streak-meter = { $tier } · { $amount }
daily-streak-count = Streak { $days }
daily-streak-best = Best: { $best }
daily-streak-best-current = current
daily-streak-none = —
daily-streak-claim = Claim { $amount } coins
daily-streak-claimed = Available in
daily-streak-claimed-notice = { $amount } coins added to your balance
daily-streak-claim-ready = Claim your coins
countdown-done = Ready
countdown-refresh = Refresh to collect

pack-opening-title = Pack Opening
pack-opening-open = Open pack
pack-opening-open-another = Open another pack
pack-opening-error-collection = Collection not found
pack-opening-error-empty = This booster has no cards
pack-opening-error-internal = Failed to open the pack
pack-opening-back = Back to inventory
pack-opening-remaining = { $count ->
    [0] No boosters left
    [one] { $count } booster left
    *[other] { $count } boosters left
}
pack-opening-error-booster = This booster is not in your inventory
pack-opening-error-pile = No default booster available
pack-opening-buy = Open again
pack-opening-buy-label = Open again for { $price }
pack-opening-sound = Sound
pack-opening-motion = Animations
pack-opening-odds-title = Drop rates
pack-opening-odds-foil = Foil
pack-opening-odds-note = Each card of the booster is rolled on its own.
pack-opening-progress-title = Collection progress
pack-opening-progress-note = Different cards of this collection you own.
pack-opening-summary = Cards drawn: { $cards }
pack-opening-card = { $title } ({ $rarity })
pack-opening-card-foil = { $title } ({ $rarity }, foil)
pack-opening-spinning = Opening the pack…
pack-opening-skip = Skip

shop-balance = Balance
shop-login-hint = Log in to get coins and buy boosters
shop-booster-buy = Buy
shop-price = { $amount } coins
shop-topup-title = Get coins
shop-topup-note = Demo checkout: no real payment is taken. Coins are credited to your account immediately.
shop-topup-error-unconfirmed = Unable to confirm the top-up. Refresh the page to check your balance before trying again.
shop-topup-price = €{ $euros }.{ $cents }
shop-topup-open = Buy more coins
shop-topup-buy = Buy { $amount } for { $price }
shop-topup-success = { $amount } coins have been added to your balance
shop-error-unauthenticated = You must be logged in to get coins
shop-error-pack = Unknown coin pack
shop-error-funds = You do not have enough coins for this purchase
shop-error-quantity = Enter a quantity between 1 and { $max }
shop-error-internal = Failed to update your balance
shop-buy-close = Close
shop-buy-quantity = Quantity
shop-buy-total = Total
shop-coins = coins
shop-coins-locked = { $amount } up in trades

index-title = Transcendance, collect trading cards
meta-description = Transcendance is a trading card game: buy boosters, open packs and trade cards with other players to complete your collections.
index-heading = Collect, trade and show off your cards
index-tagline = Transcendance is a trading card game: buy boosters and trade cards to grow your collection.
index-stat-cards = { $count ->
    [one] { $count } card
    *[other] { $count } cards
}
index-stat-collections = { $count ->
    [one] { $count } collection
    *[other] { $count } collections
}
index-action-register = Create an account
index-action-browse = Browse the cards
index-action-inventory = Open your inventory
index-step-buy-title = Buy boosters
index-step-buy-text = Spend your coins on boosters of the collection you like, right from your inventory.
index-step-open-title = Open packs
index-step-open-text = Every booster holds a handful of cards, some of them rare, legendary or foil.
index-step-trade-title = Trade cards
index-step-trade-text = Keep your cards in your inventory and trade with other players to complete your collections.
index-install-title = Install the app
index-install-text = Add Transcendance to your home screen to open it like an app, without the browser's bars.
index-install-ios-title = iPhone and iPad:
index-install-ios-text = in Safari, tap Share, then Add to Home Screen.
index-install-android-title = Android:
index-install-android-text = in Chrome, open the ⋮ menu, then tap Add to Home screen or Install app.
index-install-desktop-title = Computer:
index-install-desktop-text = in Chrome or Edge, click the install icon at the end of the address bar.
index-cards-title = Rarest cards
index-cards-all = See every card
index-collections-title = Collections
index-collections-all = See every collection

footer-label = Site information
footer-project = Transcendance, a student project

info-page-updated = Last updated: October 29, 2026

terms-title = Terms of Service
terms-intro = These terms govern your use of Transcendance. By creating an account or using the site, you agree to them.
terms-service-title = 1. The service
terms-service-text = Transcendance is a non-commercial student project where you collect, open and trade virtual cards. It is provided free of charge, for entertainment and educational purposes.
terms-account-title = 2. Your account
terms-account-text = You need a username, an email and a password to create an account. Your email is not verified and is used only to log in; we do not send email. You may log in using your username or email. You can edit your email, and administrators can edit it for you. You are responsible for keeping your password secret and for everything done with your account. Offensive usernames, or usernames impersonating someone, may be changed or removed.
terms-conduct-title = 3. Acceptable use
terms-conduct-intro = When using the site, you agree not to:
terms-conduct-harass = harass, threaten or insult other users;
terms-conduct-content = post illegal, hateful, sexual or violent content, or someone else's personal information;
terms-conduct-spam = spam, advertise, or flood the site with requests;
terms-conduct-abuse = exploit bugs, automate the game, or try to access accounts or data that aren't yours.
terms-content-title = 4. Your content
terms-content-text = You keep the rights to what you post on your profile (bio, contact info, avatar). By posting it, you allow us to store it and display it on the site. Your username, profile and avatar are public. We may remove content that breaks these terms.
terms-items-title = 5. Virtual items
terms-items-text = Coins, cards and boosters have no monetary value. They cannot be bought with or exchanged for real money, and only exist within the site. Balances and inventories may be adjusted or reset, for instance to fix a bug.
terms-termination-title = 6. Ending your account
terms-termination-text = You can delete your account at any time from your profile, which permanently erases its data. We may suspend or delete accounts that break these terms.
terms-liability-title = 7. Availability and liability
terms-liability-text = The site is provided as is, without any warranty. It may be unavailable, change or shut down at any time, and data may be lost. To the extent permitted by law, the team is not liable for any damage resulting from its use.
terms-changes-title = 8. Changes
terms-changes-text = We may update these terms. The date at the top of this page shows when they last changed, and continuing to use the site after a change means you accept it.
terms-law-title = 9. Applicable law
terms-law-text = These terms are governed by French law.

legal-title = Legal notice & privacy
legal-intro = This page describes who runs Transcendance, which personal data it processes and why, and which cookies it uses.
legal-publisher-title = Publisher
legal-publisher-text = Transcendance is a non-commercial student project, published and run by its development team:
legal-publisher-source = The source code is public:
legal-host-text = The site and its database are hosted by the development team.
legal-privacy-title = Personal data
legal-privacy-text = The development team is responsible for processing your personal data, in accordance with the General Data Protection Regulation (GDPR).
legal-data-title = Data we collect
legal-data-account = Account: your username, email, password (stored only as a bcrypt hash) and registration date. Your email is not public: it is visible to you and users of a strictly higher rank. You and administrators can edit it.
legal-data-profile = Profile: your bio, birth date, contact info and avatar, only if you fill them in. They are public.
legal-data-activity = Activity: your login sessions, coins, cards, boosters, trade offers, daily streak, notifications, your friends, the users you block or ignore, and your chat messages, public ones in the shoutbox and private ones. Other users can see whether you are online (active in the last 5 minutes), and the people you trade with see how many coins you have available for the trade.
legal-data-logs = Technical logs: the server logs each request along with your IP address.
legal-purpose-title = Why we use it
legal-purpose-text = This data is only used to provide the service (your account, collection and profile) and to keep it secure. Your email is used only as a login identifier. It is not verified and we do not send email, including marketing or password recovery messages. Data is processed to fulfil the terms you accept when registering, and in our legitimate interest of protecting the site. It is never sold, used for advertising, or shared with third parties.
legal-thirdparty-text = The site loads no external resources: fonts and scripts are served by our own server, so no third party is notified of your visit.
legal-retention-title = How long we keep it
legal-retention-text = Account, profile and activity data are kept until you delete your account. Login sessions expire after 3 days. Technical logs are kept 2 years, as French law requires for hosting providers, then deleted; nightly backups of them are kept 30 more days. Moderators can delete chat messages and mute users in the chat for up to a week.
legal-rights-title = Your rights
legal-rights-text = You have the right to access, correct, erase and export your data, and to object to or restrict its processing.
legal-rights-edit = Your profile can be edited at any time:
legal-rights-delete = Deleting your account from your profile permanently erases your account, profile, avatar, sessions, cards, boosters, coins, trade offers, friends, chat messages and notifications.
legal-rights-other = For any other request, contact the team:
legal-rights-complaint = If you believe your rights are not respected, you can file a complaint with the French data protection authority (CNIL):
legal-cookies-title = Cookies
legal-cookies-text = The site only uses the following cookies, none of which track you:
legal-cookies-name = Name
legal-cookies-purpose = Purpose
legal-cookies-duration = Duration
legal-cookies-session = Keeps you logged in
legal-cookies-lang = Remembers your language
legal-cookies-theme = Remembers your light or dark theme
legal-cookies-days = 3 days
legal-cookies-year = 1 year
legal-cookies-consent = These cookies are needed for the site to work or to remember your own choices, so they do not require your consent. There are no analytics or advertising cookies.

contact-title = Contact
contact-intro = Transcendance is made by a small team of students. Here is how to reach us.
contact-issues-title = Bugs, suggestions and questions
contact-issues-text = The best way to reach us is to open an issue on the project's GitHub repository. Issues are public, so do not include personal information.
contact-issues-link = Open an issue on GitHub
contact-team-title = The team
contact-team-text = You can also reach a team member directly through their GitHub profile:
contact-privacy-title = Personal data requests
contact-privacy-text = For a request about your personal data, contact a team member privately rather than opening a public issue. You can delete your account at any time from your profile.

inventory-trade = Propose a trade
inventory-trades = Your trades
inventory-trades-incoming = { $count ->
    [one] { $count } offer waiting for your answer
   *[other] { $count } offers waiting for your answer
}

trades-title = Trades
trade-menu-title = Trade with { $username }
trade-you-give = You give
trade-you-get = You get
trade-coins = Coins
trade-send = Send offer
trade-counter-send = Send counter-offer
trade-back = Back to trades
trade-accept = Accept
trade-decline = Decline
trade-counter = Counter
trade-cancel = Cancel
trade-incoming = Incoming offers
trade-outgoing = Outgoing offers
trade-history = Past trades
trade-state-accepted = Accepted
trade-state-declined = Declined
trade-state-cancelled = Cancelled
trade-none-incoming = No incoming offers
trade-none-outgoing = No pending offers
trade-none-history = No past trades
trade-stack = { $count }× { $title }
trade-menu-empty = Nothing to trade here
trade-menu-loading = Loading…
trade-menu-increase = Add one
trade-menu-decrease = Remove one
trade-menu-error-network = Unable to contact the server
trade-menu-noscript = Building a trade offer needs JavaScript, to pick the cards of both inventories. Without it, you can still accept, decline or cancel your offers from your trades.
trade-error-settle = This trade offer cannot be completed anymore
trade-error-invalid = Invalid trade offer
trade-error-internal = Failed to save the trade offer
trade-error-not-found = Trade offer not found
trade-error-user = User not found
trade-error-self = You cannot trade with yourself
trade-error-too-many = You have too many offers waiting for an answer: wait for them, or cancel some.
trade-error-empty-give = Offer at least one card or coins
trade-error-empty-want = Request at least one card or coins
trade-error-count = Invalid card count
trade-error-items = Too many card stacks in the offer
trade-error-card = Unknown card
trade-error-duplicate = The same card stack appears twice
trade-error-owned = Not enough copies of a card
trade-error-locked = Some copies of a card are up in another trade
trade-error-coins = Not enough coins
trade-error-locked-coins = Some coins are up in another trade
trade-error-want-coins = The other user does not have that many coins
trade-error-state = This trade offer is not pending anymore
trade-error-forbidden = This trade offer is not yours
trade-error-blocked = You cannot trade with this user
shoutbox-title = Chat
shoutbox-unread = new messages
shoutbox-close = Close
shoutbox-connected = Live
shoutbox-connecting = Connecting...
shoutbox-empty = No messages yet, say hi!
shoutbox-placeholder = Write a message
shoutbox-send = Send
shoutbox-error-empty = Your message is empty
shoutbox-error-too-long = Your message is too long, 500 characters at most
shoutbox-error-rate-limited = You're sending messages too fast, wait a moment
shoutbox-error-internal = Your message could not be sent, try again
shoutbox-closed-unauthorized = Your session expired, log in again to chat
shoutbox-closed-too-many = The shoutbox is open in too many tabs, close some and reload the page
shoutbox-closed-error = Disconnected from the shoutbox, reload the page
shoutbox-tab-shout = Shoutbox
shoutbox-tab-messages = Messages
shoutbox-private-placeholder = Write a private message
shoutbox-conversation-empty = No messages yet, say hi!
shoutbox-conversations-empty = No conversations yet
shoutbox-start = Start a conversation with a user
shoutbox-start-placeholder = Username
shoutbox-start-button = Open
shoutbox-back = Back to conversations
shoutbox-you = You
shoutbox-user-menu = User actions
shoutbox-message-user = Send a private message
shoutbox-view-profile = View profile
shoutbox-ignore-hour = Ignore for 1 hour
shoutbox-ignore-day = Ignore for 1 day
shoutbox-ignore-week = Ignore for 1 week
shoutbox-unignore = Stop ignoring
shoutbox-ignored-title = Ignored users
shoutbox-ignored-until = Until
shoutbox-ignoring-notice = You are ignoring this user: their shoutbox messages are hidden and they can't message you.
shoutbox-error-user-not-found = This user doesn't exist
shoutbox-error-self = You can't message yourself
shoutbox-error-blocked = This user isn't accepting your messages right now
shoutbox-error-forbidden = Only moderators can do this
shoutbox-error-muted = A moderator muted you in the chat for now
shoutbox-error-message-not-found = This message no longer exists

tradeup-rules = Trade { $count } cards of the same rarity for one random card of the next rarity. Each collection's chance follows how many of the cards come from it.
tradeup-empty = No cards here can be traded up
tradeup-slots = Trade-up slots
tradeup-remove = Remove
tradeup-slot-empty = Empty slot
tradeup-clear = Clear
tradeup-submit = Trade up
tradeup-result = You obtained
tradeup-result-close = Close
tradeup-error-count = A trade-up needs exactly { $count } cards
tradeup-error-rarity = Every card in a trade-up must share the same rarity
tradeup-error-tier = These cards cannot be traded up
tradeup-error-missing = You do not own enough of these cards
tradeup-error-locked = Some of these cards are up in trades
tradeup-error-empty = None of these collections holds a card of the next rarity
tradeup-error-internal = Failed to complete the trade-up
