package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/models"
	"transcendance/views"
)

type ProfileController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

type profile struct {
	Username     string
	Avatar       string
	CreationDate string
}

func (pc *ProfileController) ProfileGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	var username string

	username = c.Query("username")
	if username == "" {
		if user != nil {
			username = user.Username
		}
	}
	if username == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": pc.Renderer.T(c, "profile-error-username"),
		})
		return
	}

	ctx := c.Request.Context()
	query, err := gorm.G[models.User](pc.DB).
		Where("username = ?", username).
		First(ctx)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": pc.Renderer.T(c, "profile-error-username"),
		})
		return
	}
	avatar := "/static/profile-picture.svg"
	if query.Avatar != "" {
		avatar = query.Avatar
	}
	prof := profile{
		Username: query.Username,
		Avatar: avatar,
		CreationDate: query.CreatedAt.Format(pc.Renderer.T(c, "profile-time")),
	}


	builder := views.PageBuilder("base", map[string]any{
		"Title": pc.Renderer.T(c, "profile-title"),
		"User":  user,
	})
	builder.Add("profile", "Content", map[string]any{
		"User":     user,
		"Profile": prof,
	})
	pc.Renderer.Render(c, &builder)
}
