package database

import (
	"fmt"
	"time"

	"deepseek-monitor/config"
	"deepseek-monitor/models"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Init(cfg *config.DatabaseConfig) error {
	var dialector gorm.Dialector

	switch cfg.Driver {
	case "sqlite":
		dialector = sqlite.Open(cfg.DSN)
	case "postgres":
		// For PostgreSQL migration, use:
		// dialector = postgres.Open(cfg.DSN)
		return fmt.Errorf("postgres driver not yet configured, use sqlite first")
	default:
		return fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}

	db, err := gorm.Open(dialector, &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn),
	})
	if err != nil {
		return fmt.Errorf("connect database: %w", err)
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get underlying sql.DB: %w", err)
	}

	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)

	DB = db

	return autoMigrate()
}

func autoMigrate() error {
	return DB.AutoMigrate(
		&models.BalanceSnapshot{},
		&models.UsageRecord{},
		&models.RateLimitRecord{},
		&models.DailyUsageSummary{},
		&models.APIErrorRecord{},
		&models.User{},
		&models.APIKey{},
		&models.SystemConfig{},
		&models.ProxyLog{},
	)
}

func Close() error {
	if DB != nil {
		sqlDB, err := DB.DB()
		if err != nil {
			return err
		}
		return sqlDB.Close()
	}
	return nil
}

// HealthCheck returns nil if database is reachable
func HealthCheck() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
