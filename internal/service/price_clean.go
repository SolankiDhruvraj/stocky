package service

import (
	"context"
	"math/rand"
	"time"

	"stocky/internal/database"

	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type PriceProvider interface {
	GetPrice(ctx context.Context, symbol string) (decimal.Decimal, time.Time, error)
	Start(ctx context.Context, interval time.Duration)
}

type CleanPriceService struct {
	repo *database.Repo
	log  *logrus.Logger
}

func NewCleanPriceService(r *database.Repo, log *logrus.Logger) *CleanPriceService {
	return &CleanPriceService{repo: r, log: log}
}

func (p *CleanPriceService) GetPrice(ctx context.Context, symbol string) (decimal.Decimal, time.Time, error) {
	price, ts, err := p.repo.GetLatestPrice(ctx, symbol)
	if err == nil && time.Since(ts) < 15*time.Minute {
		return price, ts, nil
	}
	val := decimal.NewFromFloat(50 + rand.Float64()*(5000-50))
	ts = time.Now().UTC()
	_ = p.repo.UpsertPrice(ctx, symbol, val, ts) 
	return val, ts, nil
}

func (p *CleanPriceService) Start(ctx context.Context, interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				p.log.Info("price updater stopping")
				return
			case <-ticker.C:
				symbols, err := p.repo.GetAllSymbols(ctx)
				if err != nil {
					p.log.Warnf("failed to fetch symbols: %v", err)
					continue
				}
				for _, s := range symbols {
					val := decimal.NewFromFloat(50 + rand.Float64()*(5000-50))
					_ = p.repo.UpsertPrice(ctx, s, val, time.Now().UTC())
				}
			}
		}
	}()
}
