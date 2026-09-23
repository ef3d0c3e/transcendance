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

	id, err := strconv.Atoi(c.Param("ID"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "cards-error-not-found"),
		})
		return
	}

	card, collection := dc.Data.GetCard(id)
	if card == nil || collection == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"message": dc.Renderer.T(c, "cards-error-not-found"),
		})
		return
	}

	method := c.Param("METHOD")
	switch method {
	case "info":
		builder := views.PageBuilder("base", map[string]any{
			"Title": "Login",
			"User":  user,
		})
		builder.Add("card", "Content", map[string]any{
			"User":       user,
			"Card":       card,
			"Collection": collection,
			"ID":         id,
		})
		dc.Renderer.Render(c, &builder)
	case "artwork":
		c.File(card.ArtworkPath)
	case "thumbnail":
		c.File(card.ThumbnailPath)
	}
}
