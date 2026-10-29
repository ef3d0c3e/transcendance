package controllers

import (
	"log"
	"net/http"
	"transcendance/models"

	"github.com/gin-gonic/gin"
)

// /api/account_delete POST: permanently delete the authenticated user's account, or the `username` account
func (ac *Controller) AccountDeletePost(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	username := ""
	if user != nil {
		username = user.Username
	}
	if v := c.PostForm("username"); v != "" {
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
	err := ac.DB.Omit("avatar").Where("username = ?", username).First(&otherUser).Error
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

	// Permanently erase the user, related rows are removed by their ON DELETE CASCADE constraint
	result := ac.DB.WithContext(c.Request.Context()).Unscoped().Delete(&models.User{}, otherUser.ID)
	if result.Error != nil || result.RowsAffected != 1 {
		log.Printf("Failed to delete user `%s'", username)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": ac.Renderer.T(c, "account-delete-error-generic"),
		})
		return
	}
	log.Printf("Deleted user `%s'", username)
	if otherUser.ID == user.ID {
		c.SetCookie("session_token", "", -1, "/", "", false, true)
	}
	c.Status(http.StatusNoContent)
}
