package models

import "time"

// --- Existing models kept as-is from original ---

type BalanceSnapshot struct {
	ID             uint      `gorm:"primaryKey" json:"id"`
	APIKeyID       uint      `gorm:"index;default:0" json:"api_key_id"`
	TotalBalance   float64   `gorm:"not null" json:"total_balance"`
	GrantedBalance float64   `gorm:"not null;default:0" json:"granted_balance"`
	ToppedUpBalance float64  `gorm:"not null;default:0" json:"topped_up_balance"`
	IsAvailable    bool      `gorm:"not null" json:"is_available"`
	CollectedAt    time.Time `gorm:"index;not null" json:"collected_at"`
}

func (BalanceSnapshot) TableName() string { return "balance_snapshots" }

type UsageRecord struct {
	ID                   uint      `gorm:"primaryKey" json:"id"`
	APIKeyID             uint      `gorm:"index;default:0" json:"api_key_id"`
	Model                string    `gorm:"index;not null" json:"model"`
	PromptTokens         int64     `gorm:"not null" json:"prompt_tokens"`
	CompletionTokens     int64     `gorm:"not null" json:"completion_tokens"`
	TotalTokens          int64     `gorm:"not null" json:"total_tokens"`
	PromptCacheHitTokens int64     `gorm:"default:0" json:"prompt_cache_hit_tokens"`
	PromptCacheMissTokens int64    `gorm:"default:0" json:"prompt_cache_miss_tokens"`
	EstimatedCost        float64   `gorm:"not null;default:0" json:"estimated_cost"`
	CollectedAt          time.Time `gorm:"index;not null" json:"collected_at"`
}

func (UsageRecord) TableName() string { return "usage_records" }

type RateLimitRecord struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	APIKeyID         uint      `gorm:"index;default:0" json:"api_key_id"`
	RequestsLimit    int64     `gorm:"not null" json:"requests_limit"`
	RequestsRemaining int64    `gorm:"not null" json:"requests_remaining"`
	TokensLimit      int64     `gorm:"not null" json:"tokens_limit"`
	TokensRemaining  int64     `gorm:"not null" json:"tokens_remaining"`
	CollectedAt      time.Time `gorm:"index;not null" json:"collected_at"`
}

func (RateLimitRecord) TableName() string { return "rate_limit_records" }

type DailyUsageSummary struct {
	ID                    uint      `gorm:"primaryKey" json:"id"`
	Date                  string    `gorm:"uniqueIndex:idx_date_model;type:text;not null" json:"date"`
	Model                 string    `gorm:"uniqueIndex:idx_date_model;not null" json:"model"`
	TotalPromptTokens     int64     `gorm:"not null" json:"total_prompt_tokens"`
	TotalCompletionTokens int64     `gorm:"not null" json:"total_completion_tokens"`
	TotalTokens           int64     `gorm:"not null" json:"total_tokens"`
	TotalRequests         int64     `gorm:"not null;default:0" json:"total_requests"`
	EstimatedCost         float64   `gorm:"not null;default:0" json:"estimated_cost"`
	CreatedAt             time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt             time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (DailyUsageSummary) TableName() string { return "daily_usage_summaries" }

type APIErrorRecord struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	APIKeyID    uint      `gorm:"index;default:0" json:"api_key_id"`
	ErrorType   string    `gorm:"index;not null" json:"error_type"`
	StatusCode  int       `gorm:"not null" json:"status_code"`
	Message     string    `gorm:"type:text" json:"message"`
	CollectedAt time.Time `gorm:"index;not null" json:"collected_at"`
}

func (APIErrorRecord) TableName() string { return "api_error_records" }

// --- NEW MODELS ---

// User represents an authenticated user
type User struct {
	ID           uint      `gorm:"primaryKey" json:"id"`
	Username     string    `gorm:"uniqueIndex;not null;size:64" json:"username"`
	PasswordHash string    `gorm:"not null;size:255" json:"-"`
	Role         string    `gorm:"not null;default:'admin';size:32" json:"role"`
	CreatedAt    time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt    time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (User) TableName() string { return "users" }

// APIKey stores a managed API key for rotation
type APIKey struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"not null;size:128" json:"name"`
	KeyValue    string    `gorm:"not null;size:512" json:"-"`
	KeyPrefix   string    `gorm:"not null;size:32" json:"key_prefix"` // first 8 chars for display
	IsActive    bool      `gorm:"not null;default:true" json:"is_active"`
	Priority    int       `gorm:"not null;default:0" json:"priority"` // higher = used first
	UsageCount  int64     `gorm:"not null;default:0" json:"usage_count"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	LastError   string    `gorm:"size:512;default:''" json:"last_error"`
	CreatedAt   time.Time `gorm:"autoCreateTime" json:"created_at"`
	UpdatedAt   time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (APIKey) TableName() string { return "api_keys" }

// SystemConfig stores key-value configuration
type SystemConfig struct {
	ID        uint      `gorm:"primaryKey" json:"id"`
	Key       string    `gorm:"uniqueIndex;not null;size:128" json:"key"`
	Value     string    `gorm:"not null;size:2048" json:"value"`
	UpdatedAt time.Time `gorm:"autoUpdateTime" json:"updated_at"`
}

func (SystemConfig) TableName() string { return "system_config" }

// ProxyLog records every proxied API call
type ProxyLog struct {
	ID               uint      `gorm:"primaryKey" json:"id"`
	Model            string    `gorm:"index;not null;size:64" json:"model"`
	APIModel         string    `gorm:"not null;size:64" json:"api_model"` // actual model from response
	PromptTokens     int64     `gorm:"not null;default:0" json:"prompt_tokens"`
	CompletionTokens int64     `gorm:"not null;default:0" json:"completion_tokens"`
	TotalTokens      int64     `gorm:"not null;default:0" json:"total_tokens"`
	LatencyMs        int64     `gorm:"not null;default:0" json:"latency_ms"`
	StatusCode       int       `gorm:"index;not null;default:0" json:"status_code"`
	ErrorType        string    `gorm:"size:32;default:''" json:"error_type"` // "", "4xx", "5xx", "timeout"
	PromptPreview    string    `gorm:"size:512;default:''" json:"prompt_preview"`
	ResponsePreview  string    `gorm:"size:512;default:''" json:"response_preview"`
	APIKeyID         uint      `gorm:"index;default:0" json:"api_key_id"`
	RequestID        string    `gorm:"size:64;default:''" json:"request_id"`
	CreatedAt        time.Time `gorm:"index;not null" json:"created_at"`
}

func (ProxyLog) TableName() string { return "proxy_logs" }

// Log file constants
const MaxLogPreviewLen = 500 // max chars to save for prompt/response preview


// Default config keys
const (
	ConfigCollectInterval = "monitor.collect_interval" // e.g. "5m"
	ConfigRetentionDays   = "monitor.retention_days"   // e.g. "90"
	ConfigBalanceAlert    = "alert.balance_threshold"  // e.g. "5.0"
	ConfigErrorAlert      = "alert.error_enabled"      // "true"/"false"
)
