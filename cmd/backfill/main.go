package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	godotenv.Load()
	dbURL := os.Getenv("POSTGRES_URL")
	if dbURL == "" {
		log.Fatal("POSTGRES_URL is required")
	}

	db, err := sqlx.Connect("postgres", dbURL)
	if err != nil {
		log.Fatalf("failed to connect to db: %v", err)
	}
	defer db.Close()

	ctx := context.Background()
	userID := "demo-user"
	targetDay := time.Now().AddDate(0, 0, -1).UTC().Truncate(24 * time.Hour)
	yesterdayTS := targetDay.Add(-1 * time.Second) // 1 second before yesterday starts

	fmt.Printf("Backfilling data for %s with timestamp %s (to count for %s)...\n", userID, yesterdayTS.Format(time.RFC3339), targetDay.Format("2006-01-02"))

	// 1. Add historical prices for that timestamp
	prices := map[string]string{
		"RELIANCE": "2500.50",
		"TCS":      "3400.75",
		"INFY":     "1500.25",
	}

	for sym, p := range prices {
		_, err := db.ExecContext(ctx, `INSERT INTO price_history (symbol, price_inr, timestamp) VALUES ($1, $2::numeric, $3)`, sym, p, yesterdayTS)
		if err != nil {
			fmt.Printf("Warning: could not insert price for %s: %v\n", sym, err)
		}
	}

	// 2. Add a reward for that timestamp
	rewardID := "backfill-reward-v2"
	symbol := "RELIANCE"
	qty := "10.0"
	
	// Insert reward
	_, err = db.ExecContext(ctx, `
		INSERT INTO rewards (id, user_id, symbol, quantity, timestamp, idempotency_key, source, status) 
		VALUES (gen_random_uuid(), $1, $2, $3::numeric, $4, $5, $6, 'COMPLETED')`,
		userID, symbol, qty, yesterdayTS, rewardID, "backfill")
	if err != nil {
		fmt.Printf("Warning: could not insert reward: %v\n", err)
	}

	// 3. Update holdings (since holdings is a snapshot, we just ensure the user has these shares now)
	_, err = db.ExecContext(ctx, `
		INSERT INTO holdings (user_id, symbol, quantity, last_updated) 
		VALUES ($1, $2, $3::numeric, now()) 
		ON CONFLICT (user_id, symbol) DO UPDATE SET quantity = holdings.quantity + $3::numeric`,
		userID, symbol, qty)
	
	fmt.Println("Successfully backfilled yesterday's data!")
	fmt.Println("Now refresh: https://api.dhruvrajsolanki.in/historical-inr/demo-user")
}
