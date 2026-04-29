package api

import (
	"log"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"deepseek-monitor/database"
	"deepseek-monitor/models"
)

// MetricsTracker provides real-time QPS, latency, error rate, and cost tracking
type MetricsTracker struct {
	mu sync.RWMutex

	// Sliding window (last 60 seconds)
	windowDur time.Duration

	// Raw data points
	requests []metricPoint
}

type metricPoint struct {
	time      time.Time
	latencyMs int64
	status    int
	tokens    int64
	cost      float64
}

// AggregatedMetrics is the summary exposed via API
type AggregatedMetrics struct {
	QPS              float64   `json:"qps"`
	AvgLatencyMs     float64   `json:"avg_latency_ms"`
	P95LatencyMs     float64   `json:"p95_latency_ms"`
	P99LatencyMs     float64   `json:"p99_latency_ms"`
	ErrorRate        float64   `json:"error_rate"`
	TotalRequests    int64     `json:"total_requests"`
	ErrorCount       int64     `json:"error_count"`
	TotalTokens      int64     `json:"total_tokens"`
	TotalCost        float64   `json:"total_cost"`
	TokensPerSec     float64   `json:"tokens_per_sec"`
	SuccessCount     int64     `json:"success_count"`

	// Time-series (last 60 data points, one per second)
	QPSHistory       []TimePoint `json:"qps_history"`
	LatencyHistory   []TimePoint `json:"latency_history"`
	ErrorRateHistory []TimePoint `json:"error_rate_history"`
	TokenHistory     []TimePoint `json:"token_history"`

	WindowSeconds int `json:"window_seconds"`
}

type TimePoint struct {
	Time  string  `json:"time"`
	Value float64 `json:"value"`
}

func NewMetricsTracker() *MetricsTracker {
	return &MetricsTracker{
		windowDur: 60 * time.Second,
		requests:  make([]metricPoint, 0, 10000),
	}
}

func (m *MetricsTracker) Record(latencyMs int64, status int, tokens int64, cost float64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	m.requests = append(m.requests, metricPoint{
		time:      now,
		latencyMs: latencyMs,
		status:    status,
		tokens:    tokens,
		cost:      cost,
	})

	// Trim old data
	cutoff := now.Add(-m.windowDur)
	keepIdx := 0
	for i, r := range m.requests {
		if r.time.After(cutoff) {
			keepIdx = i
			break
		}
		if i == len(m.requests)-1 {
			keepIdx = len(m.requests)
		}
	}
	if keepIdx > 0 && keepIdx < len(m.requests) {
		m.requests = m.requests[keepIdx:]
	} else if keepIdx >= len(m.requests) {
		m.requests = m.requests[:0]
	}
}

func (m *MetricsTracker) GetMetrics() *AggregatedMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now()
	cutoff := now.Add(-m.windowDur)

	// Filter to current window
	var windowPoints []metricPoint
	for _, r := range m.requests {
		if r.time.After(cutoff) {
			windowPoints = append(windowPoints, r)
		}
	}

	result := &AggregatedMetrics{
		WindowSeconds: int(m.windowDur.Seconds()),
	}

	n := len(windowPoints)
	if n == 0 {
		return result
	}

	result.TotalRequests = int64(n)

	var totalLatency int64
	totalTokens := int64(0)
	totalCost := 0.0
	errorCount := int64(0)
	latencies := make([]int64, 0, n)

	for _, p := range windowPoints {
		totalLatency += p.latencyMs
		totalTokens += p.tokens
		totalCost += p.cost
		latencies = append(latencies, p.latencyMs)

		if p.status >= 400 {
			errorCount++
		}
	}

	result.TotalTokens = totalTokens
	result.TotalCost = math.Round(totalCost*1000000) / 1000000
	result.ErrorCount = errorCount
	result.SuccessCount = int64(n) - errorCount

	// Averages
	windowSecs := m.windowDur.Seconds()
	result.QPS = math.Round(float64(n)/windowSecs*100) / 100
	result.AvgLatencyMs = math.Round(float64(totalLatency)/float64(n)*100) / 100
	result.ErrorRate = math.Round(float64(errorCount)/float64(n)*10000) / 100
	result.TokensPerSec = math.Round(float64(totalTokens)/windowSecs*100) / 100

	// Sort latencies for percentiles
	sortInt64s(latencies)
	if len(latencies) > 0 {
		p95Idx := int(float64(len(latencies)) * 0.95)
		p99Idx := int(float64(len(latencies)) * 0.99)
		if p95Idx >= len(latencies) {
			p95Idx = len(latencies) - 1
		}
		if p99Idx >= len(latencies) {
			p99Idx = len(latencies) - 1
		}
		result.P95LatencyMs = float64(latencies[p95Idx])
		result.P99LatencyMs = float64(latencies[p99Idx])
	}

	// Generate time-series (one point per second for last 60s)
	nowSec := time.Now().Truncate(time.Second)
	secondBuckets := make(map[int64]*secBucket)
	for _, p := range windowPoints {
		secKey := p.time.Truncate(time.Second).Unix()
		if _, ok := secondBuckets[secKey]; !ok {
			secondBuckets[secKey] = &secBucket{}
		}
		b := secondBuckets[secKey]
		b.count++
		b.latencySum += p.latencyMs
		b.tokensSum += p.tokens
		if p.status >= 400 {
			b.errors++
		}
	}

	result.QPSHistory = make([]TimePoint, 0, 60)
	result.LatencyHistory = make([]TimePoint, 0, 60)
	result.ErrorRateHistory = make([]TimePoint, 0, 60)
	result.TokenHistory = make([]TimePoint, 0, 60)

	for i := 59; i >= 0; i-- {
		secTime := nowSec.Add(-time.Duration(i) * time.Second)
		ts := secTime.Format("15:04:05")
		b, exists := secondBuckets[secTime.Unix()]

		var qps, lat, errRate, tps float64
		if exists && b.count > 0 {
			qps = float64(b.count)
			lat = float64(b.latencySum) / float64(b.count)
			errRate = float64(b.errors) / float64(b.count) * 100
			tps = float64(b.tokensSum)
		}
		result.QPSHistory = append(result.QPSHistory, TimePoint{ts, qps})
		result.LatencyHistory = append(result.LatencyHistory, TimePoint{ts, lat})
		result.ErrorRateHistory = append(result.ErrorRateHistory, TimePoint{ts, errRate})
		result.TokenHistory = append(result.TokenHistory, TimePoint{ts, tps})
	}

	return result
}

type secBucket struct {
	count      int
	latencySum int64
	tokensSum  int64
	errors     int
}

func sortInt64s(a []int64) {
	if len(a) <= 1 {
		return
	}
	// Simple insertion sort for small slices
	for i := 1; i < len(a); i++ {
		for j := i; j > 0 && a[j] < a[j-1]; j-- {
			a[j], a[j-1] = a[j-1], a[j]
		}
	}
}

// Global metrics tracker
var GlobalMetrics = NewMetricsTracker()

// ProxyLogRepository provides DB operations for proxy logs
type ProxyLogRepo struct{}

func NewProxyLogRepo() *ProxyLogRepo {
	return &ProxyLogRepo{}
}

func (r *ProxyLogRepo) Save(log *models.ProxyLog) error {
	return database.DB.Create(log).Error
}

func (r *ProxyLogRepo) Query(offset, limit int, model, errorType string, minStatus int) ([]models.ProxyLog, int64, error) {
	var logs []models.ProxyLog
	query := database.DB.Model(&models.ProxyLog{})

	if model != "" {
		query = query.Where("model = ?", model)
	}
	if errorType != "" {
		query = query.Where("error_type = ?", errorType)
	}
	if minStatus > 0 {
		query = query.Where("status_code >= ?", minStatus)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := query.Order("created_at DESC").Offset(offset).Limit(limit).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	return logs, total, nil
}

func (r *ProxyLogRepo) GetStats(since time.Time) (*models.DailyUsageSummary, error) {
	query := database.DB.Model(&models.ProxyLog{}).
		Where("created_at >= ?", since).
		Where("status_code < 400")

	var result models.DailyUsageSummary
	err := query.Select(
		"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens",
		"COALESCE(SUM(completion_tokens), 0) as total_completion_tokens",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COUNT(*) as total_requests",
	).Scan(&result).Error
	return &result, err
}

// FileLogger writes structured API call logs to a file
type FileLogger struct {
	logger *log.Logger
}

func NewFileLogger(filePath string) (*FileLogger, error) {
	f, err := openLogFile(filePath)
	if err != nil {
		return nil, err
	}
	return &FileLogger{
		logger: log.New(f, "", 0),
	}, nil
}

func openLogFile(path string) (*os.File, error) {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	return os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0644)
}

func (l *FileLogger) LogCall(logEntry map[string]interface{}) {
	// Simple JSON logging without external dep
	l.logger.Printf(
		`{"time":"%s","type":"api_call","model":"%s","status":%v,"latency_ms":%v,"prompt_tokens":%v,"completion_tokens":%v,"total_tokens":%v,"cost":%.8f,"key":"%s","error":"%s"}`,
		logEntry["time"],
		logEntry["model"],
		logEntry["status"],
		logEntry["latency_ms"],
		logEntry["prompt_tokens"],
		logEntry["completion_tokens"],
		logEntry["total_tokens"],
		logEntry["cost"],
		logEntry["key_name"],
		logEntry["error"],
	)
}
