package controllers

import (
	"log"
	"net/http"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type AccountController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func (ac *AccountController) AccountDeleteGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	username := ""
	if user != nil {
		username = user.Username
	}
	if v := c.Query("username"); v != "" {
		username = v
	}

	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "account-delete-error-missing"),
		})
		return
	}
	if user == nil {
		c.JSON(http.StatusForbidden, gin.H{
			"message": ac.Renderer.T(c, "account-delete-error-permission", "username", username),
		})
		return
	}

	var otherUser models.User
	err := ac.DB.Where("username = ?", username).First(&otherUser).Error
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": ac.Renderer.T(c, "account-delete-error-generic"),
		})
		return
	}

	// Admins can delete everyone, otherwise user rank must be strictly above
	if user.Username != username && (user.Rank != 2 && user.Rank <= otherUser.Rank) {
		c.JSON(http.StatusForbidden, gin.H{
			"message": ac.Renderer.T(c, "account-delete-error-permission", "username", username),
		})
		return
	}

	count, err := gorm.G[models.User](ac.DB).Where("id = ?", otherUser.ID).Delete(c.Request.Context())
	if err != nil || count != 1 {
		log.Printf("Failed to delete user `%s'", username)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "account-delete-error-generic"),
		})
		return
	}
	log.Printf("Deleted user `%s'", username)
	c.JSON(http.StatusNoContent, gin.H{
		"message": ac.Renderer.T(c, "account-delete-success", "username", username),
	})
}
