package controllers

import (
	"context"
	"log"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Longest wait for the database before reporting it down
const healthTimeout = 2 * time.Second

type HealthController struct {
	DB *gorm.DB
}

// healthz: 200 when the database answers, 503 otherwise
func (hc *HealthController) HealthzGet(c *gin.Context) {
	sqlDB, err := hc.DB.DB()
	if err == nil {
		ctx, cancel := context.WithTimeout(c.Request.Context(), healthTimeout)
		defer cancel()
		err = sqlDB.PingContext(ctx)
	}
	if err != nil {
		log.Printf("WARN health check failed: %s", err)
		c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}
