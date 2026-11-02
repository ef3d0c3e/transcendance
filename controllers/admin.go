package controllers

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const adminUsersPageSize = 25

var adminRoles = map[string]uint{
	"user":      models.RankUser,
	"moderator": models.RankModerator,
	"admin":     models.RankAdmin,
}

// Order of the role filters and of the rank options
var adminRoleNames = []string{"user", "moderator", "admin"}

// Name of a rank, the key of its `profile-rank-<name>` translation
func rankName(rank uint) string {
	for name, value := range adminRoles {
		if value == rank {
			return name
		}
	}
	return "user"
}

type adminUserRow struct {
	ID         uint
	Username   string
	Email      string
	Rank       uint
	RankName   string
	CreatedAt  time.Time
	LastSeenAt *time.Time
	Online     bool
	CardCount  int64
	// The admin's own account is managed from their profile
	Own bool
}

// Ask an admin to sign in, or tell others the page is not theirs
func (ac *Controller) requireAdmin(c *gin.Context) (*AuthenticatedUser, bool) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return nil, false
	}
	if !user.IsAdmin() {
		ac.Renderer.RenderError(c, views.ErrorPage{
			User:       user,
			StatusCode: http.StatusForbidden,
			Message:    ac.Renderer.T(c, "error-unauthorized"),
		})
		return nil, false
	}
	return user, true
}

// /admin/users GET: the accounts, with their role, for admins to manage. Query: `q` searches usernames and
// e-mails, `role` keeps one role, `p` is the page.
func (ac *Controller) AdminUsersGet(c *gin.Context) {
	user, ok := ac.requireAdmin(c)
	if !ok {
		return
	}

	query := strings.TrimSpace(c.Query("q"))
	role := c.Query("role")
	if _, ok := adminRoles[role]; !ok {
		role = ""
	}
	page := PageParam(c.Query("p"))

	db := ac.DB.WithContext(c.Request.Context()).Model(&models.User{})
	if query != "" {
		pattern := "%" + escapeLike(query) + "%"
		db = db.Where("users.username LIKE ? ESCAPE '!' OR users.email LIKE ? ESCAPE '!'", pattern, pattern)
	}
	if role != "" {
		db = db.Where("users.`rank` = ?", adminRoles[role])
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		InternalError(c, ac.Renderer, err)
		return
	}
	var rows []adminUserRow
	if err := db.Select("users.id, users.username, users.email, users.`rank`, users.created_at, users.last_seen_at, " +
		userCardCount + " AS card_count").
		Order("users.`rank` DESC, users.username").
		Offset((page - 1) * adminUsersPageSize).
		Limit(adminUsersPageSize).
		Scan(&rows).Error; err != nil {
		InternalError(c, ac.Renderer, err)
		return
	}
	for i := range rows {
		rows[i].RankName = rankName(rows[i].Rank)
		rows[i].Online = models.IsOnline(rows[i].LastSeenAt)
		rows[i].Own = rows[i].ID == user.ID
	}

	params := url.Values{}
	if query != "" {
		params.Set("q", query)
	}
	if role != "" {
		params.Set("role", role)
	}
	roleOptions := SelectOptions(append([]string{""}, adminRoleNames...), role, func(value string) string {
		if value == "" {
			return ac.Renderer.T(c, "admin-users-role-all")
		}
		return ac.Renderer.T(c, "profile-rank-"+value)
	})
	rankOptions := make([]SelectOption, len(adminRoleNames))
	for i, name := range adminRoleNames {
		rankOptions[i] = SelectOption{Value: strconv.FormatUint(uint64(adminRoles[name]), 10), Label: ac.Renderer.T(c, "profile-rank-"+name)}
	}

	notice := ""
	switch c.Query("done") {
	case "rank", "delete", "password":
		notice = ac.Renderer.T(c, "admin-users-done-"+c.Query("done"))
	}
	errorMessage := ""
	if code := c.Query("error"); slices.Contains([]string{"rank", "self", "missing", "internal", "password", "password-match"}, code) {
		errorMessage = ac.Renderer.T(c, "admin-users-error-"+code)
	}

	totalPages := int((total + adminUsersPageSize - 1) / adminUsersPageSize)
	builder := views.PageBuilder("base", map[string]any{
		"Title": ac.Renderer.T(c, "admin-users-title"),
		"User":  user,
	})
	content := builder.Add("admin_users", "Content", map[string]any{
		"Users":       rows,
		"Query":       query,
		"Role":        role,
		"RoleOptions": roleOptions,
		"RankOptions": rankOptions,
		"Count":       ac.Renderer.T(c, "search-user-count", "count", total),
		"Notice":      notice,
		"Error":       errorMessage,
		"Locale":      ac.Renderer.Localizer.GetLocale(c),
		// Actions come back to the list as it is shown
		"Redirect": "/admin/users?" + params.Encode() + "&p=" + strconv.Itoa(page),
	})
	paginatorURL := "?p="
	if len(params) != 0 {
		paginatorURL = "?" + params.Encode() + "&p="
	}
	content.AddPaginator(page, totalPages, paginatorURL)
	ac.Renderer.Render(c, &builder)
}

// The account an admin action targets, from the `id` param, never the admin's own. Sends the admin back to the
// list with an error code otherwise.
func (ac *Controller) adminTarget(c *gin.Context, admin *AuthenticatedUser) (*models.User, bool) {
	if !sameOriginRequest(c) {
		c.String(http.StatusForbidden, ac.Renderer.T(c, "error-unauthorized"))
		return nil, false
	}
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		adminBack(c, "error=missing")
		return nil, false
	}
	if uint(id) == admin.ID {
		adminBack(c, "error=self")
		return nil, false
	}
	var target models.User
	if err := ac.DB.Select("id", "username", "rank").First(&target, uint(id)).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			adminBack(c, "error=missing")
		} else {
			log.Printf("WARN failed to load account %d: %s", id, err)
			adminBack(c, "error=internal")
		}
		return nil, false
	}
	return &target, true
}

func adminBack(c *gin.Context, outcome string) {
	target := c.PostForm("redirect")
	if !localPath(target) || !strings.HasPrefix(target, "/admin/users") {
		target = "/admin/users?"
	}
	if !strings.Contains(target, "?") {
		target += "?"
	}
	c.Redirect(http.StatusSeeOther, target+"&"+outcome)
}

// /admin/users/:id/rank POST: give another account the `rank`
func (ac *Controller) AdminUserRankPost(c *gin.Context) {
	admin, ok := ac.requireAdmin(c)
	if !ok {
		return
	}
	target, ok := ac.adminTarget(c, admin)
	if !ok {
		return
	}
	rank, err := strconv.ParseUint(c.PostForm("rank"), 10, 8)
	if err != nil || uint(rank) > models.RankAdmin {
		adminBack(c, "error=rank")
		return
	}
	if err := ac.DB.Model(&models.User{}).Where("id = ?", target.ID).Update("rank", uint(rank)).Error; err != nil {
		log.Printf("WARN failed to change the rank of %s: %s", target.Username, err)
		adminBack(c, "error=internal")
		return
	}
	log.Printf("INFO admin %s gave %s rank %d", admin.Username, target.Username, rank)
	adminBack(c, "done=rank")
}

// /admin/users/:id/password POST: give another account a new password, `password` confirmed by `confirm`, for a
// user who lost theirs. Their sessions end, they log in again with it.
func (ac *Controller) AdminUserPasswordPost(c *gin.Context) {
	admin, ok := ac.requireAdmin(c)
	if !ok {
		return
	}
	target, ok := ac.adminTarget(c, admin)
	if !ok {
		return
	}
	password := c.PostForm("password")
	if len(password) < 8 || len(password) > 72 {
		adminBack(c, "error=password")
		return
	}
	if password != c.PostForm("confirm") {
		adminBack(c, "error=password-match")
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), PasswordHashCost)
	if err != nil {
		adminBack(c, "error=internal")
		return
	}
	err = ac.DB.Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&models.User{}).Where("id = ?", target.ID).Update("password_hash", hash).Error; err != nil {
			return err
		}
		return tx.Where("user_id = ?", target.ID).Delete(&models.UserSession{}).Error
	})
	if err != nil {
		log.Printf("WARN failed to reset the password of %s: %s", target.Username, err)
		adminBack(c, "error=internal")
		return
	}
	log.Printf("INFO admin %s reset the password of %s", admin.Username, target.Username)
	adminBack(c, "done=password")
}

// /admin/users/:id/delete POST: permanently delete another account
func (ac *Controller) AdminUserDeletePost(c *gin.Context) {
	admin, ok := ac.requireAdmin(c)
	if !ok {
		return
	}
	target, ok := ac.adminTarget(c, admin)
	if !ok {
		return
	}
	// Related rows are removed by their ON DELETE CASCADE constraint
	if err := ac.DB.Unscoped().Delete(&models.User{}, target.ID).Error; err != nil {
		log.Printf("WARN failed to delete %s: %s", target.Username, err)
		adminBack(c, "error=internal")
		return
	}
	log.Printf("INFO admin %s deleted user %s", admin.Username, target.Username)
	adminBack(c, "done=delete")
}
