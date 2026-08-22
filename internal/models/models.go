package models

import (
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Module represents a deployable enterprise module tracked in the system.
// Stateless design: persistent state lives exclusively in the database container.
type Module struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"uniqueIndex:notNull;size:255"`
	Description string         `json:"description" gorm:"type:text"`
	Version       string         `json:"version" gorm:"size:50"`
	Status      string         `json:"status" gorm:"size:20;default:active"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`
}

// TableName returns the table name for the Module model.
func (Module) TableName() string {
	return "modules"
}

// Validate checks required fields before creation.
func (m *Module) Validate() error {
	if m.Name == "" {
		return fmt.Errorf("name is required")
	}
	if m.Version == "" {
		return fmt.Errorf("version is required")
	}
	return nil
}

// Deployment represents an air-gapped deployment record.
type Deployment struct {
	ID         uint       `json:"id" gorm:"primaryKey"`
	ModuleID    uint       `json:"module_id" gorm:"index"`
	Environment  string     `json:"environment" gorm:"size:50;index"`
	Region      string     `json:"region" gorm:"size:100"`
	Status      string     `json:"status" gorm:"size:20;default:pending"`
	StartedAt   *time.Time `json:"started_at"`
	CompletedAt *time.Time `json:"completed_at"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
}

// TableName returns the table name for the Deployment model.
func (Deployment) TableName() string {
	return "deployments"
}

// HealthStatus holds the aggregated health status for the /healthz endpoint.
type HealthStatus struct {
	Status    string `json:"status"`
	Timestamp string `json:"timestamp"`
	Database  string `json:"database"`
	Redis   string `json:"redis"`
	Version   string `json:"version"`
}