package main

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"time"

	"stocky/internal/database"
	"stocky/internal/handlers"
	"stocky/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"github.com/jmoiron/sqlx"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
)

func main() {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)

	// Load .env file if it exists, but don't fail if it's missing (e.g. in production)
	_ = godotenv.Load()

	dsn := os.Getenv("POSTGRES_URL")
	if dsn == "" {
		logger.Fatal("POSTGRES_URL is required; set to postgres://user:pass@localhost:5432/assignment?sslmode=disable")
	}

	db, err := initDB(dsn)
	if err != nil {
		logger.Fatalf("db connect failed: %v", err)
	}
	defer db.Close()

	r := database.New(db, logger)
	priceSvc := service.NewCleanPriceService(r, logger)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	interval := 3600
	if v := os.Getenv("PRICE_UPDATE_INTERVAL"); v != "" {
		if iv, err := strconv.Atoi(v); err == nil && iv > 0 {
			interval = iv
		}
	}
	priceSvc.Start(ctx, time.Duration(interval)*time.Second)

	_ = r.EnsureStockExists(ctx, "RELIANCE", "Reliance Industries")
	_ = r.EnsureStockExists(ctx, "TCS", "Tata Consultancy Services")
	_ = r.EnsureStockExists(ctx, "INFY", "Infosys")

	h := handlers.NewHandler(r, priceSvc, logger)

	rg := gin.Default()
	rg.GET("/health", func(c *gin.Context) { c.JSON(200, gin.H{"status": "ok"}) })

	rg.POST("/reward", h.PostReward)
	rg.POST("/reward/:id/revert", h.RevertReward)
	rg.GET("/today-stocks/:userId", h.GetTodayStocks)
	rg.GET("/stats/:userId", h.GetStats)
	rg.GET("/historical-inr/:userId", h.GetHistoricalINR)
	rg.GET("/portfolio/:userId", h.GetPortfolio)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	logger.Infof("server starting on :%s", port)
	rg.Run(fmt.Sprintf(":" + port))
}

func initDB(dsn string) (*sqlx.DB, error) {
	db, err := sqlx.Open("postgres", dsn)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(10)
	db.SetMaxIdleConns(5)
	return db, nil
}