register-title = Créer un compte
register-username = Nom d'utilisateur
register-password = Mot de passe
register-password-confirm = Confirmer votre mot de passe
register-tos = J'ai lu et j'accepte les
register-tos-link = Conditions d'Utilisation
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
login-type = Se connecter avec
login-username = Nom d'utilisateur
login-password = Mot de passe
login-submit = Connexion
login-error-authenticated = Vous êtes déjà connecté.
login-error-type = Choisissez le nom d’utilisateur ou l’adresse e-mail
login-error-username = Nom d'utilisateur invalide
login-error-password = Mot de passe invalide
login-error-invalid = Identifiants de connexion invalides
login-error-internal = Échec de la connexion
login-success = Bienvenu, { $username }

logout-unauthenticated = Vous n'êtes pas connecté
logout-success = Vous avez été déconnecté
logout-error-all = Impossible de déconnecter tous les appareilles

account-delete-error-generic = Echec de la suppression du compte
account-delete-title = Supprimer le compte
account-email = Adresse e-mail
account-error-email = Une adresse e-mail est requise (254 octets maximum)
account-error-email-taken = Adresse e-mail déjà utilisée

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
search-user-filter-status = Statut
search-user-filter-status-all = Tout le monde
search-user-filter-status-online = En ligne
user-status-online = En ligne
user-status-offline = Hors ligne
user-status-last-seen = Dernière activité :

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

profile-username = Nom d’utilisateur
profile-bio = Bio
profile-birthdate = Date de naissance
profile-contact = Contact
profile-new-password = Nouveau mot de passe
profile-confirm-password = Confirmer le mot de passe
profile-current-password = Mot de passe actuel
profile-username-hint = 3 à 16 caractères : lettres, chiffres ou '_'.
profile-title = Profil
profile-error-user = L'utilisateur n'existe pas
profile-error-username = Nom d'utilisateur invalide. Le nom d'utilisateur doit faire entre 3 et 16 caractères et ne peut contenir que des lettres, des chiffres et '_'
profile-error-username-taken = Nom d'utilisateur déjà utilisé
profile-error-bio = La bio ne doit pas dépasser 244 octets
profile-error-birthdate = La date de naissance doit être comprise entre l'an 0 et aujourd'hui
profile-error-contactinfo = Le contact ne doit pas dépasser 256 octets
profile-error-password = Le mot de passe ne correspond pas au mot de passe actuel
profile-error-new_password = Mot de passe invalide. Votre mot de passe doit faire entre 8 et 72 caractères
profile-error-passwords-do-not-match = Les mots de passe ne correspondent pas
profile-error-password-missing = Ce champ est requis pour changer de mot de passe
profile-empty-field = Non renseigné
profile-submit = Enregistrer
profile-error-network = Impossible de contacter le serveur

error-unexpected = Une erreur est survenue
error-unauthorized = Vous devez être connecté pour accéder à cette ressource

error-title = Erreur

profile-avatar-label = Photo de profil
profile-avatar-alt = Photo de profil de { $username }
profile-avatar-help = JPEG, PNG ou WebP. Maximum 5 Mio, 1024 par 1024 pixels.
profile-avatar-upload = Modifier la photo
profile-avatar-delete = Supprimer la photo
profile-avatar-invalid = Image invalide. Formats acceptés : JPEG, PNG ou WebP, 5 Mio maximum, 1024 par 1024 pixels.
profile-avatar-view = Ouvrir la photo de profil en taille réelle (nouvel onglet)
profile-avatar-delete-title = Supprimer la photo de profil ?
profile-avatar-delete-description = La photo de ce profil sera supprimée et remplacée par l’avatar par défaut.
profile-avatar-cancel = Annuler
profile-inventory = Voir l’inventaire

profile-rank = Rôle
profile-rank-user = Utilisateur
profile-rank-moderator = Modérateur
profile-rank-admin = Administrateur
profile-error-rank = Le rôle doit être 0, 1 ou 2.
account-delete-profile-text = Supprimer définitivement { $username } et toutes les données associées. Cette action est irréversible.
account-delete-profile-confirm = Supprimer définitivement { $username } et toutes les données associées ?
account-delete-profile-submit = Supprimer le compte

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

index-title = Transcendance, collectionnez des cartes
index-heading = Collectionnez, troquez et exposez vos cartes
index-tagline = Transcendance est un jeu de cartes à collectionner : achetez des boosters et troquez vos cartes pour enrichir votre collection.
index-stat-cards = { $count ->
    [one] { $count } carte
    *[other] { $count } cartes
}
index-stat-collections = { $count ->
    [one] { $count } collection
    *[other] { $count } collections
}
index-action-register = Créer un compte
index-action-browse = Parcourir les cartes
index-action-inventory = Ouvrir votre inventaire
index-step-buy-title = Achetez des boosters
index-step-buy-text = Dépensez vos pièces en boosters de la collection de votre choix, directement depuis votre inventaire.
index-step-open-title = Ouvrez des paquets
index-step-open-text = Chaque booster contient quelques cartes, dont certaines rares, légendaires ou holographiques.
index-step-trade-title = Troquez vos cartes
index-step-trade-text = Retrouvez vos cartes dans votre inventaire et troquez avec d’autres joueurs pour compléter vos collections.
index-install-title = Installer l’application
index-install-text = Ajoutez Transcendance à votre écran d’accueil pour l’ouvrir comme une application, sans les barres du navigateur.
index-install-ios-title = iPhone et iPad :
index-install-ios-text = dans Safari, touchez Partager, puis Sur l’écran d’accueil.
index-install-android-title = Android :
index-install-android-text = dans Chrome, ouvrez le menu ⋮, puis touchez Ajouter à l’écran d’accueil ou Installer l’application.
index-install-desktop-title = Ordinateur :
index-install-desktop-text = dans Chrome ou Edge, cliquez sur l’icône d’installation au bout de la barre d’adresse.
index-cards-title = Cartes les plus rares
index-cards-all = Voir toutes les cartes
index-collections-title = Collections
index-collections-all = Voir toutes les collections

footer-label = Informations sur le site
footer-project = Transcendance, un projet étudiant

info-page-updated = Dernière mise à jour : 29 octobre 2026

terms-title = Conditions d'Utilisation
terms-intro = Ces conditions encadrent votre utilisation de Transcendance. En créant un compte ou en utilisant le site, vous les acceptez.
terms-service-title = 1. Le service
terms-service-text = Transcendance est un projet étudiant non commercial dans lequel vous collectionnez, ouvrez et échangez des cartes virtuelles. Il est fourni gratuitement, à des fins de divertissement et d'apprentissage.
terms-account-title = 2. Votre compte
terms-account-text = Un nom d'utilisateur, une adresse e-mail et un mot de passe sont nécessaires pour créer un compte. Votre adresse e-mail n’est pas vérifiée et sert uniquement à vous connecter ; aucun e-mail n’est envoyé. Vous pouvez vous connecter avec votre nom d’utilisateur ou votre adresse e-mail. Vous êtes responsable de la confidentialité de votre mot de passe et de toute activité réalisée avec votre compte. Les noms d’utilisateur injurieux ou usurpant une identité peuvent être modifiés ou supprimés.
terms-conduct-title = 3. Règles d'usage
terms-conduct-intro = En utilisant le site, vous vous engagez à ne pas :
terms-conduct-harass = harceler, menacer ou insulter d'autres utilisateurs ;
terms-conduct-content = publier de contenu illégal, haineux, sexuel ou violent, ni les informations personnelles d'autrui ;
terms-conduct-spam = envoyer du spam, faire de la publicité ou surcharger le site de requêtes ;
terms-conduct-abuse = exploiter des bugs, automatiser le jeu, ou tenter d'accéder à des comptes ou données qui ne sont pas les vôtres.
terms-content-title = 4. Votre contenu
terms-content-text = Vous conservez vos droits sur ce que vous publiez sur votre profil (bio, contact, avatar). En le publiant, vous nous autorisez à le stocker et à l'afficher sur le site. Votre nom d'utilisateur, votre profil et votre avatar sont publics. Nous pouvons retirer tout contenu qui enfreint ces conditions.
terms-items-title = 5. Objets virtuels
terms-items-text = Les pièces, cartes et boosters n'ont aucune valeur monétaire. Ils ne peuvent être ni achetés ni échangés contre de l'argent réel, et n'existent qu'au sein du site. Les soldes et inventaires peuvent être ajustés ou réinitialisés, par exemple pour corriger un bug.
terms-termination-title = 6. Fin de votre compte
terms-termination-text = Vous pouvez supprimer votre compte à tout moment depuis votre profil, ce qui efface définitivement ses données. Nous pouvons suspendre ou supprimer les comptes qui enfreignent ces conditions.
terms-liability-title = 7. Disponibilité et responsabilité
terms-liability-text = Le site est fourni en l'état, sans aucune garantie. Il peut être indisponible, évoluer ou fermer à tout moment, et des données peuvent être perdues. Dans les limites permises par la loi, l'équipe n'est pas responsable des dommages résultant de son utilisation.
terms-changes-title = 8. Modifications
terms-changes-text = Ces conditions peuvent être mises à jour. La date en haut de cette page indique leur dernière modification, et continuer à utiliser le site après une modification vaut acceptation.
terms-law-title = 9. Droit applicable
terms-law-text = Ces conditions sont régies par le droit français.

legal-title = Mentions légales et confidentialité
legal-intro = Cette page indique qui édite Transcendance, quelles données personnelles sont traitées et pourquoi, et quels cookies sont utilisés.
legal-publisher-title = Éditeur
legal-publisher-text = Transcendance est un projet étudiant non commercial, édité et exploité par son équipe de développement :
legal-publisher-source = Le code source est public :
legal-host-text = Le site et sa base de données sont hébergés par l'équipe de développement.
legal-privacy-title = Données personnelles
legal-privacy-text = L'équipe de développement est responsable du traitement de vos données personnelles, conformément au Règlement Général sur la Protection des Données (RGPD).
legal-data-title = Données collectées
legal-data-account = Compte : votre nom d’utilisateur, votre adresse e-mail, votre mot de passe (stocké uniquement sous forme de hash bcrypt) et votre date d’inscription. Votre adresse e-mail n’est pas publique : elle est visible par vous et par les utilisateurs de rôle strictement supérieur. Vous et les administrateurs pouvez la modifier.
legal-data-profile = Profil : votre bio, date de naissance, contact et avatar, uniquement si vous les renseignez. Ils sont publics.
legal-data-activity = Activité : vos sessions de connexion, pièces, cartes, boosters et notifications.
legal-data-logs = Journaux techniques : le serveur enregistre chaque requête avec votre adresse IP.
legal-purpose-title = Utilisation
legal-purpose-text = Ces données servent uniquement à fournir le service (votre compte, votre collection et votre profil) et à le sécuriser. Votre adresse e-mail sert uniquement d’identifiant de connexion. Elle n’est pas vérifiée et aucun e-mail n’est envoyé, y compris pour la publicité ou la récupération du mot de passe. Le traitement repose sur l’exécution des conditions acceptées à l’inscription et sur notre intérêt légitime à protéger le site. Les données ne sont jamais vendues, utilisées à des fins publicitaires ni partagées avec des tiers.
legal-thirdparty-text = Le site ne charge aucune ressource externe : les polices et scripts sont servis par notre propre serveur, aucun tiers n'est donc informé de votre visite.
legal-retention-title = Durée de conservation
legal-retention-text = Les données de compte, de profil et d'activité sont conservées jusqu'à la suppression de votre compte. Les sessions de connexion expirent après 3 jours. Les journaux techniques ne sont conservés que le temps nécessaire à l'analyse des incidents.
legal-rights-title = Vos droits
legal-rights-text = Vous disposez d'un droit d'accès, de rectification, d'effacement et de portabilité de vos données, ainsi que d'un droit d'opposition et de limitation de leur traitement.
legal-rights-edit = Votre profil est modifiable à tout moment :
legal-rights-delete = Supprimer votre compte depuis votre profil efface définitivement votre compte, profil, avatar, sessions, cartes, boosters, pièces et notifications.
legal-rights-other = Pour toute autre demande, contactez l'équipe :
legal-rights-complaint = Si vous estimez que vos droits ne sont pas respectés, vous pouvez adresser une réclamation à la CNIL :
legal-cookies-title = Cookies
legal-cookies-text = Le site utilise uniquement les cookies suivants, dont aucun ne sert à vous pister :
legal-cookies-name = Nom
legal-cookies-purpose = Finalité
legal-cookies-duration = Durée
legal-cookies-session = Maintient votre connexion
legal-cookies-lang = Mémorise votre langue
legal-cookies-theme = Mémorise votre thème clair ou sombre
legal-cookies-days = 3 jours
legal-cookies-year = 1 an
legal-cookies-consent = Ces cookies sont nécessaires au fonctionnement du site ou à la mémorisation de vos propres choix, ils ne requièrent donc pas votre consentement. Il n'y a aucun cookie de mesure d'audience ni de publicité.

contact-title = Contact
contact-intro = Transcendance est réalisé par une petite équipe d'étudiants. Voici comment nous joindre.
contact-issues-title = Bugs, suggestions et questions
contact-issues-text = Le meilleur moyen de nous joindre est d'ouvrir une issue sur le dépôt GitHub du projet. Les issues sont publiques, n'y mettez donc pas d'informations personnelles.
contact-issues-link = Ouvrir une issue sur GitHub
contact-team-title = L'équipe
contact-team-text = Vous pouvez aussi contacter directement un membre de l'équipe via son profil GitHub :
contact-privacy-title = Demandes relatives à vos données
contact-privacy-text = Pour une demande concernant vos données personnelles, contactez un membre de l'équipe en privé plutôt que d'ouvrir une issue publique. Vous pouvez supprimer votre compte à tout moment depuis votre profil.
