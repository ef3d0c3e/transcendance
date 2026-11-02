package controllers

import (
	"log"
	"net/http"
	"slices"
	"strings"
	"transcendance/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Lists of the friends page
const (
	friendsListFriends  = "friends"
	friendsListSent     = "sent"
	friendsListReceived = "received"
)

var friendsLists = []string{friendsListFriends, friendsListSent, friendsListReceived}

// Only keep the users of the view's friends list
func friendsJoin(db *gorm.DB, v *userSearchView) *gorm.DB {
	switch v.List {
	case friendsListFriends:
		return db.Joins("JOIN relations ON relations.target_id = users.id AND relations.user_id = ? AND relations.type = ?",
			v.Owner, models.RelationFriends)
	case friendsListSent:
		return db.Joins("JOIN relations ON relations.target_id = users.id AND relations.user_id = ? AND relations.type = ?",
			v.Owner, models.RelationPending)
	case friendsListReceived:
		return db.Joins("JOIN relations ON relations.user_id = users.id AND relations.target_id = ? AND relations.type = ?",
			v.Owner, models.RelationPending)
	}
	return db
}

// /friends GET: the user's friends, or the friend requests they sent or received, with the user search filters
func (sc *Controller) FriendsGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}
	v := &userSearchView{
		Query: strings.TrimSpace(c.Query("q")),
		Page:  1,
		Owner: user.ID,
		List:  friendsListFriends,
	}
	if list := c.Query("list"); slices.Contains(friendsLists, list) {
		v.List = list
	}
	v.Page = PageParam(c.Query("p"))
	if joined := c.Query("joined"); slices.Contains(userJoinedFilters, joined) {
		v.Joined = joined
	}
	if status := c.Query("status"); slices.Contains(userStatusFilters, status) {
		v.Status = status
	}
	v.Sort = defaultUserSort(v.Query)
	if sort := c.Query("sort"); slices.Contains(userSorts, sort) &&
		(sort != userSortRelevance || v.Sort == userSortRelevance) {
		v.Sort = sort
	}

	if err := sc.searchUsers(c, v); err != nil {
		log.Printf("WARN friends search failed for term '%s': %s", v.Query, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": sc.Renderer.T(c, "search-user-error-internal"),
		})
		return
	}
	sc.renderSearch(c, v, "friends-title")
}
