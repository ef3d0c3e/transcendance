register-title = Créer un compte
register-username = Nom d'utilisateur
register-password = Mot de passe
register-password-confirm = Confirmer votre mot de passe
register-tos = J'ai lu et j'accepte les Conditions d'Utilisation
register-submit = Créer un compte
register-error-authenticated = Vous êtes déjà connecté.
register-error-username = Nom d'utilisateur invalide. Le nom d'utilisateur doit faire entre 3 et 16 caractères et peut être composé de lettres, chiffres et '_'
register-error-password = Le mot de passe est invalide. Le mot de passe doit faire entre 8 et 72 caractères.
register-error-passwords-do-not-match = Les mots de passe doivent correspondre
register-error-tos = Vous devez accepter les Conditions d'Utilisation
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

account-delete-success = Le compte de { $username } a été supprimé
account-delete-success-self = Votre compte a été supprimé
account-delete-error-generic = Echec de la suppression du compte
account-delete-error-missing = Il manque le nom du compte
account-delete-error-permission = Vous n'avez pas la permission pour supprimer le compte de { $username }

search-user-title = Rechercher des utilisateurs
search-user-submit = Rechercher
search-user-form-placeholder = Nom d'utilisateur
search-user-count = { $count ->
    [one] { $count } utilisateur
    *[other] { $count } utilisateurs
}
search-user-empty = Aucun utilisateur trouvé
search-user-error-internal = La recherche a échoué
search-user-filter-joined = Inscription
search-user-filter-joined-all = Toutes les dates
search-user-filter-joined-day = Dernières 24 heures
search-user-filter-joined-week = Dernière semaine
search-user-filter-joined-month = Dernier mois
search-user-filter-joined-year = Dernière année
search-user-sort-relevance = Pertinence
search-user-sort-name = Nom (A-Z)
search-user-sort-name-desc = Nom (Z-A)
search-user-sort-newest = Membres récents
search-user-sort-oldest = Membres anciens
search-user-sort-cards = Plus de cartes
search-user-joined = Inscrit :
search-user-cards = Cartes :

navbar-label-toggle = Afficher/masquer le menu
navbar-label-main = Navigation principale
navbar-cards = Cartes
navbar-users = Utilisateurs
navbar-inventory = Inventaire
navbar-shop = Boutique
navbar-profile = Votre profil
navbar-logout = Se déconnecter

theme-switcher-light = Theme clair
theme-switcher-dark = Theme sombre

language-switcher = Choisissez un langage

cards-error-not-found = Carte non trouvée
cards-error-collection-not-found = Collection non trouvée

paginator-label-nav = Pagination
paginator-label-previous = Page précédente
paginator-label-next = Page suivante

card-rarity-common = Commune
card-rarity-rare = Rare
card-rarity-legendary = Legendaire

card-search-placeholder = Rechercher des cartes
card-search-empty = Aucune carte trouvée
card-search-submit = Rechercher
card-search-error-internal = Echec de la recherche

card-error-not-found = Carte non trouvée

collection-title = Collection { $collection }
collection-booster = { $count ->
    [one] Booster de { $count } carte
    *[other] Booster de { $count } cartes
}
collection-count = { $count ->
    [one] { $count } CARTE
    *[other] { $count } CARTES
}
collection-id = NO. { $id }

collection-search-placeholder = Rechercher des collections
collection-search-empty = Aucune collection trouvée
collection-search-submit = Rechercher
collection-search-error-internal = Echec de la recherche

collection-error-not-found = Collection non trouvée

notification-title = Notifications
notification-unread = non lues
notification-loading = Chargement des notifications...
notification-none = Vous êtes à jour
notification-error = Impossible de charger les notifications
notification-view-all = Voir toutes les notifications
notification-mark-all-read = Tout marquer comme lu
notification-mark-read = Marquer comme lue
notification-mark-unread = Marquer comme non lue
notification-delete = Supprimer
notification-tabs-label = Filtrer les notifications
notification-tab-all = Toutes
notification-tab-unread = Non lues
notification-empty-all = Aucune notification pour le moment
notification-empty-unread = Aucune notification non lue

notification-friend-request-title = Demande en Ami
notification-friend-request-desc = { $username } vous a envoyé une demande d'ami
notification-info-title = { $title }
notification-info-desc = { $description }

catalog-title = Catalogue de cartes
catalog-tabs-label = Sections du catalogue
catalog-tab-cards = Cartes
catalog-tab-collections = Collections
catalog-collection-count = { $count ->
    [one] { $count } collection
    *[other] { $count } collections
}
catalog-filter-collection = Collection
catalog-filter-collection-all = Toutes les collections
catalog-sort-number = Ordre de la collection

inventory-title = Votre inventaire
inventory-title-other = Inventaire de { $username }
inventory-unauthenticated = Vous devez être connecté pour acceder aux inventaires
inventory-message-empty = Rien n'a voir ici
inventory-message-user-not-found = Utilisateur non trouvé
inventory-message-fetch-fail = Impossible d'accéder à cet inventaire
inventory-message-no-match = Aucune carte de cet inventaire ne correspond à votre recherche
inventory-search-placeholder = Rechercher dans cet inventaire
inventory-search-submit = Rechercher
inventory-search-clear = Effacer la recherche
inventory-card-acquired = Obtenue le
inventory-card-close = Fermer
inventory-card-acquired-first = Première obtenue le
inventory-card-acquired-last = Dernière obtenue le
inventory-card-copies = Exemplaires
inventory-filter-rarity = Rareté
inventory-filter-rarity-all = Toutes les raretés
inventory-filter-effect = Effet
inventory-filter-effect-all = Tous les effets
inventory-filter-effect-none = Sans effet
inventory-filter-effect-foil = Holographique
inventory-sort = Trier par
inventory-sort-relevance = Pertinence
inventory-sort-recent = Dernière obtention
inventory-sort-oldest = Première obtention
inventory-sort-rarity = Rareté
inventory-sort-count = Nombre
inventory-sort-name = Nom (A-Z)
inventory-sort-name-desc = Nom (Z-A)
inventory-tabs-label = Sections de l'inventaire
inventory-tab-cards = Cartes
inventory-tab-boosters = Boosters
inventory-message-no-boosters = Aucun booster pour l'instant
inventory-booster-open-confirm = Ouvrir le booster { $name } ?
inventory-booster-open-yes = Oui
inventory-booster-open-no = Non
inventory-booster-count = { $count ->
    [one] { $count } booster
    *[other] { $count } boosters
}
inventory-count = { $count ->
    [one] { $count } carte
    *[other] { $count } cartes
}
inventory-search-count = { $count ->
    [one] { $count } carte trouvée
    *[other] { $count } cartes trouvées
}

pack-opening-title = Ouverture de booster
pack-opening-open = Ouvrir le booster
pack-opening-open-another = Ouvrir un autre booster
pack-opening-error-collection = Collection introuvable
pack-opening-error-empty = Ce booster ne contient aucune carte
pack-opening-error-internal = Impossible d'ouvrir le booster
pack-opening-obtained = Cartes obtenues
pack-opening-back = Retour à l'inventaire
pack-opening-return = Retour
pack-opening-open-again = Ouvrir à nouveau
pack-opening-remaining = { $count ->
    [0] Plus aucun booster
    [one] { $count } booster restant
    *[other] { $count } boosters restants
}
pack-opening-error-booster = Ce booster n'est pas dans votre inventaire

shop-title = Boutique
shop-balance = Solde
shop-login-hint = Connectez-vous pour obtenir des pièces et acheter des boosters
shop-boosters-title = Boosters
shop-boosters-empty = Aucun booster n'est en vente pour le moment
shop-booster-buy = Acheter
shop-price = { $amount } pièces
shop-topup-title = Obtenir des pièces
shop-topup-note = Paiement de démonstration : aucun paiement réel n'est effectué. Les pièces sont créditées immédiatement sur votre compte.
shop-topup-price = { $euros },{ $cents } €
shop-topup-open = Acheter des pièces
shop-topup-success = { $amount } pièces ont été ajoutées à votre solde
shop-error-unauthenticated = Vous devez être connecté pour obtenir des pièces
shop-error-pack = Paquet de pièces inconnu
shop-error-funds = Vous n'avez pas assez de pièces pour cet achat
shop-error-quantity = Saisissez une quantité entre 1 et { $max }
shop-error-internal = Impossible de mettre à jour votre solde
shop-buy-close = Fermer
shop-buy-success = { $count ->
    [one] Un booster { $collection } a été ajouté à votre inventaire
   *[other] { $count } boosters { $collection } ont été ajoutés à votre inventaire
}
shop-buy-view-inventory = Voir l'inventaire
shop-buy-quantity = Quantité
shop-buy-total = Total
shop-coins = pièces
