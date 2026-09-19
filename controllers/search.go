package controllers

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"transcendance/models"
	"transcendance/views"
)

type SearchController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
}

func (sc *SearchController) SearchUsersGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	username := c.Query("username")
	if username == "" && user != nil {
		username = user.Username
	}
	page, err := strconv.Atoi(c.Query("p"))
	if err != nil || page <= 0 {
		page = 1
	}

	var users []models.User
	total := int64(0)
	if username != "" {

		// NOTE: This can be upgraded using SOUNDEX or FULLTEXT, then using LEVENSHTEIN for ranking
		total, _ = gorm.G[models.User](sc.DB).
			Where("username LIKE ?", "%"+username+"%").
			Count(c.Request.Context(), "username")

		users, err = gorm.G[models.User](sc.DB).
			Where("username LIKE ?", "%"+username+"%").
			Offset((page - 1) * 50).
			Limit(50).
			Order("username").
			Find(c.Request.Context())
		if err != nil {
			users = make([]models.User, 0)
		}
		total = max(total,  int64(len(users)))
	} else {
		users = make([]models.User, 0)
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": sc.Renderer.T(c, "search-user-title"),
		"User":  user,
	})
	content := builder.Add("search_user", "Content", map[string]any{
		"User":     user,
		"Username": username,
		"Users":    users,
	})
	content.Add("paginator", "Paginator", map[string]any{
		"Current": page,
		"Total":   int(total / 50),
		"Url":     "?username=" + username + "&p=",
	})
	sc.Renderer.Render(c, &builder)
}
