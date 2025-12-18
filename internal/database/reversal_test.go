package database

import (
	"context"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

func TestReverseReward(t *testing.T) {
	db := setupDB(t)
	logger := logrus.New()
	r := New(db, logger)

	userID := "test-reverse-user"
	_, err := db.Exec("INSERT INTO users (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING", userID, "Test Reverse User")
	if err != nil {
		t.Fatalf("create user failed: %v", err)
	}

	symbol := "RELIANCE"
	q := decimal.NewFromFloat(10.0)
	idKey := "test-reverse-key"
	_, _ = db.Exec("DELETE FROM ledger_entries WHERE reward_id IN (SELECT id FROM rewards WHERE idempotency_key = $1)", idKey)
	_, _ = db.Exec("DELETE FROM rewards WHERE idempotency_key = $1", idKey)
	_, _ = db.Exec("DELETE FROM holdings WHERE user_id = $1 AND symbol = $2", userID, symbol)


	id, created, err := r.CreateReward(context.Background(), userID, symbol, q, time.Now().UTC(), idKey, "test", decimal.NewFromFloat(100))
	if err != nil {
		t.Fatalf("create reward failed: %v", err)
	}
	if !created {
		t.Fatalf("expected created true")
	}


	holdings, err := r.GetHoldings(context.Background(), userID)
	if err != nil {
		t.Fatalf("get holdings failed: %v", err)
	}
	if len(holdings) != 1 || !holdings[0].Quantity.Equal(q) {
		t.Fatalf("expected holdings %s, got %v", q, holdings)
	}


	if err := r.ReverseReward(context.Background(), id); err != nil {
		t.Fatalf("reverse reward failed: %v", err)
	}


	holdings, err = r.GetHoldings(context.Background(), userID)
	if err != nil {
		t.Fatalf("get holdings failed: %v", err)
	}
	if len(holdings) != 1 || !holdings[0].Quantity.Equal(decimal.Zero) {
		t.Fatalf("expected holdings 0, got %v", holdings)
	}


	var status string
	err = db.QueryRow("SELECT status FROM rewards WHERE id = $1", id).Scan(&status)
	if err != nil {
		t.Fatalf("get status failed: %v", err)
	}
	if status != "REVERSED" {
		t.Fatalf("expected status REVERSED, got %s", status)
	}
}
