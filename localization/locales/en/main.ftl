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


card-search-title = Search Cards
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

collection-search-title = Search Collections
collection-search-placeholder = Search for collections
collection-search-empty = No collections found
collection-search-submit = Search
collection-search-error-internal = Search failed

collection-error-not-found = Collection not found
profile-title = Profile
profile-error-user = User doesn't exist
profile-error-username = Invalid username. Username must be between 3 and 16 characters long and may contain letters, numbers and '_'
profile-error-username-taken = Username already taken
profile-error-bio = Bio must be no more than 244 characters long
profile-error-birthdate = Birthdate must be between year 0 and the present day
profile-error-contactinfo = Contact must be no more than 256 characters long
profile-error-password = Password doesn't not match the current password
profile-error-new_password = Invalid password. Your password must be between 8 and 72 characters long
profile-error-passwords-do-not-match = Passwords do not match
profile-error-password-missing = Must provide this field to change password
profile-empty-field = Not Specified
profile-submit = Save

error-unexpected = An error occured
error-unauthorized = You must be logged in to access this ressource

Error = Error

profile-avatar-label = Profile picture
profile-avatar-help = JPEG, PNG or WebP. Maximum 5 MiB, 1024 by 1024 pixels.
profile-avatar-upload = Upload picture
profile-avatar-delete = Delete picture
profile-avatar-invalid = Invalid image. JPEG, PNG or WebP image up to 5 MiB, 1024 by 1024 pixels.
