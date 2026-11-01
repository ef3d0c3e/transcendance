package models

// Relation types. Friends have a relation each way, requests and blocks only go from the user to the target.
const (
	RelationPending = "pending"
	RelationFriends = "friends"
	RelationBlocked = "blocked"
)

type Relation struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"not null;uniqueIndex:idx_relation_pair,priority:1"`
	User     User   `gorm:"constraint:OnDelete:CASCADE"`
	TargetID uint   `gorm:"not null;uniqueIndex:idx_relation_pair,priority:2;index"`
	Target   User   `gorm:"constraint:OnDelete:CASCADE"`
	Type     string `gorm:"not null;size:16"`
}
