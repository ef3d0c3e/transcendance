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

	if password != passwordConfirm {
		ac.Renderer.Render(c, "register", map[string]any{
			"Title":    "Register",
			"Username": username,
			"Error":    "Passwords do not match",
		})
		return
	}

	if tosAgree != "on" {
		ac.Renderer.Render(c, "register", map[string]any{
			"Title":    "Register",
			"Username": username,
			"Error":    "You must agree to the Terms of Service",
		})
		return
	}

	passwordHash, err := bcrypt.GenerateFromPassword(
		[]byte(password),
		bcrypt.DefaultCost,
	)
	if err != nil {
		c.String(http.StatusInternalServerError, "could not hash password")
		return
	}

	user := models.User{
		Username:     username,
		PasswordHash: string(passwordHash),
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.String(http.StatusInternalServerError, "could not create user")
		return
	}

	c.Redirect(http.StatusSeeOther, "/login")
}
