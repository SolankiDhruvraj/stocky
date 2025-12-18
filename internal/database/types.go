package database

import "github.com/shopspring/decimal"

type DailyValuation struct {
	TotalINR decimal.Decimal `db:"total_inr" json:"total_inr"`
	Date     string          `db:"date" json:"date"`
}

type PortfolioItem struct {
	Symbol       string          `json:"symbol"`
	Quantity     decimal.Decimal `json:"quantity"`
	CurrentPrice decimal.Decimal `json:"current_price"`
	CurrentValue decimal.Decimal `json:"current_value"`
}

type Holding struct {
	Symbol   string          `db:"symbol" json:"symbol"`
	Quantity decimal.Decimal `db:"quantity" json:"quantity"`
}
