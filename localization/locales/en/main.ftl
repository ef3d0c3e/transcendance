register-title = Create account
register-username = Username
register-password = Password
register-password-confirm = Confirm password
register-tos = I have read and agree to the Terms of Service
register-submit = Create account
registed-error-authenticated = You are already logged in.
register-error-username = Invalid username. Username must be between 3 and 16 characters long and may contain letters, numbers and '_'
register-error-password = Invalid password. Your password must be between 12 and 72 characters long.
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

notification-time-ago-just-now = just now
notification-time-ago-seconds = { $count ->
    [one] { $count } second ago
   *[other] { $count } seconds ago
}
notification-time-ago-minutes = { $count ->
    [one] { $count } minute ago
   *[other] { $count } minutes ago
}
notification-time-ago-hours = { $count ->
    [one] { $count } hour ago
   *[other] { $count } hours ago
}
notification-time-ago-days = { $count ->
    [one] { $count } day ago
   *[other] { $count } days ago
}
notification-time-ago-weeks = { $count ->
    [one] { $count } week ago
   *[other] { $count } weeks ago
}
notification-time-ago-months = { $count ->
    [one] { $count } month ago
   *[other] { $count } months ago
}
notification-time-ago-years = { $count ->
    [one] { $count } year ago
   *[other] { $count } years ago
}

notification-friend-request-title = Friend Request
notification-friend-request-desc = { $username } sent you a friend request
notification-friend-request-accept = Accept
notification-friend-request-deny = Deny
