package controllers

import (
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm/clause"
)

// Fill the view with the matching users of the page
func (sc *Controller) searchFriends(c *gin.Context, v *userSearchView) error {
	if err := sc.userQuery(c, v).Count(&v.Count).Error; err != nil {
		return err
	}
	v.Users = make([]userSearchResult, 0, userSearchPageSize)
	return sc.userQuery(c, v).
		Raw("SELECT users.id, username, users.created_at from users RIGHT JOIN relations ON users.id=relations.target_id WHERE type != ? AND relations.user_id = ?", "blocked", GetAuthenticatedUser(c).ID).
		Order(clause.OrderBy{Expression: userOrder(v)}).
		Offset((v.Page - 1) * userSearchPageSize).
		Limit(userSearchPageSize).
		Scan(&v.Users).Error
}

// search: every user, or the ones matching the search and filters
func (sc *Controller) FriendsGet(c *gin.Context) {
	v := &userSearchView{
		Query: strings.TrimSpace(c.Query("q")),
		Page:  1,
	}
	if page, err := strconv.Atoi(c.Query("p")); err == nil && page > 1 {
		v.Page = page
	}
	if joined := c.Query("joined"); slices.Contains(userJoinedFilters, joined) {
		v.Joined = joined
	}
	v.Sort = defaultUserSort(v.Query)
	if sort := c.Query("sort"); slices.Contains(userSorts, sort) &&
		(sort != userSortRelevance || v.Sort == userSortRelevance) {
		v.Sort = sort
	}

	if err := sc.searchFriends(c, v); err != nil {
		log.Printf("WARN user search failed for term '%s': %s", v.Query, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": sc.Renderer.T(c, "search-user-error-internal"),
		})
		return
	}
	sc.renderSearch(c, v, "friends-title")
}
