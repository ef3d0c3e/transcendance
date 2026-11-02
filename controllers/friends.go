package controllers

import (
	"errors"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"transcendance/models"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// Relation of the authenticated user to another user, as sent to clients
const (
	friendNone = ""
	// The user sent a request
	friendPending = "pending"
	// The other user sent a request
	friendRespond   = "respond"
	friendFriends   = "friends"
	friendBlocked   = "blocked"
	friendBlockedBy = "block"
)

// Actions of `/friendaction`
var friendActions = []string{"add", "cancel", "accept", "deny", "unfriend", "block", "unblock"}

// The relation of a user to others, by ID. A block takes precedence over anything else.
func relationsOf(db *gorm.DB, userID uint, others []uint) (map[uint]string, error) {
	result := make(map[uint]string, len(others))
	if len(others) == 0 {
		return result, nil
	}
	var rows []models.Relation
	if err := db.
		Where("(user_id = ? AND target_id IN ?) OR (target_id = ? AND user_id IN ?)", userID, others, userID, others).
		Find(&rows).
		Error; err != nil {
		return nil, err
	}

	mine := make(map[uint]string, len(rows))
	theirs := make(map[uint]string, len(rows))
	for _, row := range rows {
		if row.UserID == userID {
			mine[row.TargetID] = row.Type
		} else {
			theirs[row.UserID] = row.Type
		}
	}
	for _, id := range others {
		switch {
		case mine[id] == models.RelationBlocked:
			result[id] = friendBlocked
		case theirs[id] == models.RelationBlocked:
			result[id] = friendBlockedBy
		case mine[id] == models.RelationFriends:
			result[id] = friendFriends
		// Answering wins when both sent a request at once
		case theirs[id] == models.RelationPending:
			result[id] = friendRespond
		case mine[id] == models.RelationPending:
			result[id] = friendPending
		default:
			result[id] = friendNone
		}
	}
	return result, nil
}

func relationOf(db *gorm.DB, userID uint, otherID uint) (string, error) {
	relations, err := relationsOf(db, userID, []uint{otherID})
	return relations[otherID], err
}

// Delete the relations from a user to another, of the given types
func deleteRelations(tx *gorm.DB, userID uint, targetID uint, types ...string) error {
	return tx.Where("user_id = ? AND target_id = ? AND type IN ?", userID, targetID, types).Delete(&models.Relation{}).Error
}

func createRelation(tx *gorm.DB, userID uint, targetID uint, kind string) error {
	return tx.Clauses(clause.OnConflict{UpdateAll: true}).
		Create(&models.Relation{UserID: userID, TargetID: targetID, Type: kind}).
		Error
}

// Notify a user of something another user did, linking to the other user's profile
func notifyFriend(tx *gorm.DB, userID uint, fromID uint, fromUsername string, kind string) error {
	return tx.Create(&models.Notification{
		UserID: userID,
		Type:   kind,
		Data:   datatypes.JSONMap{"sender": fromID, "username": fromUsername},
		Action: "/profile/" + url.PathEscape(fromUsername),
	}).Error
}

// The unread friend request notifications a user got from another
func friendRequestNotifications(tx *gorm.DB, userID uint, senderID uint) *gorm.DB {
	return tx.Model(&models.Notification{}).
		Where("user_id = ? AND type = ?", userID, models.NotificationFriendRequest).
		Where("JSON_VALUE(data, '$.sender') = ?", senderID)
}

// Become friends with a user who sent a request, which is then read
func acceptFriend(tx *gorm.DB, user *AuthenticatedUser, target *models.User) error {
	if err := deleteRelations(tx, target.ID, user.ID, models.RelationPending); err != nil {
		return err
	}
	if err := deleteRelations(tx, user.ID, target.ID, models.RelationPending); err != nil {
		return err
	}
	if err := createRelation(tx, user.ID, target.ID, models.RelationFriends); err != nil {
		return err
	}
	if err := createRelation(tx, target.ID, user.ID, models.RelationFriends); err != nil {
		return err
	}
	if err := friendRequestNotifications(tx, user.ID, target.ID).
		Where("read_at IS NULL").
		Update("read_at", gorm.Expr("NOW()")).
		Error; err != nil {
		return err
	}
	return notifyFriend(tx, target.ID, user.ID, user.Username, models.NotificationFriendAccepted)
}

// Apply an action of the user on their relation to the target, returns the translation key of the reason it
// can't be done. Actions that don't apply to the current relation do nothing.
func applyFriendAction(tx *gorm.DB, user *AuthenticatedUser, target *models.User, current string, action string) (string, error) {
	switch action {
	case "add":
		switch current {
		case friendNone:
			if err := createRelation(tx, user.ID, target.ID, models.RelationPending); err != nil {
				return "", err
			}
			return "", notifyFriend(tx, target.ID, user.ID, user.Username, models.NotificationFriendRequest)
		case friendRespond:
			return "", acceptFriend(tx, user, target)
		case friendBlocked:
			return "friends-unblock-error", nil
		case friendBlockedBy:
			return "friends-blocked-error", nil
		}
	case "cancel":
		if current == friendPending {
			if err := deleteRelations(tx, user.ID, target.ID, models.RelationPending); err != nil {
				return "", err
			}
			// The request is gone
			return "", friendRequestNotifications(tx, target.ID, user.ID).Delete(&models.Notification{}).Error
		}
	case "accept":
		if current == friendRespond {
			return "", acceptFriend(tx, user, target)
		}
	case "deny":
		if current == friendRespond {
			if err := deleteRelations(tx, target.ID, user.ID, models.RelationPending); err != nil {
				return "", err
			}
			return "", friendRequestNotifications(tx, user.ID, target.ID).
				Where("read_at IS NULL").
				Update("read_at", gorm.Expr("NOW()")).
				Error
		}
	case "unfriend":
		if current == friendFriends {
			if err := deleteRelations(tx, user.ID, target.ID, models.RelationFriends); err != nil {
				return "", err
			}
			return "", deleteRelations(tx, target.ID, user.ID, models.RelationFriends)
		}
	case "block":
		if current == friendBlocked {
			return "", nil
		}
		// Their own block of the user stays
		if err := deleteRelations(tx, target.ID, user.ID, models.RelationPending, models.RelationFriends); err != nil {
			return "", err
		}
		if err := createRelation(tx, user.ID, target.ID, models.RelationBlocked); err != nil {
			return "", err
		}
		if err := friendRequestNotifications(tx, user.ID, target.ID).Delete(&models.Notification{}).Error; err != nil {
			return "", err
		}
		return "", friendRequestNotifications(tx, target.ID, user.ID).Delete(&models.Notification{}).Error
	case "unblock":
		if current == friendBlocked {
			return "", deleteRelations(tx, user.ID, target.ID, models.RelationBlocked)
		}
	}
	return "", nil
}

// /friendaction POST: change the relation to the `target` user ID with an `action`: add, cancel, accept, deny,
// unfriend, block or unblock. Answers with the new relation, and a message when the action failed.
func (fc *Controller) FriendsPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": fc.Renderer.T(c, "error-unauthorized")})
		return
	}

	action := c.Query("action")
	if !slices.Contains(friendActions, action) {
		c.JSON(http.StatusBadRequest, gin.H{"message": fc.Renderer.T(c, "friends-unknown-error")})
		return
	}
	var target models.User
	err := gorm.ErrRecordNotFound
	if targetID, parseErr := strconv.ParseUint(c.Query("target"), 10, 0); parseErr == nil {
		err = fc.DB.Select("id", "username").First(&target, targetID).Error
	}
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"message": fc.Renderer.T(c, "friends-not-found-error")})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fc.Renderer.T(c, "friends-unknown-error")})
		return
	}
	if target.ID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"message": fc.Renderer.T(c, "friends-yourself-error")})
		return
	}

	var relation, failure string
	err = fc.DB.WithContext(c.Request.Context()).Transaction(func(tx *gorm.DB) error {
		current, err := relationOf(tx, user.ID, target.ID)
		if err != nil {
			return err
		}
		if failure, err = applyFriendAction(tx, user, &target, current, action); err != nil {
			return err
		}
		relation, err = relationOf(tx, user.ID, target.ID)
		return err
	})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": fc.Renderer.T(c, "friends-unknown-error")})
		return
	}
	if failure != "" {
		c.JSON(http.StatusConflict, gin.H{"relation": relation, "message": fc.Renderer.T(c, failure)})
		return
	}
	c.JSON(http.StatusOK, gin.H{"relation": relation})
}
