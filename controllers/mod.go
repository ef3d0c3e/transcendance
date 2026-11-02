package controllers

import (
	"transcendance/data"
	"transcendance/views"

	"gorm.io/gorm"
)

type Controller struct {
	DB       *gorm.DB
	Renderer *views.Renderer
	Data     *data.Data
	// Failed login attempts, see NewLoginLimits
	Logins *LoginLimits
}
