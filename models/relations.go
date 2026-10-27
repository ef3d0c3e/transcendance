package models

type Relation struct {
	ID uint `gorm:"primaryKey"`
	UserID uint
	User   User `gorm:"constraint:OnDelete:CASCADE"`
	TargetID uint
	Target   User `gorm:"constraint:OnDelete:CASCADE"`
	Type string `gorm:"not null"`
}
