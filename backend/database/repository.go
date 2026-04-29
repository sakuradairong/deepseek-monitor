package database

import (
	"time"

	"deepseek-monitor/models"

	"gorm.io/gorm"
)

// --- Balance Snapshot ---

func SaveBalanceSnapshot(snap *models.BalanceSnapshot) error {
	return DB.Create(snap).Error
}

func GetLatestBalance() (*models.BalanceSnapshot, error) {
	var snap models.BalanceSnapshot
	err := DB.Order("collected_at DESC").First(&snap).Error
	if err != nil {
		return nil, err
	}
	return &snap, nil
}

func GetBalanceHistory(since time.Time, limit int) ([]models.BalanceSnapshot, error) {
	var snaps []models.BalanceSnapshot
	err := DB.Where("collected_at >= ?", since).
		Order("collected_at ASC").
		Limit(limit).
		Find(&snaps).Error
	return snaps, err
}

// --- Usage Record ---

func SaveUsageRecord(record *models.UsageRecord) error {
	return DB.Create(record).Error
}

func SaveUsageRecords(records []models.UsageRecord) error {
	if len(records) == 0 {
		return nil
	}
	return DB.Create(&records).Error
}

func GetUsageSummary(since time.Time, model string) (*models.DailyUsageSummary, error) {
	query := DB.Model(&models.UsageRecord{})
	if model != "" {
		query = query.Where("model = ?", model)
	}
	var result models.DailyUsageSummary
	err := query.Select(
		"COALESCE(SUM(prompt_tokens), 0) as total_prompt_tokens",
		"COALESCE(SUM(completion_tokens), 0) as total_completion_tokens",
		"COALESCE(SUM(total_tokens), 0) as total_tokens",
		"COUNT(*) as total_requests",
		"COALESCE(SUM(estimated_cost), 0) as estimated_cost",
	).Where("collected_at >= ?", since).
		Scan(&result).Error
	return &result, err
}

func GetUsageTrend(since time.Time, interval string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	// Group by date for SQLite
	query := DB.Model(&models.UsageRecord{}).
		Select("DATE(collected_at) as date, model, SUM(prompt_tokens) as prompt_tokens, SUM(completion_tokens) as completion_tokens, SUM(total_tokens) as total_tokens, COUNT(*) as requests, SUM(estimated_cost) as cost").
		Where("collected_at >= ?", since).
		Group("DATE(collected_at), model").
		Order("date ASC").
		Find(&results)
	return results, query.Error
}

func GetModelUsageDistribution(since time.Time) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	query := DB.Model(&models.UsageRecord{}).
		Select("model, SUM(total_tokens) as total_tokens, COUNT(*) as requests, SUM(estimated_cost) as cost").
		Where("collected_at >= ?", since).
		Group("model").
		Order("total_tokens DESC").
		Find(&results)
	return results, query.Error
}

// --- Rate Limit Record ---

func SaveRateLimitRecord(record *models.RateLimitRecord) error {
	return DB.Create(record).Error
}

func GetLatestRateLimit() (*models.RateLimitRecord, error) {
	var record models.RateLimitRecord
	err := DB.Order("collected_at DESC").First(&record).Error
	if err != nil {
		return nil, err
	}
	return &record, nil
}

func GetRateLimitHistory(since time.Time, limit int) ([]models.RateLimitRecord, error) {
	var records []models.RateLimitRecord
	err := DB.Where("collected_at >= ?", since).
		Order("collected_at ASC").
		Limit(limit).
		Find(&records).Error
	return records, err
}

// --- Daily Usage Summary ---

func UpsertDailySummary(summary *models.DailyUsageSummary) error {
	// Use OnConflict for PostgreSQL, for SQLite use a different approach
	var existing models.DailyUsageSummary
	result := DB.Where("date = ? AND model = ?", summary.Date, summary.Model).
		First(&existing)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			return DB.Create(summary).Error
		}
		return result.Error
	}

	// Update existing
	existing.TotalPromptTokens = summary.TotalPromptTokens
	existing.TotalCompletionTokens = summary.TotalCompletionTokens
	existing.TotalTokens = summary.TotalTokens
	existing.TotalRequests = summary.TotalRequests
	existing.EstimatedCost = summary.EstimatedCost
	return DB.Save(&existing).Error
}

func GetDailySummaries(since time.Time, limit int) ([]models.DailyUsageSummary, error) {
	var summaries []models.DailyUsageSummary
	err := DB.Where("date >= ?", since.Format("2006-01-02")).
		Order("date DESC").
		Limit(limit).
		Find(&summaries).Error
	return summaries, err
}

func RefreshDailySummary(date string) error {
	// Aggregate from usage_records for a specific date
	var records []models.UsageRecord
	startDate, _ := time.Parse("2006-01-02", date)
	endDate := startDate.Add(24 * time.Hour)

	if err := DB.Where("collected_at >= ? AND collected_at < ?", startDate, endDate).
		Find(&records).Error; err != nil {
		return err
	}

	// Group by model
	modelSummary := make(map[string]*models.DailyUsageSummary)
	for _, r := range records {
		s, ok := modelSummary[r.Model]
		if !ok {
			s = &models.DailyUsageSummary{
				Date:  date,
				Model: r.Model,
			}
			modelSummary[r.Model] = s
		}
		s.TotalPromptTokens += r.PromptTokens
		s.TotalCompletionTokens += r.CompletionTokens
		s.TotalTokens += r.TotalTokens
		s.TotalRequests++
		s.EstimatedCost += r.EstimatedCost
	}

	for _, s := range modelSummary {
		if err := UpsertDailySummary(s); err != nil {
			return err
		}
	}

	return nil
}

// --- API Error Record ---

func SaveAPIErrorRecord(record *models.APIErrorRecord) error {
	return DB.Create(record).Error
}

func GetRecentErrors(limit int) ([]models.APIErrorRecord, error) {
	var records []models.APIErrorRecord
	err := DB.Order("collected_at DESC").
		Limit(limit).
		Find(&records).Error
	return records, err
}

// --- Cleanup ---

func DeleteOldData(before time.Time) error {
	return DB.Where("collected_at < ?", before).
		Delete(&models.UsageRecord{}).Error
}

func DeleteOldBalanceSnapshots(before time.Time) error {
	return DB.Where("collected_at < ?", before).
		Delete(&models.BalanceSnapshot{}).Error
}

// ==================== USER ====================

func CreateUser(user *models.User) error {
	return DB.Create(user).Error
}

func FindUserByUsername(username string) (*models.User, error) {
	var user models.User
	err := DB.Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func GetUserByID(id uint) (*models.User, error) {
	var user models.User
	err := DB.First(&user, id).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

func CountUsers() (int64, error) {
	var count int64
	err := DB.Model(&models.User{}).Count(&count).Error
	return count, err
}

// ==================== API KEY ====================

func CreateAPIKey(key *models.APIKey) error {
	return DB.Create(key).Error
}

func GetAPIKeyByID(id uint) (*models.APIKey, error) {
	var key models.APIKey
	err := DB.First(&key, id).Error
	return &key, err
}

func ListAPIKeys() ([]models.APIKey, error) {
	var keys []models.APIKey
	err := DB.Order("priority DESC, created_at ASC").Find(&keys).Error
	return keys, err
}

func ListActiveAPIKeys() ([]models.APIKey, error) {
	var keys []models.APIKey
	err := DB.Where("is_active = ?", true).
		Order("priority DESC, usage_count ASC, created_at ASC").
		Find(&keys).Error
	return keys, err
}

func UpdateAPIKey(key *models.APIKey) error {
	return DB.Save(key).Error
}

func DeleteAPIKey(id uint) error {
	return DB.Delete(&models.APIKey{}, id).Error
}

func IncrementAPIKeyUsage(id uint) error {
	return DB.Model(&models.APIKey{}).Where("id = ?", id).
		Updates(map[string]interface{}{
			"usage_count":  gorm.Expr("usage_count + 1"),
			"last_used_at": time.Now(),
		}).Error
}

func SetAPIKeyError(id uint, errMsg string) {
	DB.Model(&models.APIKey{}).Where("id = ?", id).
		Update("last_error", errMsg)
}

// SelectNextAPIKey returns the next API key to use (round-robin by usage_count ASC)
func SelectNextAPIKey() (*models.APIKey, error) {
	var key models.APIKey
	err := DB.Where("is_active = ?", true).
		Order("priority DESC, usage_count ASC, last_used_at ASC NULLS FIRST").
		First(&key).Error
	if err != nil {
		return nil, err
	}
	return &key, nil
}

// ==================== SYSTEM CONFIG ====================

func GetConfig(key string) (string, error) {
	var cfg models.SystemConfig
	err := DB.Where("key = ?", key).First(&cfg).Error
	if err != nil {
		return "", err
	}
	return cfg.Value, nil
}

func SetConfig(key, value string) error {
	var cfg models.SystemConfig
	result := DB.Where("key = ?", key).First(&cfg)
	if result.Error != nil {
		// Create new
		return DB.Create(&models.SystemConfig{
			Key:   key,
			Value: value,
		}).Error
	}
	// Update existing
	return DB.Model(&cfg).Update("value", value).Error
}

func GetAllConfig() ([]models.SystemConfig, error) {
	var configs []models.SystemConfig
	err := DB.Find(&configs).Error
	return configs, err
}

func GetConfigWithDefault(key, defaultVal string) string {
	val, err := GetConfig(key)
	if err != nil {
		return defaultVal
	}
	return val
}

