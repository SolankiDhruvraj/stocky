package models

import "time"

type Reward struct {
	ID        string    `db:"id" json:"reward_id"`
	UserID    string    `db:"user_id" json:"user_id"`
	Symbol    string    `db:"symbol" json:"symbol"`
	Quantity  string    `db:"quantity" json:"quantity"`
	Timestamp time.Time `db:"timestamp" json:"timestamp"`
}
