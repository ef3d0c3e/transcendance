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
			"error": "Username is required",
		})
		return
	}

	if password != passwordConfirm {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Passwords do not match",
		})
		return
	}

	if tosAgree != "on" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "You must agree to the Terms of Service",
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Could not hash password",
		})
		return
	}

	user := models.User{
		Username:     username,
		PasswordHash: string(passwordHash),
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "Could not create user",
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
	})
}
