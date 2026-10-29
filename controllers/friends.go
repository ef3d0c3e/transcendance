package controllers

import (
	"strconv"
	"transcendance/models"
	"transcendance/views"

	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func GetRelation(target models.User, c *gin.Context, db *gorm.DB) string {
	user := models.User{}
	db.First(&user, GetAuthenticatedUser(c).ID)
	relation := models.Relation{}
	res := db.First(&relation, "user_id = ? and target_id = ?", user.ID, target.ID)
	if res.RowsAffected == 0 {
		res = db.First(&relation, "user_id = ? and target_id = ?", target.ID, user.ID)
		if res.RowsAffected == 0 {
			return ""
		}
		switch relation.Type {
		case "blocked":
			return ""
		case "pending":
			return "respond"
		default:
			return "default"
		}
	}
	if res.Error != nil {
		return ""
	}
	return relation.Type
}

func (fc *Controller) ApiFriendsGet(c *gin.Context) {
	targetID := c.Param("target")
	if (targetID == "/") {
		c.JSON(http.StatusBadRequest, "No specified user")
		return;
	} else {
		targetID = strings.ReplaceAll(targetID, "/", "")
		targetID, _ := strconv.Atoi(targetID)
		target := models.User{}
		res := fc.DB.First(&target, targetID)
		if res.RowsAffected == 0 {
			c.JSON(http.StatusBadRequest, "No such user")
			return
		}
		c.JSON(http.StatusOK, gin.H{"relation": GetRelation(target, c, fc.DB)})
	}
}

func updateRelation(db *gorm.DB, user, target models.User, action string){
	var relation models.Relation
	switch action {
	case "delete":
		res := db.Where("user_id = ? and target_id = ?", user.ID, target.ID).
		Delete(&relation)
		if res.Error != nil {
			return
		}
		if  res.RowsAffected ==  0  {
			return
		}
	case "pending":
		//fails silently if user is blocked
		db.First(&relation).
		Where("user_id = ? and target_id = ?", target.ID, user.ID)
		if relation.Type == "blocked" {
			return
		}
		fallthrough
	default:
		relation = models.Relation{
			Type: action, 
			UserID: user.ID,
			User: user,
			TargetID: target.ID,
			Target: target,
		}
		res := db.Create(&relation)
		if res.Error != nil {
			//something went wrong
			return
		}
	}
}

func (fc *Controller) FriendsPost(c *gin.Context) {
	if GetAuthenticatedUser(c) == nil {
		return
	}
	var user  models.User
	fc.DB.First(&user, GetAuthenticatedUser(c).ID)
	targetID := c.Query("target")
	action := c.Query("action")
	if targetID == "" || action == "" {
		return
	}
	var target  models.User
	fc.DB.First(&target, targetID)
	if target.ID == user.ID {
		return;
	}
	var relation models.Relation
	res := fc.DB.First(&relation, "user_id = ? and type = ?", user.ID, "pending")
	activeInvite := res.RowsAffected > 0
	relationStr := GetRelation(target, c, fc.DB)
	switch {
	case relationStr == "" && action == "add":
		if activeInvite {
			c.JSON(http.StatusOK, gin.H{
				"message": fc.Renderer.T(c, "friends-spam-error"),
			})
			return
		}
		updateRelation(fc.DB, user, target, "pending")
	case relationStr == "respond" && (action == "add" || action == "accept"):
		updateRelation(fc.DB, target, user, "delete")
		updateRelation(fc.DB, user, target, "friends")
		updateRelation(fc.DB, target, user, "friends")
	case relationStr == "respond" && action == "deny":
		updateRelation(fc.DB, target, user, "delete")
	case relationStr == "blocked" && action == "unblock":
		updateRelation(fc.DB, user, target, "delete")
	case (relationStr == "friends" ) && (action == "unfriend"):
		updateRelation(fc.DB, user, target, "delete")
		updateRelation(fc.DB, target, user, "delete")
	case relationStr == "pending" && action == "cancel":
		updateRelation(fc.DB, user, target, "delete")
	case action == "block":
		updateRelation(fc.DB, user, target, "delete")
		updateRelation(fc.DB, target, user, "delete")
		updateRelation(fc.DB, user, target, "blocked")
	default:
		c.JSON(http.StatusOK, gin.H{
			"message": fc.Renderer.T(c, "friend-unknown-error"),
		})
	}
}

func (fc *Controller) FriendsGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if (user == nil) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}

	var relation models.Relation
	var friends []models.User
	fc.DB.Model(&relation).
	Where("user_id = ? and type = ?", user.ID, "friends").
	Association("Target").
	Find(&user)

	builder := views.PageBuilder("base", map[string]any{
		"Title": fc.Renderer.T(c, "friends-title"),
		"User":  user,
	})

	builder.Add("search_user", "Content", map[string]any{
		"User":     user,
		"Username": user.Username,
		"Users": friends,
		"FormAction": "/friends",
	})

	fc.Renderer.Render(c, &builder)
}
