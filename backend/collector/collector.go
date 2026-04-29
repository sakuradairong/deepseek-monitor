package collector

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// DeepSeekPricing holds model pricing per 1M tokens (in USD)
// Based on https://api-docs.deepseek.com/quick_start/pricing
var DeepSeekPricing = map[string]struct {
	Input     float64 // per 1M input tokens
	Output    float64 // per 1M output tokens
	CacheHit  float64 // per 1M cached input tokens
	CacheMiss float64 // per 1M uncached input tokens
}{
	"deepseek-v4-flash":   {Input: 0.15, Output: 0.60, CacheHit: 0.015, CacheMiss: 0.15},
	"deepseek-v4-pro":       {Input: 0.40, Output: 1.60, CacheHit: 0.04, CacheMiss: 0.40},
	"deepseek-chat":         {Input: 0.27, Output: 1.10, CacheHit: 0.027, CacheMiss: 0.27},
	"deepseek-reasoner":     {Input: 0.55, Output: 2.19, CacheHit: 0.055, CacheMiss: 0.55},
}

func GetPricing(model string) (input, output, cacheHit, cacheMiss float64) {
	p, ok := DeepSeekPricing[model]
	if !ok {
		// Default to v4-flash pricing for unknown models
		p = DeepSeekPricing["deepseek-v4-flash"]
	}
	return p.Input, p.Output, p.CacheHit, p.CacheMiss
}

func EstimateCost(model string, promptTokens, completionTokens, cacheHitTokens, cacheMissTokens int64) float64 {
	inputPrice, outputPrice, cacheHitPrice, cacheMissPrice := GetPricing(model)

	inputCost := float64(promptTokens) / 1_000_000 * inputPrice
	outputCost := float64(completionTokens) / 1_000_000 * outputPrice
	cacheHitCost := float64(cacheHitTokens) / 1_000_000 * cacheHitPrice
	cacheMissCost := float64(cacheMissTokens) / 1_000_000 * cacheMissPrice

	// Effective input cost uses cache prices
	effectiveInputCost := cacheHitCost + cacheMissCost
	if effectiveInputCost == 0 {
		effectiveInputCost = inputCost
	}

	return effectiveInputCost + outputCost
}

// Client wraps the DeepSeek HTTP API
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client
}

func NewClient(baseURL, apiKey string) *Client {
	return &Client{
		baseURL:    strings.TrimRight(baseURL, "/"),
		apiKey:     apiKey,
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}
}

// BalanceResponse represents the /user/balance response
type BalanceResponse struct {
	IsAvailable  bool `json:"is_available"`
	BalanceInfos []struct {
		TotalBalance  string `json:"total_balance"`
		GrantedBalance string `json:"granted_balance"`
		ToppedUpBalance string `json:"topped_up_balance"`
	} `json:"balance_infos"`
}

func (c *Client) GetBalance() (*BalanceResponse, error) {
	req, err := http.NewRequest("GET", c.baseURL+"/user/balance", nil)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	var result BalanceResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &result, nil
}

// RateLimitInfo parsed from response headers
type RateLimitInfo struct {
	RequestsLimit     int64
	RequestsRemaining int64
	TokensLimit       int64
	TokensRemaining   int64
}

// parseRateLimitHeaders extracts rate limit info from HTTP response headers
func parseRateLimitHeaders(headers http.Header) *RateLimitInfo {
	info := &RateLimitInfo{}

	if v := headers.Get("X-RateLimit-Limit-Requests"); v != "" {
		info.RequestsLimit, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := headers.Get("X-RateLimit-Remaining-Requests"); v != "" {
		info.RequestsRemaining, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := headers.Get("X-RateLimit-Limit-Tokens"); v != "" {
		info.TokensLimit, _ = strconv.ParseInt(v, 10, 64)
	}
	if v := headers.Get("X-RateLimit-Remaining-Tokens"); v != "" {
		info.TokensRemaining, _ = strconv.ParseInt(v, 10, 64)
	}

	return info
}

// ProbeAPI sends a minimal request to the chat API to get usage/rate limit info
// Uses a "test" request that minimizes token consumption
func (c *Client) ProbeAPI() (*UsageResponse, *RateLimitInfo, error) {
	payload := `{
		"model": "deepseek-v4-flash",
		"messages": [{"role": "user", "content": "ping"}],
		"max_tokens": 1,
		"stream": false
	}`

	req, err := http.NewRequest("POST", c.baseURL+"/chat/completions",
		strings.NewReader(payload))
	if err != nil {
		return nil, nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("read response: %w", err)
	}

	rateLimit := parseRateLimitHeaders(resp.Header)

	// If it's an error, still return the rate limit info but no usage
	if resp.StatusCode != http.StatusOK {
		var errResp struct {
			Error struct {
				Message string `json:"message"`
				Type    string `json:"type"`
				Code    string `json:"code"`
			} `json:"error"`
		}
		if jsonErr := json.Unmarshal(body, &errResp); jsonErr == nil && errResp.Error.Message != "" {
			return nil, rateLimit, fmt.Errorf("API error (%s): %s", errResp.Error.Type, errResp.Error.Message)
		}
		return nil, rateLimit, fmt.Errorf("API error (status %d)", resp.StatusCode)
	}

	var usageResp UsageResponse
	if err := json.Unmarshal(body, &usageResp); err != nil {
		return nil, rateLimit, fmt.Errorf("parse response: %w", err)
	}

	return &usageResp, rateLimit, nil
}

// UsageResponse captures the usage portion of chat completions response
type UsageResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Usage   struct {
		PromptTokens         int64 `json:"prompt_tokens"`
		CompletionTokens     int64 `json:"completion_tokens"`
		TotalTokens         int64 `json:"total_tokens"`
		PromptCacheHitTokens  int64 `json:"prompt_cache_hit_tokens"`
		PromptCacheMissTokens int64 `json:"prompt_cache_miss_tokens"`
	} `json:"usage"`
}
