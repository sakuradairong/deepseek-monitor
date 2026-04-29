package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Server   ServerConfig   `yaml:"server"`
	Database DatabaseConfig `yaml:"database"`
	DeepSeek DeepSeekConfig `yaml:"deepseek"`
	Monitor  MonitorConfig  `yaml:"monitor"`
	Log      LogConfig      `yaml:"log"`
}

type ServerConfig struct {
	Port    int    `yaml:"port"`
	Host    string `yaml:"host"`
	BaseURL string `yaml:"base_url"`
}

type DatabaseConfig struct {
	Driver string `yaml:"driver"` // sqlite or postgres
	DSN    string `yaml:"dsn"`    // file:data.db or host=localhost user=... dbname=...
}

type DeepSeekConfig struct {
	APIKey  string `yaml:"api_key"`
	BaseURL string `yaml:"base_url"`
}

type MonitorConfig struct {
	CollectInterval string `yaml:"collect_interval"` // e.g., "5m"
	RetentionDays   int    `yaml:"retention_days"`
}

type LogConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Port:    8080,
			Host:    "0.0.0.0",
			BaseURL: "http://localhost:8080",
		},
		Database: DatabaseConfig{
			Driver: "sqlite",
			DSN:    "data/deepseek_monitor.db",
		},
		DeepSeek: DeepSeekConfig{
			BaseURL: "https://api.deepseek.com",
			APIKey:  os.Getenv("DEEPSEEK_API_KEY"),
		},
		Monitor: MonitorConfig{
			CollectInterval: "5m",
			RetentionDays:   90,
		},
		Log: LogConfig{
			Level:  "info",
			Format: "text",
		},
	}
}

func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return nil, fmt.Errorf("read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parse config file: %w", err)
	}

	// Override API key from env if set
	if envKey := os.Getenv("DEEPSEEK_API_KEY"); envKey != "" {
		cfg.DeepSeek.APIKey = envKey
	}

	return cfg, nil
}
