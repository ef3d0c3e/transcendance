package controllers

import (
	"net/http"
	"strconv"
	"transcendance/data"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type DataController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data
}

func (dc *DataController) CardGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)

	id, err := strconv.Atoi(c.Query("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": dc.Renderer.T(c, "cards-error-not-found"),
		})
		return
	}

	card, collection := dc.Data.GetCard(id)
	if card == nil || collection == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"message": dc.Renderer.T(c, "cards-error-not-found"),
		})
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		"Title": "Login",
		"User":  user,
	})
	builder.Add("card", "Content", map[string]any{
		"User":  user,
		"Card": card,
		"Collection": collection,
	})
	dc.Renderer.Render(c, &builder)
}
