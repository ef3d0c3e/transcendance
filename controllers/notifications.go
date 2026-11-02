package controllers

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
	"transcendance/models"
	"transcendance/views"

	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

const (
	notificationPageSize    = 15
	notificationMaxPageSize = 50

	notificationFilterUnread = "unread"
)

// Translate a notification's title and description, using its data as arguments
func (nc *Controller) translate(c *gin.Context, notif *models.Notification) (string, string) {
	args := make([]any, 0, len(notif.Data)*2)
	for key, value := range notif.Data {
		// Numbers are read back as `json.Number`, which isn't one for plurals
		if number, ok := value.(json.Number); ok {
			if f, err := number.Float64(); err == nil {
				value = f
			}
		}
		args = append(args, key, value)
	}
	title := nc.Renderer.T(c, "notification-"+notif.Type+"-title", args...)
	desc := nc.Renderer.T(c, "notification-"+notif.Type+"-desc", args...)
	return title, desc
}

// Get the user's notification by the `ID` param, writes the error response if any
func (nc *Controller) find(c *gin.Context, userID uint) (*models.Notification, bool) {
	notif, err := gorm.G[models.Notification](nc.DB).
		Where("id = ? AND user_id = ?", c.Param("ID"), userID).
		First(c.Request.Context())
	if errors.Is(err, gorm.ErrRecordNotFound) {
		c.String(http.StatusNotFound, "notification not found")
		return nil, false
	}
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return nil, false
	}
	return &notif, true
}

// /notifications GET: notifications page
func (nc *Controller) NotificationsGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	filter := c.Query("filter")
	if filter != notificationFilterUnread {
		filter = ""
	}
	page, err := strconv.Atoi(c.Query("p"))
	if err != nil || page < 1 {
		page = 1
	}

	list, err := nc.list(c, user.ID, filter, page, notificationPageSize)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	// The last page can empty out after dismissing or reading notifications
	if list.Pages > 0 && int64(page) > list.Pages {
		c.Redirect(http.StatusSeeOther, "/notifications?filter="+filter+"&p="+strconv.FormatInt(list.Pages, 10))
		return
	}
	var all int64
	if err := nc.DB.Model(&models.Notification{}).Where("user_id = ?", user.ID).Count(&all).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}

	// Actions come back to the page being viewed
	current := "/notifications?filter=" + filter + "&p=" + strconv.Itoa(page)
	builder := views.PageBuilder("base", map[string]any{
		"Title": nc.Renderer.T(c, "notification-title"),
		"User":  user,
	})
	content := builder.Add("notifications", "Content", map[string]any{
		"Locale":        nc.Renderer.Localizer.GetLocale(c),
		"Filter":        filter,
		"Notifications": list.Items,
		"Unread":        list.Unread,
		"All":           all,
		"Redirect":      current,
	})
	if list.Pages > 1 {
		content.AddPaginator(page, int(list.Pages), "?filter="+filter+"&p=")
	}
	nc.Renderer.Render(c, &builder)
}

// /notification/:ID GET: mark the notification as read and open its page
func (nc *Controller) NotificationGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	notif, ok := nc.find(c, user.ID)
	if !ok {
		return
	}
	if notif.ReadAt == nil {
		if err := nc.DB.Model(notif).Update("read_at", time.Now()).Error; err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}

	target := notif.Action
	if target == "" {
		target = "/notifications"
	}
	c.Redirect(http.StatusSeeOther, target)
}

type notificationList struct {
	Items  []map[string]any
	Unread int64
	Total  int64
	Pages  int64
}

// A page of the user's notifications, newest first, with the user's unread count
func (nc *Controller) list(c *gin.Context, userID uint, filter string, page int, limit int) (notificationList, error) {
	var list notificationList

	query := nc.DB.Model(&models.Notification{}).Where("user_id = ?", userID)
	if filter == notificationFilterUnread {
		query = query.Where("read_at IS NULL")
	}
	if err := query.Session(&gorm.Session{}).Count(&list.Total).Error; err != nil {
		return list, err
	}
	if err := nc.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", userID).
		Count(&list.Unread).
		Error; err != nil {
		return list, err
	}
	list.Pages = (list.Total + int64(limit) - 1) / int64(limit)

	var notifs []models.Notification
	if err := query.
		Order("created_at DESC").
		Order("id DESC").
		Offset((page - 1) * limit).
		Limit(limit).
		Find(&notifs).
		Error; err != nil {
		return list, err
	}

	// Only expose what's required, relative times are formatted client-side
	list.Items = make([]map[string]any, 0, len(notifs))
	for i := range notifs {
		title, desc := nc.translate(c, &notifs[i])
		list.Items = append(list.Items, map[string]any{
			"ID":        notifs[i].ID,
			"Type":      notifs[i].Type,
			"Data":      notifs[i].Data,
			"Read":      notifs[i].ReadAt != nil,
			"CreatedAt": notifs[i].CreatedAt.Format(time.RFC3339),
			"Title":     title,
			"Desc":      desc,
		})
	}
	return list, nil
}

// Respond to a successful action: go back to the `redirect` form value if it is a local page, nothing otherwise
func respondAction(c *gin.Context) {
	target := c.PostForm("redirect")
	if strings.HasPrefix(target, "/") && !strings.HasPrefix(target, "//") && !strings.HasPrefix(target, "/\\") {
		c.Redirect(http.StatusSeeOther, target)
		return
	}
	c.Status(http.StatusNoContent)
}

// /api/notifications GET: a page of the user's notifications, newest first
//
// Query: `page` (from 1), `limit` (page size), `filter=unread` to only list unread ones
func (nc *Controller) ApiNotificationsGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You must be authenticated"})
		return
	}

	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil || page < 1 {
		page = 1
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", strconv.Itoa(notificationPageSize)))
	if err != nil || limit < 1 {
		limit = notificationPageSize
	}
	limit = min(limit, notificationMaxPageSize)

	list, err := nc.list(c, user.ID, c.Query("filter"), page, limit)
	if err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"locale":        nc.Renderer.Localizer.GetLocale(c),
		"notifications": list.Items,
		"unread":        list.Unread,
		"total":         list.Total,
		"page":          page,
		"pages":         list.Pages,
	})
}

// Set whether a notification is read
func (nc *Controller) setRead(c *gin.Context, read bool) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You must be authenticated"})
		return
	}

	notif, ok := nc.find(c, user.ID)
	if !ok {
		return
	}
	if (notif.ReadAt != nil) != read {
		var readAt *time.Time
		if read {
			now := time.Now()
			readAt = &now
		}
		if err := nc.DB.Model(notif).Update("read_at", readAt).Error; err != nil {
			c.String(http.StatusInternalServerError, err.Error())
			return
		}
	}
	respondAction(c)
}

// /api/notifications/:ID/read POST: mark a notification as read
func (nc *Controller) ApiNotificationReadPost(c *gin.Context) {
	nc.setRead(c, true)
}

// /api/notifications/:ID/unread POST: mark a notification as unread
func (nc *Controller) ApiNotificationUnreadPost(c *gin.Context) {
	nc.setRead(c, false)
}

// /api/notifications/:ID/delete POST: dismiss a notification
func (nc *Controller) ApiNotificationDeletePost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You must be authenticated"})
		return
	}

	notif, ok := nc.find(c, user.ID)
	if !ok {
		return
	}
	if err := nc.DB.Delete(notif).Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	respondAction(c)
}

// /api/notifications/read_all POST: mark all of the user's notifications as read
func (nc *Controller) ApiNotificationsReadAllPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": "You must be authenticated"})
		return
	}

	if err := nc.DB.Model(&models.Notification{}).
		Where("user_id = ? AND read_at IS NULL", user.ID).
		Update("read_at", time.Now()).
		Error; err != nil {
		c.String(http.StatusInternalServerError, err.Error())
		return
	}
	respondAction(c)
}

// /add_notification GET: debug page to send a notification to yourself
func (nc *Controller) AddNotificationGet(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.Redirect(http.StatusSeeOther, "/login")
		return
	}

	builder := views.PageBuilder("base", map[string]any{
		// Debug page, not localized
		"Title": "Add a notification (debug)",
		"User":  user,
	})
	builder.Add("add-notification", "Content", map[string]any{})
	nc.Renderer.Render(c, &builder)
}

// /add_notification POST: send a notification to yourself, then show the notifications
func (nc *Controller) AddNotificationPost(c *gin.Context) {
	user := GetAuthenticatedUser(c)
	if user == nil {
		c.String(http.StatusForbidden, "You must be authenticated")
		return
	}

	notif := models.Notification{
		UserID: user.ID,
		Type:   c.PostForm("type"),
	}
	switch notif.Type {
	case models.NotificationFriendRequest:
		notif.Data = datatypes.JSONMap{"username": c.PostForm("username")}
		notif.Action = "/profile/" + url.PathEscape(c.PostForm("username"))
	case models.NotificationInfo:
		notif.Data = datatypes.JSONMap{"title": c.PostForm("title"), "description": c.PostForm("description")}
	default:
		c.String(http.StatusBadRequest, "invalid notification type")
		return
	}
	if err := nc.DB.Create(&notif).Error; err != nil {
		c.String(http.StatusInternalServerError, "failed to insert notification: %s", err)
		return
	}

	c.Redirect(http.StatusSeeOther, "/notifications")
}
