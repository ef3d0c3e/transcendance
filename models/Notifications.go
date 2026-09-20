package models

import "time"

type Notif struct {
	ID			uint		`gorm:"primaryKey"`
	UserID		uint		`gorm:"not null;index"`
	User		User		`gorm:"constraint:OnDelete:CASCADE"`
	Icon		string		`gorm:"not null"`
	Title		string		`gorm:"not null"`
	Description	string		`gorm:"not null"`
	Status		string		`gorm:"not null"`
	CreatedAt	time.Time	`gorm:"not null"`
	Action		string		`gorm:"not null"`
}
