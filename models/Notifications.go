package models

import (
	"time"
	"gorm.io/datatypes"
)

type Notif struct {
	ID			uint			`gorm:"primaryKey"`
	UserID		uint			`gorm:"not null;index"`
	User		User			`gorm:"constraint:OnDelete:CASCADE"`
	EmmiterID	uint
	Icon		string			`gorm:"not null"`
	Type		string			`gorm:"not null"`
	Data		datatypes.JSONMap
	Status		string			`gorm:"not null"`
	CreatedAt	time.Time
	Action		string
}
