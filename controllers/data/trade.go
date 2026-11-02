package data

import (
	"net/http"
	"strconv"
	"strings"
	"unicode"

	"transcendance/controllers"
	"transcendance/data"
	"transcendance/models"

	"github.com/gin-gonic/gin"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"
)

// Stacks per side and request, the picker asks for more as it scrolls
const tradeStacksPageSize = 24

type tradeStack struct {
	CardID uint   `json:"card_id"`
	Foil   string `json:"foil"`
	Count  uint   `json:"count"`
	Title  string `json:"title"`
	// Title of the card's collection, the menu's search matches it too
	Collection string `json:"collection"`
	Rarity     string `json:"rarity"`
	Image      string `json:"image"`
	// Rendered card, left out of the prefilled stacks which only show in the
	// deal box
	HTML string `json:"html,omitempty"`
	// Copies a counter starts with on this stack, capped to the free ones
	Picked uint `json:"picked,omitempty"`

	card       *data.Card
	collection *data.Collection
	copies     uint
}

type tradeStackRow struct {
	CardID uint
	Foil   string
	Count  uint
}

// Case and accents are ignored when searching the stacks
func foldTradeSearch(text string) string {
	folder := transform.Chain(norm.NFD, runes.Remove(runes.In(unicode.Mn)), norm.NFC)
	folded, _, err := transform.String(folder, text)
	if err != nil {
		folded = text
	}
	return strings.ToLower(folded)
}

// tradeStacks counts the copies free to be put up, leaving out those committed
// to other pending offers. No card is rendered here, so listing a whole
// inventory stays cheap
func (dc *DataController) tradeStacks(loc string, userID uint, committed *controllers.TradeCommitments, query string) ([]tradeStack, error) {
	var rows []tradeStackRow
	if err := dc.DB.Model(&models.Card{}).
		Select("card_id, foil, COUNT(*) AS count").
		Where("owner_id = ?", userID).
		Group("card_id, foil").
		Order("card_id, foil").
		Scan(&rows).Error; err != nil {
		return nil, err
	}

	query = foldTradeSearch(strings.TrimSpace(query))
	// Collection titles are shared by many stacks
	collectionTitles := map[int]string{}
	stacks := make([]tradeStack, 0, len(rows))
	for _, row := range rows {
		card, collection := dc.Data.GetCard(int(row.CardID))
		if card == nil || collection == nil {
			continue
		}
		title, err := card.Translate(loc, "title")
		if err != nil {
			return nil, err
		}
		collectionTitle, ok := collectionTitles[collection.ID]
		if !ok {
			if collectionTitle, err = collection.Translate(loc, "title"); err != nil {
				return nil, err
			}
			collectionTitles[collection.ID] = collectionTitle
		}
		if query != "" && !strings.Contains(foldTradeSearch(title+" "+collectionTitle), query) {
			continue
		}
		free := row.Count
		if locked := committed.Copies(row.CardID, row.Foil); free > locked {
			free -= locked
		} else {
			free = 0
		}
		stacks = append(stacks, tradeStack{
			CardID:     row.CardID,
			Foil:       row.Foil,
			Count:      free,
			Title:      title,
			Collection: collectionTitle,
			Rarity:     card.Rarity,
			Image:      "/cards/" + strconv.Itoa(collection.ID*1000+card.ID) + "/thumbnail",
			card:       card,
			collection: collection,
			copies:     row.Count,
		})
	}
	return stacks, nil
}

// tradeStacksPage renders the cards of one page only, and counts every match
func (dc *DataController) tradeStacksPage(c *gin.Context, loc string, userID uint, committed *controllers.TradeCommitments, query string, page int) ([]tradeStack, int, error) {
	stacks, err := dc.tradeStacks(loc, userID, committed, query)
	if err != nil {
		return nil, 0, err
	}
	start := min((page-1)*tradeStacksPageSize, len(stacks))
	end := min(start+tradeStacksPageSize, len(stacks))
	shown := stacks[start:end]
	for i := range shown {
		stack := &shown[i]
		cardData, err := GetCardData(c, loc, dc.Renderer, stack.collection, stack.card)
		if err != nil {
			return nil, 0, err
		}
		cardData["CardCopies"] = stack.copies
		setCardFoil(cardData, stack.Foil)
		if stack.HTML, err = dc.Renderer.RenderToString("card", map[string]any{"Data": cardData}); err != nil {
			return nil, 0, err
		}
	}
	return shown, len(stacks), nil
}

// The stacks of a user a counter starts with, whatever page they are on
func (dc *DataController) tradePrefill(loc string, userID uint, committed *controllers.TradeCommitments, items []models.TradeItem) ([]tradeStack, error) {
	picked := make([]tradeStack, 0, len(items))
	if len(items) == 0 {
		return picked, nil
	}
	stacks, err := dc.tradeStacks(loc, userID, committed, "")
	if err != nil {
		return nil, err
	}
	for _, item := range items {
		for _, stack := range stacks {
			if stack.CardID == item.CardID && stack.Foil == item.Foil && stack.Count > 0 {
				stack.Picked = min(item.Count, stack.Count)
				picked = append(picked, stack)
				break
			}
		}
	}
	return picked, nil
}

// /api/trade/stacks GET: the card stacks of the user and of the trade partner
//
// Query: `u` is the partner; `counter` is the offer being countered, whose
// contents stay unlocked on the partner's side since it will be replaced;
// `side` (`mine` or `theirs`) restricts the reply to one side; `page` (from 1)
// pages each side; `q` filters by card or collection title. Without a side, a
// counter's reply also holds the stacks of the offer it replaces, in
// `mine_picked` and `theirs_picked`
func (dc *DataController) TradeStacksGet(c *gin.Context) {
	user := controllers.GetAuthenticatedUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"message": dc.Renderer.T(c, "error-unauthorized")})
		return
	}

	side := c.Query("side")
	if side != "" && side != "mine" && side != "theirs" {
		c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-invalid")})
		return
	}
	page := 1
	if raw := c.Query("page"); raw != "" {
		parsed, err := strconv.Atoi(raw)
		if err != nil || parsed < 1 {
			c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-invalid")})
			return
		}
		// Bounded, huge pages would overflow the offsets
		page = controllers.PageParam(raw)
	}

	other := c.Query("u")
	if strings.EqualFold(other, user.Username) {
		c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-self")})
		return
	}
	var target models.User
	if err := dc.DB.Omit("avatar").Where("username = ?", other).First(&target).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"message": dc.Renderer.T(c, "trade-error-user")})
		return
	}
	// Compared by ID too: usernames compare ignoring case and trailing spaces in the database
	if target.ID == user.ID {
		c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-self")})
		return
	}
	if blocked, err := controllers.TradeBlocked(dc.DB, user.ID, target.ID); err != nil || blocked {
		c.JSON(http.StatusForbidden, gin.H{"message": dc.Renderer.T(c, "trade-error-blocked")})
		return
	}

	// The countered offer must be one made by the partner to the user: it is
	// the only one its counter replaces and unlocks
	var countered *models.Trade
	if counter := c.Query("counter"); counter != "" {
		id, err := strconv.ParseUint(counter, 10, 32)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-invalid")})
			return
		}
		var old models.Trade
		if err := dc.DB.Where("id = ? AND from_id = ? AND to_id = ? AND state = ?",
			uint(id), target.ID, user.ID, models.TradePending).First(&old).Error; err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": dc.Renderer.T(c, "trade-error-state")})
			return
		}
		countered = &old
	}
	var skipTradeID uint
	if countered != nil {
		skipTradeID = countered.ID
	}

	// Copies put up in other pending offers are locked there. The countered
	// offer is the partner's, so only their side skips it
	mine, err := controllers.TradeCommitmentsOf(dc.DB, user.ID, 0)
	if err != nil {
		controllers.InternalError(c, dc.Renderer, err)
		return
	}
	theirs, err := controllers.TradeCommitmentsOf(dc.DB, target.ID, skipTradeID)
	if err != nil {
		controllers.InternalError(c, dc.Renderer, err)
		return
	}

	loc := dc.Renderer.Localizer.GetLocale(c)
	query := c.Query("q")
	reply := gin.H{
		"coins":       mine.Free(user.Coins),
		"their_coins": theirs.Free(target.Coins),
		"page_size":   tradeStacksPageSize,
	}
	sides := []struct {
		name      string
		userID    uint
		committed *controllers.TradeCommitments
	}{
		{"mine", user.ID, mine},
		{"theirs", target.ID, theirs},
	}
	for _, s := range sides {
		if side != "" && side != s.name {
			continue
		}
		stacks, total, err := dc.tradeStacksPage(c, loc, s.userID, s.committed, query, page)
		if err != nil {
			controllers.InternalError(c, dc.Renderer, err)
			return
		}
		reply[s.name] = stacks
		reply[s.name+"_total"] = total
	}

	// A counter starts mirrored: the cards the partner offered become what
	// the user wants, and what they asked for what the user gives
	if side == "" && countered != nil {
		items, err := countered.Items(dc.DB)
		if err != nil {
			controllers.InternalError(c, dc.Renderer, err)
			return
		}
		if reply["mine_picked"], err = dc.tradePrefill(loc, user.ID, mine, items.Want); err != nil {
			controllers.InternalError(c, dc.Renderer, err)
			return
		}
		if reply["theirs_picked"], err = dc.tradePrefill(loc, target.ID, theirs, items.Give); err != nil {
			controllers.InternalError(c, dc.Renderer, err)
			return
		}
	}

	c.JSON(http.StatusOK, reply)
}
