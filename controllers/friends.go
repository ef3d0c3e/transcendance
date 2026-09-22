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

type FriendsController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func GetRelation(target models.User, c *gin.Context, db *gorm.DB) string {
	user := models.User{}
	db.First(&user, GetAuthenticatedUser(c).ID)
	ctx := c.Request.Context()
	relation, err := gorm.G[models.Relation](db).
		Where("user_id = ? and target_id = ?", user.ID, target.ID).
		First(ctx)
	if err == gorm.ErrRecordNotFound {
		relation, err = gorm.G[models.Relation](db).
		Where("user_id = ? and target_id = ?", target.ID, user.ID).
			First(ctx)
	}
	if err != nil {
		return ""
	}
	if relation.User.ID == target.ID {
		switch relation.Type {
		case "blocked":
			return ""
		case "pending":
			return "respond"
		}
	}
	return relation.Type
}

func (fc *FriendsController) ApiFriendsGet(c *gin.Context) {
	var user  models.User
	fc.DB.First(&user, GetAuthenticatedUser(c).ID)
	targetID := c.Param("target")
	if (targetID == "/") { //no target, query all relations from active user
		rel := models.Relation{}
		res := fc.DB.Find(&rel, user.ID).Order("Type")
		if res.RowsAffected == 0 {
			c.JSON(http.StatusOK, "No friends or pending requests")
			return
		}
	c.JSON(http.StatusOK, gin.H{"Relations:": rel})
	} else { //query specific relation
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

func updateRelation(db *gorm.DB, user, target models.User, action string) {
	var relation models.Relation
	switch action {
	case "friends":
		db.Find(&relation).
		Where("ID = ? and TargetID = ?", user.ID, target.ID).
		Update("Type", "friends")
		//send notification
	case "delete":
		db.Where("ID = ? and TargetID = ?", user.ID, target.ID).
		Delete(&relation)
	case "pending":
		//fails silently if user is blocked
		db.First(&relation).
		Where("ID = ? and TargetID = ?", target.ID, user.ID)
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
		if action == "friends" {
			//send notification
		}
	}
}

func (fc *FriendsController) FriendsPost(c *gin.Context) {
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
	ctx := c.Request.Context()
	target, err := gorm.G[models.User](fc.DB).Where("ID = ?", targetID).First(ctx)
	if err != nil {
		return
	}
	relationStr := GetRelation(target, c, fc.DB)
	switch {
	case relationStr == "" && action == "add":
		updateRelation(fc.DB, user, target, "pending")
	case relationStr == "respond" && (action == "add" || action == "accept"):
		updateRelation(fc.DB, user, target, "friends")
		updateRelation(fc.DB, target, user, "friends")
	case relationStr == "respond" && action == "deny":
		updateRelation(fc.DB, target, user, "delete")
	case relationStr == "block" && action == "unblock":
		updateRelation(fc.DB, user, target, "delete")
	case (relationStr == "friend" || relationStr == "pending" || relationStr == "respond") &&
		 (action == "unfriend" || action == "block"):
		updateRelation(fc.DB, user, target, "delete")
		updateRelation(fc.DB, target, user, "delete")
		fallthrough
	case action == "block":
		updateRelation(fc.DB, user, target, "blocked")
	default:
		//invalid action somehow, send error
	}
}

func (fc *FriendsController) FriendsGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if (user == nil) {
		c.Redirect(http.StatusTemporaryRedirect, "/login")
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": fc.Renderer.T(c, "search-user-title"),
		"User":  user,
	})

	builder.Add("friends", "Content", map[string]any{
		"User":     user,
	})

	fc.Renderer.Render(c, &builder)
}
