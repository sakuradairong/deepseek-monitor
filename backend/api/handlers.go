package api

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"deepseek-monitor/database"
	"deepseek-monitor/models"

	"github.com/gin-gonic/gin"
)

type Handler struct {
	CollectorAPI  string
	CollectorKey  string
}

func NewHandler() *Handler {
	return &Handler{}
}

// Health returns service health status
func (h *Handler) Health(c *gin.Context) {
	dbErr := database.HealthCheck()
	status := http.StatusOK
	response := gin.H{
		"status":   "ok",
		"database": "ok",
	}
	if dbErr != nil {
		status = http.StatusServiceUnavailable
		response["status"] = "degraded"
		response["database"] = dbErr.Error()
	}
	c.JSON(status, response)
}

// GetOverview returns the dashboard overview data
func (h *Handler) GetOverview(c *gin.Context) {
	now := time.Now()
	today := now.Format("2006-01-02")

	// Latest balance
	var balance *models.BalanceSnapshot
	latestBalance, err := database.GetLatestBalance()
	if err == nil {
		balance = latestBalance
	}

	// Today's usage summary
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	todaySummary, _ := database.GetUsageSummary(todayStart, "")

	// This month usage
	monthStart := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, now.Location())
	monthSummary, _ := database.GetUsageSummary(monthStart, "")

	// Latest rate limit
	rateLimit, _ := database.GetLatestRateLimit()

	// Recent errors
	recentErrors, _ := database.GetRecentErrors(5)

	c.JSON(http.StatusOK, gin.H{
		"current_balance":        balance,
		"today_usage":            todaySummary,
		"month_usage":            monthSummary,
		"today_date":             today,
		"latest_rate_limit":      rateLimit,
		"recent_errors":          recentErrors,
	})
}

// GetBalance returns the latest balance
func (h *Handler) GetBalance(c *gin.Context) {
	balance, err := database.GetLatestBalance()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no balance data available"})
		return
	}
	c.JSON(http.StatusOK, balance)
}

// GetBalanceHistory returns balance history
func (h *Handler) GetBalanceHistory(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	snapshots, err := database.GetBalanceHistory(since, days*48) // max 48 samples per day (5min interval)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if snapshots == nil {
		snapshots = []models.BalanceSnapshot{}
	}
	c.JSON(http.StatusOK, snapshots)
}

// GetUsageTrend returns usage over time
func (h *Handler) GetUsageTrend(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "7")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 7
	}
	since := time.Now().AddDate(0, 0, -days)

	trend, err := database.GetUsageTrend(since, "day")
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if trend == nil {
		trend = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, trend)
}

// GetUsageSummary returns aggregated usage summary
func (h *Handler) GetUsageSummary(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)
	model := c.Query("model")

	summary, err := database.GetUsageSummary(since, model)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, summary)
}

// GetModelDistribution returns usage breakdown by model
func (h *Handler) GetModelDistribution(c *gin.Context) {
	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}
	since := time.Now().AddDate(0, 0, -days)

	dist, err := database.GetModelUsageDistribution(since)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if dist == nil {
		dist = []map[string]interface{}{}
	}

	// Calculate total for percentages
	totalTokens := float64(0)
	totalCost := float64(0)
	for _, d := range dist {
		if tokens, ok := d["total_tokens"]; ok {
			totalTokens += toFloat64(tokens)
		}
		if cost, ok := d["cost"]; ok {
			totalCost += toFloat64(cost)
		}
	}

	type ModelInfo struct {
		Model          string  `json:"model"`
		TotalTokens    int64   `json:"total_tokens"`
		Requests       int64   `json:"requests"`
		Cost           float64 `json:"cost"`
		TokenPercent   float64 `json:"token_percent"`
		CostPercent    float64 `json:"cost_percent"`
	}

	var models []ModelInfo
	for _, d := range dist {
		mi := ModelInfo{
			Model:        toString(d["model"]),
			TotalTokens:  toInt64(d["total_tokens"]),
			Requests:     toInt64(d["requests"]),
			Cost:         roundTo(toFloat64(d["cost"]), 4),
		}
		if totalTokens > 0 {
			mi.TokenPercent = math.Round(float64(mi.TotalTokens)/totalTokens*10000) / 100
		}
		if totalCost > 0 {
			mi.CostPercent = math.Round(mi.Cost/totalCost*10000) / 100
		}
		models = append(models, mi)
	}

	c.JSON(http.StatusOK, gin.H{
		"models":      models,
		"total_tokens": int64(totalTokens),
		"total_cost":  roundTo(totalCost, 4),
	})
}

// GetRateLimit returns the latest rate limit info
func (h *Handler) GetRateLimit(c *gin.Context) {
	rateLimit, err := database.GetLatestRateLimit()
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no rate limit data available"})
		return
	}
	c.JSON(http.StatusOK, rateLimit)
}

// GetRecentErrors returns recent API errors
func (h *Handler) GetRecentErrors(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}

	errors, err := database.GetRecentErrors(limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if errors == nil {
		errors = []models.APIErrorRecord{}
	}
	c.JSON(http.StatusOK, errors)
}

// Helper functions
func toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case int64:
		return float64(val)
	case int:
		return float64(val)
	case uint:
		return float64(val)
	case uint64:
		return float64(val)
	case string:
		f, _ := strconv.ParseFloat(val, 64)
		return f
	default:
		return 0
	}
}

func toInt64(v interface{}) int64 {
	switch val := v.(type) {
	case int64:
		return val
	case float64:
		return int64(val)
	case int:
		return int64(val)
	case uint:
		return int64(val)
	case uint64:
		return int64(val)
	case string:
		i, _ := strconv.ParseInt(val, 10, 64)
		return i
	default:
		return 0
	}
}

func toString(v interface{}) string {
	if s, ok := v.(string); ok {
		return s
	}
	return ""
}

func roundTo(f float64, decimals int) float64 {
	pow := math.Pow(10, float64(decimals))
	return math.Round(f*pow) / pow
}
