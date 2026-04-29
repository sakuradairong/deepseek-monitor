package scheduler

import (
	"fmt"
	"log"
	"time"

	"deepseek-monitor/collector"
	"deepseek-monitor/database"
	"deepseek-monitor/models"

	"github.com/robfig/cron/v3"
)

type Scheduler struct {
	cron      *cron.Cron
	interval  time.Duration
	retention time.Duration
}

func New(interval time.Duration, retentionDays int) *Scheduler {
	return &Scheduler{
		cron:      cron.New(cron.WithSeconds()),
		interval:  interval,
		retention: time.Duration(retentionDays) * 24 * time.Hour,
	}
}

func (s *Scheduler) Start() error {
	// Run immediately on start
	if err := s.collectAll(); err != nil {
		log.Printf("[scheduler] initial collection error: %v", err)
	}

	// Schedule periodic collection
	schedule := fmt.Sprintf("@every %s", s.interval.String())
	if _, err := s.cron.AddFunc(schedule, func() {
		if err := s.collectAll(); err != nil {
			log.Printf("[scheduler] collection error: %v", err)
		}
	}); err != nil {
		return fmt.Errorf("add cron job: %w", err)
	}

	// Schedule daily summary refresh at midnight
	if _, err := s.cron.AddFunc("0 0 0 * * *", func() {
		yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")
		if err := database.RefreshDailySummary(yesterday); err != nil {
			log.Printf("[scheduler] daily summary error: %v", err)
		}
		retentionCutoff := time.Now().Add(-s.retention)
		if err := database.DeleteOldData(retentionCutoff); err != nil {
			log.Printf("[scheduler] cleanup error: %v", err)
		}
	}); err != nil {
		return fmt.Errorf("add daily cron job: %w", err)
	}

	s.cron.Start()
	log.Printf("[scheduler] started with interval %s, retention %d days, multi-key rotation enabled",
		s.interval, int(s.retention.Hours()/24))
	return nil
}

func (s *Scheduler) Stop() {
	if s.cron != nil {
		s.cron.Stop()
	}
}

// collectAll runs one collection cycle using the next available API key
func (s *Scheduler) collectAll() error {
	now := time.Now()

	// Select the next API key to use (round-robin rotation)
	apiKey, err := database.SelectNextAPIKey()
	if err != nil {
		log.Printf("[scheduler] no active API keys available")
		return fmt.Errorf("no active API keys")
	}

	log.Printf("[scheduler] using key: %s (prefix: %s, usage: %d)",
		apiKey.Name, apiKey.KeyPrefix, apiKey.UsageCount)

	// Create collector client with the selected key
	client := collector.NewClient("https://api.deepseek.com", apiKey.KeyValue)

	// 1. Collect balance info
	balanceResp, balanceErr := client.GetBalance()
	if balanceErr != nil {
		log.Printf("[collector] balance error for key %s: %v", apiKey.Name, balanceErr)
		database.SetAPIKeyError(apiKey.ID, balanceErr.Error())
		database.SaveAPIErrorRecord(&models.APIErrorRecord{
			APIKeyID:    apiKey.ID,
			ErrorType:   "balance",
			StatusCode:  0,
			Message:     fmt.Sprintf("[%s] %s", apiKey.Name, balanceErr.Error()),
			CollectedAt: now,
		})
	} else {
		database.SetAPIKeyError(apiKey.ID, "")
		for _, bi := range balanceResp.BalanceInfos {
			totalBal := parseFloat(bi.TotalBalance)
			grantedBal := parseFloat(bi.GrantedBalance)
			toppedUpBal := parseFloat(bi.ToppedUpBalance)

			snapshot := &models.BalanceSnapshot{
				APIKeyID:       apiKey.ID,
				TotalBalance:   totalBal,
				GrantedBalance: grantedBal,
				ToppedUpBalance: toppedUpBal,
				IsAvailable:    balanceResp.IsAvailable,
				CollectedAt:    now,
			}
			if err := database.SaveBalanceSnapshot(snapshot); err != nil {
				log.Printf("[collector] save balance error: %v", err)
			}
		}
	}

	// 2. Probe API for usage and rate limit info (only if balance succeeded)
	//    This avoids using tokens if the key is invalid
	if balanceErr == nil {
		usageResp, rateLimit, probeErr := client.ProbeAPI()
		if probeErr != nil {
			log.Printf("[collector] probe error for key %s: %v", apiKey.Name, probeErr)
			database.SetAPIKeyError(apiKey.ID, probeErr.Error())
			database.SaveAPIErrorRecord(&models.APIErrorRecord{
				APIKeyID:    apiKey.ID,
				ErrorType:   "probe",
				StatusCode:  0,
				Message:     fmt.Sprintf("[%s] %s", apiKey.Name, probeErr.Error()),
				CollectedAt: now,
			})
		} else {
			// Save usage record
			cost := collector.EstimateCost(
				usageResp.Model,
				usageResp.Usage.PromptTokens,
				usageResp.Usage.CompletionTokens,
				usageResp.Usage.PromptCacheHitTokens,
				usageResp.Usage.PromptCacheMissTokens,
			)
			record := &models.UsageRecord{
				APIKeyID:              apiKey.ID,
				Model:                 usageResp.Model,
				PromptTokens:          usageResp.Usage.PromptTokens,
				CompletionTokens:      usageResp.Usage.CompletionTokens,
				TotalTokens:           usageResp.Usage.TotalTokens,
				PromptCacheHitTokens:  usageResp.Usage.PromptCacheHitTokens,
				PromptCacheMissTokens: usageResp.Usage.PromptCacheMissTokens,
				EstimatedCost:         cost,
				CollectedAt:           now,
			}
			if err := database.SaveUsageRecord(record); err != nil {
				log.Printf("[collector] save usage error: %v", err)
			}

			// Save rate limit info
			if rateLimit != nil {
				rlRecord := &models.RateLimitRecord{
					APIKeyID:         apiKey.ID,
					RequestsLimit:    rateLimit.RequestsLimit,
					RequestsRemaining: rateLimit.RequestsRemaining,
					TokensLimit:      rateLimit.TokensLimit,
					TokensRemaining:  rateLimit.TokensRemaining,
					CollectedAt:      now,
				}
				if err := database.SaveRateLimitRecord(rlRecord); err != nil {
					log.Printf("[collector] save rate limit error: %v", err)
				}
			}

			// Increment key usage count
			database.IncrementAPIKeyUsage(apiKey.ID)
		}
	}

	return nil
}

func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}
