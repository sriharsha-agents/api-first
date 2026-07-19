package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"api-first/internal/cache"
	"api-first/internal/config"
	"api-first/internal/database"
	"api-first/internal/handler"
	"api-first/internal/logger"
	"api-first/internal/models"

	"github.com/gin-gonic/gin"
)

// Build-time metadata injected via ldflags during Docker build.
var (
	Version = "dev"
	BuildAt = "unknown"
)

func main() {
	log := logger.Logger()

	// Load all configuration from environment variables (zero hardcoded configs).
	cfg := config.Load()

	log.WithFields(map[string]interface{}{
		"version": Version,
		"build":   BuildAt,
		"port":    cfg.Server.Port,
	}).Info("starting API-First enterprise module")

	// Connect to PostgreSQL (local database, data never leaves air-gapped boundary).
	db, err := database.Connect(&cfg.Database)
	if err != nil {
		log.WithError(err).Fatal("failed to connect to database")
	}

	// Run auto-migrations.
	if err := database.AutoMigrate(db, &models.Module{}, &models.Deployment{}); err != nil {
		log.WithError(err).Fatal("auto-migration failed")
	}

	// Connect to Redis cache.
	if err := cache.Init(&cfg.Redis); err != nil {
		log.WithError(err).Warn("redis unavailable, continuing without cache")
	} else {
		defer cache.Close()
	}

	// Set Gin to release mode in production.
	gin.SetMode(gin.ReleaseMode)

	// Create router with structured JSON logging middleware.
	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(requestLogger())

	// Health check endpoints (required for Kubernetes orchestration).
	r.GET("/healthz", handler.LivenessCheck)
	r.GET("/readyz", handler.ReadinessCheck(db))

	// API v1 routes.
	v1 := r.Group("/api/v1")
	api := handler.New(db)
	api.RegisterRoutes(v1)

	// Create HTTP server with timeouts for graceful shutdown.
	srv := &http.Server{
		Addr:           ":" + cfg.Server.Port,
		Handler:        r,
		ReadTimeout:    cfg.Server.ReadTimeout,
		WriteTimeout:   cfg.Server.WriteTimeout,
		IdleTimeout:    30 * time.Second,
		MaxHeaderBytes: 1 << 20,
	}

	// Start server in a goroutine.
	go func() {
		log.WithField("addr", srv.Addr).Info("HTTP server listening")
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.WithError(err).Fatal("server failed to start")
		}
	}()

	// Wait for interrupt signal for graceful shutdown.
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info("shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), cfg.Server.ShutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.WithError(err).Error("server forced to shutdown")
		os.Exit(1)
	}

	log.Info("server exited cleanly")
}

// requestLogger returns a Gin middleware that logs every request in structured JSON format
// to stdout for SIEM ingestion (Splunk, etc.).
func requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path

		c.Next()

		log := logger.Logger()
		log.WithFields(map[string]interface{}{
			"method":     c.Request.Method,
			"path":       path,
			"status":     c.Writer.Status(),
			"latency":    time.Since(start).String(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		}).Info("request")
	}
}
