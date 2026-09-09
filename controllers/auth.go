package controllers

import (
	"net/http"

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

func (ac *AuthController) Register(c *gin.Context) {
	username := c.PostForm("username")
	password := c.PostForm("password")
	passwordConfirm := c.PostForm("password_confirm")
	tosAgree := c.PostForm("tos_agree")

	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "register-error-username-required"),
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

	user := models.User{
		Username:     username,
		PasswordHash: string(passwordHash),
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "register-error-user"),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": ac.Renderer.T(c, "register-success"),
	})
}
