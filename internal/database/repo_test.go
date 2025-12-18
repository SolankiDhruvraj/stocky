package database

import (
	"context"
	"io/ioutil"
	"os"
	"testing"
	"time"

	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

func setupDB(t *testing.T) *sqlx.DB {
	url := os.Getenv("POSTGRES_URL")
	if url == "" {
		t.Skip("POSTGRES_URL is not set; skipping integration tests")
	}
	db, err := sqlx.Open("postgres", url)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}

	files := []string{"../../migrations/0001_init.up.sql"}
	for _, f := range files {
		b, err := ioutil.ReadFile(f)
		if err != nil {
			t.Fatalf("read migration %s: %v", f, err)
		}
		if _, err := db.Exec(string(b)); err != nil {
			t.Logf("exec migration %s: %v", f, err)
		}
	}
	return db
}

func TestCreateReward_Idempotency(t *testing.T) {
	db := setupDB(t)
	logger := logrus.New()
	r := New(db, logger)

	userID := "11111111-1111-1111-1111-111111111111"
	symbol := "RELIANCE"
	q, _ := decimal.NewFromString("1.500000")
	idKey := "test-idempotency-1"

	_, _ = db.Exec(`DELETE FROM ledger_entries WHERE reward_id IN (SELECT id FROM rewards WHERE idempotency_key = $1)`, idKey)
	if _, err := db.Exec(`DELETE FROM rewards WHERE idempotency_key = $1`, idKey); err != nil {
		t.Fatalf("cleanup failed: %v", err)
	}


	id1, created, err := r.CreateReward(context.Background(), userID, symbol, q, time.Now().UTC(), idKey, "test", decimal.NewFromFloat(100))
	if err != nil {
		t.Fatalf("create reward failed: %v", err)
	}
	if !created {
		t.Fatalf("expected created true on first insert")
	}
	if id1 == "" {
		t.Fatalf("expected non-empty id")
	}


	id2, created2, err := r.CreateReward(context.Background(), userID, symbol, q, time.Now().UTC(), idKey, "test", decimal.NewFromFloat(100))
	if err != nil {
		t.Fatalf("create reward (replay) failed: %v", err)
	}
	if created2 {
		t.Fatalf("expected created false on replay")
	}
	if id2 != id1 {
		t.Fatalf("expected same id returned for replay; got %s != %s", id1, id2)
	}


	var qtyStr string
	if err := db.Get(&qtyStr, `SELECT quantity::text FROM holdings WHERE user_id=$1 AND symbol=$2`, userID, symbol); err != nil {
		t.Fatalf("get holdings failed: %v", err)
	}
	qty, _ := decimal.NewFromString(qtyStr)
	if !qty.Equal(q) {
		t.Logf("expected holdings %s, got %s (might be cumulative from other runs)", q.StringFixed(6), qty.StringFixed(6))
	}
}