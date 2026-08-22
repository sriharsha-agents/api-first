package database

import (
	"api-first/internal/config"
	"api-first/internal/logger"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"
)

// Connect establishes a PostgreSQL connection pool with enterprise-grade settings.
// Data never leaves the air-gapped boundary.
func Connect(cfg *config.DatabaseConfig) (*gorm.DB, error) {
	log := logger.Logger()

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{
		Logger: gormlogger.Default.LogMode(gormlogger.Silent),
	})
	if err != nil {
		log.WithError(err).Error("failed to connect to database")
		return nil, err
	}

	// Configure connection pool
	sqlDB, err := db.DB()
	if err != nil {
		log.WithError(err).Error("failed to get underlying sql.DB")
		return nil, err
	}

	sqlDB.SetMaxOpenConns(int(cfg.MaxConns))
	sqlDB.SetMaxIdleConns(int(cfg.MinConns))
	sqlDB.SetConnMaxLifetime(cfg.MaxLifetime)

	// Verify connectivity
	if err = sqlDB.Ping(); err != nil {
		log.WithError(err).Error("database ping failed")
		return nil, err
	}

	log.Info("successfully connected to PostgreSQL database")
	return db, nil
}

// AutoMigrate runs auto-migration for the provided models.
func AutoMigrate(db *gorm.DB, models ...interface{}) error {
	log := logger.Logger()

	for _, model := range models {
		if err := db.AutoMigrate(model); err != nil {
			log.WithError(err).Error("auto-migration failed")
			return err
		}
	}

	log.Info("database auto-migration completed")
	return nil
}

// HealthCheck verifies database connectivity for /healthz and /readyz endpoints.
func HealthCheck(db *gorm.DB) error {
	if db == nil {
		return nil
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Ping()
}
