package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"math"
	"net/http"
	"net/mail"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"transcendance/models"
	"transcendance/views"
)

// PasswordHashCost is the bcrypt cost of new password hashes. The tests
// lower it, hashing at the default cost is slow under the race detector
var PasswordHashCost = bcrypt.DefaultCost

// Lifetime of a session
const sessionDuration = 3 * 24 * time.Hour

// Failed logins allowed in a row per client address and per account, then one more every refill period
const (
	loginBurstPerAddress  = 20
	loginRefillPerAddress = 30 * time.Second
	loginBurstPerAccount  = 5
	loginRefillPerAccount = time.Minute
)

// LoginLimits slows down password guessing, see LoginPost
type LoginLimits struct {
	addresses *AttemptLimiter
	accounts  *AttemptLimiter
}

func NewLoginLimits() *LoginLimits {
	return &LoginLimits{
		addresses: NewAttemptLimiter(loginBurstPerAddress, loginRefillPerAddress),
		accounts:  NewAttemptLimiter(loginBurstPerAccount, loginRefillPerAccount),
	}
}

// Compared against when the account does not exist, so that answering takes as long as for a wrong password
var dummyPasswordHash, _ = bcrypt.GenerateFromPassword([]byte("not a password of any account"), bcrypt.DefaultCost)

type AuthenticatedUser struct {
	ID       uint
	Username string
	// One of the ranks, e.g models.RankAdmin
	Rank uint
	// In-shop currency balance
	Coins uint
	// Coins of the balance up in pending trades, which cannot be spent
	LockedCoins uint
}

// FreeCoins is the balance the user can spend
func (u *AuthenticatedUser) FreeCoins() uint {
	return FreeCoins(u.Coins, u.LockedCoins)
}

// IsModerator reports whether the user can moderate the chat, which admins can too
func (u *AuthenticatedUser) IsModerator() bool {
	return u != nil && u.Rank >= models.RankModerator
}

// IsAdmin reports whether the user can manage accounts and roles
func (u *AuthenticatedUser) IsAdmin() bool {
	return u != nil && u.Rank >= models.RankAdmin
}

// Files served as they are, which don't need to know the user
func staticPath(path string) bool {
	for _, prefix := range []string{"/static/", "/frontend/", "/cards/assets/"} {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	if strings.HasPrefix(path, "/cards/") && (strings.HasSuffix(path, "/thumbnail") || strings.HasSuffix(path, "/artwork")) {
		return true
	}
	return path == "/favicon.ico" || path == "/robots.txt"
}

func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")

		if err != nil || token == "" || staticPath(c.Request.URL.Path) {
			c.Set("authenticatedUser", (*AuthenticatedUser)(nil))
			c.Next()
			return
		}

		// Invalid or expired tokens are unauthenticated
		var user models.User
		err = db.Model(&models.User{}).
			Select("users.id", "users.username", "users.rank", "users.coins", "users.last_seen_at").
			Joins("JOIN user_sessions ON user_sessions.user_id = users.id").
			Where("user_sessions.token = ? AND user_sessions.expires_at > ?", models.HashSessionToken(token), time.Now()).
			Take(&user).
			Error

		if err != nil {
			if !errors.Is(err, gorm.ErrRecordNotFound) {
				log.Printf("WARN failed to look up a session: %s", err)
			}
			c.Set("authenticatedUser", (*AuthenticatedUser)(nil))
			c.Next()
			return
		}

		// Activity of the user, for the online status
		if user.LastSeenAt == nil || time.Since(*user.LastSeenAt) >= models.LastSeenRefresh {
			if err := db.Model(&models.User{}).Where("id = ?", user.ID).
				UpdateColumn("last_seen_at", time.Now()).Error; err != nil {
				log.Printf("WARN failed to update last seen of user %d: %s", user.ID, err)
			}
		}

		locked, err := TradeCommittedCoins(db, user.ID)
		if err != nil {
			log.Printf("WARN failed to get the coins user %d put up in trades: %s", user.ID, err)
		}

		c.Set("authenticatedUser", &AuthenticatedUser{
			ID:          user.ID,
			Username:    user.Username,
			Rank:        user.Rank,
			Coins:       user.Coins,
			LockedCoins: locked,
		})

		c.Next()
	}
}

func GetAuthenticatedUser(c *gin.Context) *AuthenticatedUser {
	user, _ := c.Get("authenticatedUser")
	if user == nil {
		return nil
	}

	return user.(*AuthenticatedUser)
}

func (ac *Controller) RegisterGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	if v := c.Query("fragment"); v == "1" {
		builder := views.PageBuilder("register", map[string]any{
			"User": user,
		})
		ac.Renderer.Render(c, &builder)
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": ac.Renderer.T(c, "register-title"),
		"User":  user,
	})
	builder.Add("register", "Content", map[string]any{
		"User": user,
	})
	ac.Renderer.Render(c, &builder)
}

var isAlphanumeric = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

// validEmail reports whether an address is a plain e-mail address, e.g "name@example.org", as browsers check
// <input type="email">
func validEmail(email string) bool {
	if email == "" || len(email) > 254 {
		return false
	}
	address, err := mail.ParseAddress(email)
	return err == nil && address.Name == "" && address.Address == email
}

func (ac *Controller) RegisterPost(c *gin.Context) {
	username := c.PostForm("username")
	email := strings.TrimSpace(c.PostForm("email"))
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")
	tosAgree := c.PostForm("tos_agree")

	if username == "" || len(username) < 3 || len(username) > 16 || !isAlphanumeric.MatchString(username) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-username"),
		})
		return
	}

	if !validEmail(email) {
		c.JSON(http.StatusBadRequest, gin.H{"message": ac.Renderer.T(c, "account-error-email")})
		return
	}

	if len(password) < 8 || len(password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-password"),
		})
		return
	}

	if password != passwordConfirm {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-passwords-do-not-match"),
		})
		return
	}

	if tosAgree != "on" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-tos"),
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		PasswordHashCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "register-error-password"),
		})
		return
	}

	var conflicts struct {
		UsernameTaken bool
		EmailTaken    bool
	}
	err = ac.DB.WithContext(c.Request.Context()).Raw(`
		SELECT EXISTS(SELECT 1 FROM users WHERE username = ?) AS username_taken,
		       EXISTS(SELECT 1 FROM users WHERE email = ?) AS email_taken
	`, username, email).Scan(&conflicts).Error
	if err != nil {
		log.Printf("Failed to query user database: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": ac.Renderer.T(c, "register-error-user")})
		return
	}
	if conflicts.UsernameTaken || conflicts.EmailTaken {
		key := "register-error-username-taken"
		if !conflicts.UsernameTaken {
			key = "account-error-email-taken"
		}
		c.JSON(http.StatusConflict, gin.H{"message": ac.Renderer.T(c, key)})
		return
	}

	now := time.Now()
	user := models.User{
		Email:        email,
		Username:     username,
		PasswordHash: string(passwordHash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	err = ac.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&user).Error; err != nil {
			return err
		}
		if err := tx.Create(&models.UserProfile{
			UserID: user.ID,
		}).Error; err != nil {
			return err
		}
		// The account comes with its pile of default boosters
		_, err := ensureBoosterPile(tx, user.ID, now)
		return err
	})
	if err != nil {
		key := ""
		switch duplicateUserField(err) {
		case "username":
			key = "register-error-username-taken"
		case "email":
			key = "account-error-email-taken"
		}
		if key != "" {
			c.JSON(http.StatusConflict, gin.H{"message": ac.Renderer.T(c, key)})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "register-error-user"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": ac.Renderer.T(c, "register-success"),
	})
}

func (ac *Controller) LoginGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	if v := c.Query("fragment"); v == "1" {
		builder := views.PageBuilder("login", map[string]any{
			"User": user,
		})
		ac.Renderer.Render(c, &builder)
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": ac.Renderer.T(c, "login-title"),
		"User":  user,
	})
	builder.Add("login", "Content", map[string]any{
		"User": user,
	})
	ac.Renderer.Render(c, &builder)
}

func (ac *Controller) LoginPost(c *gin.Context) {
	column := c.PostForm("login_type")
	if column == "" {
		column = "username"
	}
	var identifier string
	switch column {
	case "username":
		identifier = c.PostForm("username")
		if len(identifier) < 3 || len(identifier) > 16 || !isAlphanumeric.MatchString(identifier) {
			c.JSON(http.StatusBadRequest, gin.H{"message": ac.Renderer.T(c, "login-error-username")})
			return
		}
	case "email":
		identifier = strings.TrimSpace(c.PostForm("email"))
		if identifier == "" || len(identifier) > 254 {
			c.JSON(http.StatusBadRequest, gin.H{"message": ac.Renderer.T(c, "account-error-email")})
			return
		}
	default:
		c.JSON(http.StatusBadRequest, gin.H{"message": ac.Renderer.T(c, "login-error-type")})
		return
	}
	password := c.PostForm("password")
	if len(password) < 8 || len(password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{"message": ac.Renderer.T(c, "login-error-password")})
		return
	}

	// Too many failed attempts from this address or on this account: wait before guessing again
	address, account := c.ClientIP(), column+":"+strings.ToLower(identifier)
	blocked, wait := ac.Logins.addresses.Blocked(address)
	if accountBlocked, accountWait := ac.Logins.accounts.Blocked(account); accountBlocked {
		blocked, wait = true, max(wait, accountWait)
	}
	if blocked {
		seconds := int(math.Ceil(wait.Seconds()))
		c.Header("Retry-After", strconv.Itoa(seconds))
		c.JSON(http.StatusTooManyRequests, gin.H{
			"message": ac.Renderer.T(c, "login-error-rate-limited", "seconds", seconds),
		})
		return
	}

	var user models.User
	err := ac.DB.Omit("avatar").
		Where(column+" = ?", identifier).
		First(&user).
		Error
	if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
		log.Printf("WARN failed to look up an account to log in: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{"message": ac.Renderer.T(c, "login-error-internal")})
		return
	}
	hash := dummyPasswordHash
	if err == nil {
		hash = []byte(user.PasswordHash)
	}
	if bcrypt.CompareHashAndPassword(hash, []byte(password)) != nil || err != nil {
		ac.Logins.addresses.Fail(address)
		ac.Logins.accounts.Fail(account)
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "login-error-invalid"),
		})
		return
	}

	// Token
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		log.Printf("Failed to acquire randomness: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "login-error-internal"),
		})
		return
	}
	token := base64.RawURLEncoding.EncodeToString(buf)

	now := time.Now()
	session := models.UserSession{
		UserID:    user.ID,
		Token:     models.HashSessionToken(token),
		CreatedAt: now,
		ExpiresAt: now.Add(sessionDuration),
	}

	if err := ac.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "login-error-internal"),
		})
		return
	}
	// Expired sessions of every user are cleared along the way, the expiry is indexed
	if err := ac.DB.Where("expires_at <= ?", now).Delete(&models.UserSession{}).Error; err != nil {
		log.Printf("WARN failed to clear expired sessions: %s", err)
	}

	setSessionCookie(c, token, int(sessionDuration.Seconds()))
	c.JSON(http.StatusOK, gin.H{
		"message": ac.Renderer.T(c, "login-success", "username", user.Username),
	})
}

// Secure when the browser reached the site over HTTPS, through the gateway in production
func secureRequest(c *gin.Context) bool {
	return c.Request.TLS != nil || c.GetHeader("X-Forwarded-Proto") == "https"
}

// Set the session cookie, cleared with an empty value and a negative age. It is not sent along cross-site
// requests, so other sites cannot act as the user, nor over plain HTTP once the site was reached over HTTPS.
func setSessionCookie(c *gin.Context, value string, maxAge int) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_token", value, maxAge, "/", "", secureRequest(c), true)
}

// /api/logout POST: end the session, or every session of the user with `all=yes`. Pages get JSON, plain form
// submissions are sent back home.
func (ac *Controller) LogoutPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	if user == nil {
		if !wantsJSON(c) {
			c.Redirect(http.StatusSeeOther, "/")
			return
		}
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "logout-unauthenticated"),
		})
		return
	}

	if v := c.Query("all"); v == "yes" {
		// Log out all devices
		ctx := c.Request.Context()
		_, err := gorm.G[models.UserSession](ac.DB).
			Where("user_id = ?", user.ID).
			Delete(ctx)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"message": ac.Renderer.T(c, "logout-error-all"),
			})
			return
		}
	} else {
		// Log out current device
		token, err := c.Cookie("session_token")
		if err == nil && token != "" {
			ctx := c.Request.Context()
			// Silently fail, since the user's cookie is cleared anyway, that just means keeping data in the DB a little longer
			_, _ = gorm.G[models.UserSession](ac.DB).
				Where("user_id = ? and token = ?", user.ID, models.HashSessionToken(token)).
				Delete(ctx)
		}
	}

	setSessionCookie(c, "", -1)
	if !wantsJSON(c) {
		c.Redirect(http.StatusSeeOther, "/")
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"message": ac.Renderer.T(c, "logout-success"),
	})
}

// duplicateUserField identifies the unique index that rejected a user update.
func duplicateUserField(err error) string {
	var duplicate *mysql.MySQLError
	if !errors.As(err, &duplicate) || duplicate.Number != 1062 {
		return ""
	}
	for _, field := range []string{"username", "email"} {
		if strings.HasSuffix(duplicate.Message, "idx_users_"+field+"'") {
			return field
		}
	}
	return ""
}
