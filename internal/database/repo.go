package database

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type Repo struct {
	db  *sqlx.DB
	log *logrus.Logger
}

func New(db *sqlx.DB, log *logrus.Logger) *Repo {
	return &Repo{db: db, log: log}
}

func (r *Repo) CreateReward(ctx context.Context, userID, symbol string, quantity decimal.Decimal, ts time.Time, idempotencyKey, source string, price decimal.Decimal) (string, bool, error) {
	var existingID sql.NullString
	if idempotencyKey != "" {
		err := r.db.GetContext(ctx, &existingID, "SELECT id FROM rewards WHERE idempotency_key = $1 LIMIT 1", idempotencyKey)
		if err == nil && existingID.Valid {
			return existingID.String, false, nil
		}
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return "", false, err
	}
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		}
	}()

	var rewardID string
	q := `INSERT INTO rewards (id, user_id, symbol, quantity, timestamp, idempotency_key, source, created_at, status) VALUES (gen_random_uuid(), $1, $2, $3::numeric, $4, $5, $6, now(), 'COMPLETED') RETURNING id`
	if err := tx.QueryRowContext(ctx, q, userID, symbol, quantity.String(), ts, idempotencyKey, source).Scan(&rewardID); err != nil {
		tx.Rollback()
		if pqErr, ok := err.(*pq.Error); ok && pqErr.Code == "23505" {
			var existing string
			if err := r.db.GetContext(ctx, &existing, "SELECT id FROM rewards WHERE idempotency_key=$1 LIMIT 1", idempotencyKey); err == nil {
				return existing, false, nil
			}
		}
		return "", false, err
	}


	amountINR := quantity.Mul(price)
	fees := amountINR.Mul(decimal.NewFromFloat(0.01)).Round(4)
	totalCashOut := amountINR.Add(fees)

	ledgerQ := `INSERT INTO ledger_entries (id, reward_id, entry_time, account_debit, account_credit, amount_inr, stock_symbol, stock_quantity, description) VALUES (gen_random_uuid(), $1, now(), $2, $3, $4::numeric, $5, $6::numeric, $7)`
	if _, err := tx.ExecContext(ctx, ledgerQ, rewardID, "company_cash", "stock_inventory", totalCashOut.StringFixed(4), symbol, quantity.StringFixed(6), "reward purchase"); err != nil {
		tx.Rollback()
		return "", false, err
	}
	if fees.Cmp(decimal.Zero) > 0 {
		if _, err := tx.ExecContext(ctx, ledgerQ, rewardID, "company_expense", "company_cash", fees.StringFixed(4), nil, nil, "fees for reward"); err != nil {
			tx.Rollback()
			return "", false, err
		}
	}


	upsert := `INSERT INTO holdings (user_id, symbol, quantity, last_updated) VALUES ($1, $2, $3::numeric, now()) ON CONFLICT (user_id, symbol) DO UPDATE SET quantity = holdings.quantity + $3::numeric, last_updated = now()`
	if _, err := tx.ExecContext(ctx, upsert, userID, symbol, quantity.String()); err != nil {
		tx.Rollback()
		return "", false, err
	}

	if err := tx.Commit(); err != nil {
		return "", false, err
	}
	return rewardID, true, nil
}

func (r *Repo) ReverseReward(ctx context.Context, rewardID string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()

	var status string
	var userID, symbol string
	var quantity decimal.Decimal
	if err := tx.QueryRowContext(ctx, `SELECT status, user_id, symbol, quantity FROM rewards WHERE id = $1 FOR UPDATE`, rewardID).Scan(&status, &userID, &symbol, &quantity); err != nil {
		return err
	}
	if status != "COMPLETED" {
		return sql.ErrNoRows
	}


	if _, err := tx.ExecContext(ctx, `UPDATE rewards SET status = 'REVERSED' WHERE id = $1`, rewardID); err != nil {
		return err
	}

	if _, err := tx.ExecContext(ctx, `UPDATE holdings SET quantity = quantity - $1::numeric, last_updated = now() WHERE user_id = $2 AND symbol = $3`, quantity.String(), userID, symbol); err != nil {
		return err
	}

	return tx.Commit()
}

type Reward struct {
	ID        string          `db:"id" json:"id"`
	Symbol    string          `db:"symbol" json:"symbol"`
	Quantity  decimal.Decimal `db:"quantity" json:"quantity"`
	Timestamp time.Time       `db:"timestamp" json:"timestamp"`
}

func (r *Repo) GetTodayRewards(ctx context.Context, userID string) ([]Reward, error) {
	start := time.Now().UTC().Truncate(24 * time.Hour)
	end := start.Add(24 * time.Hour)
	rows, err := r.db.QueryxContext(ctx, `SELECT id, symbol, quantity, timestamp FROM rewards WHERE user_id = $1 AND timestamp >= $2 AND timestamp < $3 ORDER BY timestamp ASC`, userID, start, end)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []Reward{}
	for rows.Next() {
		var m Reward
		if err := rows.StructScan(&m); err != nil {
			r.log.Warnf("scan row failed: %v", err)
			continue
		}
		res = append(res, m)
	}
	return res, nil
}

func (r *Repo) GetHoldings(ctx context.Context, userID string) ([]Holding, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT symbol, quantity FROM holdings WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []Holding{}
	for rows.Next() {
		var h Holding
		if err := rows.StructScan(&h); err != nil {
			r.log.Warnf("scan holding failed: %v", err)
			continue
		}
		res = append(res, h)
	}
	return res, nil
}

func (r *Repo) GetLatestPrice(ctx context.Context, symbol string) (decimal.Decimal, time.Time, error) {
	var priceStr string
	var ts time.Time
	if err := r.db.QueryRowContext(ctx, `SELECT price_inr, timestamp FROM price_history WHERE symbol = $1 ORDER BY timestamp DESC LIMIT 1`, symbol).Scan(&priceStr, &ts); err != nil {
		return decimal.Zero, time.Time{}, err
	}
	p, err := decimal.NewFromString(priceStr)
	if err != nil {
		return decimal.Zero, time.Time{}, err
	}
	return p, ts, nil
}

func (r *Repo) UpsertPrice(ctx context.Context, symbol string, price decimal.Decimal, ts time.Time) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO price_history (symbol, price_inr, timestamp) VALUES ($1, $2::numeric, $3)`, symbol, price.StringFixed(4), ts)
	return err
}

func (r *Repo) GetAllSymbols(ctx context.Context) ([]string, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT symbol FROM stocks`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	res := []string{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			r.log.Warnf("scan symbol failed: %v", err)
			continue
		}
		res = append(res, s)
	}
	return res, nil
}

func (r *Repo) GetDailyValuations(ctx context.Context, userID string) ([]DailyValuation, error) {
	rows, err := r.db.QueryxContext(ctx, `SELECT date, total_inr FROM daily_valuations WHERE user_id = $1 AND date < CURRENT_DATE ORDER BY date ASC`, userID)
	if err == nil {
		defer rows.Close()
		res := []DailyValuation{}
		for rows.Next() {
			var d DailyValuation
			var totalStr string
			if err := rows.Scan(&d.Date, &totalStr); err != nil {
				r.log.Warnf("scan daily valuation failed: %v", err)
				continue
			}
			p, _ := decimal.NewFromString(totalStr)
			d.TotalINR = p
			res = append(res, d)
		}
		if len(res) > 0 {
			return res, nil
		}
	}

	return r.ComputeHistoricalValuations(ctx, userID)
}

func (r *Repo) ComputeHistoricalValuations(ctx context.Context, userID string) ([]DailyValuation, error) {
	// find the earliest reward date
	var minDate sql.NullTime
	if err := r.db.GetContext(ctx, &minDate, `SELECT MIN(timestamp) FROM rewards WHERE user_id = $1`, userID); err != nil {
		return nil, err
	}
	if !minDate.Valid {
		return []DailyValuation{}, nil
	}
	start := minDate.Time.UTC().Truncate(24 * time.Hour)
	end := time.Now().UTC().Truncate(24 * time.Hour).Add(-24 * time.Hour)
	if !start.Before(end) && !start.Equal(end) {
		return []DailyValuation{}, nil
	}
	res := []DailyValuation{}
	for d := start; !d.After(end); d = d.Add(24 * time.Hour) {
		// targetTS is the very last moment of day 'd' in UTC
		targetTS := d.Add(24 * time.Hour).Add(-1 * time.Microsecond)
		
		rows, err := r.db.QueryxContext(ctx, `
			SELECT symbol, COALESCE(SUM(quantity)::text,'0') AS qty 
			FROM rewards 
			WHERE user_id = $1 AND timestamp <= $2 AND status = 'COMPLETED' 
			GROUP BY symbol`, userID, targetTS)
		if err != nil {
			r.log.Warnf("get cumulative quantities failed for %v: %v", d, err)
			continue
		}
		
		var total decimal.Decimal
		for rows.Next() {
			var sym string
			var qtyStr string
			if err := rows.Scan(&sym, &qtyStr); err != nil {
				r.log.Warnf("scan cum qty failed: %v", err)
				continue
			}
			qty, _ := decimal.NewFromString(qtyStr)
			
			var priceStr sql.NullString
			err := r.db.GetContext(ctx, &priceStr, `
				SELECT price_inr 
				FROM price_history 
				WHERE symbol = $1 AND timestamp <= $2 
				ORDER BY timestamp DESC LIMIT 1`, sym, targetTS)
			
			if err != nil || !priceStr.Valid {
				continue
			}
			
			p, _ := decimal.NewFromString(priceStr.String)
			total = total.Add(qty.Mul(p))
		}
		rows.Close()
		res = append(res, DailyValuation{Date: d.Format("2006-01-02"), TotalINR: total})
	}
	return res, nil
}

func (r *Repo) GetPortfolio(ctx context.Context, userID string) ([]PortfolioItem, decimal.Decimal, error) {
	holdings, err := r.GetHoldings(ctx, userID)
	if err != nil {
		return nil, decimal.Zero, err
	}
	items := []PortfolioItem{}
	total := decimal.Zero
	for _, h := range holdings {
		price, _, err := r.GetLatestPrice(ctx, h.Symbol)
		if err != nil {
			r.log.Warnf("no price for symbol %s: %v", h.Symbol, err)
			continue
		}
		value := h.Quantity.Mul(price)
		items = append(items, PortfolioItem{Symbol: h.Symbol, Quantity: h.Quantity, CurrentPrice: price, CurrentValue: value})
		total = total.Add(value)
	}
	return items, total, nil
}

func (r *Repo) EnsureStockExists(ctx context.Context, symbol, name string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO stocks (symbol, name) VALUES ($1, $2) ON CONFLICT (symbol) DO NOTHING`, symbol, name)
	return err
}

func (r *Repo) EnsureUserExists(ctx context.Context, userID, name string) error {
	_, err := r.db.ExecContext(ctx, `INSERT INTO users (id, name) VALUES ($1, $2) ON CONFLICT (id) DO NOTHING`, userID, name)
	return err
}
