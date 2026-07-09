package config

import (
	"os"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration loaded entirely from environment variables.
// No hardcoded database strings, API keys, or certificates per enterprise policy.
type Config struct {
	Server   ServerConfig
	Database DatabaseConfig
	Redis  RedisConfig
	Log    LogConfig
}

type ServerConfig struct {
	Port         string
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ShutdownTimeout time.Duration
}

type DatabaseConfig struct {
	Host     string
	Port     string
	User     string
	Password   string
	DBName   string
	SSLMode  string
	MaxConns int32
	MinConns int32
	MaxLifetime time.Duration
}

// DSN returns the PostgreSQL connection string built dynamically from config.
func (d DatabaseConfig) DSN() string {
	return "host=" + d.Host +
		" port=" + d.Port +
		" user=" + d.User +
		" password=" + d.Password +
		" dbname=" + d.DBName +
		" sslmode=" + d.SSLMode
}

type RedisConfig struct {
	Addr      string
	Password   string
	DB        int
	MinIdleConns int
	MaxIdleConns int
	MaxIdleTime  time.Duration
}

type LogConfig struct {
	Level string // panic, fatal, error, warn, info, debug
	Format string // json, text
}

// Load reads configuration from environment variables with sensible defaults for air-gapped deployment.
func Load() *Config {
	viper.AutomaticEnv()

	// Server defaults
	viper.SetDefault("SERVER_PORT", "8080")
	viper.SetDefault("SERVER_READ_TIMEOUT", "5s")
	viper.SetDefault("SERVER_WRITE_TIMEOUT", "10s")
	viper.SetDefault("SERVER_SHUTDOWN_TIMEOUT", "5s")

	// Database defaults (enterprise PostgreSQL)
	viper.SetDefault("DATABASE_HOST", "localhost")
	viper.SetDefault("DATABASE_PORT", "5432")
	viper.SetDefault("DATABASE_USER", "postgres")
	viper.SetDefault("DATABASE_PASSWORD", "postgres")
	viper.SetDefault("DATABASE_NAME", "apifirst")
	viper.SetDefault("DATABASE_SSLMODE", "disable")
	viper.SetDefault("DATABASE_MAX_CONNS", 25)
	viper.SetDefault("DATABASE_MIN_CONNS", 2)
	viper.SetDefault("DATABASE_MAX_LIFETIME", "3600s")

	// Redis defaults
	viper.SetDefault("REDIS_ADDR", "localhost:6379")
	viper.SetDefault("REDIS_PASSWORD", "")
	viper.SetDefault("REDIS_DB", 0)
	viper.SetDefault("REDIS_MIN_IDLE_CONNS", 2)
	viper.SetDefault("REDIS_MAX_IDLE_CONNS", 10)
	viper.SetDefault("REDIS_MAX_IDLE_TIME", "300s")

	// Log defaults
	viper.SetDefault("LOG_LEVEL", "info")
	viper.SetDefault("LOG_FORMAT", "json")

	return &Config{
		Server: ServerConfig{
			Port:         viper.GetString("SERVER_PORT"),
			ReadTimeout:    viper.GetDuration("SERVER_READ_TIMEOUT"),
			WriteTimeout:   viper.GetDuration("SERVER_WRITE_TIMEOUT"),
			ShutdownTimeout: viper.GetDuration("SERVER_SHUTDOWN_TIMEOUT"),
		},
		Database: DatabaseConfig{
			Host:      viper.GetString("DATABASE_HOST"),
			Port:      viper.GetString("DATABASE_PORT"),
			User:      viper.GetString("DATABASE_USER"),
			Password:  viper.GetString("DATABASE_PASSWORD"),
			DBName:    viper.GetString("DATABASE_NAME"),
			SSLMode:   viper.GetString("DATABASE_SSLMODE"),
			MaxConns:  viper.GetInt32("DATABASE_MAX_CONNS"),
			MinConns:  viper.GetInt32("DATABASE_MIN_CONNS"),
			MaxLifetime: viper.GetDuration("DATABASE_MAX_LIFETIME"),
		},
		Redis: RedisConfig{
			Addr:       viper.GetString("REDIS_ADDR"),
			Password:   viper.GetString("REDIS_PASSWORD"),
			DB:         viper.GetInt("REDIS_DB"),
			MinIdleConns: viper.GetInt("REDIS_MIN_IDLE_CONNS"),
			MaxIdleConns: viper.GetInt("REDIS_MAX_IDLE_CONNS"),
			MaxIdleTime:  viper.GetDuration("REDIS_MAX_IDLE_TIME"),
		},
		Log: LogConfig{
			Level:  viper.GetString("LOG_LEVEL"),
			Format: viper.GetString("LOG_FORMAT"),
		},
	}
}

// EnvOr returns the environment variable value or the provided default.
func EnvOr(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}