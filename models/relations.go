package models

type Relation struct {
	ID uint `gorm:"primaryKey"`
	UserID uint `gorm:"not null;index"`
	User   User `gorm:"constraint:OnDelete:CASCADE"`
	TargetID uint `gorm:"not null;index"`
	Target   User `gorm:"constraint:OnDelete:CASCADE"`
	Type string `gorm:"not null"`
}
