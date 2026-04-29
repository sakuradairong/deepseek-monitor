package api

import (
	"fmt"
	"net/http"
	"strconv"

	"deepseek-monitor/collector"
	"deepseek-monitor/database"
	"deepseek-monitor/models"

	"github.com/gin-gonic/gin"
)

type KeyHandler struct{}

func NewKeyHandler() *KeyHandler {
	return &KeyHandler{}
}

// KeyResponse is a safe representation of APIKey (without the full key value)
type KeyResponse struct {
	ID         uint    `json:"id"`
	Name       string  `json:"name"`
	KeyPrefix  string  `json:"key_prefix"`
	IsActive   bool    `json:"is_active"`
	Priority   int     `json:"priority"`
	UsageCount int64   `json:"usage_count"`
	LastUsedAt *string `json:"last_used_at"`
	LastError  string  `json:"last_error"`
	CreatedAt  string  `json:"created_at"`
	UpdatedAt  string  `json:"updated_at"`
}

func toKeyResponse(k *models.APIKey) KeyResponse {
	r := KeyResponse{
		ID:         k.ID,
		Name:       k.Name,
		KeyPrefix:  k.KeyPrefix,
		IsActive:   k.IsActive,
		Priority:   k.Priority,
		UsageCount: k.UsageCount,
		LastError:  k.LastError,
		CreatedAt:  k.CreatedAt.Format("2006-01-02T15:04:05Z"),
		UpdatedAt:  k.UpdatedAt.Format("2006-01-02T15:04:05Z"),
	}
	if k.LastUsedAt != nil {
		s := k.LastUsedAt.Format("2006-01-02T15:04:05Z")
		r.LastUsedAt = &s
	}
	return r
}

// ListKeys returns all API keys (masked)
func (h *KeyHandler) ListKeys(c *gin.Context) {
	keys, err := database.ListAPIKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	resp := make([]KeyResponse, 0, len(keys))
	for i := range keys {
		resp = append(resp, toKeyResponse(&keys[i]))
	}
	c.JSON(http.StatusOK, resp)
}

// CreateKey adds a new API key
func (h *KeyHandler) CreateKey(c *gin.Context) {
	var req struct {
		Name     string `json:"name" binding:"required"`
		KeyValue string `json:"key_value" binding:"required"`
		Priority int    `json:"priority"`
		IsActive *bool  `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Extract prefix for display
	prefix := req.KeyValue
	if len(prefix) > 12 {
		prefix = prefix[:12] + "..."
	}

	key := &models.APIKey{
		Name:      req.Name,
		KeyValue:  req.KeyValue,
		KeyPrefix: prefix,
		Priority:  req.Priority,
		IsActive:  true,
	}
	if req.IsActive != nil {
		key.IsActive = *req.IsActive
	}

	if err := database.CreateAPIKey(key); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, toKeyResponse(key))
}

// UpdateKey updates an existing API key
func (h *KeyHandler) UpdateKey(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	existing, err := database.GetAPIKeyByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	var req struct {
		Name     *string `json:"name"`
		KeyValue *string `json:"key_value"`
		IsActive *bool   `json:"is_active"`
		Priority *int    `json:"priority"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name != nil {
		existing.Name = *req.Name
	}
	if req.KeyValue != nil {
		existing.KeyValue = *req.KeyValue
		prefix := *req.KeyValue
		if len(prefix) > 12 {
			prefix = prefix[:12] + "..."
		}
		existing.KeyPrefix = prefix
	}
	if req.IsActive != nil {
		existing.IsActive = *req.IsActive
	}
	if req.Priority != nil {
		existing.Priority = *req.Priority
	}

	if err := database.UpdateAPIKey(existing); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Clear error on successful update
	database.SetAPIKeyError(existing.ID, "")

	c.JSON(http.StatusOK, toKeyResponse(existing))
}

// DeleteKey removes an API key
func (h *KeyHandler) DeleteKey(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	if err := database.DeleteAPIKey(uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// TestKey tests an API key against DeepSeek API
func (h *KeyHandler) TestKey(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	key, err := database.GetAPIKeyByID(uint(id))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "key not found"})
		return
	}

	client := collector.NewClient("https://api.deepseek.com", key.KeyValue)
	balance, err := client.GetBalance()

	if err != nil {
		errMsg := err.Error()
		database.SetAPIKeyError(key.ID, errMsg)
		c.JSON(http.StatusOK, gin.H{
			"success": false,
			"error":   errMsg,
			"message": "API Key 测试失败",
		})
		return
	}

	// Clear error on success
	database.SetAPIKeyError(key.ID, "")

	total := 0.0
	granted := 0.0
	toppedUp := 0.0
	for _, bi := range balance.BalanceInfos {
		fmt.Sscanf(bi.TotalBalance, "%f", &total)
		fmt.Sscanf(bi.GrantedBalance, "%f", &granted)
		fmt.Sscanf(bi.ToppedUpBalance, "%f", &toppedUp)
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"is_available": balance.IsAvailable,
		"total_balance":  total,
		"granted_balance": granted,
		"topped_up_balance": toppedUp,
		"message": "API Key 有效",
	})
}

// ListKeyNames returns only the key names and IDs for dropdown selection
func (h *KeyHandler) ListKeyNames(c *gin.Context) {
	keys, err := database.ListActiveAPIKeys()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	type KeyName struct {
		ID        uint   `json:"id"`
		Name      string `json:"name"`
		KeyPrefix string `json:"key_prefix"`
	}
	names := make([]KeyName, 0, len(keys))
	for _, k := range keys {
		names = append(names, KeyName{ID: k.ID, Name: k.Name, KeyPrefix: k.KeyPrefix})
	}
	c.JSON(http.StatusOK, names)
}
