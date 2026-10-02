package models

import (
	"gorm.io/datatypes"
	"time"
)

type Notif struct {
	ID        uint `gorm:"primaryKey"`
	UserID    uint `gorm:"not null;index"`
	User      User `gorm:"constraint:OnDelete:CASCADE"`
	Icon      string `gorm:"not null"`
	Type      string `gorm:"not null"`
	Data      datatypes.JSONMap
	Status    string `gorm:"not null"`
	CreatedAt time.Time
	Action    string
}
