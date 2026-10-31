package controllers

import (
	"log"
	"net/http"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"transcendance/models"
	"transcendance/views"
)

const (
	userSearchPageSize = 24
	// Shorter searches only match usernames containing them
	userSearchFuzzyLength = 4
)

const (
	userSortRelevance = "relevance"
	userSortName      = "name"
	userSortNameDesc  = "name-desc"
	userSortNewest    = "newest"
	userSortOldest    = "oldest"
	userSortCards     = "cards"
)

// User sorts, relevance is only available when searching
var userSorts = []string{
	userSortRelevance,
	userSortName,
	userSortNameDesc,
	userSortNewest,
	userSortOldest,
	userSortCards,
}

// Join date filters, as how long ago users registered
var userJoinedFilters = []string{"day", "week", "month", "year"}

var userJoinedDurations = map[string]time.Duration{
	"day":   24 * time.Hour,
	"week":  7 * 24 * time.Hour,
	"month": 30 * 24 * time.Hour,
	"year":  365 * 24 * time.Hour,
}

// Status filters, only online users for now
var userStatusFilters = []string{"online"}

// Number of cards owned by the user of the row
const userCardCount = "(SELECT COUNT(*) FROM cards WHERE cards.owner_id = users.id)"

// Sort used when none is picked: search relevance, or usernames
func defaultUserSort(query string) string {
	if query != "" {
		return userSortRelevance
	}
	return userSortName
}

// Escape LIKE wildcards, '!' being the escape character
func escapeLike(s string) string {
	return strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(s)
}

// LIKE pattern matching the query's characters in order, e.g "jdoe" matches "john_doe"
func fuzzyPattern(query string) string {
	var b strings.Builder
	b.WriteString("%")
	for _, r := range query {
		b.WriteString(escapeLike(string(r)))
		b.WriteString("%")
	}
	return b.String()
}

// A user shown in the results
type userSearchResult struct {
	ID        uint
	Username  string
	CreatedAt time.Time
	CardCount int64
	// nil when the user was never seen
	LastSeenAt *time.Time
	Online     bool
}

// Everything needed to render the user search page
type userSearchView struct {
	Query string
	// Join date filter, "" when unfiltered
	Joined string
	// Status filter, "" when unfiltered
	Status string
	// Always set
	Sort  string
	Page  int
	Count int64
	Users []userSearchResult
}

func (v *userSearchView) filtered() bool {
	return v.Query != "" || v.Joined != "" || v.Status != ""
}

// Query parameters of the search, without the page
func (v *userSearchView) params() url.Values {
	params := url.Values{}
	if v.Query != "" {
		params.Set("q", v.Query)
	}
	if v.Joined != "" {
		params.Set("joined", v.Joined)
	}
	if v.Status != "" {
		params.Set("status", v.Status)
	}
	if v.Sort != defaultUserSort(v.Query) {
		params.Set("sort", v.Sort)
	}
	return params
}

// Users matching the search and filters, without ordering
func (sc *Controller) userQuery(c *gin.Context, v *userSearchView) *gorm.DB {
	db := sc.DB.WithContext(c.Request.Context()).Model(&models.User{})
	if v.Query != "" {
		pattern := "%" + escapeLike(v.Query) + "%"
		if len([]rune(v.Query)) >= userSearchFuzzyLength {
			pattern = fuzzyPattern(v.Query)
		}
		db = db.Where("users.username LIKE ? ESCAPE '!'", pattern)
	}
	if v.Joined != "" {
		db = db.Where("users.created_at >= ?", time.Now().Add(-userJoinedDurations[v.Joined]))
	}
	if v.Status == "online" {
		db = db.Where("users.last_seen_at > ?", time.Now().Add(-models.OnlineWindow))
	}
	return db
}

// Order of the results, the username is the tie-break
func userOrder(v *userSearchView) clause.Expression {
	switch v.Sort {
	case userSortRelevance:
		// Exact match, prefix, substring then fuzzy match, earlier and shorter matches first
		escaped := escapeLike(v.Query)
		return clause.Expr{
			SQL: "CASE WHEN users.username = ? THEN 0 " +
				"WHEN users.username LIKE ? ESCAPE '!' THEN 1 " +
				"WHEN users.username LIKE ? ESCAPE '!' THEN 2 ELSE 3 END, " +
				"LOCATE(?, users.username), CHAR_LENGTH(users.username), users.username",
			Vars:               []any{v.Query, escaped + "%", "%" + escaped + "%", v.Query},
			WithoutParentheses: true,
		}
	case userSortNameDesc:
		return clause.Expr{SQL: "users.username DESC"}
	case userSortNewest:
		return clause.Expr{SQL: "users.created_at DESC, users.username"}
	case userSortOldest:
		return clause.Expr{SQL: "users.created_at, users.username"}
	case userSortCards:
		return clause.Expr{SQL: "card_count DESC, users.username"}
	default:
		return clause.Expr{SQL: "users.username"}
	}
}

// Fill the view with the matching users of the page
func (sc *Controller) searchUsers(c *gin.Context, v *userSearchView) error {
	if err := sc.userQuery(c, v).Count(&v.Count).Error; err != nil {
		return err
	}
	v.Users = make([]userSearchResult, 0, userSearchPageSize)
	err := sc.userQuery(c, v).
		Select("users.id, users.username, users.created_at, users.last_seen_at, " + userCardCount + " AS card_count").
		Order(clause.OrderBy{Expression: userOrder(v)}).
		Offset((v.Page - 1) * userSearchPageSize).
		Limit(userSearchPageSize).
		Scan(&v.Users).Error
	for i := range v.Users {
		v.Users[i].Online = models.IsOnline(v.Users[i].LastSeenAt)
	}
	return err
}

// search: every user, or the ones matching the search and filters
func (sc *Controller) SearchUsersGet(c *gin.Context) {
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
	if status := c.Query("status"); slices.Contains(userStatusFilters, status) {
		v.Status = status
	}
	v.Sort = defaultUserSort(v.Query)
	if sort := c.Query("sort"); slices.Contains(userSorts, sort) &&
		(sort != userSortRelevance || v.Sort == userSortRelevance) {
		v.Sort = sort
	}

	if err := sc.searchUsers(c, v); err != nil {
		log.Printf("WARN user search failed for term '%s': %s", v.Query, err)
		c.JSON(http.StatusInternalServerError, gin.H{
			"message": sc.Renderer.T(c, "search-user-error-internal"),
		})
		return
	}
	sc.renderSearch(c, v)
}

// Options of the filter and sort selects
func (sc *Controller) searchOptions(c *gin.Context, v *userSearchView) map[string]any {
	joined := SelectOptions(append([]string{""}, userJoinedFilters...), v.Joined, func(value string) string {
		if value == "" {
			return sc.Renderer.T(c, "search-user-filter-joined-all")
		}
		return sc.Renderer.T(c, "search-user-filter-joined-"+value)
	})

	status := SelectOptions(append([]string{""}, userStatusFilters...), v.Status, func(value string) string {
		if value == "" {
			return sc.Renderer.T(c, "search-user-filter-status-all")
		}
		return sc.Renderer.T(c, "search-user-filter-status-"+value)
	})

	defaultSort := defaultUserSort(v.Query)
	sorts := userSorts
	if defaultSort != userSortRelevance {
		sorts = sorts[1:]
	}
	sortOptions := SelectOptions(sorts, v.Sort, func(value string) string {
		return sc.Renderer.T(c, "search-user-sort-"+value)
	})
	// The default sort isn't submitted, so a new search ranks by relevance
	for i := range sortOptions {
		if sortOptions[i].Value == defaultSort {
			sortOptions[i].Value = ""
		}
	}
	return map[string]any{
		"JoinedOptions": joined,
		"StatusOptions": status,
		"SortOptions":   sortOptions,
	}
}

func (sc *Controller) renderSearch(c *gin.Context, v *userSearchView) {
	user := GetAuthenticatedUser(c)
	title := sc.Renderer.T(c, "search-user-title")
	builder := views.PageBuilder("base", map[string]any{
		"Title": title,
		"User":  user,
	})

	message := ""
	if len(v.Users) == 0 {
		message = sc.Renderer.T(c, "search-user-empty")
	}
	count := ""
	if v.Count > 0 {
		count = sc.Renderer.T(c, "search-user-count", "count", v.Count)
	}
	totalPages := int((v.Count + userSearchPageSize - 1) / userSearchPageSize)
	fields := map[string]any{
		"Title":    title,
		"User":     user,
		"Locale":   sc.Renderer.Localizer.GetLocale(c),
		"Query":    v.Query,
		"Users":    v.Users,
		"Count":    count,
		"Message":  message,
		"Filtered": v.filtered(),
		"Pages":    totalPages,
	}
	for key, options := range sc.searchOptions(c, v) {
		fields[key] = options
	}
	content := builder.Add("search_user", "Content", fields)

	paginatorUrl := "?p="
	if params := v.params(); len(params) != 0 {
		paginatorUrl = "?" + params.Encode() + "&p="
	}
	content.AddPaginator(v.Page, totalPages, paginatorUrl)
	sc.Renderer.Render(c, &builder)
}
