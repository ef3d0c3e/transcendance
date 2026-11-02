package config_test

import (
	"os"
	"testing"
	"time"

	"transcendance/config"
	"transcendance/models"

	"github.com/stretchr/testify/require"
)

// Databases made before trade stacks got their own table and foils a binary collation are migrated in place
func TestMigrateLegacyTrades(t *testing.T) {
	require.NoError(t, os.Chdir(".."))
	db, cleanup, err := config.ConnectTestDatabase()
	require.NoError(t, err)
	defer cleanup()

	require.NoError(t, db.AutoMigrate(&models.User{}))
	// The former layout: JSON stacks in trades, foils with the default collation
	require.NoError(t, db.Exec(`CREATE TABLE trades (
		id bigint unsigned AUTO_INCREMENT PRIMARY KEY,
		from_id bigint unsigned NOT NULL, to_id bigint unsigned NOT NULL,
		give longtext NOT NULL, coins bigint unsigned NOT NULL DEFAULT 0,
		want longtext NOT NULL, want_coins bigint unsigned NOT NULL DEFAULT 0,
		state varchar(191) NOT NULL, created_at datetime(3), updated_at datetime(3))`).Error)
	require.NoError(t, db.Exec(`CREATE TABLE cards (
		id bigint unsigned AUTO_INCREMENT PRIMARY KEY, card_id bigint unsigned NOT NULL,
		acquired datetime(3) NOT NULL, foil varchar(32) NOT NULL DEFAULT '', owner_id bigint unsigned NOT NULL)`).Error)

	now := time.Now()
	alice := models.User{Email: "alice@example.test", Username: "alice", PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	bob := models.User{Email: "bob@example.test", Username: "bob", PasswordHash: "x", CreatedAt: now, UpdatedAt: now}
	require.NoError(t, db.Create(&alice).Error)
	require.NoError(t, db.Create(&bob).Error)
	require.NoError(t, db.Exec(`INSERT INTO trades (from_id, to_id, give, coins, want, want_coins, state, created_at, updated_at)
		VALUES (?, ?, ?, 10, ?, 0, 'pending', NOW(), NOW()), (?, ?, '[]', 5, 'null', 0, 'cancelled', NOW(), NOW())`,
		alice.ID, bob.ID,
		`[{"card_id":1001,"foil":"","count":2},{"card_id":1002,"foil":"foil3","count":1},{"card_id":1001,"foil":"","count":1}]`,
		`[{"card_id":2001,"foil":"","count":1}]`,
		bob.ID, alice.ID).Error)
	require.NoError(t, db.Exec(`INSERT INTO cards (card_id, acquired, foil, owner_id) VALUES (1001, NOW(), 'foil1', ?)`, alice.ID).Error)

	require.NoError(t, config.MigrateDatabase(db))

	require.False(t, db.Migrator().HasColumn(&models.Trade{}, "give"))
	require.False(t, db.Migrator().HasColumn(&models.Trade{}, "want"))
	var trades []models.Trade
	require.NoError(t, db.Order("id").Find(&trades).Error)
	require.Len(t, trades, 2)
	items, err := trades[0].Items(db)
	require.NoError(t, err)
	// Repeated stacks are merged
	require.ElementsMatch(t, []models.TradeItem{{CardID: 1001, Count: 3}, {CardID: 1002, Foil: "foil3", Count: 1}}, items.Give)
	require.Equal(t, []models.TradeItem{{CardID: 2001, Count: 1}}, items.Want)
	require.Equal(t, uint(10), trades[0].Coins)
	items, err = trades[1].Items(db)
	require.NoError(t, err)
	require.Empty(t, items.Give)
	require.Empty(t, items.Want)

	// Foils no longer match their case or space variants
	var matches int64
	require.NoError(t, db.Model(&models.Card{}).Where("foil = ?", "FOIL1 ").Count(&matches).Error)
	require.Zero(t, matches)
	require.NoError(t, db.Model(&models.Card{}).Where("foil = ?", "foil1").Count(&matches).Error)
	require.Equal(t, int64(1), matches)

	// Migrating again changes nothing
	require.NoError(t, config.MigrateDatabase(db))
	var lines int64
	require.NoError(t, db.Model(&models.TradeLine{}).Count(&lines).Error)
	require.Equal(t, int64(3), lines)
}
