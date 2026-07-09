package handler

import (
	"net/http"
	"strconv"
	"time"

	"api-first/internal/cache"
	"api-first/internal/database"
	"api-first/internal/logger"
	"api-first/internal/models"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
)

// Handler holds the dependencies injected into all HTTP handlers.
type Handler struct {
	db *gorm.DB
}

// New creates a new Handler with the given database connection.
func New(db *gorm.DB) *Handler {
	return &Handler{db: db}
}

// RegisterRoutes mounts all API routes on the given router group.
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
	modules := r.Group("/modules")
	{
		modules.GET("", h.ListModules)
		modules.GET("/:id", h.GetModule)
		modules.POST("", h.CreateModule)
		modules.PUT("/:id", h.UpdateModule)
		modules.DELETE("/:id", h.DeleteModule)
	}

	deployments := r.Group("/deployments")
	{
		deployments.GET("", h.ListDeployments)
		deployments.POST("", h.CreateDeployment)
		deployments.PUT("/:id/status", h.UpdateDeploymentStatus)
	}
}

// --- Health Check Endpoints (standalone, not part of Handler) ---

// LivenessCheck returns 200 to indicate the process is alive (/healthz).
func LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "ok",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// ReadinessCheck verifies DB and Redis connectivity before reporting ready (/readyz).
func ReadinessCheck(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		log := logger.Logger()
		status := "ok"
		dbStatus := "ok"
		redisStatus := "ok"

		// Check database
		if err := database.HealthCheck(db); err != nil {
			dbStatus = "error"
			status = "not_ready"
			log.WithError(err).Error("readiness check: database unhealthy")
		}

		// Check redis
		rc := cache.Get()
		if rc == nil || rc.Ping(c.Request.Context()).Err() != nil {
			redisStatus = "error"
			status = "not_ready"
			log.Error("readiness check: redis unhealthy")
		}

		if status != "ok" {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    status,
				"database":  dbStatus,
				"redis":   redisStatus,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"status":    status,
			"database":  dbStatus,
			"redis":   redisStatus,
			"timestamp": time.Now().UTC().Format(time.RFC3339),
		})
	}
}

// --- Module Handlers ---

func (h *Handler) ListModules(c *gin.Context) {
	log := logger.Logger()
	var modules []models.Module

	if err := h.db.Find(&modules).Error; err != nil {
		log.WithError(err).Error("failed to list modules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  modules,
		"count": len(modules),
	})
}

func (h *Handler) GetModule(c *gin.Context) {
	log := logger.Logger()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}

	var module models.Module
	if err := h.db.First(&module, id).Error; err != nil {
		log.WithError(err).Warn("module not found")
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": module})
}

func (h *Handler) CreateModule(c *gin.Context) {
	log := logger.Logger()
	var module models.Module

	if err := c.ShouldBindJSON(&module); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := module.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := h.db.Create(&module).Error; err != nil {
		log.WithError(err).Error("failed to create module")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.WithField("id", module.ID).Info("module created")
	c.JSON(http.StatusCreated, gin.H{"data": module})
}

func (h *Handler) UpdateModule(c *gin.Context) {
	log := logger.Logger()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}

	var module models.Module
	if err := h.db.First(&module, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "module not found"})
		return
	}

	var updates models.Module
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if err := h.db.Model(&module).Updates(updates).Error; err != nil {
		log.WithError(err).Error("failed to update module")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.WithField("id", module.ID).Info("module updated")
	c.JSON(http.StatusOK, gin.H{"data": module})
}

func (h *Handler) DeleteModule(c *gin.Context) {
	log := logger.Logger()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid module id"})
		return
	}

	if err := h.db.Delete(&models.Module{}, id).Error; err != nil {
		log.WithError(err).Error("failed to delete module")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.WithField("id", id).Info("module deleted")
	c.JSON(http.StatusOK, gin.H{"message": "module deleted"})
}

// --- Deployment Handlers ---

func (h *Handler) ListDeployments(c *gin.Context) {
	log := logger.Logger()
	var deployments []models.Deployment

	if err := h.db.Order("created_at DESC").Find(&deployments).Error; err != nil {
		log.WithError(err).Error("failed to list deployments")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  deployments,
		"count": len(deployments),
	})
}

func (h *Handler) CreateDeployment(c *gin.Context) {
	log := logger.Logger()
	var deployment models.Deployment

	if err := c.ShouldBindJSON(&deployment); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}

	if deployment.Environment == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "environment is required"})
		return
	}

	if err := h.db.Create(&deployment).Error; err != nil {
		log.WithError(err).Error("failed to create deployment")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.WithFields(gin.H{
		"id":        deployment.ID,
		"module_id": deployment.ModuleID,
		"env":       deployment.Environment,
	}).Info("deployment created")

	c.JSON(http.StatusCreated, gin.H{"data": deployment})
}

func (h *Handler) UpdateDeploymentStatus(c *gin.Context) {
	log := logger.Logger()
	id, err := strconv.ParseUint(c.Param("id"), 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid deployment id"})
		return
	}

	var body struct {
		Status string `json:"status" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "status is required"})
		return
	}

	var deployment models.Deployment
	if err := h.db.First(&deployment, id).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "deployment not found"})
		return
	}

	now := time.Now()
	updates := map[string]interface{}{"status": body.Status}

	switch body.Status {
	case "running":
		updates["started_at"] = &now
	case "completed", "failed":
		updates["completed_at"] = &now
	}

	if err := h.db.Model(&deployment).Updates(updates).Error; err != nil {
		log.WithError(err).Error("failed to update deployment status")
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal server error"})
		return
	}

	log.WithFields(gin.H{
		"id":     id,
		"status": body.Status,
	}).Info("deployment status updated")

	c.JSON(http.StatusOK, gin.H{"data": deployment})
}