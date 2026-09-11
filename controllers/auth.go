package controllers

import (
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

func (ac *AuthController) ShowRegister(c *gin.Context) {
	ac.Renderer.Render(c, "register", map[string]any{
		"Title": "Register",
	})
}

var is_alphanumeric = regexp.MustCompile(`^[a-zA-Z0-9_]+$`)

func (ac *AuthController) Register(c *gin.Context) {
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
			"message": ac.Renderer.T(c, "register-error-password-hash"),
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
