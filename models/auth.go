package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID           uint           `gorm:"primaryKey"`

	Username     string         `gorm:"size:16;uniqueIndex;not null"`
	PasswordHash string         `gorm:"not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	UpdatedAt    time.Time      `gorm:"not null"`
	DeletedAt    gorm.DeletedAt `gorm:"index"`
	Avatar       string
}

type UserSesssion struct {
	ID           uint           `gorm:"primaryKey"`
	UserID       uint           `gorm:"not null;index"`
	User         User           `gorm:"constraint:OnDelete:CASCADE"`

	Token        string         `gorm:"not null"`
	CreatedAt    time.Time      `gorm:"not null"`
	ExpiresAt    time.Time      `gorm:"not null"`
}
