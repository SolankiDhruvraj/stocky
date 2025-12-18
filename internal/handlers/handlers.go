package handlers

import (
	"context"
	"net/http"
	"time"

	"stocky/internal/database"
	"stocky/internal/service"
	"github.com/gin-gonic/gin"
	"github.com/shopspring/decimal"
	"github.com/sirupsen/logrus"
)

type Handler struct {
	repo     *database.Repo
	priceSvc service.PriceProvider
	log      *logrus.Logger
}

func NewHandler(r *database.Repo, p service.PriceProvider, log *logrus.Logger) *Handler {
	return &Handler{repo: r, priceSvc: p, log: log}
}

type RewardRequest struct {
	IdempotencyKey string    `json:"idempotency_key"`
	UserID         string    `json:"user_id" binding:"required"`
	Symbol         string    `json:"symbol" binding:"required"`
	Quantity       string    `json:"quantity" binding:"required"`
	Timestamp      time.Time `json:"timestamp" binding:"required"`
	Source         string    `json:"source"`
}

func (h *Handler) PostReward(c *gin.Context) {
	var req RewardRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.log.Warnf("invalid post body: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// parse quantity
	q, err := decimal.NewFromString(req.Quantity)
	if err != nil {
		h.log.Warnf("invalid quantity: %v", err)
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid quantity format"})
		return
	}

	ctx := context.Background()
	if err := h.repo.EnsureUserExists(ctx, req.UserID, ""); err != nil {
		h.log.Warnf("ensure user: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if err := h.repo.EnsureStockExists(ctx, req.Symbol, req.Symbol); err != nil {
		h.log.Warnf("ensure stock: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}


	price, _, err := h.priceSvc.GetPrice(ctx, req.Symbol)
	if err != nil {
		h.log.Warnf("price fetch failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "price fetch failed"})
		return
	}

	id, created, err := h.repo.CreateReward(ctx, req.UserID, req.Symbol, q, req.Timestamp, req.IdempotencyKey, req.Source, price)
	if err != nil {
		h.log.Errorf("create reward failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "create failed"})
		return
	}
	if !created {
		c.JSON(http.StatusOK, gin.H{"reward_id": id, "status": "already_exists"})
		return
	}
	c.JSON(http.StatusCreated, gin.H{"reward_id": id})
}

func (h *Handler) RevertReward(c *gin.Context) {
	id := c.Param("id")
	if err := h.repo.ReverseReward(context.Background(), id); err != nil {
		h.log.Errorf("revert reward failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "revert failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "reversed"})
}

func (h *Handler) GetTodayStocks(c *gin.Context) {
	userId := c.Param("userId")
	rows, err := h.repo.GetTodayRewards(context.Background(), userId)
	if err != nil {
		h.log.Errorf("query today rewards failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, rows)
}

func (h *Handler) GetPortfolio(c *gin.Context) {
	userId := c.Param("userId")
	items, total, err := h.repo.GetPortfolio(context.Background(), userId)
	if err != nil {
		h.log.Errorf("get portfolio failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"items": items, "total_inr": total.StringFixed(4)})
}

func (h *Handler) GetStats(c *gin.Context) {
	userId := c.Param("userId")
	rows, err := h.repo.GetTodayRewards(context.Background(), userId)
	if err != nil {
		h.log.Errorf("get today rewards failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	sharesToday := map[string]decimal.Decimal{}
	for _, r := range rows {
		sym := r.Symbol
		q := r.Quantity
		sharesToday[sym] = sharesToday[sym].Add(q)
	}


	_, total, err := h.repo.GetPortfolio(context.Background(), userId)
	if err != nil {
		h.log.Errorf("get portfolio failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}

	sharesTodayStr := map[string]string{}
	for k, v := range sharesToday {
		sharesTodayStr[k] = v.StringFixed(6)
	}
	c.JSON(http.StatusOK, gin.H{"shares_today": sharesTodayStr, "current_inr_value": total.StringFixed(4)})
}

func (h *Handler) GetHistoricalINR(c *gin.Context) {
	userId := c.Param("userId")
	rows, err := h.repo.GetDailyValuations(context.Background(), userId)
	if err != nil {
		h.log.Errorf("get daily valuations failed: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query failed"})
		return
	}
	res := []map[string]string{}
	for _, r := range rows {
		res = append(res, map[string]string{"date": r.Date, "inr_value": r.TotalINR.StringFixed(4)})
	}
	c.JSON(http.StatusOK, res)
}