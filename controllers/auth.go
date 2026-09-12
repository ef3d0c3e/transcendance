package controllers

import (
	"crypto/rand"
	"encoding/base64"
	"log"
	"net/http"
	"regexp"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"transcendance/models"
	"transcendance/views"
)

type AuthController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func (ac *AuthController) RegisterGet(c *gin.Context) {
	ac.Renderer.Render(c, "register", map[string]any{
		"Title": "Register",
	})
}

var is_alphanumeric = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (ac *AuthController) RegisterPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")
	tosAgree := c.PostForm("tos_agree")

	if username == "" || len(username) < 3 || len(username) > 16 || !is_alphanumeric.MatchString(username) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-username"),
		})
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

	ctx := c.Request.Context()
	count, err := gorm.G[models.User](ac.DB).Where("username = ?", username).Count(ctx, "username")
	if err != nil {
		log.Printf("Failed to query user database: %s", err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "register-error-user"),
		})
		return
	}
	if count != 0 {
		c.JSON(http.StatusConflict, gin.H{
			"message": ac.Renderer.T(c, "register-error-username-taken"),
		})
		return
	}

	now := time.Now()
	user := models.User{
		Username:     username,
		PasswordHash: string(passwordHash),
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "register-error-user"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message": ac.Renderer.T(c, "register-success"),
	})
}

func (ac *AuthController) LoginGet(c *gin.Context) {
	ac.Renderer.Render(c, "login", map[string]any{
		"Title": "Login",
	})
}

func (ac *AuthController) LoginPost(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")

	if username == "" || len(username) < 3 || len(username) > 16 || !is_alphanumeric.MatchString(username) {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "login-error-username"),
		})
		return
	}

	if len(password) < 8 || len(password) > 72 {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "login-error-password"),
		})
		return
	}

	ctx := c.Request.Context()
	user, err := gorm.G[models.User](ac.DB).
		Where("username = ?", username).
		First(ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "login-error-username"),
		})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "login-error-password"),
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
	session := models.UserSesssion{
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

	c.SetCookie("session_token", token, 3600*24*3, "/", "", false, true)
	c.JSON(http.StatusOK, gin.H{
		"message": ac.Renderer.T(c, "login-success", "username", username),
	})
}
