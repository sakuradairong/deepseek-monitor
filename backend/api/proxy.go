package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"deepseek-monitor/collector"
	"deepseek-monitor/database"
	"deepseek-monitor/models"

	"github.com/gin-gonic/gin"
)

var requestIDCounter uint64

const maxRequestBodySize = 10 * 1024 * 1024 // 10MB max request body

// logContentEnabled controls whether prompt/response content is logged
var logContentEnabled = os.Getenv("LOG_CONTENT") != "false"

func nextRequestID() string {
	id := atomic.AddUint64(&requestIDCounter, 1)
	return fmt.Sprintf("req-%d-%d", time.Now().UnixMilli(), id)
}

// ProxyHandler forwards OpenAI-compatible requests to DeepSeek with key rotation
type ProxyHandler struct {
	baseURL    string
	logRepo    *ProxyLogRepo
	fileLogger *FileLogger
}

func NewProxyHandler(fileLogPath string) *ProxyHandler {
	fl, err := NewFileLogger(fileLogPath)
	if err != nil {
		// File logging is optional
		fl = nil
	}

	return &ProxyHandler{
		baseURL:    "https://api.deepseek.com",
		logRepo:    NewProxyLogRepo(),
		fileLogger: fl,
	}
}

func (h *ProxyHandler) HandleProxy(c *gin.Context) {
	reqID := nextRequestID()
	startTime := time.Now()

	// Read request body with size limit
	limitedReader := io.LimitReader(c.Request.Body, maxRequestBodySize)
	bodyBytes, err := io.ReadAll(limitedReader)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read request body"})
		return
	}
	if len(bodyBytes) >= maxRequestBodySize {
		c.JSON(http.StatusRequestEntityTooLarge, gin.H{"error": "request body too large (max 10MB)"})
		return
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	// Extract model name from request
	modelName := extractModel(bodyBytes)

	// Select next API key from rotation
	apiKey, err := database.SelectNextAPIKey()
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"error": "No active API keys available. Add one in the settings.",
		})
		h.recordError(reqID, modelName, 503, "no_active_key", startTime, 0, 0)
		return
	}

	// Build proxy request to DeepSeek
	proxyURL := h.baseURL + "/chat/completions"
	proxyReq, err := http.NewRequest("POST", proxyURL, bytes.NewReader(bodyBytes))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create proxy request"})
		return
	}

	// Copy headers
	proxyReq.Header.Set("Authorization", "Bearer "+apiKey.KeyValue)
	proxyReq.Header.Set("Content-Type", c.GetHeader("Content-Type"))
	if accept := c.GetHeader("Accept"); accept != "" {
		proxyReq.Header.Set("Accept", accept)
	} else {
		proxyReq.Header.Set("Accept", "application/json")
	}

	// Execute proxy call
	client := &http.Client{Timeout: 120 * time.Second}
	resp, err := client.Do(proxyReq)
	latencyMs := time.Since(startTime).Milliseconds()

	// Handle network errors
	if err != nil {
		errMsg := err.Error()
		c.JSON(http.StatusBadGateway, gin.H{
			"error": fmt.Sprintf("DeepSeek API request failed: %s", errMsg),
		})
		h.recordError(reqID, modelName, 502, "network_error", startTime, latencyMs, apiKey.ID)
		database.SetAPIKeyError(apiKey.ID, errMsg)
		database.IncrementAPIKeyUsage(apiKey.ID)
		return
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to read response"})
		return
	}

	// Parse response for token usage
	promptTokens, completionTokens, totalTokens := extractTokens(respBody, resp.StatusCode)
	apiModel := extractResponseModel(respBody)

	// Calculate estimated cost
	cost := collector.EstimateCost(apiModel, promptTokens, completionTokens, 0, promptTokens)

	// Determine error type
	errorType := ""
	statusCode := resp.StatusCode
	if statusCode >= 500 {
		errorType = "5xx"
	} else if statusCode >= 400 {
		errorType = "4xx"
	}

	// Extract previews (safely) — controlled by LOG_CONTENT env var
	promptPreview := ""
	responsePreview := ""
	if logContentEnabled {
		promptPreview = truncateStr(extractPromptPreview(bodyBytes), models.MaxLogPreviewLen)
		responsePreview = truncateStr(extractResponsePreview(respBody), models.MaxLogPreviewLen)
	}

	// Save to DB
	proxyLog := &models.ProxyLog{
		Model:            modelName,
		APIModel:         apiModel,
		PromptTokens:     promptTokens,
		CompletionTokens: completionTokens,
		TotalTokens:      totalTokens,
		LatencyMs:        latencyMs,
		StatusCode:       statusCode,
		ErrorType:        errorType,
		PromptPreview:    promptPreview,
		ResponsePreview:  responsePreview,
		APIKeyID:         apiKey.ID,
		RequestID:        reqID,
		CreatedAt:        startTime,
	}
	if err := h.logRepo.Save(proxyLog); err != nil {
		// Non-fatal error
	}

	// Update real-time metrics
	GlobalMetrics.Record(latencyMs, statusCode, totalTokens, cost)

	// Update key usage
	database.IncrementAPIKeyUsage(apiKey.ID)
	if errorType != "" {
		database.SetAPIKeyError(apiKey.ID, fmt.Sprintf("HTTP %d on request %s", statusCode, reqID))
	} else {
		database.SetAPIKeyError(apiKey.ID, "")
	}

	// Write to log file
	if h.fileLogger != nil {
		h.fileLogger.LogCall(map[string]interface{}{
			"time":              startTime.Format(time.RFC3339),
			"model":             apiModel,
			"status":            statusCode,
			"latency_ms":        latencyMs,
			"prompt_tokens":     promptTokens,
			"completion_tokens": completionTokens,
			"total_tokens":      totalTokens,
			"cost":              cost,
			"key_name":          apiKey.Name,
			"error":             errorType,
			"request_id":        reqID,
		})
	}

	// Stream response back to caller
	for k, v := range resp.Header {
		if len(v) > 0 {
			c.Header(k, v[0])
		}
	}
	c.Data(statusCode, resp.Header.Get("Content-Type"), respBody)
}

func (h *ProxyHandler) recordError(reqID, model string, status int, errType string, start time.Time, latencyMs int64, keyID uint) {
	proxyLog := &models.ProxyLog{
		Model:      model,
		APIModel:   model,
		StatusCode: status,
		ErrorType:  errType,
		LatencyMs:  latencyMs,
		APIKeyID:   keyID,
		RequestID:  reqID,
		CreatedAt:  start,
	}
	h.logRepo.Save(proxyLog)
	GlobalMetrics.Record(latencyMs, status, 0, 0)
}

// --- Helpers ---

func extractModel(body []byte) string {
	var req struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(body, &req) == nil && req.Model != "" {
		return req.Model
	}
	return "unknown"
}

func extractTokens(body []byte, status int) (prompt, completion, total int64) {
	if status >= 400 {
		return 0, 0, 0
	}
	var resp struct {
		Usage *struct {
			PromptTokens     int64 `json:"prompt_tokens"`
			CompletionTokens int64 `json:"completion_tokens"`
			TotalTokens      int64 `json:"total_tokens"`
		} `json:"usage"`
	}
	if json.Unmarshal(body, &resp) == nil && resp.Usage != nil {
		return resp.Usage.PromptTokens, resp.Usage.CompletionTokens, resp.Usage.TotalTokens
	}
	return 0, 0, 0
}

func extractResponseModel(body []byte) string {
	var resp struct {
		Model string `json:"model"`
	}
	if json.Unmarshal(body, &resp) == nil && resp.Model != "" {
		return resp.Model
	}
	return "unknown"
}

func extractPromptPreview(body []byte) string {
	var req struct {
		Messages []struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"messages"`
	}
	if json.Unmarshal(body, &req) != nil {
		return ""
	}
	var parts []string
	for _, msg := range req.Messages {
		if len(parts) >= 2 {
			break
		}
		content := strings.TrimSpace(msg.Content)
		if len(content) > 200 {
			content = content[:200] + "..."
		}
		if content != "" {
			parts = append(parts, fmt.Sprintf("[%s] %s", msg.Role, content))
		}
	}
	return strings.Join(parts, " | ")
}

func extractResponsePreview(body []byte) string {
	var resp struct {
		Choices []struct {
			Message *struct {
				Content string `json:"content"`
			} `json:"message"`
			Delta *struct {
				Content string `json:"content"`
			} `json:"delta"`
		} `json:"choices"`
	}
	if json.Unmarshal(body, &resp) != nil {
		return ""
	}
	for _, choice := range resp.Choices {
		if choice.Message != nil && choice.Message.Content != "" {
			content := strings.TrimSpace(choice.Message.Content)
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			return content
		}
		if choice.Delta != nil && choice.Delta.Content != "" {
			content := strings.TrimSpace(choice.Delta.Content)
			if len(content) > 200 {
				content = content[:200] + "..."
			}
			return content
		}
	}
	return ""
}

func truncateStr(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}

// ProxyQueryHandler handles GET /api/v1/proxy/logs
func (h *ProxyHandler) QueryLogs(c *gin.Context) {
	offset := parseIntQuery(c.Query("offset"), 0)
	limit := parseIntQuery(c.Query("limit"), 50)
	if limit > 200 {
		limit = 200
	}
	model := c.Query("model")
	errorType := c.Query("error_type")
	minStatus := parseIntQuery(c.Query("min_status"), 0)

	logs, total, err := h.logRepo.Query(offset, limit, model, errorType, minStatus)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if logs == nil {
		logs = []models.ProxyLog{}
	}

	c.JSON(http.StatusOK, gin.H{
		"logs":  logs,
		"total": total,
		"limit": limit,
		"offset": offset,
	})
}

// ProxyRealtimeHandler handles GET /api/v1/proxy/realtime
func (h *ProxyHandler) RealtimeMetrics(c *gin.Context) {
	metrics := GlobalMetrics.GetMetrics()
	c.JSON(http.StatusOK, metrics)
}

func parseIntQuery(s string, defaultVal int) int {
	if s == "" {
		return defaultVal
	}
	var val int
	if _, err := fmt.Sscanf(s, "%d", &val); err != nil {
		return defaultVal
	}
	return val
}
