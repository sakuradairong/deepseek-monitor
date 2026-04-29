package api

import (
	"net/http"

	"deepseek-monitor/database"
	"deepseek-monitor/models"

	"github.com/gin-gonic/gin"
)

type SettingsHandler struct{}

func NewSettingsHandler() *SettingsHandler {
	return &SettingsHandler{}
}

// GetSettings returns all system settings
func (h *SettingsHandler) GetSettings(c *gin.Context) {
	configs, err := database.GetAllConfig()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build a map for easy frontend consumption
	settings := make(map[string]string)
	for _, cfg := range configs {
		settings[cfg.Key] = cfg.Value
	}

	// Ensure defaults are present
	defaults := map[string]string{
		models.ConfigCollectInterval: "5m",
		models.ConfigRetentionDays:   "90",
		models.ConfigBalanceAlert:    "5.0",
		models.ConfigErrorAlert:      "true",
	}
	for k, v := range defaults {
		if _, exists := settings[k]; !exists {
			settings[k] = v
		}
	}

	c.JSON(http.StatusOK, settings)
}

// UpdateSettings updates one or more settings
func (h *SettingsHandler) UpdateSettings(c *gin.Context) {
	var updates map[string]string
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	validKeys := map[string]bool{
		models.ConfigCollectInterval: true,
		models.ConfigRetentionDays:   true,
		models.ConfigBalanceAlert:    true,
		models.ConfigErrorAlert:      true,
	}

	for k, v := range updates {
		if !validKeys[k] {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid config key: " + k})
			return
		}
		if err := database.SetConfig(k, v); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to set " + k + ": " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"status": "saved"})
}
