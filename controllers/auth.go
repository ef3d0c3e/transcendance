package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-sql-driver/mysql"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"transcendance/models"
	"transcendance/views"
)

type AuthenticatedUser struct {
	ID       uint
	Username string
	// 2: Admin, 1: Moderator, 0: User
	Rank uint
	// In-shop currency balance
	Coins uint
}

func AuthMiddleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		token, err := c.Cookie("session_token")

		if err != nil || token == "" {
			c.Set("authenticatedUser", (*AuthenticatedUser)(nil))
			c.Next()
			return
		}

		var session models.UserSession
		err = db.
			Where("token = ? AND expires_at > ?", token, time.Now()).
			First(&session).
			Error

		if err != nil {
			// Invalid/expired token = unauthenticated.
			c.Set("authenticatedUser", (*AuthenticatedUser)(nil))
			c.Next()
			return
		}

		var user models.User
		err = db.Select("id", "username", "rank", "coins", "last_seen_at").
			Where("id = ?", session.UserID).
			First(&user).
			Error

		if err != nil {
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

		c.Set("authenticatedUser", &AuthenticatedUser{
			ID:       session.UserID,
			Username: user.Username,
			Rank:     user.Rank,
			Coins:    user.Coins,
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
		"Title": "Register",
		"User":  user,
	})
	builder.Add("register", "Content", map[string]any{
		"User": user,
	})
	ac.Renderer.Render(c, &builder)
}

var is_alphanumeric = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (ac *Controller) RegisterPost(c *gin.Context) {
	username := c.PostForm("username")
	email := c.PostForm("email")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")
	tosAgree := c.PostForm("tos_agree")

	if username == "" || len(username) < 3 || len(username) > 16 || !is_alphanumeric.MatchString(username) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-username"),
		})
		return
	}

	if email == "" || len(email) > 254 {
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
		bcrypt.DefaultCost,
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
		"Title": "Login",
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
		if len(identifier) < 3 || len(identifier) > 16 || !is_alphanumeric.MatchString(identifier) {
			c.JSON(http.StatusBadRequest, gin.H{"message": ac.Renderer.T(c, "login-error-username")})
			return
		}
	case "email":
		identifier = c.PostForm("email")
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

	var user models.User
	err := ac.DB.Omit("avatar").
		Where(column+" = ?", identifier).
		First(&user).
		Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "login-error-invalid"),
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
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
		Token:     token,
		CreatedAt: now,
		ExpiresAt: now.AddDate(0, 0, 3),
	}

	if err := ac.DB.Create(&session).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "login-error-internal"),
		})
		return
	}

	// Not sent along cross-site requests, so other sites cannot act as the user
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("session_token", token, 3600*24*3, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": ac.Renderer.T(c, "login-success", "username", user.Username),
	})
}

func (ac *Controller) LogoutGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	if user == nil {
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
			c.JSON(http.StatusOK, gin.H{
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
				Where("user_id = ? and token = ?", user.ID, token).
				Delete(ctx)
		}
	}

	c.SetCookie("session_token", "", -1, "/", "", false, true)
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
