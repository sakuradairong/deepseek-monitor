package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"deepseek-monitor/api"
	"deepseek-monitor/config"
	"deepseek-monitor/database"
	"deepseek-monitor/models"
	"deepseek-monitor/scheduler"
)

func main() {
	log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

	// Load config
	configPath := "config.yaml"
	if len(os.Args) > 1 {
		configPath = os.Args[1]
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// Ensure data directory exists
	if cfg.Database.Driver == "sqlite" && cfg.Database.DSN != "" {
		for i := len(cfg.Database.DSN) - 1; i >= 0; i-- {
			if cfg.Database.DSN[i] == '/' {
				if dir := cfg.Database.DSN[:i]; dir != "" {
					os.MkdirAll(dir, 0755)
				}
				break
			}
		}
	}

	// Initialize database
	if err := database.Init(&cfg.Database); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}

	// Ensure logs directory exists
	os.MkdirAll("logs", 0755)

	// Initialize default settings in database
	initDefaultSettings()

	// Determine collection interval from config (may be overridden by DB settings)
	interval := parseInterval(cfg.Monitor.CollectInterval, 5*time.Minute)
	if dbInterval := database.GetConfigWithDefault(models.ConfigCollectInterval, ""); dbInterval != "" {
		if parsed, err := time.ParseDuration(dbInterval); err == nil {
			interval = parsed
		}
	}

	retention := cfg.Monitor.RetentionDays
	if retention <= 0 || retention > 365 {
		retention = 90
	}
	if dbRetention := database.GetConfigWithDefault(models.ConfigRetentionDays, ""); dbRetention != "" {
		if _, err := fmt.Sscanf(dbRetention, "%d", &retention); err == nil {
			if retention <= 0 || retention > 365 {
				retention = 90
			}
		}
	}

	// Start scheduler (no longer needs a pre-configured API key - selects from DB)
	sched := scheduler.New(interval, retention)
	if err := sched.Start(); err != nil {
		log.Fatalf("Failed to start scheduler: %v", err)
	}

	// Setup HTTP server
	handler := api.NewHandler()
	webDist := "web/dist"
	if _, err := os.Stat(webDist); os.IsNotExist(err) {
		webDist = ""
	}
	router := api.SetupRouter(handler, webDist)

	addr := fmt.Sprintf("%s:%d", cfg.Server.Host, cfg.Server.Port)
	log.Printf("DeepSeek Monitor API starting on %s", addr)

	srv := &http.Server{
		Addr:    addr,
		Handler: router,
	}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Failed to start server: %v", err)
		}
	}()

	// Wait for shutdown signal, then gracefully shut down
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	sig := <-quit
	log.Printf("Received signal %v, shutting down...", sig)

	sched.Stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server forced to shutdown: %v", err)
	}

	database.Close()
	log.Println("Shutdown complete.")
}

func initDefaultSettings() {
	defaults := map[string]string{
		models.ConfigCollectInterval: "5m",
		models.ConfigRetentionDays:   "90",
		models.ConfigBalanceAlert:    "5.0",
		models.ConfigErrorAlert:      "true",
	}
	for k, v := range defaults {
		database.SetConfigDefault(k, v)
	}
}

func parseInterval(s string, fallback time.Duration) time.Duration {
	if d, err := time.ParseDuration(s); err == nil {
		return d
	}
	return fallback
}
