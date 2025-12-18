package database

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestComputeHistoricalValuations_Reproduction(t *testing.T) {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set; skipping integration tests")
	}
	db, err := sqlx.Open("postgres", url)
	require.NoError(t, err)
	defer db.Close()

	logger := logrus.New()
	r := New(db, logger)

	ctx := context.Background()
	userID := "historical-test-user"

	// Cleanup
	_, _ = db.ExecContext(ctx, "DELETE FROM ledger_entries WHERE reward_id IN (SELECT id FROM rewards WHERE user_id = $1)", userID)
	_, _ = db.ExecContext(ctx, "DELETE FROM rewards WHERE user_id = $1", userID)
	_, _ = db.ExecContext(ctx, "DELETE FROM holdings WHERE user_id = $1", userID)
	_, _ = db.ExecContext(ctx, "DELETE FROM users WHERE id = $1", userID)
	_, _ = db.ExecContext(ctx, "DELETE FROM price_history WHERE symbol IN ('RELIANCE', 'TCS')")

	// Setup
	require.NoError(t, r.EnsureUserExists(ctx, userID, "Historical Test User"))
	require.NoError(t, r.EnsureStockExists(ctx, "RELIANCE", "Reliance Industries"))
	require.NoError(t, r.EnsureStockExists(ctx, "TCS", "Tata Consultancy Services"))

	// Seed data for 2 days ago
	twoDaysAgo := time.Now().UTC().AddDate(0, 0, -2).Truncate(24 * time.Hour).Add(12 * time.Hour)
	require.NoError(t, r.UpsertPrice(ctx, "RELIANCE", decimal.NewFromFloat(2500.0), twoDaysAgo))
	_, created, err := r.CreateReward(ctx, userID, "RELIANCE", decimal.NewFromFloat(10.0), twoDaysAgo, "hist-test-1", "test", decimal.NewFromFloat(2500.0))
	require.NoError(t, err)
	require.True(t, created)

	// Seed data for yesterday
	yesterday := time.Now().UTC().AddDate(0, 0, -1).Truncate(24 * time.Hour).Add(12 * time.Hour)
	require.NoError(t, r.UpsertPrice(ctx, "RELIANCE", decimal.NewFromFloat(2600.0), yesterday))
	require.NoError(t, r.UpsertPrice(ctx, "TCS", decimal.NewFromFloat(3500.0), yesterday))
	_, created, err = r.CreateReward(ctx, userID, "TCS", decimal.NewFromFloat(5.0), yesterday, "hist-test-2", "test", decimal.NewFromFloat(3500.0))
	require.NoError(t, err)
	require.True(t, created)

	// Compute historical valuations
	valuations, err := r.ComputeHistoricalValuations(ctx, userID)
	require.NoError(t, err)

	// We expect at least 2 entries: one for 2 days ago and one for yesterday
	// The logic computes up to yesterday.
	
	t.Logf("Valuations: %+v", valuations)

	foundYesterday := false
	foundTwoDaysAgo := false

	yesterdayStr := time.Now().UTC().AddDate(0, 0, -1).Format("2006-01-02")
	twoDaysAgoStr := time.Now().UTC().AddDate(0, 0, -2).Format("2006-01-02")

	for _, v := range valuations {
		if v.Date == yesterdayStr {
			foundYesterday = true
			// Expected: (10 RELIANCE * 2600) + (5 TCS * 3500) = 26000 + 17500 = 43500
			assert.True(t, v.TotalINR.Equal(decimal.NewFromFloat(43500)), "Yesterday valuation mismatch: expected 43500, got %s", v.TotalINR.String())
		}
		if v.Date == twoDaysAgoStr {
			foundTwoDaysAgo = true
			// Expected: (10 RELIANCE * 2500) = 25000
			assert.True(t, v.TotalINR.Equal(decimal.NewFromFloat(25000)), "Two days ago valuation mismatch: expected 25000, got %s", v.TotalINR.String())
		}
	}

	assert.True(t, foundYesterday, "Yesterday's valuation not found")
	assert.True(t, foundTwoDaysAgo, "Two days ago valuation not found")
}
