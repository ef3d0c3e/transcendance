package data

import (
	"transcendance/data"
	"transcendance/views"

	"gorm.io/gorm"
)

type DataController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data

	CardSearch map[string]*CardSearchSet
	CollectionSearch map[string]*CollectionSearchSet
}
