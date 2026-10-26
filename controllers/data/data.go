package data

import (
	"sync"
	"transcendance/data"
	"transcendance/views"

	"gorm.io/gorm"
)

type DataController struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data

	CardSearch       map[string]*CardSearchSet
	CollectionSearch map[string]*CollectionSearchSet
	// Guards lazy building of the search sets
	searchMu sync.Mutex
}
