register-title = Créer un compte
register-username = Nom d'utilisateur
register-password = Mot de passe
register-password-confirm = Confirmer votre mot de passe
register-tos = J'ai lu et j'accepte les Conditions d'Utilisation
register-submit = Créer un compte
register-error-authenticated = Vous êtes déjà connecté.
register-error-username = Nom d'utilisateur invalide. Le nom d'utilisateur doit faire entre 3 et 16 caractères et peut être composé de lettres, chiffres et '_'
register-error-password = Le mot de passe est invalide. Le mot de passe doit faire entre 12 et 72 caractères.
register-error-passwords-do-not-match = Les mots de passe doivent correspondre
register-error-tos = Vous devez accepter les Conditions d'Utilisation
register-error-password-hash = Impossible de hash le mot de passe
register-error-user = Impossible de créer l'utilisateur
register-error-username-taken = Nom d'utilisateur déjà utilisé
register-success = Compte crée avec succès

login-title = Connexion
login-username = Nom d'utilisateur
login-password = Mot de passe
login-submit = Connexion
login-error-authenticated = Vous êtes déjà connecté.
login-error-username = Nom d'utilisateur invalide
login-error-password = Mot de passe invalide
login-error-invalid = Nom d'utilisateur ou mot de passe invalide
login-error-internal = Échec de la connexion
login-success = Bienvenu, { $username }

logout-unauthenticated = Vous n'êtes pas connecté
logout-success = Vous avez été déconnecté
logout-error-all = Impossible de déconnecter tous les appareilles

notification-time-ago-just-now = à l'instant
notification-time-ago-seconds = { $count ->
    [one] il y a { $count } seconde
   *[other] il y a { $count } secondes
}
notification-time-ago-minutes = { $count ->
    [one] il y a { $count } minute
   *[other] il y a { $count } minutes
}
notification-time-ago-hours = { $count ->
    [one] il y a { $count } heure
   *[other] il y a { $count } heures
}
notification-time-ago-days = { $count ->
    [one] il y a { $count } jour
   *[other] il y a { $count } jours
}
notification-time-ago-weeks = { $count ->
    [one] il y a { $count } semaine
   *[other] il y a { $count } semaines
}
notification-time-ago-months = il y a { $count } mois
notification-time-ago-years = { $count ->
    [one] il y a { $count } an
   *[other] il y a { $count } ans
}

notification-friend-request-title = Demande en Ami
notification-friend-request-desc = { $username } vous a envoyé une demande d'ami
notification-friend-request-accept = Accepter
notification-friend-request-deny = Refuser
